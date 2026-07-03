// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Op is a boolean operation between two solids.
type Op int

const (
	Union        Op = iota // A ∪ B
	Difference             // A − B
	Intersection           // A ∩ B
)

// ErrNonPlanar is returned when an operand has a non-planar face (the planar B-rep
// boolean handles planar-faceted solids; curved-face booleans need NURBS work).
var ErrNonPlanar = errors.New("brep: boolean requires planar-faceted solids")

// subFace is one region a face is split into, with an interior point for classification
// and the outward normal it should carry in the result. lineage carries the source face's
// lineage forward so the result face's reference key survives the boolean (K1a).
type subFace struct {
	outer   []math.Point3
	holes   [][]math.Point3
	normal  math.Vector3
	point   math.Point3
	lineage topo.Lineage
	fromB   bool // which operand this came from (false=A, true=B); fuses tangent contacts
}

// Boolean computes A op B as a clean planar B-rep: it imprints the face–face intersection
// segments, splits each face along them (the 2D arrangement), classifies the sub-faces
// against the other solid, keeps the ones the operation calls for (reversing B's where
// needed), and stitches them into a watertight solid. Unlike the triangle-soup CSG this
// produces a low-face-count result and is sound under chaining.
func Boolean(op Op, a, b *topo.Body) (*topo.Body, error) {
	return BooleanDiag(op, a, b, nil)
}

// BooleanDiag is [Boolean] with a diagnostic recorder (nil to discard). A tangent/grazing contact
// (a >2-edge-use configuration) is resolved EXACTLY by the stitch first (resolveEdgeUses azimuth-
// pairs the coincident dihedrals; pinchedEndpoints splits their shared endpoints), and that exact,
// UNDISPLACED result is SHIPPED whenever it is a valid closed 2-manifold — the common flush/box
// tangency (#1600). Only when the exact result stays pinched (an Euler-inadmissible χ the planar
// pinch resolution does not yet split — a faceted-cylinder line-tangency) does it fall back to the
// geometry-moving nudge as a recorded LAST RESORT (issue option 1); that fallback is a tracked
// Defect, never silent. The exact path never perturbs output coordinates (SoS discipline).
func BooleanDiag(op Op, a, b *topo.Body, rec *diag.Recorder) (*topo.Body, error) {
	fa, oka := facesOf(a)
	fb, okb := facesOf(b)
	if !oka || !okb {
		return nil, ErrNonPlanar
	}
	res, away, err := booleanOnce(op, fa, fb, a, b)
	if err != nil || away.LengthSquared() == 0 {
		return res, err
	}
	if exactTangentIsValid(res) {
		rec.Recordf(CodeBooleanTangentContact, diag.Info,
			"tangent/grazing contact resolved exactly (no displacement) near axis (%g, %g, %g)",
			float64(away.X), float64(away.Y), float64(away.Z))
		return res, nil
	}
	return retryNudged(op, fa, fb, a, res, away, rec)
}

// exactTangentIsValid reports whether the exactly-resolved tangent result is a closed 2-manifold
// fit to ship: a solid whose every edge is shared by exactly two faces and whose χ is Euler-
// admissible. When it is NOT — a genuine line-tangency whose endpoints stay pinched (χ odd), which
// the planar pinch resolution does not yet split — the caller falls back to the recorded nudge so
// the operation never ships an invalid body nor silently degrades to triangle-soup CSG (#1600).
func exactTangentIsValid(b *topo.Body) bool {
	if b == nil || !b.IsSolid() {
		return false
	}
	for _, e := range b.Edges() {
		if len(e.Faces()) != 2 {
			return false
		}
	}
	return b.EulerAdmissible()
}

// booleanOnce runs one pass: imprint, split, classify, keep, stitch. The vector is a non-zero
// "away" direction when the pass hit a tangent/grazing contact (operand B nudged along it
// opens a clean clearance), else the zero vector.
func booleanOnce(op Op, fa, fb []planarFace, a, b *topo.Body) (*topo.Body, math.Vector3, error) {
	// One AABB-culled candidate set feeds imprint, provenance AND the coplanar-cover scans —
	// the retired brute version recomputed the O(Fa·Fb) pairing 2–3× per pass — and each
	// operand is flattened ONCE into a solidProbe for every ray-cast classification, instead
	// of per query point (#1607).
	pairs := crossingFaceCandidates(fa, fb)
	impA, impB, prov := imprintCandidates(fa, fb, pairs)
	var kept []subFace
	kept = append(kept, selectFaces(fa, impA, newSolidProbe(b), fb, pairs.bForA, op, false, prov)...)
	kept = append(kept, selectFaces(fb, impB, newSolidProbe(a), fa, pairs.aForB, op, true, prov)...)
	return stitch(kept, prov)
}

// CodeBooleanTangentContact marks a boolean whose operands met at a tangent/grazing contact — a
// coincident line where more than two faces meet — and that shipped the EXACT, undisplaced result
// (the stitch split the coincident dihedrals by radial order and pinch-split their shared
// endpoints). An informational note, not a defect: no coordinate moved (#1600, A5).
const CodeBooleanTangentContact diag.Code = "boolean.tangent-contact"

// nudgeEps is the magnitude (cm) of the clearance opened at a tangent contact the exact path could
// not resolve to a valid solid: above the weld grid (planarStitchGrid, 1e-6) so it survives, far
// below any modelled feature so it is geometrically irrelevant. A line tangency carries no
// material, so replacing it with a ~0.1 µm gap loses nothing and leaves no pinched seam. Calibrated
// with — and only meaningful relative to — the absolute planar weld grid: the two move together
// (#1602).
const nudgeEps = 10 * planarStitchGrid // tol:calibrated — imprint nudge tied to the planar weld grid

// CodeBooleanNudgedGeometry marks a boolean that shipped the displaced-geometry tangency retry: the
// EXACT topological resolution of a tangent contact stayed pinched (Euler-inadmissible), so the
// result contains operand B translated by nudgeEps (#1600). A tracked Defect — the displaced
// coordinates poison flush mating, coplanar detection and exports downstream — kept only as the
// last resort for the residual configurations the exact planar pinch resolution does not yet split.
const CodeBooleanNudgedGeometry diag.Code = "boolean.nudged-geometry"

// retryNudged re-runs the boolean with operand B nudged a hair along `away` (out of the tangent
// contact) so the degenerate touch becomes a clean clearance. It is the LAST RESORT behind the
// exact resolution (#1600): it runs only when the un-nudged result is not a valid solid, and
// shipping it records a Defect because the output carries the displaced coordinates. A nudge that
// fails too falls back to the original exact result.
func retryNudged(op Op, fa, fb []planarFace, a, original *topo.Body, away math.Vector3, rec *diag.Recorder) (*topo.Body, error) {
	fbp := translateFaces(fb, away.Scale(nudgeEps))
	bp, _, err := stitch(planarToSubFaces(fbp), nil) // materialising nudged B: no intersections to name
	if err != nil || bp == nil {
		return original, nil
	}
	res, _, err := booleanOnce(op, fa, fbp, a, bp)
	if err != nil || res == nil || !res.IsSolid() {
		return original, nil
	}
	rec.Recordf(CodeBooleanNudgedGeometry, diag.Defect,
		"tangent contact did not resolve to a valid solid exactly: shipping operand B displaced by %g along (%g, %g, %g)",
		nudgeEps, float64(away.X), float64(away.Y), float64(away.Z))
	return res, nil
}

// translateFaces returns a copy of the planar faces rigidly displaced by d (a rigid move keeps
// every face planar, so the result is still a valid planar-faceted operand).
func translateFaces(faces []planarFace, d math.Vector3) []planarFace {
	out := make([]planarFace, len(faces))
	for i, f := range faces {
		loops := make([][]math.Point3, len(f.loops))
		for li, ring := range f.loops {
			moved := make([]math.Point3, len(ring))
			for vi, p := range ring {
				moved[vi] = p.TranslateBy(d)
			}
			loops[li] = moved
		}
		pl, _ := geom.NewPlane(centroid3(loops[0]), f.normal)
		out[i] = planarFace{plane: pl, normal: f.normal, loops: loops, lineage: f.lineage}
	}
	return out
}

// planarToSubFaces adapts whole planar faces to sub-faces (outer ring first, the rest holes) so
// stitch can rebuild a body from them — used to materialise the nudged operand B.
func planarToSubFaces(faces []planarFace) []subFace {
	out := make([]subFace, 0, len(faces))
	for _, f := range faces {
		if len(f.loops) == 0 {
			continue
		}
		sf := subFace{outer: f.loops[0], normal: f.normal, point: centroid3(f.loops[0]), lineage: f.lineage, fromB: true}
		sf.holes = append(sf.holes, f.loops[1:]...)
		out = append(out, sf)
	}
	return out
}

// imprintAll computes, for every crossing face pair, the shared intersection segment and
// records it on both faces (by index). A segment lying along a face's own boundary is NOT
// recorded on that face: it splits nothing (the arrangement already contains the boundary),
// and a float-wobbled near-copy of a boundary edge destabilizes the 2D arrangement. The
// flush-cut case (#137) hits this constantly — a tool wall whose bottom edge lies exactly in
// the target's bottom plane imprints that plane with a near-duplicate of the coplanar cap
// edge, and imprints ITSELF with its own bottom edge. Since #1607 the pairing is AABB-culled
// and shared with provenanceOf through imprintCandidates; this wrapper keeps the historical
// contract for callers that only need the geometry.
func imprintAll(fa, fb []planarFace) (impA, impB [][][2]math.Point3) {
	impA, impB, _ = imprintCandidates(fa, fb, crossingFaceCandidates(fa, fb))
	return impA, impB
}

// imprint returns the 3D segments of the intersection line of two faces' planes clipped
// to where both faces overlap (empty when parallel or non-overlapping).
func imprint(a, b planarFace) [][2]math.Point3 {
	p0, dir, ok := planeLine(a.plane, b.plane)
	if !ok {
		return nil
	}
	overlap := intersectIntervals(faceLineIntervals(a, p0, dir), faceLineIntervals(b, p0, dir))
	var segs [][2]math.Point3
	for _, iv := range overlap {
		if iv[1]-iv[0] > 1e-9 { // tol:calibrated — planar imprint overlap length (see arrange2d arrTol)
			segs = append(segs, [2]math.Point3{p0.TranslateBy(dir.Scale(iv[0])), p0.TranslateBy(dir.Scale(iv[1]))})
		}
	}
	return segs
}

// intersectIntervals returns the overlaps of two sorted interval sets.
func intersectIntervals(a, b [][2]float64) [][2]float64 {
	var out [][2]float64
	for _, x := range a {
		for _, y := range b {
			lo, hi := max(x[0], y[0]), min(x[1], y[1])
			if hi > lo {
				out = append(out, [2]float64{lo, hi})
			}
		}
	}
	return out
}

// selectFaces splits each face by its imprints and keeps the material sub-faces this
// operation wants, classifying each via [classifySubFace]. `others` is the other solid's
// face list (for the coplanar overlap test), culled per face to its box-overlap candidates
// `otherCand` (#1607); `other` is the body's cached probe (for the winding-number cast).
func selectFaces(faces []planarFace, imprints [][][2]math.Point3, other *solidProbe, others []planarFace, otherCand [][]int, op Op, isB bool, prov []imprintSeg) []subFace {
	var kept []subFace
	for i, f := range faces {
		near := facesAt(others, otherCand[i])
		var fromFace []subFace
		for _, sf := range splitFace(f, imprints[i]) {
			if out, ok := classifySubFace(sf, f, other, near, op, isB); ok {
				fromFace = append(fromFace, out)
			}
		}
		fromFace = mergeFilledHoles(fromFace)
		nameFragments(fromFace, f.lineage, isB, prov)
		kept = append(kept, fromFace...)
	}
	return kept
}

// nameFragments assigns each kept piece of one source face its reference-key lineage. A face that
// survives as a single piece keeps its source lineage unchanged (K1a — its key is identical after
// the boolean). A face split into several pieces names each piece by the SET of cutting faces
// bordering it (#1154), so the piece's identity is the geometry that bounds it rather than an
// ordinal index that an upstream edit can reorder. A piece with no detectable cut border (a
// degenerate split) and any duplicate cutting set fall back to an ordinal, deferred to F05.
func nameFragments(fromFace []subFace, parent topo.Lineage, isB bool, prov []imprintSeg) {
	dups := map[string]int{}
	// The parent's cut segments (with cutting-face keys precomputed) are shared by every fragment,
	// so resolve them once here rather than rebuilding lineage keys per fragment per ring vertex
	// — the #1578 fix for the outrunner's dense-imprint fragment-naming explosion.
	border := parentBorderSegments(parent, prov)
	for k := range fromFace {
		fromFace[k].fromB = isB // operand tag, so the stitch can fuse tangent contacts
		if len(fromFace) == 1 {
			fromFace[k].lineage = parent // K1a: a single survivor keeps its key
			continue
		}
		cutting := fragmentCuttingFaces(border, fromFace[k])
		if len(cutting) == 0 {
			fromFace[k].lineage = splitLineage(parent, k) // no detectable border: ordinal fallback
			continue
		}
		setKey := string(fragmentLineage(parent, cutting, 0).Key())
		fromFace[k].lineage = fragmentLineage(parent, cutting, dups[setKey])
		dups[setKey]++
	}
}

// splitLineage derives a distinct child lineage for the k-th piece of a face split into
// several by the boolean — the fallback when a piece's cutting border cannot be resolved.
func splitLineage(parent topo.Lineage, k int) topo.Lineage {
	return topo.NewLineage(append(parent.Tokens(), topo.Tok("brep", "split", k))...)
}

// classifySubFace decides whether a sub-face survives. A fragment coplanar with a face of
// the other solid follows the ON/ON table ([coplanarKeep]); otherwise it is kept by the
// inside/outside table ([keep]) from a winding-number cast against the other solid's cached
// probe, with B's difference faces reversed to form the cut walls.
func classifySubFace(sf subFace, f planarFace, other *solidProbe, others []planarFace, op Op, isB bool) (subFace, bool) {
	if covered, sameNormal := coplanarCover(f, sf.point, others); covered {
		return sf, coplanarKeep(op, isB, sameNormal)
	}
	if !keep(op, isB, other.inside(sf.point)) {
		return sf, false
	}
	if op == Difference && isB {
		sf = reverseSubFace(sf)
	}
	return sf, true
}

// keep encodes the boolean selection table: which sub-faces (by side and inside/outside
// the other solid) survive each operation.
func keep(op Op, isB, inside bool) bool {
	switch op {
	case Union:
		return !inside // both sides keep their outside-the-other parts
	case Intersection:
		return inside // both sides keep their inside-the-other parts
	default: // Difference: keep A outside B, and B inside A (reversed)
		if isB {
			return inside
		}
		return !inside
	}
}

// reverseSubFace flips a sub-face's orientation (normal and loop windings) so it faces
// into the cavity a Difference carves.
func reverseSubFace(sf subFace) subFace {
	sf.normal = sf.normal.Scale(-1)
	sf.outer = reverseRing(sf.outer)
	for i := range sf.holes {
		sf.holes[i] = reverseRing(sf.holes[i])
	}
	return sf
}

func reverseRing(r []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(r))
	for i, p := range r {
		out[len(r)-1-i] = p
	}
	return out
}

// boundaryImprintTol is the distance at which an imprint point counts as lying on a face's
// boundary. The wobble between a boundary edge and its imprint re-derivation is float noise
// (~1e-15), far below it; genuinely interior imprints sit at feature scale, far above it.
const boundaryImprintTol = 1e-7 // tol:calibrated — planar imprint-on-boundary distance (see arrange2d arrTol)

// interiorSegments filters out the segments that lie along f's boundary, keeping only the
// ones that can actually split the face's interior.
func interiorSegments(f planarFace, segs [][2]math.Point3) [][2]math.Point3 {
	out := segs[:0:0]
	for _, s := range segs {
		if !segmentOnFaceBoundary(f, s) {
			out = append(out, s)
		}
	}
	return out
}

// segmentOnFaceBoundary reports whether the whole segment lies on f's boundary (within
// [boundaryImprintTol]). Endpoints AND midpoint are tested, so a segment that runs along a
// boundary edge's line but crosses the interior elsewhere (a concave face) is kept.
func segmentOnFaceBoundary(f planarFace, s [2]math.Point3) bool {
	mid := math.P3((s[0].X+s[1].X)/2, (s[0].Y+s[1].Y)/2, (s[0].Z+s[1].Z)/2)
	return pointOnFaceBoundary(f, s[0]) && pointOnFaceBoundary(f, mid) && pointOnFaceBoundary(f, s[1])
}

// pointOnFaceBoundary reports whether p lies within [boundaryImprintTol] of any of f's
// boundary edges.
func pointOnFaceBoundary(f planarFace, p math.Point3) bool {
	for _, ring := range f.loops {
		n := len(ring)
		for i := 0; i < n; i++ {
			if distPointSegment(p, ring[i], ring[(i+1)%n]) < boundaryImprintTol {
				return true
			}
		}
	}
	return false
}

// distPointSegment returns the distance from p to segment ab.
func distPointSegment(p, a, b math.Point3) float64 {
	ab := a.VectorTo(b)
	lenSq := ab.LengthSquared()
	if lenSq == 0 {
		return a.VectorTo(p).Length()
	}
	t := a.VectorTo(p).Dot(ab) / lenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return p.VectorTo(a.TranslateBy(ab.Scale(t))).Length()
}
