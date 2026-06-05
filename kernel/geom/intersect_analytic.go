// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// IntersectSurfacesAnalytic returns the EXACT intersection curves of two analytic surfaces
// for the pairs the curved-face boolean (K1b) relies on: plane∩plane → a line; plane∩
// cylinder (plane ⟂ axis) → a circle; plane∩sphere → a circle. handled is false for the
// pairs it does not solve in closed form (an oblique plane∩cylinder ellipse, cylinder∩
// cylinder, anything with a torus) — the caller falls back to the numeric tracer
// [IntersectSurfaceSurface]. An empty result with handled==true means the surfaces are
// known not to cross (parallel planes, a sphere clear of the plane, a tangent touch).
//
// Returning exact curves (a real circle, not a 64-gon) is what lets a drilled hole be a
// single cylindrical face with a stable reference key, rather than faceted soup.
func IntersectSurfacesAnalytic(a, b Surface) (curves []Curve3, handled bool) {
	if pl, ok := a.(Plane); ok {
		return intersectPlaneSurface(pl, b)
	}
	if pl, ok := b.(Plane); ok {
		return intersectPlaneSurface(pl, a)
	}
	return nil, false
}

func intersectPlaneSurface(pl Plane, other Surface) ([]Curve3, bool) {
	switch o := other.(type) {
	case Plane:
		return planePlaneCurve(pl, o)
	case Cylinder:
		return planeCylinderCurve(pl, o)
	case Sphere:
		return planeSphereCurve(pl, o)
	case Cone:
		return planeConeCurve(pl, o)
	default:
		return nil, false
	}
}

// planePlaneCurve returns the intersection line of two planes (empty when parallel).
func planePlaneCurve(a, b Plane) ([]Curve3, bool) {
	na, nb := unitVec3(a.Normal()), unitVec3(b.Normal())
	dir := na.Cross(nb)
	if dir.Length() < 1e-9 {
		return nil, true // parallel or coincident: no isolated intersection curve
	}
	// A point on both planes (Goldman): p = (da·(nb×dir) + db·(dir×na)) / |dir|².
	da := na.Dot(a.Origin.AsVector())
	db := nb.Dot(b.Origin.AsVector())
	num := nb.Cross(dir).Scale(da).Add(dir.Cross(na).Scale(db))
	k := 1 / dir.LengthSquared()
	p := math.P3(num.X*math.Scalar(k), num.Y*math.Scalar(k), num.Z*math.Scalar(k))
	ln, err := NewLine(p, dir)
	if err != nil {
		return nil, true
	}
	return []Curve3{ln}, true
}

// planeCylinderCurve returns the conic a plane cuts from a cylinder: a circle when the
// plane is perpendicular to the axis, an ellipse when oblique. A plane parallel to the axis
// (a line pair) is left to the numeric tracer.
func planeCylinderCurve(pl Plane, cyl Cylinder) ([]Curve3, bool) {
	n := unitVec3(pl.Normal())
	axis := cyl.AxisDir.AsVector()
	cosA := float64(axis.Dot(n)) // both unit; |cosA| = cos∠(axis, plane normal)
	abs := stdmath.Abs(cosA)
	if abs < 1e-6 { // axis ∥ plane → 0/1/2 lines, not a conic
		return nil, false
	}
	t := n.Dot(cyl.Origin.VectorTo(pl.Origin)) / axis.Dot(n)
	center := cyl.Origin.TranslateBy(axis.Scale(t))
	if abs > 1-1e-6 { // perpendicular → circle
		c, err := NewCircle(center, axis, cyl.Radius)
		if err != nil {
			return nil, true
		}
		return []Curve3{c}, true
	}
	// Oblique → ellipse: minor radius = r, major = r/|cosA|; major axis is the cylinder
	// axis projected into the plane.
	majorDir := axis.Add(n.Scale(math.Scalar(-cosA)))
	e, err := NewEllipseFull(center, n, majorDir, cyl.Radius/abs, cyl.Radius)
	if err != nil {
		return nil, true
	}
	return []Curve3{e}, true
}

// planeConeCurve returns the circle where a plane perpendicular to a cone's axis cuts it
// (radius grows with distance from the apex); an oblique plane (ellipse/parabola/hyperbola)
// is left to the numeric tracer, and a plane through the apex yields a point (no curve).
func planeConeCurve(pl Plane, cone Cone) ([]Curve3, bool) {
	n := unitVec3(pl.Normal())
	axis := cone.AxisDir.AsVector()
	denom := axis.Dot(n)
	if stdmath.Abs(float64(denom)) < 1-1e-6 { // not perpendicular → conic section, defer
		return nil, false
	}
	t := n.Dot(cone.Apex.VectorTo(pl.Origin)) / denom
	r := stdmath.Abs(float64(t)) * stdmath.Tan(cone.HalfAngle)
	if r < 1e-9 { // plane through the apex
		return nil, true
	}
	center := cone.Apex.TranslateBy(axis.Scale(t))
	c, err := NewCircle(center, axis, r)
	if err != nil {
		return nil, true
	}
	return []Curve3{c}, true
}

// planeSphereCurve returns the circle where a plane cuts a sphere (empty when the plane
// misses or merely touches the sphere).
func planeSphereCurve(pl Plane, sp Sphere) ([]Curve3, bool) {
	n := unitVec3(pl.Normal())
	d := float64(n.Dot(pl.Origin.VectorTo(sp.Center))) // signed distance, center to plane
	if stdmath.Abs(d) >= sp.Radius-1e-9 {
		return nil, true // clear of, or tangent to, the plane: no circle
	}
	r := stdmath.Sqrt(sp.Radius*sp.Radius - d*d)
	center := sp.Center.TranslateBy(n.Scale(math.Scalar(-d))) // foot of the center on the plane
	c, err := NewCircle(center, n, r)
	if err != nil {
		return nil, true
	}
	return []Curve3{c}, true
}

// unitVec3 normalizes v (returning it unchanged if degenerate).
func unitVec3(v math.Vector3) math.Vector3 {
	if l := float64(v.Length()); l > 0 {
		return v.Scale(math.Scalar(1 / l))
	}
	return v
}
