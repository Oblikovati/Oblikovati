// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Angular tolerances for the analytic surface-intersection classifier. These compare unit
// vectors (cosine/sine of the angle between an axis and a plane normal), so they are
// DIMENSIONLESS and stay absolute — they do not scale with the model. The length tests in
// this file (clearance/grazing of a plane against a primitive radius) instead derive from a
// model-relative [Resolution] (res.Weld()), per Oblikovati/Oblikovati#1399.
const (
	// axisAlignCosTol classifies axis↔plane-normal alignment: |cos∠| < axisAlignCosTol means
	// the axis is parallel to the plane (axis ⟂ normal); |cos∠| > 1−axisAlignCosTol means the
	// axis is perpendicular to the plane (axis ∥ normal). Below this the case is "oblique".
	axisAlignCosTol = 1e-6 // tol:angular — cosine of the axis↔normal angle
	// coincidentNormalSinTol treats two plane normals whose cross-product magnitude (the sine
	// of the angle between them) is below this as parallel/coincident — no isolated section line.
	coincidentNormalSinTol = 1e-9 // tol:angular — sine of the angle between unit normals
)

// IntersectSurfacesAnalytic returns the EXACT intersection curves of two analytic surfaces
// for the pairs the curved-face boolean (K1b) relies on: plane∩plane → a line; plane∩
// cylinder (plane ⟂ axis) → a circle; plane∩sphere → a circle. handled is false for the
// pairs it does not solve in closed form (an oblique plane∩cylinder ellipse, cylinder∩
// cylinder, anything with a torus) — the caller falls back to the numeric tracer
// [IntersectSurfaceSurface]. An empty result with handled==true means the surfaces are
// known not to cross (parallel planes, a sphere clear of the plane, a tangent touch).
//
// res is the model-relative coincidence scale (Oblikovati/Oblikovati#1399): the clearance
// and grazing tests that decide "the plane just touches the primitive" use res.Weld() rather
// than a cm-anchored absolute epsilon, so a µm or km copy of the same part classifies the
// same way. Build it from the operands' bounding box: geom.ResolutionForBox(box).
//
// Returning exact curves (a real circle, not a 64-gon) is what lets a drilled hole be a
// single cylindrical face with a stable reference key, rather than faceted soup.
func IntersectSurfacesAnalytic(a, b Surface, res Resolution) (curves []Curve3, handled bool) {
	if pl, ok := a.(Plane); ok {
		return intersectPlaneSurface(pl, b, res)
	}
	if pl, ok := b.(Plane); ok {
		return intersectPlaneSurface(pl, a, res)
	}
	return nil, false
}

func intersectPlaneSurface(pl Plane, other Surface, res Resolution) ([]Curve3, bool) {
	switch o := other.(type) {
	case Plane:
		return planePlaneCurve(pl, o)
	case Cylinder:
		return planeCylinderCurve(pl, o, res)
	case Sphere:
		return planeSphereCurve(pl, o, res)
	case Cone:
		return planeConeCurve(pl, o, res)
	case Torus:
		return planeTorusCurve(pl, o, res)
	default:
		return nil, false
	}
}

// planeTorusCurve returns the section a plane PERPENDICULAR to the torus axis cuts: two concentric circles
// (the outer R+√(r²−d²) and inner R−√(r²−d²), at the section level d along the axis), or none when the
// plane clears or grazes the tube (|d|≥r). An OBLIQUE or axis-parallel plane cuts a quartic SPIRIC curve
// (no analytic conic), so it is not solved here — handled=false defers it to the CSG fallback. Returned
// outer first, then inner — both anchored to the torus Ref so their seam lines up with the band.
func planeTorusCurve(pl Plane, t Torus, res Resolution) ([]Curve3, bool) {
	n := unitVec3(pl.Normal())
	axis := t.AxisDir.AsVector()
	cosA := float64(axis.Dot(n))
	if stdmath.Abs(cosA) < 1-axisAlignCosTol {
		// Oblique / axis-parallel: a spiric quartic when it cuts the tube. But a plane that CLEARS the whole
		// torus (its distance exceeds the torus's reach along n) carries no section — report that (handled,
		// empty) so a box's far clearing faces compose; only a genuine spiric cut defers to CSG.
		reach := (t.MajorRadius+t.MinorRadius)*float64(n.Sub(axis.Scale(math.Scalar(cosA))).Length()) + t.MinorRadius*stdmath.Abs(cosA)
		if stdmath.Abs(float64(t.Center.VectorTo(pl.Origin).Dot(n))) >= reach-res.Weld() {
			return nil, true // the plane clears the torus
		}
		return nil, false // a spiric quartic cut, not an analytic conic
	}
	d := float64(t.Center.VectorTo(pl.Origin).Dot(axis)) // section level along the axis
	r := t.MinorRadius
	if stdmath.Abs(d) >= r-res.Weld() {
		return nil, true // the plane clears or grazes the tube: no crossing section
	}
	half := stdmath.Sqrt(r*r - d*d)
	level := t.Center.TranslateBy(axis.Scale(math.Scalar(d)))
	outer := Circle{Center: level, Normal: t.AxisDir, RefDir: t.Ref, Radius: t.MajorRadius + half}
	inner := Circle{Center: level, Normal: t.AxisDir, RefDir: t.Ref, Radius: t.MajorRadius - half}
	return []Curve3{outer, inner}, true
}

// PlanePlaneLine returns a point on, and the UNIT direction of, two planes' intersection line, or ok=false
// when they are parallel/coincident (no isolated line). It is the plane∩plane fast path of the general
// intersector, exposed so the planar B-rep imprint computes this surface pair ONCE here rather than
// re-deriving it: the kernel ground rule keeps SSI and geometry-kind knowledge in kernel/geom.
//
//	p0, dir, ok := geom.PlanePlaneLine(faceA.Plane(), faceB.Plane())
func PlanePlaneLine(a, b Plane) (p0 math.Point3, dir math.Vector3, ok bool) {
	curves, _ := planePlaneCurve(a, b)
	if len(curves) == 0 {
		return math.Point3{}, math.Vector3{}, false // parallel or coincident
	}
	ln := curves[0].(Line)
	return ln.Origin, ln.Dir.AsVector(), true
}

// planePlaneCurve returns the intersection line of two planes (empty when parallel).
func planePlaneCurve(a, b Plane) ([]Curve3, bool) {
	na, nb := unitVec3(a.Normal()), unitVec3(b.Normal())
	dir := na.Cross(nb)
	if dir.Length() < coincidentNormalSinTol {
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
// plane is perpendicular to the axis, an ellipse when oblique, and — when the plane is
// parallel to the axis — the pair of axis-parallel lines it grazes the cylinder along (none
// when it clears or is tangent to the cylinder). The axis-parallel pair is what lets a box
// wall be subtracted from a cylinder exactly (M2 Phase 1, Oblikovati/Oblikovati#1334).
func planeCylinderCurve(pl Plane, cyl Cylinder, res Resolution) ([]Curve3, bool) {
	n := unitVec3(pl.Normal())
	axis := cyl.AxisDir.AsVector()
	cosA := float64(axis.Dot(n)) // both unit; |cosA| = cos∠(axis, plane normal)
	abs := stdmath.Abs(cosA)
	if abs < axisAlignCosTol { // axis ∥ plane → 0/1/2 lines along the axis
		return cylinderAxisParallelLines(pl, cyl, n, axis, res)
	}
	t := n.Dot(cyl.Origin.VectorTo(pl.Origin)) / axis.Dot(n)
	center := cyl.Origin.TranslateBy(axis.Scale(t))
	if abs > 1-axisAlignCosTol { // perpendicular → circle
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

// cylinderAxisParallelLines returns the lines where a plane parallel to the cylinder axis grazes
// the cylinder: two when the plane cuts inside the radius, none when it clears or is tangent to it
// (a single tangent line carries no enclosed region, so the half-space cut keeps the cylinder
// whole). Both lines run along the axis; they are the section edges of a box-wall subtraction.
func cylinderAxisParallelLines(pl Plane, cyl Cylinder, n, axis math.Vector3, res Resolution) ([]Curve3, bool) {
	d := float64(n.Dot(pl.Origin.VectorTo(cyl.Origin))) // signed distance of the axis from the plane
	if stdmath.Abs(d) >= cyl.Radius-res.Weld() {
		return nil, true // plane clears or merely touches the cylinder: no line pair
	}
	half := stdmath.Sqrt(cyl.Radius*cyl.Radius - d*d) // half-chord of the cross-section
	tangent := unitVec3(n.Cross(axis))                // in-plane, perpendicular to the axis
	foot := cyl.Origin.TranslateBy(n.Scale(math.Scalar(-d)))
	var out []Curve3
	for _, s := range []float64{half, -half} {
		ln, err := NewLine(foot.TranslateBy(tangent.Scale(math.Scalar(s))), axis)
		if err != nil {
			return nil, true
		}
		out = append(out, ln)
	}
	return out, true
}

// planeConeCurve returns the conic a plane cuts from a cone: a circle when the plane is
// perpendicular to the axis (radius grows with distance from the apex), and — when the plane is
// PARALLEL to the axis — the hyperbola branch it cuts (Oblikovati/Oblikovati#1372, the section that
// lets a box face parallel to the cone axis be subtracted exactly). The intermediate oblique cases
// (an ellipse, a parabola, or a hyperbola from a plane steeper than the generators but not
// axis-parallel) are still left to the numeric tracer; a plane through the apex yields a point.
func planeConeCurve(pl Plane, cone Cone, res Resolution) ([]Curve3, bool) {
	n := unitVec3(pl.Normal())
	axis := cone.AxisDir.AsVector()
	along := stdmath.Abs(float64(axis.Dot(n)))
	if along < axisAlignCosTol { // plane ∥ axis → hyperbola
		return coneAxisParallelHyperbola(pl, cone, n, axis, res)
	}
	if along < 1-axisAlignCosTol { // oblique → ellipse / hyperbola (parabolic boundary deferred inside)
		return coneObliqueConic(pl, cone, n, axis)
	}
	t := n.Dot(cone.Apex.VectorTo(pl.Origin)) / axis.Dot(n)
	r := stdmath.Abs(float64(t)) * stdmath.Tan(cone.HalfAngle)
	if r < res.Weld() { // plane through the apex
		return nil, true
	}
	center := cone.Apex.TranslateBy(axis.Scale(t))
	c, err := NewCircle(center, axis, r)
	if err != nil {
		return nil, true
	}
	return []Curve3{c}, true
}

// coneAxisParallelHyperbola returns the hyperbola branch where a plane parallel to the cone axis
// cuts the cone. With the apex-to-plane signed distance D = n·(Origin − Apex) (n ⟂ axis here), the
// section is z²/A² − y²/B² = 1 in the plane: vertex on the axis side, transverse semi A = |D|/tanα
// along the axis (the branch climbs the cone as v=A·coshθ grows), conjugate semi B = |D| along
// a×n. Its centre sits at the foot of the apex on the plane, Apex + D·n. A plane through the apex
// (D≈0, the two coincident generators) is degenerate and deferred.
func coneAxisParallelHyperbola(pl Plane, cone Cone, n, axis math.Vector3, res Resolution) ([]Curve3, bool) {
	d := float64(n.Dot(cone.Apex.VectorTo(pl.Origin)))
	if stdmath.Abs(d) < res.Weld() {
		return nil, false // plane through the apex/axis: two coincident generators, not a hyperbola
	}
	conjugate := axis.Cross(n)
	tanA := stdmath.Tan(cone.HalfAngle)
	center := cone.Apex.TranslateBy(n.Scale(math.Scalar(d)))
	h, err := NewHyperbola(center, axis, conjugate, stdmath.Abs(d)/tanA, stdmath.Abs(d))
	if err != nil {
		return nil, true
	}
	return []Curve3{h}, true
}

// planeSphereCurve returns the circle where a plane cuts a sphere (empty when the plane
// misses or merely touches the sphere).
func planeSphereCurve(pl Plane, sp Sphere, res Resolution) ([]Curve3, bool) {
	n := unitVec3(pl.Normal())
	d := float64(n.Dot(pl.Origin.VectorTo(sp.Center))) // signed distance, center to plane
	if stdmath.Abs(d) >= sp.Radius-res.Weld() {
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
