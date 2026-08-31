// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Self-intersection detection (M07 PBI-084, Oblikovati/Oblikovati#300): pairs of faces of one body
// that pass through each other. Topologically adjacent faces (sharing an edge or vertex) legitimately
// touch along that boundary and are excluded; what remains crossing is real interpenetration — the
// classic outcome of a bad import or an over-folded offset.
//
// ★ IT READS NO TESSELLATION (M48/C3, Oblikovati/Oblikovati#3477). Until this slice the detector
// tessellated every face, scanned triangle pairs with a Möller test, and then SUBTRACTED a per-face
// faceting allowance (faceMeshDeviation) so tangent blends stopped reporting crossings that only the
// facets had. That is a validity verdict taken on facets and then corrected with a facet-error budget:
// the same body could be valid at one Quality and invalid at another. The decision now runs on the
// exact B-rep, exactly as OCCT's IntTools_FaceFace decides face interference:
//
//   - two faces trimmed out of the SAME surface sheet interpenetrate where their TRIMS overlap, which
//     no surface-surface intersection can see (the two surfaces have no isolated intersection curve).
//     That arm lives in self_intersect_coincident.go.
//   - otherwise they interpenetrate where their surfaces' exact intersection curve
//     (geom.SurfaceIntersect) runs INSIDE both trims (brep.PointInFaceTrim) and away from the two
//     faces' shared boundary — the same shape kernel/ops/boundary_cross.go uses for containment.
//
// The whole triangle narrow phase (the Möller straddle test, its coplanar SAT/Sutherland-Hodgman
// branch, the triangle BVH and the faceting allowance) is DELETED with this change rather than kept
// as a fallback: a generalization is only complete when the special cases it replaces are gone.

// SelfIntersection is one interpenetrating, non-adjacent face pair, with a witness point on the
// crossing.
type SelfIntersection struct {
	FaceA, FaceB *topo.Face
	Witness      math.Point3
}

// SelfIntersections reports every face pair of b whose TRIMMED surfaces genuinely pass through each
// other, decided on the exact B-rep — no tessellation, and so no dependence on facet density. The
// Quality argument is kept only so the many call sites that thread one through do not churn; it is
// deliberately unused (the same reason [FillInternalVoids] ignores its own).
//
// Faces that merely touch along shared topology (an edge or a vertex they have in common) are contact,
// not interpenetration, and are excluded. A face pair whose surfaces the analytic intersector cannot
// resolve is DECLINED rather than reported — see [declineUnresolvedSurfacePair].
//
// Example: if hits := ops.SelfIntersections(body, ops.DefaultQuality()); len(hits) > 0 { reject(body) }
func SelfIntersections(b *topo.Body, _ Quality) []SelfIntersection {
	res := geom.ResolutionForBox(b.RangeBox())
	scan := newFaceScan(b.Faces())
	var out []SelfIntersection
	for i := range scan.faces {
		out = append(out, scan.crossingsWith(i, res)...)
	}
	return out
}

// crossingsWith reports face i's interpenetrations with the later faces its range box reaches. The
// candidates come from the shared box tree rather than an all-pairs walk: a body dense enough to need
// this check at all is dense in FACES (a fine-pitch coil ships 18434 of them, #2080), and 170 million
// box tests cost more than the exact geometry that follows them.
func (s *faceScan) crossingsWith(i int, res geom.Resolution) []SelfIntersection {
	var out []SelfIntersection
	s.tree.Query(s.boxes[i], func(j int) bool {
		if j <= i {
			return false // each pair is tested once, from its lower-indexed face
		}
		if p, hit := s.interpenetrate(i, j, res); hit {
			out = append(out, SelfIntersection{FaceA: s.faces[i], FaceB: s.faces[j], Witness: p})
		}
		return false
	})
	return out
}

// faceScan holds the per-face data a pairwise scan would otherwise recompute for every pair: each
// face's range box, and (lazily) its trim probes.
//
// It is not a micro-optimization. topo.Face.RangeBox walks the face's edges and evaluates their
// curves, so recomputing it inside the pair loop is O(F²·E) — measured, that alone took a fine-pitch
// coil's turn-clearance check (#2080, a body of 18434 swept faces) past twenty minutes. The boxes then
// index a geom.BoxTree, the same broad phase the deleted triangle scan used one level down.
type faceScan struct {
	faces  []*topo.Face
	boxes  []math.Box
	tree   *geom.BoxTree   // broad phase over boxes, so the scan is not all-pairs
	probes [][]math.Point3 // nil per face until a same-sheet test needs it
}

// newFaceScan captures the faces, their range boxes and the broad-phase index over those boxes.
func newFaceScan(faces []*topo.Face) *faceScan {
	s := &faceScan{faces: faces, boxes: make([]math.Box, len(faces)), probes: make([][]math.Point3, len(faces))}
	for i, f := range faces {
		s.boxes[i] = f.RangeBox()
	}
	s.tree = geom.NewBoxTree(s.boxes)
	return s
}

// trimProbes returns face i's boundary probes, computing them the first time they are asked for.
func (s *faceScan) trimProbes(i int) []math.Point3 {
	if s.probes[i] == nil {
		s.probes[i] = faceTrimProbes(s.faces[i])
	}
	return s.probes[i]
}

// interpenetrate returns a witness point where faces i and j pass through each other, on the exact
// B-rep. Their range boxes already overlap (the caller's broad phase); a pair trimmed out of one
// surface sheet takes the trim-overlap arm; every other pair takes the surface-surface intersection arm. The shared topology
// is collected only once a crossing curve exists to test against it, because that walk is O(E) too.
func (s *faceScan) interpenetrate(i, j int, res geom.Resolution) (math.Point3, bool) {
	fa, fb := s.faces[i], s.faces[j]
	if sheetHoldsBoth(fa.Geometry(), fb.Geometry(), s.trimProbes(i), s.trimProbes(j), res) {
		return coincidentTrimOverlap(fa, fb, sharedFaceContact(fa, fb), res)
	}
	curves, handled := geom.SurfaceIntersect(fa.Geometry(), fb.Geometry(), s.boxes[i].Union(s.boxes[j]), res)
	if !handled {
		return declineUnresolvedSurfacePair()
	}
	if len(curves) == 0 {
		return math.Point3{}, false // the surfaces are known not to cross
	}
	return crossingWitness(curves, boxOverlap(s.boxes[i], s.boxes[j]), fa, fb, sharedFaceContact(fa, fb), res)
}

// declineUnresolvedSurfacePair is the ONE named decline of this detector, so the skip is on the record
// instead of being silent: it answers "no crossing" for a face pair the analytic intersector could not
// resolve.
//
// geom.SurfaceIntersect reports handled=false when neither the closed form nor the general marcher
// produced a curve and the two surfaces may still cross (an oblique cone section the conic solver
// declines, a fitted patch the marcher fails to seed). kernel/ops/boundary_cross.go treats that case
// as CROSSING, because there the conservative move is to demote the operands to the general boolean —
// always correct, only slower. Here the conservative direction is the opposite one: SelfIntersections
// is a VALIDITY verdict, and manufacturing a defect on every pair the intersector cannot resolve would
// condemn healthy bodies (and, through ValidateBodyEntities, fail features that are perfectly sound).
// An unresolvable pair is therefore reported as no crossing, which is a false-negative risk this
// detector accepts knowingly and which shrinks as geom's intersector coverage grows.
func declineUnresolvedSurfacePair() (math.Point3, bool) {
	return math.Point3{}, false
}

// crossingWitness returns the first intersection curve's witness: a point that lies inside BOTH trims
// and off the faces' shared boundary, i.e. a place the two trimmed faces really do occupy together.
func crossingWitness(curves []geom.Curve3, overlap math.Box, fa, fb *topo.Face,
	shared sharedContact, res geom.Resolution) (math.Point3, bool) {
	inBoth := func(p math.Point3) bool {
		return brep.PointInFaceTrim(fa, p) && brep.PointInFaceTrim(fb, p) && !shared.holds(p, res.Sew())
	}
	for _, c := range curves {
		if p, ok := midCrossingSample(c, overlap, inBoth); ok {
			return p, true
		}
	}
	return math.Point3{}, false
}

// midCrossingSample walks the interior samples of an intersection curve, bounded to the two faces'
// box overlap (sampleRange — the closed form returns an UNBOUNDED line for a plane pair), and returns
// the MIDDLE accepted sample.
//
// The middle, not the first: the witness is quoted to a human and filtered by callers against the
// shared boundary, so it must sit well inside the interpenetration rather than at its rim. Two faces
// that share only a vertex and then fan apart cross right from that vertex, and a first-hit witness
// lands a sampling step away from it — indistinguishable from legitimate vertex contact (#1321).
func midCrossingSample(c geom.Curve3, overlap math.Box, accept func(math.Point3) bool) (math.Point3, bool) {
	lo, hi, ok := sampleRange(c, overlap)
	if !ok || hi <= lo {
		return math.Point3{}, false
	}
	var hits []math.Point3
	for i := 1; i < curveTrimSamples; i++ {
		if p := c.PointAt(lo + (hi-lo)*float64(i)/curveTrimSamples); accept(p) {
			hits = append(hits, p)
		}
	}
	if len(hits) == 0 {
		return math.Point3{}, false
	}
	return hits[len(hits)/2], true
}

// sharedContact is the geometry two faces legitimately share: the EXACT curves of their common edges
// and the points of their common vertices. Contact within tolerance of any of it is the faces meeting
// along their shared topology, not an interpenetration.
//
// The curves are the edges' own geometry, never the chord between their vertices: a shared ARC (every
// fillet's tangent edge) bows away from its chord by the sagitta, so a chord-based filter let the whole
// tangent-blend population read as interpenetrating.
type sharedContact struct {
	curves []geom.Curve3
	points []math.Point3
}

// sharedFaceContact collects the edges and vertices faces a and b have in common.
func sharedFaceContact(a, b *topo.Face) sharedContact {
	edgesB, vertsB := map[*topo.Edge]bool{}, map[*topo.Vertex]bool{}
	for _, e := range b.Edges() {
		edgesB[e] = true
		vertsB[e.StartVertex()], vertsB[e.EndVertex()] = true, true
	}
	var out sharedContact
	seenV := map[*topo.Vertex]bool{}
	for _, e := range a.Edges() {
		if edgesB[e] {
			out.curves = append(out.curves, e.Geometry())
		}
		out.points = appendSharedVertices(out.points, e, vertsB, seenV)
	}
	return out
}

// appendSharedVertices adds edge e's ends to pts when face b uses them too, each at most once.
func appendSharedVertices(pts []math.Point3, e *topo.Edge, vertsB, seenV map[*topo.Vertex]bool) []math.Point3 {
	for _, v := range [2]*topo.Vertex{e.StartVertex(), e.EndVertex()} {
		if vertsB[v] && !seenV[v] {
			seenV[v] = true
			pts = append(pts, v.Point())
		}
	}
	return pts
}

// holds reports whether p lies within tol of the shared topology — on a common edge's exact curve or
// at a common vertex.
func (s sharedContact) holds(p math.Point3, tol float64) bool {
	for _, q := range s.points {
		if float64(p.DistanceTo(q)) <= tol {
			return true
		}
	}
	for _, c := range s.curves {
		if brep.EntityDistance(brep.PointSupport(p), brep.CurveSupport(c)) <= tol {
			return true
		}
	}
	return false
}

// faceTrimProbes returns the exact points a trim-overlap decision is taken at: every boundary vertex
// and every boundary edge's curve midpoint. Vertices alone would miss an overlap whose corners all sit
// outside the other trim (two crossing bars), and midpoints alone would miss a corner poking in.
func faceTrimProbes(f *topo.Face) []math.Point3 {
	edges := f.Edges()
	out := make([]math.Point3, 0, 2*len(edges))
	for _, e := range edges {
		out = append(out, e.StartVertex().Point(), edgeCurveMidpoint(e))
	}
	return out
}

// edgeCurveMidpoint is the point at the middle of an edge's own parameter range, falling back to the
// vertex chord midpoint for an unbounded curve (whose mid-parameter is not a number).
func edgeCurveMidpoint(e *topo.Edge) math.Point3 {
	lo, hi := e.Geometry().Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return e.StartVertex().Point().Lerp(e.EndVertex().Point(), 0.5)
	}
	return e.Geometry().PointAt((lo + hi) / 2)
}
