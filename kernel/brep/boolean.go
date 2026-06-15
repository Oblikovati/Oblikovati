// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"

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
// produces a low-face-count result and is sound under chaining. Coplanar (shared-plane)
// faces are a follow-up — operands that don't share a face plane work today.
func Boolean(op Op, a, b *topo.Body) (*topo.Body, error) {
	fa, oka := facesOf(a)
	fb, okb := facesOf(b)
	if !oka || !okb {
		return nil, ErrNonPlanar
	}
	res, away, err := booleanOnce(op, fa, fb, a, b)
	if err != nil || away.LengthSquared() == 0 {
		return res, err
	}
	return retryNudged(op, fa, fb, a, res, away)
}

// booleanOnce runs one pass: imprint, split, classify, keep, stitch. The vector is a non-zero
// "away" direction when the pass hit a tangent/grazing contact (operand B nudged along it
// opens a clean clearance), else the zero vector.
func booleanOnce(op Op, fa, fb []planarFace, a, b *topo.Body) (*topo.Body, math.Vector3, error) {
	impA, impB := imprintAll(fa, fb)
	var kept []subFace
	kept = append(kept, selectFaces(fa, impA, b, fb, op, false)...)
	kept = append(kept, selectFaces(fb, impB, a, fa, op, true)...)
	return stitch(kept)
}

// nudgeEps is the magnitude (cm) of the clearance opened at a tangent contact: above the weld
// grid (1e-6) so it survives, far below any modelled feature so it is geometrically
// irrelevant. A line tangency carries no material, so replacing it with a ~0.1 µm gap loses
// nothing and — unlike the exact tangent — leaves no coincident edge for a re-weld to collapse.
const nudgeEps = 1e-5

// retryNudged re-runs the boolean with operand B nudged a hair along `away` (out of the
// tangent contact) so the degenerate touch becomes a clean clearance. It keeps the nudged
// result only when it is a proper solid, so a nudge that would disconnect the part falls back
// to the original (topologically resolved) result.
func retryNudged(op Op, fa, fb []planarFace, a, original *topo.Body, away math.Vector3) (*topo.Body, error) {
	fbp := translateFaces(fb, away.Scale(nudgeEps))
	bp, _, err := stitch(planarToSubFaces(fbp))
	if err != nil || bp == nil {
		return original, nil
	}
	res, _, err := booleanOnce(op, fa, fbp, a, bp)
	if err != nil || res == nil || !res.IsSolid() {
		return original, nil
	}
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

// planarToSubFaces adapts whole planar faces to sub-faces (outer ring first, the rest holes)
// so stitch can rebuild a body from them — used to materialise the nudged operand B.
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
// edge, and imprints ITSELF with its own bottom edge.
func imprintAll(fa, fb []planarFace) (impA, impB [][][2]math.Point3) {
	impA = make([][][2]math.Point3, len(fa))
	impB = make([][][2]math.Point3, len(fb))
	for i := range fa {
		for j := range fb {
			if coplanar(fa[i], fb[j]) {
				impA[i] = append(impA[i], coplanarOverlapSegments(fa[i], faceEdges3D(fb[j]))...)
				impB[j] = append(impB[j], coplanarOverlapSegments(fb[j], faceEdges3D(fa[i]))...)
				continue
			}
			segs := imprint(fa[i], fb[j])
			impA[i] = append(impA[i], interiorSegments(fa[i], segs)...)
			impB[j] = append(impB[j], interiorSegments(fb[j], segs)...)
		}
	}
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
		if iv[1]-iv[0] > 1e-9 {
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
			lo, hi := maxf(x[0], y[0]), minf(x[1], y[1])
			if hi > lo {
				out = append(out, [2]float64{lo, hi})
			}
		}
	}
	return out
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// selectFaces splits each face by its imprints and keeps the material sub-faces this
// operation wants, classifying each via [classifySubFace]. `others` is the other solid's
// face list (for the coplanar overlap test); `other` is the body itself (for the ray cast).
func selectFaces(faces []planarFace, imprints [][][2]math.Point3, other *topo.Body, others []planarFace, op Op, isB bool) []subFace {
	var kept []subFace
	for i, f := range faces {
		var fromFace []subFace
		for _, sf := range splitFace(f, imprints[i]) {
			if out, ok := classifySubFace(sf, f, other, others, op, isB); ok {
				fromFace = append(fromFace, out)
			}
		}
		fromFace = mergeFilledHoles(fromFace)
		// A face that survives as a single piece carries its source lineage unchanged, so
		// its reference key is identical after the boolean (K1a). A face split into several
		// kept pieces gives each a distinct child lineage (parent + split#k).
		for k := range fromFace {
			fromFace[k].fromB = isB // operand tag, so the stitch can fuse tangent contacts
			if len(fromFace) == 1 {
				fromFace[k].lineage = f.lineage
			} else {
				fromFace[k].lineage = splitLineage(f.lineage, k)
			}
		}
		kept = append(kept, fromFace...)
	}
	return kept
}

// splitLineage derives a distinct child lineage for the k-th piece of a face split into
// several by the boolean.
func splitLineage(parent topo.Lineage, k int) topo.Lineage {
	return topo.NewLineage(append(parent.Tokens(), topo.Tok("brep", "split", k))...)
}

// classifySubFace decides whether a sub-face survives. A fragment coplanar with a face of
// the other solid follows the ON/ON table ([coplanarKeep]); otherwise it is kept by the
// inside/outside table ([keep]) from a ray cast, with B's difference faces reversed to form
// the cut walls.
func classifySubFace(sf subFace, f planarFace, other *topo.Body, others []planarFace, op Op, isB bool) (subFace, bool) {
	if covered, sameNormal := coplanarCover(f, sf.point, others); covered {
		return sf, coplanarKeep(op, isB, sameNormal)
	}
	if !keep(op, isB, insideSolid(other, sf.point)) {
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
const boundaryImprintTol = 1e-7

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
