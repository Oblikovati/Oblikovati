// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// SurfacesApart reports whether two surfaces are separated EVERYWHERE by more than gap — so no
// patch of one can touch any patch of the other, whatever their trims.
//
// It is a statement about the SURFACES, which is what makes it useful: proving two trimmed faces
// clear normally means intersecting their surfaces and testing the curves against both trims, but
// when the surfaces never meet at all there is nothing to test. A boolean's face-pair gate can
// take that answer directly instead of declining because it cannot reason about a curved trim
// (#3459).
//
// It is exact and one-sided: true means PROVEN apart, false means "no proof", never "they touch".
// The pairs it proves are the ones whose separation is constant in closed form — parallel planes,
// coaxial cylinders, and coaxial cones of equal half-angle. Anything else (including a cylinder
// and a coaxial cone, which meet in a circle) returns false.
//
// Example:
//
//	if geom.SurfacesApart(seat, host, tol) { /* the patches cannot touch */ }
func SurfacesApart(a, b Surface, gap float64) bool {
	switch x := a.(type) {
	case Plane:
		y, ok := b.(Plane)
		return ok && parallelPlanesApart(x, y, gap)
	case Cylinder:
		y, ok := b.(Cylinder)
		return ok && coaxialCylindersApart(x, y, gap)
	case Cone:
		y, ok := b.(Cone)
		return ok && parallelConesApart(x, y, gap)
	}
	return false
}

// parallelPlanesApart: two parallel planes are everywhere |d| apart, where d is the offset along
// the shared normal. Non-parallel planes always meet in a line.
func parallelPlanesApart(a, b Plane, gap float64) bool {
	n1, n2 := a.Normal(), b.Normal()
	if !parallelDirs(n1, n2) {
		return false
	}
	return stdmath.Abs(float64(a.Origin.VectorTo(b.Origin).Dot(n1))) > gap
}

// coaxialCylindersApart: two cylinders on the SAME axis line are everywhere |r1−r2| apart. Axes
// that are parallel but distinct, or skew, are not proven here — the surfaces can still be apart,
// but the separation is no longer this expression.
func coaxialCylindersApart(a, b Cylinder, gap float64) bool {
	if !parallelDirs(a.AxisDir.AsVector(), b.AxisDir.AsVector()) {
		return false
	}
	if !onAxisLine(a.Origin, a.AxisDir, b.Origin) {
		return false
	}
	return stdmath.Abs(a.Radius-b.Radius) > gap
}

// parallelConesApart: two cones sharing an axis LINE and a half-angle are translates of one
// another along that axis, so like parallel planes they never meet — and their constant
// separation is the apex offset resolved perpendicular to the surface, |Δ|·sin(halfAngle).
//
// This is the case an emboss pad on a chamfer cone produces: the pad's seat is the host cone sunk
// by the wrap sagitta, which is exactly an apex shift. Unequal half-angles are NOT proven — two
// coaxial cones of different angles meet in a circle.
func parallelConesApart(a, b Cone, gap float64) bool {
	if !parallelDirs(a.AxisDir.AsVector(), b.AxisDir.AsVector()) {
		return false
	}
	if stdmath.Abs(a.HalfAngle-b.HalfAngle) > coneAngleWeld {
		return false
	}
	if !onAxisLine(a.Apex, a.AxisDir, b.Apex) {
		return false
	}
	delta := stdmath.Abs(float64(a.Apex.VectorTo(b.Apex).Dot(a.AxisDir.AsVector())))
	return delta*stdmath.Sin(a.HalfAngle) > gap
}

// coneAngleWeld is how close two half-angles must be to count as the same cone family. It is an
// ANGLE on unit directions, so it carries no model scale.
const coneAngleWeld = 1e-9 // tol:angular — half-angle equality for the parallel-cone test

// parallelDirs reports whether two unit-length directions are parallel, either sense.
func parallelDirs(a, b math.Vector3) bool {
	return stdmath.Abs(stdmath.Abs(float64(a.Dot(b)))-1) <= coneAngleWeld
}

// onAxisLine reports whether p lies on the line through origin along dir — the test that two
// coaxial-looking surfaces really do share ONE axis rather than two parallel ones.
func onAxisLine(origin math.Point3, dir math.UnitVector3, p math.Point3) bool {
	v := origin.VectorTo(p)
	perp := v.Sub(dir.AsVector().Scale(v.Dot(dir.AsVector())))
	return float64(perp.Length()) <= ResolutionForSize(float64(v.Length())).Weld()
}

// ConicForm is a curve recognised as a conic, in the terms every consumer of one needs: its centre,
// its two principal directions, the matching semi-axes, and which of the two signed metrics it
// satisfies —
//
//	Hyperbolic false:  (p·Major/A)² + (p·Minor/B)² = 1   (circle, ellipse)
//	Hyperbolic true:   (p·Major/A)² − (p·Minor/B)² = 1   (one hyperbola BRANCH, Major-ward)
//
// One description for all of them is the point. A plane cuts a cone in a circle, an ellipse or a
// hyperbola branch depending only on how it is tilted, and code that clips or classifies that
// section should not care which it got — the kernel ground rules keep geometry-kind switches in
// this package, and this is the accessor that lets them.
type ConicForm struct {
	Center       math.Point3
	Major, Minor math.UnitVector3
	A, B         float64
	Hyperbolic   bool
}

// AsConic recognises the conics a plane∩quadric section can be. ok=false for anything else,
// including a parabola (no centre) and a bounded arc, which is not the whole conic.
//
// Example:
//
//	if cf, ok := geom.AsConic(section); ok && !cf.Hyperbolic { /* a closed section */ }
func AsConic(c Curve3) (ConicForm, bool) {
	switch x := c.(type) {
	case Circle:
		return circularForm(x.Center, x.Normal, x.RefDir, x.Radius)
	case Arc3d:
		// A circular ARC runs on a circle, and the conic it runs on is what a crossing solver or a
		// section plane needs (ADR-0060); its bounds are its parameterisation, as for the hyperbolic arc.
		return circularForm(x.Center, x.Normal, x.RefDir, x.Radius)
	case EllipseFull:
		return ellipticForm(x.Center, x.Normal, x.MajorAxis, x.MajorRadius, x.MinorRadius)
	case EllipticalArc:
		return ellipticForm(x.Center, x.Normal, x.MajorAxis, x.MajorRadius, x.MinorRadius)
	case Hyperbola:
		return ConicForm{Center: x.Center, Major: x.TransverseAxis, Minor: x.ConjugateAxis,
			A: x.A, B: x.B, Hyperbolic: true}, true
	case HyperbolicArc:
		// A bounded arc runs ON a hyperbola, and the conic it runs on is what a clipper needs. Its
		// own bounds are its parameterisation, not its shape.
		return ConicForm{Center: x.Center, Major: x.TransverseAxis, Minor: x.ConjugateAxis,
			A: x.A, B: x.B, Hyperbolic: true}, true
	}
	return ConicForm{}, false
}

// circularForm is the conic form of a circle: equal semi-axes along its reference direction and
// the in-plane direction a quarter turn on.
func circularForm(center math.Point3, normal, ref math.UnitVector3, r float64) (ConicForm, bool) {
	minor, err := math.UnitVector3FromVector(normal.AsVector().Cross(ref.AsVector()))
	if err != nil {
		return ConicForm{}, false
	}
	return ConicForm{Center: center, Major: ref, Minor: minor, A: r, B: r}, true
}

// ellipticForm is the conic form of an ellipse from its centre, plane normal, major axis and radii.
func ellipticForm(center math.Point3, normal, major math.UnitVector3, a, b float64) (ConicForm, bool) {
	minor, err := math.UnitVector3FromVector(normal.AsVector().Cross(major.AsVector()))
	if err != nil {
		return ConicForm{}, false
	}
	return ConicForm{Center: center, Major: major, Minor: minor, A: a, B: b}, true
}

// AxialAmplitude is the conic's half-extent along axis about its centre — how far the curve reaches
// up and down a wall band. A closed conic in a plane perpendicular to the axis gives 0.
//
// A hyperbola branch is UNBOUNDED along the axis, so it returns +Inf. It used to return 0 with a
// caveat that callers must read that as "no amplitude" rather than "flat" — and the caveat's own
// premise was wrong: a crossing scan decides the verdict FIRST only where the crossings fall inside
// the band being tested, and an arm can pass through the band well inside a large trim while crossing
// its boundary far outside. The mixed boolean then read amplitude 0 as a flat conic that clears every
// wall, so a plane sectioning a cone parallel to its axis dropped the tool's own face and left the cut
// open (ADR-0062). +Inf is the honest value and needs no reading.
func (c ConicForm) AxialAmplitude(axis math.Vector3) float64 {
	if c.Hyperbolic {
		return stdmath.Inf(1)
	}
	a := c.A * float64(c.Major.AsVector().Dot(axis))
	b := c.B * float64(c.Minor.AsVector().Dot(axis))
	return stdmath.Sqrt(a*a + b*b)
}

// ConicParamAt inverts a conic's own parameterisation: the parameter t with c.PointAt(t) == p, for
// a point on the curve. ok=false for a curve kind that is not a conic.
//
// Each conic answers in ITS own convention — a circle and an ellipse in the [0,1) their Domain
// reports, a hyperbola branch in the hyperbolic angle theta its Domain spans — because that is what
// a caller must hand back to PointAt. The inversion lives here with the curves rather than at the
// call site, so consumers ask a geometric question instead of switching on a geometry kind.
//
// The hyperbola inverts through ASINH, not acosh: eta = sinh(theta) is single-valued over the whole
// branch, while xi = cosh(theta) loses the sign of theta.
//
// Example:
//
//	if t, ok := geom.ConicParamAt(section, hit); ok { shared := section.PointAt(t) }
func ConicParamAt(c Curve3, p math.Point3) (float64, bool) {
	switch x := c.(type) {
	case Circle:
		return wrapUnit(circleAngleAt(x.Center, x.RefDir, x.Normal, p) / (2 * stdmath.Pi)), true
	case Arc3d:
		return arcParamAt(circleAngleAt(x.Center, x.RefDir, x.Normal, p), x.StartAngle, x.SweepAngle), true
	case EllipseFull:
		return ellipseParamAt(x, p), true
	case EllipticalArc:
		theta := ellipseAngleAt(x.Center, x.MajorAxis, x.Normal, x.MajorRadius, x.MinorRadius, p)
		return arcParamAt(theta, x.StartAngle, x.SweepAngle), true
	case Hyperbola:
		return hyperbolaTheta(x.Center, x.ConjugateAxis, x.B, p), true
	case HyperbolicArc:
		if x.Theta1 == x.Theta0 {
			return 0, false
		}
		theta := hyperbolaTheta(x.Center, x.ConjugateAxis, x.B, p)
		return (theta - x.Theta0) / (x.Theta1 - x.Theta0), true
	}
	return 0, false
}

// circleAngleAt is the polar angle of p about a circle's centre, from its reference direction.
func circleAngleAt(center math.Point3, ref, normal math.UnitVector3, p math.Point3) float64 {
	d := center.VectorTo(p)
	return stdmath.Atan2(float64(d.Dot(normal.Cross(ref))), float64(d.Dot(ref.AsVector())))
}

// ellipseAngleAt is the eccentric angle of p on an ellipse: each axis component scaled by its radius.
func ellipseAngleAt(center math.Point3, major, normal math.UnitVector3, a, b float64, p math.Point3) float64 {
	d := center.VectorTo(p)
	return stdmath.Atan2(float64(d.Dot(normal.Cross(major)))/b, float64(d.Dot(major.AsVector()))/a)
}

// ellipseParamAt inverts a full ellipse onto its [0,1) parameter.
func ellipseParamAt(e EllipseFull, p math.Point3) float64 {
	return wrapUnit(ellipseAngleAt(e.Center, e.MajorAxis, e.Normal, e.MajorRadius, e.MinorRadius, p) / (2 * stdmath.Pi))
}

// hyperbolaTheta inverts a hyperbola branch onto its hyperbolic angle through ASINH, which is
// single-valued over the whole branch — acosh would lose the sign of theta.
func hyperbolaTheta(center math.Point3, conjugate math.UnitVector3, b float64, p math.Point3) float64 {
	d := center.VectorTo(p)
	return stdmath.Asinh(float64(d.Dot(conjugate.AsVector())) / b)
}

// wrapUnit folds a real onto [0,1) — the domain a closed conic reports.
func wrapUnit(t float64) float64 {
	t -= stdmath.Floor(t)
	if t >= 1 {
		return 0
	}
	return t
}

// ConicSubArc restricts a conic to the parameter span [t0, t1] IN THAT CURVE'S OWN PARAMETER, and
// returns the bounded arc.
//
// The two hyperbola forms differ in what their parameter means — a Hyperbola's is the hyperbolic
// angle theta, a HyperbolicArc's is its own [0,1] — and getting that wrong stores an edge whose
// curve spans more than its two vertices. Asking here means a caller never has to know which form
// it holds.
//
// ok=false for a curve this does not bound (a closed conic, whose sub-arc has its own
// representation, or a non-conic).
//
// Example:
//
//	if arc, ok := geom.ConicSubArc(section, lo, hi); ok { edge.curve = arc }
func ConicSubArc(c Curve3, t0, t1 float64) (Curve3, bool) {
	switch x := c.(type) {
	case Hyperbola:
		return x.Arc(t0, t1), true
	case HyperbolicArc:
		span := x.Theta1 - x.Theta0
		return HyperbolicArc{
			Center: x.Center, TransverseAxis: x.TransverseAxis, ConjugateAxis: x.ConjugateAxis,
			A: x.A, B: x.B,
			Theta0: x.Theta0 + t0*span, Theta1: x.Theta0 + t1*span,
		}, true
	}
	return nil, false
}
