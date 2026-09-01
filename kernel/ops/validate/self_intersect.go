// SPDX-License-Identifier: GPL-2.0-only

package validate

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/ops/internal/probe"
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
// the same body could be valid at one mesh.Quality and invalid at another. The decision now runs on the
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
// mesh.Quality argument is kept only so the many call sites that thread one through do not churn; it is
// deliberately unused (the same reason [FillInternalVoids] ignores its own).
//
// Faces that merely touch along shared topology (an edge or a vertex they have in common) are contact,
// not interpenetration, and are excluded. A face pair whose surfaces the analytic intersector cannot
// resolve is DECLINED rather than reported — see [declineUnresolvedSurfacePair].
//
// Example: if hits := ops.SelfIntersections(body, ops.DefaultQuality()); len(hits) > 0 { reject(body) }
func SelfIntersections(b *topo.Body, _ mesh.Quality) []SelfIntersection {
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
// surface sheet takes the trim-overlap arm, every other pair the surface-surface intersection arm, and
// each arm then has to separate a crossing from a TOUCH — facesOverlapMaterial plus a STRICTLY interior
// witness here, an overlap REGION rather than a graze there.
//
// ORDER IS PERFORMANCE, NOT VERDICT. facesOverlapMaterial is a NECESSARY condition, so asking it before
// geom.SurfaceIntersect cannot change any answer — but it costs a handful of point projections, while
// the intersector MARCHES for every pair with no closed form. Two B-spline faces put the marcher's
// corrector inside BSplineSurface.ParamAt for thousands of steps; on the OCCT blend-parity corpus, a
// scan that marched every box-overlapping pair before asking whether the faces reach each other at all
// did not finish in 45 minutes. The shared topology is collected last, for the same reason: that walk
// is O(E).
//
// THE GATE IS A Sew() COMPARISON, and the class matters (ADR-0042). A probe on face i's boundary and
// face j's surface come from INDEPENDENT computations — two surfaces fitted by different operations,
// their common edge marched — so how far apart they read is a Sew() question, never a Weld() one. Read
// at Weld() it let a face and the blend that runs tangent to it through the gate on 3.3e-7 of marcher
// noise (Weld() is 2.5e-7 on that body): corpus bfuseblend/B5's cylinder against its B-spline flank,
// which then put geom's SSI seed field on a NURBS surface and did not return in nine minutes
// (Oblikovati/Oblikovati#3477). Tangency is contact, and Sew() is the tolerance that says so.
//
// The intersector is bounded by the two boxes' OVERLAP, not their union: a point on both trimmed faces
// lies in both range boxes, so nothing outside the overlap can witness a crossing, and the union asks
// the marcher to trace a domain several times larger for answers that are discarded.
func (s *faceScan) interpenetrate(i, j int, res geom.Resolution) (math.Point3, bool) {
	fa, fb := s.faces[i], s.faces[j]
	if sheetHoldsBoth(fa.Geometry(), fb.Geometry(), s.trimProbes(i), s.trimProbes(j), res) {
		return coincidentTrimOverlap(fa, fb, sharedFaceContact(fa, fb), res)
	}
	if !facesOverlapMaterial(fa, fb, s.trimProbes(i), s.trimProbes(j), res.Sew()) {
		return math.Point3{}, false
	}
	overlap := probe.BoxOverlap(s.boxes[i], s.boxes[j])
	curves, handled := geom.SurfaceIntersect(fa.Geometry(), fb.Geometry(), overlap, res)
	if !handled {
		return declineUnresolvedSurfacePair()
	}
	if len(curves) == 0 {
		return math.Point3{}, false // the surfaces are known not to cross
	}
	return crossingWitness(curves, overlap, fa, fb, sharedFaceContact(fa, fb), res)
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

// crossingWitness returns the first intersection curve's witness: a point STRICTLY inside both trims
// and off the faces' shared boundary, i.e. a place the two trimmed faces really do occupy together.
//
// Strictly, because brep.PointInFaceTrim is inclusive and the boundary is exactly where CONTACT lives.
// Two faces that meet at a right angle along one line — a side facet standing on the floor of the turn
// above it, or a block set beside and above another — have that whole line inside both trims, so an
// inclusive test reads every such touch as a crossing (543 of them on the coil #2080's
// TestCoilAcceptsTurnsThatClear must build).
func crossingWitness(curves []geom.Curve3, overlap math.Box, fa, fb *topo.Face,
	shared sharedContact, res geom.Resolution) (math.Point3, bool) {
	for _, c := range curves {
		// A witness is only as certain as the CURVE IT CAME FROM. A marched intersection is a chord
		// approximation carrying a measured deviation, so a point it places d off the true curve can
		// read as d inside a trim it actually lies ON, and as d clear of a boundary it actually touches.
		// Judging that against Weld/Sew alone compares a marched position to a tolerance three decades
		// tighter than its own accuracy, and six filleted corpus bodies were condemned as
		// self-intersecting by witnesses sitting 5.5e-5 to 7e-5 off both surfaces — the marcher's error,
		// not an interpenetration (Oblikovati/Oblikovati#3477). The achieved tolerance is a measured
		// output of the intersector, so it is what this decision reads.
		stamped := geom.CurveDeviation(c)
		inBoth := func(p math.Point3) bool {
			// How far the witness actually sits off the two surfaces IS its uncertainty, measured on
			// the spot rather than taken from metadata the curve may not carry. A point the marcher
			// placed d away cannot be trusted to be more than d inside a trim, nor more than d clear
			// of a boundary.
			reach := stdmath.Max(stamped, stdmath.Max(offSurface(fa, p), offSurface(fb, p)))
			insideBy, clearOf := stdmath.Max(res.Weld(), reach), stdmath.Max(res.Sew(), reach)
			return strictlyInsideTrim(fa, p, insideBy) && strictlyInsideTrim(fb, p, insideBy) &&
				!shared.holds(p, clearOf)
		}
		if p, ok := midCrossingSample(c, overlap, inBoth); ok {
			return p, true
		}
	}
	return math.Point3{}, false
}

// offSurface is how far p lies off the face's own surface — the witness's measured uncertainty.
func offSurface(f *topo.Face, p math.Point3) float64 {
	s := f.Geometry()
	u, v := s.ParamAt(p)
	return float64(p.DistanceTo(s.PointAt(u, v)))
}

// midCrossingSample walks the interior samples of an intersection curve, bounded to the two faces'
// box overlap, and returns the MIDDLE accepted sample.
//
// Bounded TWICE, because probe.SampleRange only narrows the PARAMETER interval and only does so for an
// unbounded curve (the closed form returns an infinite line for a plane pair); a bounded curve — the
// closed loop two curved faces meet on — is sampled over its whole domain and wanders far outside
// either face. A witness lies on both TRIMMED faces, so it lies in both range boxes: sampling outside
// their overlap can only produce a point neither face owns, and the trim test is then the only thing
// standing between the scan and a fabricated crossing (Oblikovati/Oblikovati#3477).
//
// The middle, not the first: the witness is quoted to a human and filtered by callers against the
// shared boundary, so it must sit well inside the interpenetration rather than at its rim. Two faces
// that share only a vertex and then fan apart cross right from that vertex, and a first-hit witness
// lands a sampling step away from it — indistinguishable from legitimate vertex contact (#1321).
func midCrossingSample(c geom.Curve3, overlap math.Box, accept func(math.Point3) bool) (math.Point3, bool) {
	lo, hi, ok := probe.SampleRange(c, overlap)
	if !ok || hi <= lo {
		return math.Point3{}, false
	}
	var hits []math.Point3
	for i := 1; i < probe.CurveTrimSamples; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/probe.CurveTrimSamples)
		if overlap.Contains(p) && accept(p) {
			hits = append(hits, p)
		}
	}
	if len(hits) == 0 {
		return math.Point3{}, false
	}
	return hits[len(hits)/2], true
}

// facesOverlapMaterial reports whether either face has a boundary point lying strictly INSIDE the
// material the other one bounds — the invariant that separates INTERPENETRATION from CONTACT (#2075,
// #2080), and the exact form of what the deleted Möller narrow phase measured on facets.
//
// An intersection curve running inside both trims is NOT enough on its own. Two faces that merely TOUCH
// have one: a side facet meeting the top of the turn below it along their common line, or two facets
// lying face to face, both put the curve inside both trims while the solids only kiss. Measured on the
// coil fixture #2080 uses, the two populations are eight decades apart — at pitch = profile depth,
// which must build, no probe reaches more than 4e-16 into the other face's material; at pitch 0.8,
// which must be refused, they reach 0.7997.
//
// Both halves of the test are load-bearing. The depth alone is a HALF-SPACE test, and a face at a
// concave corner legitimately sits behind its neighbour's plane — so the probe's foot must also land
// STRICTLY inside that neighbour's own TRIM, which is what makes the question "is this point inside the
// material THIS FACE bounds" rather than "is it behind its plane". Strictly, because a face that merely
// abuts f lands its feet exactly ON f's trim boundary: two blocks stacked face to face put every probe
// of the upper one's floor on the edge of the lower one's wall, two units deep behind its plane.
//
// It is a necessary condition read off the faces' own boundaries, so a crossing that dips through
// without either boundary reaching in (a tangential dimple) is not seen. That is the same class the
// corner ring declines in face_loop_corners.go, and for the same reason: seeing it needs the interior.
func facesOverlapMaterial(fa, fb *topo.Face, probesA, probesB []math.Point3, tol float64) bool {
	return probeInsideMaterial(fb, probesA, tol) || probeInsideMaterial(fa, probesB, tol)
}

// probeInsideMaterial reports whether any probe lies deeper than tol on f's material side, with its
// foot STRICTLY inside f's own trim — clear of the trim boundary, where a face merely abutting f lands.
func probeInsideMaterial(f *topo.Face, probes []math.Point3, tol float64) bool {
	for _, p := range probes {
		gap, foot := signedGapToFace(f, p)
		if gap < -tol && strictlyInsideTrim(f, foot, tol) {
			return true
		}
	}
	return false
}

// signedGapToFace is p's signed distance to f's surface along f's own outward normal at the foot of the
// projection — positive on the side f faces, negative inside the material it bounds — with that foot.
func signedGapToFace(f *topo.Face, p math.Point3) (float64, math.Point3) {
	_, _, foot := geom.ClosestPointOnSurface(f.Geometry(), p)
	return float64(foot.VectorTo(p).Dot(probe.OutwardFaceNormalAt(f, foot))), foot
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

// edgeProbesPerEdge is how many points each boundary edge contributes to a face's probe set: its start
// vertex and the quarter points of its own parameter range. It is a COUNT, not a tolerance.
//
// Two per edge is not enough, and a closed face is why. brep.SolidCylinder's side face is bounded by
// two full-circle rims and a seam, so start-and-midpoint samples it at exactly TWO angles — the seam's
// and its antipode. Measured on two cylinders overlapping by 0.01 whose seams sit 90° from the overlap,
// both angles report the SAME distance to the other cylinder, so the crossing is invisible to any
// predicate read off that set. Quartering the rims samples the near and far sides as well.
const edgeProbesPerEdge = 4

// faceTrimProbes returns the exact points a trim-overlap or straddle decision is taken at: every
// boundary vertex and the quarter points of every boundary edge. Vertices alone would miss an overlap
// whose corners all sit outside the other trim (two crossing bars), and interior points alone would
// miss a corner poking in.
func faceTrimProbes(f *topo.Face) []math.Point3 {
	edges := f.Edges()
	out := make([]math.Point3, 0, edgeProbesPerEdge*len(edges))
	for _, e := range edges {
		out = append(out, e.StartVertex().Point())
		for k := 1; k < edgeProbesPerEdge; k++ {
			out = append(out, edgeCurvePointAt(e, float64(k)/edgeProbesPerEdge))
		}
	}
	return out
}

// edgeCurvePointAt is the point at fraction tau of an edge's own parameter range, falling back to the
// vertex chord for an unbounded curve (whose interior parameters are not numbers).
func edgeCurvePointAt(e *topo.Edge, tau float64) math.Point3 {
	lo, hi := e.Geometry().Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return e.StartVertex().Point().Lerp(e.EndVertex().Point(), math.Scalar(tau))
	}
	return e.Geometry().PointAt(lo + tau*(hi-lo))
}
