// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The DEGENERATE cylinder∩cylinder case: two cylinders of EQUAL radius whose axes meet. It is the one
// place the ruled∩quadric closed form declines for a reason that is not ill-conditioning but exactness —
// the two roots of its quadratic coincide at a fold, and the section is not two azimuth wraps but two
// planar ELLIPSES crossing at the folds. Subtracting the two implicit forms leaves a difference of
// squares, so the section lies in the two planes through the axes' bisectors, and in each of them it is
// an ellipse of semi-axes r and r/sin(half-angle).
//
// It belongs here, inside the general intersector, and not in a caller: a Steinmetz solid is a boolean
// of two cylinders like any other, and the only thing special about it is that its section has a closed
// form the generic one cannot express (ADR-0061 stage 4, kernel ground rules — a surface-pair closed
// form is a fast path INSIDE the general intersector, never a top-level handler).

// equalCylinderSection returns the two ellipses two EQUAL-radius cylinders with intersecting,
// non-parallel axes cut from one another. ok=false for any other pair, so the caller falls through to
// the general ruled∩quadric form.
func equalCylinderSection(a, b Surface, res Resolution) ([]Curve3, bool) {
	ca, okA := a.(Cylinder)
	cb, okB := b.(Cylinder)
	if !okA || !okB || stdmath.Abs(ca.Radius-cb.Radius) > res.Weld() {
		return nil, false
	}
	e1, e2 := ca.AxisDir.AsVector(), cb.AxisDir.AsVector()
	w := e1.Cross(e2)
	if float64(w.Length()) <= equalCylinderSkewFloor {
		return nil, false // parallel or nearly so: the section is not two ellipses
	}
	centre, ok := axesMeetingPoint(ca, cb, e1, e2, w, res)
	if !ok {
		return nil, false // the axes pass by one another: no common centre, no planar section
	}
	half := stdmath.Acos(math.Clamp(float64(e1.AsUnit().Dot(e2.AsUnit())), -1, 1)) / 2
	return equalCylinderEllipses(centre, e1, e2, w, ca.Radius, half)
}

// equalCylinderSkewFloor is the smallest |e1×e2| the pair may have. Below it the axes are parallel, the
// section is not two ellipses at all, and the bisector planes are not defined.
const equalCylinderSkewFloor = 1e-9 // tol:numeric — |sin| between two unit axes (dimensionless)

// equalCylinderEllipses builds the two bisector-plane ellipses. Each has the axes' common perpendicular
// as one semi-axis (length r, where the two cylinders' walls are closest) and the bisector as the other,
// stretched by the angle between the axes.
func equalCylinderEllipses(centre math.Point3, e1, e2, w math.Vector3, r, half float64) ([]Curve3, bool) {
	var out []Curve3
	for _, arm := range [2]struct {
		major math.Vector3
		scale float64
	}{
		{e1.Add(e2), stdmath.Sin(half)},
		{e1.Sub(e2), stdmath.Cos(half)},
	} {
		if arm.scale <= equalCylinderSkewFloor {
			return nil, false
		}
		el, err := NewEllipseFull(centre, arm.major.Cross(w), arm.major, r/arm.scale, r)
		if err != nil {
			return nil, false
		}
		out = append(out, el)
	}
	return out, true
}

// axesMeetingPoint returns the point the two axes share, when they do. Two skew axes have no common
// point and their equal-radius section is not planar, so the pair falls back to the general form.
func axesMeetingPoint(ca, cb Cylinder, e1, e2, w math.Vector3, res Resolution) (math.Point3, bool) {
	d := ca.Origin.VectorTo(cb.Origin)
	if stdmath.Abs(float64(d.Dot(w))/float64(w.Length())) > res.Sew() {
		return math.Point3{}, false // skew: the axes miss each other
	}
	// Solve ca.Origin + s·e1 = cb.Origin + t·e2 in the plane the two axes span.
	den := float64(w.LengthSquared())
	s := float64(d.Cross(e2).Dot(w)) / den
	return ca.Origin.TranslateBy(e1.Scale(math.Scalar(s))), true
}
