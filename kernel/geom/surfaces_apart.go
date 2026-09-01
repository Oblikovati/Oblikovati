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
		minor, err := math.UnitVector3FromVector(x.Normal.AsVector().Cross(x.RefDir.AsVector()))
		if err != nil {
			return ConicForm{}, false
		}
		return ConicForm{Center: x.Center, Major: x.RefDir, Minor: minor, A: x.Radius, B: x.Radius}, true
	case EllipseFull:
		minor, err := math.UnitVector3FromVector(x.Normal.AsVector().Cross(x.MajorAxis.AsVector()))
		if err != nil {
			return ConicForm{}, false
		}
		return ConicForm{Center: x.Center, Major: x.MajorAxis, Minor: minor,
			A: x.MajorRadius, B: x.MinorRadius}, true
	case Hyperbola:
		return ConicForm{Center: x.Center, Major: x.TransverseAxis, Minor: x.ConjugateAxis,
			A: x.A, B: x.B, Hyperbolic: true}, true
	}
	return ConicForm{}, false
}

// AxialAmplitude is the conic's half-extent along axis about its centre — how far the curve reaches
// up and down a wall band. A closed conic in a plane perpendicular to the axis gives 0.
//
// A hyperbola branch is UNBOUNDED and has no such extent; it returns 0, which callers must read as
// "no amplitude" rather than "flat". That is sound where it is used: an unbounded curve with any
// point inside a bounded trim must cross that trim's boundary, so the exact crossing scan decides
// the verdict before any amplitude is consulted.
func (c ConicForm) AxialAmplitude(axis math.Vector3) float64 {
	if c.Hyperbolic {
		return 0
	}
	a := c.A * float64(c.Major.AsVector().Dot(axis))
	b := c.B * float64(c.Minor.AsVector().Dot(axis))
	return stdmath.Sqrt(a*a + b*b)
}
