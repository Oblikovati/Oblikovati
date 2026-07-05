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

// steinmetzSnapCeiling is the radius gap |Δr| below which two near-equal perpendicular crossing cylinders
// snap to a common radius and take the exact bicylinder constructor (#1780). It is the model-relative stitch
// weld (1e-6·size), the SSI producer's OWN noise floor: below it the two-loop neck the general rod-band path
// would otherwise build has closest-approach √(2R·Δr) that is itself unresolvable, so representing the
// near-pinch as the exact pinch is honest, not a loss — the only error is each wall displaced by ≤|Δr|/2,
// well under the weld. ABOVE it the neck is macroscopic, genuine two-loop geometry that must NOT snap (that
// residual band stays with the deterministic faceted route; unifying it analytically is #1780 Direction 2).
// It is deliberately the same value both steinmetzFrame (accept) and crossingCylinderImprint (decline) read,
// so the two paths meet at exactly one radius gap with neither a hole nor an overlap between them.
func steinmetzSnapCeiling(res geom.Resolution) float64 { return res.Stitch() }

// steinmetzSnapRadius returns the common radius two crossing cylinders snap to and whether they are close
// enough to snap (|Δr| ≤ steinmetzSnapCeiling). The common radius is the MEAN, which minimises the maximum
// wall displacement to |Δr|/2 (min/max would move one wall by the full |Δr|); the snap is symmetric because
// both operands are rebuilt at the common radius, so no operation-aware bias is needed — crossing cylinders
// meet transversally, never near-tangentially, so a sub-noise radius nudge cannot open a sliver (#1780).
// Δr is formed directly as ra−rb, never via ra²−rb², which would square the floating-point cancellation.
func steinmetzSnapRadius(ra, rb float64, res geom.Resolution) (radius float64, ok bool) {
	if stdmath.Abs(ra-rb) > steinmetzSnapCeiling(res) {
		return 0, false
	}
	return (ra + rb) / 2, true
}

// steinmetzFrame resolves two bodies into the Steinmetz frame: the axis crossing point O, the two unit axis
// directions, and the common radius R the bicylinder is built at. ok=false unless both are bare cylinders
// whose radii are within steinmetzSnapCeiling (equal, or near-equal and snapped to their mean — #1780) and
// whose axes are perpendicular, cross, and whose finite extents each reach at least R either side of O (so
// the bicylinder is not clipped by a cap).
func steinmetzFrame(a, b *topo.Body) (o math.Point3, dirA, dirB math.Vector3, r float64, ok bool) {
	ca, baseA, hA, okA := cylinderSolidParams(facesOfAny(a))
	cb, baseB, hB, okB := cylinderSolidParams(facesOfAny(b))
	if !okA || !okB {
		return math.Point3{}, math.Vector3{}, math.Vector3{}, 0, false
	}
	// Axis-intersection coincidence and the radius-snap ceiling are both model-relative (#1399/#1780),
	// from the two operands' combined extent.
	res := geom.ResolutionForBox(a.RangeBox().Union(b.RangeBox()))
	rc, snap := steinmetzSnapRadius(ca.Radius, cb.Radius, res)
	if !snap {
		return math.Point3{}, math.Vector3{}, math.Vector3{}, 0, false
	}
	dirA, dirB = ca.AxisDir.AsVector(), cb.AxisDir.AsVector()
	if !math.IsNearZero(dirA.Dot(dirB), 1e-7) { // tol:angular — perpendicular gate, kept isolated from the radius snap (#1780)
		return math.Point3{}, math.Vector3{}, math.Vector3{}, 0, false
	}
	lineA, _ := geom.NewLine(ca.Origin, dirA)
	lineB, _ := geom.NewLine(cb.Origin, dirB)
	o, ok = geom.LineLineIntersection(lineA, lineB, res.Plane())
	if !ok || !axisReachesR(ca, baseA, hA, o) || !axisReachesR(cb, baseB, hB, o) {
		return math.Point3{}, math.Vector3{}, math.Vector3{}, 0, false
	}
	return o, dirA, dirB, rc, true
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
