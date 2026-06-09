// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"

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
	impA, impB := imprintAll(fa, fb)
	var kept []subFace
	kept = append(kept, selectFaces(fa, impA, b, fb, op, false)...)
	kept = append(kept, selectFaces(fb, impB, a, fa, op, true)...)
	return stitch(kept)
}

// imprintAll computes, for every crossing face pair, the shared intersection segment and
// records it on both faces (by index).
func imprintAll(fa, fb []planarFace) (impA, impB [][][2]math.Point3) {
	impA = make([][][2]math.Point3, len(fa))
	impB = make([][][2]math.Point3, len(fb))
	for i := range fa {
		for j := range fb {
			if coplanar(fa[i], fb[j]) {
				impA[i] = append(impA[i], faceEdges3D(fb[j])...)
				impB[j] = append(impB[j], faceEdges3D(fa[i])...)
				continue
			}
			segs := imprint(fa[i], fb[j])
			impA[i] = append(impA[i], segs...)
			impB[j] = append(impB[j], segs...)
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
		// A face that survives as a single piece carries its source lineage unchanged, so
		// its reference key is identical after the boolean (K1a). A face split into several
		// kept pieces gives each a distinct child lineage (parent + split#k).
		for k := range fromFace {
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
