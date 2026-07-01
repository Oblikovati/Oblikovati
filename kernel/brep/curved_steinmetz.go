// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Equal-radius Steinmetz frame + arcs (M2 Phase 2, Oblikovati/Oblikovati#1335). This file holds the shared
// analytic geometry the GENERAL curved∩curved pipeline (curved_steinmetz_general.go, Oblikovati#1403) uses to
// build every equal-radius perpendicular-bicylinder boolean — intersect, cut and join all ride the general
// (u,v) arrangement now; there is no bespoke assembler left.
//
// Two EQUAL-radius perpendicular cylinders crossing at a common axis point intersect in the Steinmetz
// bicylinder, whose SSI imprint is NOT a quartic saddle but two planar ELLIPSES that cross at two pinch
// points. Because that imprint SELF-INTERSECTS, the recogniser splits it at the analytic pinches into four
// open arcs and feeds those to the arrangement (approach A, Oblikovati#1403) rather than the two crossing
// closed loops the SSI tracer would otherwise return.
//
// Geometry in the frame (a = axis A, b = axis B, n = a×b) through the axis crossing O, with equal radius R:
// a surface point O + α·a + β·b + γ·n is on cyl A when β²+γ²=R² and on cyl B when α²+γ²=R², so on both when
// β=±α. The two intersection ellipses therefore lie in the planes β=α and β=−α; they meet only where
// α=β=0, i.e. at the PINCH points O±R·n, where the two surfaces are tangent.

// steinmetzFrame resolves two bodies into the Steinmetz frame: the axis crossing point O, the two unit axis
// directions, and the shared radius R. ok=false unless both are bare cylinders of equal radius whose axes
// are perpendicular, cross, and whose finite extents each reach at least R either side of O (so the
// bicylinder is not clipped by a cap).
func steinmetzFrame(a, b *topo.Body) (o math.Point3, dirA, dirB math.Vector3, r float64, ok bool) {
	ca, baseA, hA, okA := cylinderSolidParams(facesOfAny(a))
	cb, baseB, hB, okB := cylinderSolidParams(facesOfAny(b))
	if !okA || !okB || !nearEqual(ca.Radius, cb.Radius) {
		return math.Point3{}, math.Vector3{}, math.Vector3{}, 0, false
	}
	dirA, dirB = ca.AxisDir.AsVector(), cb.AxisDir.AsVector()
	if !math.IsNearZero(dirA.Dot(dirB), 1e-7) { // tol:angular — perpendicular-axes dot of unit vectors
		return math.Point3{}, math.Vector3{}, math.Vector3{}, 0, false
	}
	lineA, _ := geom.NewLine(ca.Origin, dirA)
	lineB, _ := geom.NewLine(cb.Origin, dirB)
	// Axis-intersection coincidence is model-relative (#1399), from the two operands' extent.
	o, ok = geom.LineLineIntersection(lineA, lineB, geom.ResolutionForBox(a.RangeBox().Union(b.RangeBox())).Plane())
	if !ok || !axisReachesR(ca, baseA, hA, o) || !axisReachesR(cb, baseB, hB, o) {
		return math.Point3{}, math.Vector3{}, math.Vector3{}, 0, false
	}
	return o, dirA, dirB, ca.Radius, true
}

// axisReachesR reports whether the cylinder's finite extent reaches at least its radius either side of O
// along the axis — the condition for the bicylinder (which spans ±R about O on each axis) to lie strictly
// between the caps rather than be clipped by one.
func axisReachesR(c geom.Cylinder, base math.Point3, height float64, o math.Point3) bool {
	axis := c.AxisDir.AsVector()
	vBase := float64(c.Origin.VectorTo(base).Dot(axis))
	vO := float64(c.Origin.VectorTo(o).Dot(axis))
	tol := geom.ResolutionForSize(height + 2*c.Radius).Weld() // model-relative axial-reach margin (#1399)
	return vO-c.Radius >= vBase-tol && vO+c.Radius <= vBase+height+tol
}

// steinmetzArcs builds the four elliptical-arc edges of the bicylinder. The E+ ellipse lies in the plane
// spanned by (a+b) and n (its major axis is a+b, length R√2; its minor axis is n, length R); the E−
// ellipse lies in the plane spanned by (a−b) and n. Each ellipse passes through both pinch points O±R·n at
// angles ±π/2, so the front arc (sweep starting at −π/2) runs P− → P+ and the back arc (starting at +π/2)
// runs P+ → P−.
func steinmetzArcs(o math.Point3, dirA, dirB math.Vector3, r float64) (ePlusFront, ePlusBack, eMinusFront, eMinusBack geom.EllipticalArc) {
	majorR := stdmath.Sqrt2 * r
	sum, diff := dirA.Add(dirB), dirA.Sub(dirB)
	const start, sweep = -stdmath.Pi / 2, stdmath.Pi
	// E+ (β=α): Normal = a−b, MajorAxis = a+b, so the minor direction Normal×MajorAxis = n.
	ePlusFront, _ = geom.NewEllipticalArc(o, diff, sum, majorR, r, start, sweep)
	ePlusBack, _ = geom.NewEllipticalArc(o, diff, sum, majorR, r, start+sweep, sweep)
	// E− (β=−α): Normal = −(a+b), MajorAxis = a−b, so again the minor direction is n.
	eMinusFront, _ = geom.NewEllipticalArc(o, sum.Scale(-1), diff, majorR, r, start, sweep)
	eMinusBack, _ = geom.NewEllipticalArc(o, sum.Scale(-1), diff, majorR, r, start+sweep, sweep)
	return ePlusFront, ePlusBack, eMinusFront, eMinusBack
}

// nearEqual reports whether two lengths are equal to a small relative tolerance (the Steinmetz case needs
// the two cylinder radii to match).
func nearEqual(x, y float64) bool {
	// a relative equality test (scaled by max|x|,|y|), not a model-anchored length.
	return stdmath.Abs(x-y) <= 1e-7*(1+stdmath.Max(stdmath.Abs(x), stdmath.Abs(y))) // tol:numeric
}
