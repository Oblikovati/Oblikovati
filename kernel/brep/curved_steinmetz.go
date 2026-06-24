// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Equal-radius Steinmetz intersection (M2 Phase 2, Oblikovati/Oblikovati#1335). Two EQUAL-radius
// perpendicular cylinders crossing through a common axis point intersect in the Steinmetz bicylinder. This
// is the degenerate case the SSI imprint tracer cannot trace: the intersection curve is NOT a quartic
// saddle but two planar ELLIPSES that cross at two pinch points (where the tracer's continuation stalls),
// so it is fitted analytically here instead.
//
// Geometry in the frame (a = axis A, b = axis B, n = a×b) through the axis crossing O, with equal radius R:
// a surface point O + α·a + β·b + γ·n is on cyl A when β²+γ²=R² and on cyl B when α²+γ²=R², so on both when
// β=±α. The two intersection ellipses therefore lie in the planes β=α and β=−α; they meet only where
// α=β=0, i.e. at the PINCH points O±R·n. The bicylinder boundary is four cylinder patches (two lobes of
// each cylinder, split at the pinch points), each bounded by one arc of each ellipse, all four meeting at
// the two pinch vertices. Every lobe carries its cylinder's natural outward normal (the solid is inside
// both cylinders), and every elliptical arc is shared by one A-lobe and one B-lobe in opposite orientation,
// so the result is a closed manifold solid of exactly four analytic faces.

// steinLin tags the assembled Steinmetz body's topology (one entity per role, so the index is always 0).
func steinLin(role string) topo.Lineage { return topo.NewLineage(topo.Tok("steinmetz", role, 0)) }

// EqualRadiusSteinmetzIntersect builds the exact intersection of two equal-radius perpendicular cylinders
// (the Steinmetz bicylinder), or ok=false when the configuration is outside that case (unequal radii,
// non-perpendicular or non-intersecting axes, or a cylinder too short to contain the bicylinder) so
// kernel/ops keeps its CSG fallback.
//
// Example — two radius-3 cylinders on x and z crossing at the origin gives the four-face bicylinder:
//
//	cx, _ := brep.SolidCylinder(math.P3(-6,0,0), math.V3(1,0,0), 3, 12)
//	cz, _ := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	res, ok := brep.EqualRadiusSteinmetzIntersect(cx, cz)
func EqualRadiusSteinmetzIntersect(a, b *topo.Body) (*topo.Body, bool) {
	o, dirA, dirB, r, ok := steinmetzFrame(a, b)
	if !ok {
		return nil, false
	}
	return buildSteinmetz(o, dirA, dirB, r), true
}

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
	if !math.IsNearZero(dirA.Dot(dirB), 1e-7) { // axes must be perpendicular
		return math.Point3{}, math.Vector3{}, math.Vector3{}, 0, false
	}
	lineA, _ := geom.NewLine(ca.Origin, dirA)
	lineB, _ := geom.NewLine(cb.Origin, dirB)
	o, ok = geom.LineLineIntersection(lineA, lineB, 1e-6)
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
	return vO-c.Radius >= vBase-1e-9 && vO+c.Radius <= vBase+height+1e-9
}

// buildSteinmetz welds the four cylinder lobes of the bicylinder. The four elliptical arcs (two ellipses,
// each split at the pinch vertices into a front and a back arc) are created once and shared: every arc is
// used by one A-lobe and one B-lobe in opposite orientation, so every edge is used exactly twice. Each lobe
// cylinder is framed with its reference pointing at the lobe centre, so the lobe is a contractible patch
// (it never wraps the seam) that the trimmed-patch mesher handles directly.
func buildSteinmetz(o math.Point3, dirA, dirB math.Vector3, r float64) *topo.Body {
	n := dirA.Cross(dirB)
	bld := topo.NewBuilder(true, steinLin("body"))
	vMinus := bld.AddVertex(o.TranslateBy(n.Scale(math.Scalar(-r))), steinLin("pinchlo"))
	vPlus := bld.AddVertex(o.TranslateBy(n.Scale(math.Scalar(r))), steinLin("pinchhi"))
	ePF, ePB, eMF, eMB := steinmetzArcs(o, dirA, dirB, r)
	e1 := bld.AddEdge(ePF, vMinus, vPlus, steinLin("e+front"))    // E+ front: P− → P+
	e2 := bld.AddEdge(ePB, vPlus, vMinus, steinLin("e+back"))     // E+ back:  P+ → P−
	e3 := bld.AddEdge(eMF, vMinus, vPlus, steinLin("e-front"))    // E− front: P− → P+
	e4 := bld.AddEdge(eMB, vPlus, vMinus, steinLin("e-back"))     // E− back:  P+ → P−
	cA1, _ := geom.NewCylinderWithRef(o, dirA, dirB, r)           // A lobe on +b
	cA2, _ := geom.NewCylinderWithRef(o, dirA, dirB.Scale(-1), r) // A lobe on −b
	cB1, _ := geom.NewCylinderWithRef(o, dirB, dirA, r)           // B lobe on +a
	cB2, _ := geom.NewCylinderWithRef(o, dirB, dirA.Scale(-1), r) // B lobe on −a
	bld.AddFace(cA1, steinLin("a1"), topo.OuterLoop(topo.Fwd(e1), topo.Fwd(e4)))
	bld.AddFace(cA2, steinLin("a2"), topo.OuterLoop(topo.Rev(e2), topo.Rev(e3)))
	bld.AddFace(cB1, steinLin("b1"), topo.OuterLoop(topo.Rev(e1), topo.Fwd(e3)))
	bld.AddFace(cB2, steinLin("b2"), topo.OuterLoop(topo.Rev(e4), topo.Fwd(e2)))
	return bld.Build()
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
	return stdmath.Abs(x-y) <= 1e-7*(1+stdmath.Max(stdmath.Abs(x), stdmath.Abs(y)))
}
