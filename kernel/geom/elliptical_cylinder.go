// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// EllipticalCylinder is an infinite cylinder of elliptical cross-section (contract:
// EllipticalCylinder) about the axis line through Origin along AxisDir. Ref is the in-plane
// major-axis direction (perpendicular to AxisDir) with semi-axis MajorRadius; the minor
// semi-axis MinorRadius runs along AxisDir×Ref. Parameters are (u = angle in [0,2π], v =
// signed distance along the axis):
// P(u,v) = Origin + v·AxisDir + MajorRadius·cos u·Ref + MinorRadius·sin u·(AxisDir×Ref).
type EllipticalCylinder struct {
	Origin      math.Point3
	AxisDir     math.UnitVector3
	Ref         math.UnitVector3
	MajorRadius float64
	MinorRadius float64
	binormal    math.Vector3
}

// NewEllipticalCylinder builds an elliptical cylinder from an axis point and direction, the
// major-axis direction (projected onto the plane perpendicular to the axis), and the two
// semi-radii. Errors on a zero axis direction.
func NewEllipticalCylinder(origin math.Point3, axisDir, majorAxis math.Vector3, majorR, minorR float64) (EllipticalCylinder, error) {
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return EllipticalCylinder{}, err
	}
	ref := planarRef(a, majorAxis)
	return EllipticalCylinder{
		Origin: origin, AxisDir: a, Ref: ref, MajorRadius: majorR, MinorRadius: minorR,
		binormal: a.Cross(ref),
	}, nil
}

// PointAt returns the point at (u, v).
func (c EllipticalCylinder) PointAt(u, v float64) math.Point3 {
	cos, sin := cosSin(u)
	radial := c.Ref.AsVector().Scale(c.MajorRadius * cos).Add(c.binormal.Scale(c.MinorRadius * sin))
	return c.Origin.TranslateBy(c.AxisDir.AsVector().Scale(v)).TranslateBy(radial)
}

// DerivativesAt returns ∂P/∂u (around the ellipse) and ∂P/∂v (the axis direction).
func (c EllipticalCylinder) DerivativesAt(u, _ float64) (du, dv math.Vector3) {
	cos, sin := cosSin(u)
	du = c.Ref.AsVector().Scale(-c.MajorRadius * sin).Add(c.binormal.Scale(c.MinorRadius * cos))
	return du, c.AxisDir.AsVector()
}

// NormalAt returns the outward unit normal (du×dv normalized).
func (c EllipticalCylinder) NormalAt(u, v float64) math.Vector3 {
	du, dv := c.DerivativesAt(u, v)
	return normalFromPartials(du, dv)
}

// UDomain returns the periodic angular range [0, 2π].
func (c EllipticalCylinder) UDomain() (lo, hi float64) { return fullCircleDomain() }

// VDomain returns the unbounded axial range.
func (c EllipticalCylinder) VDomain() (lo, hi float64) { return unboundedDomain() }

// ParamAt inverts PointAt: the angle of the ellipse and the signed distance along the
// axis. The radii rescale the projected components before the atan2 so an on-surface point
// reproduces its u exactly.
func (c EllipticalCylinder) ParamAt(q math.Point3) (u, v float64) {
	d := c.Origin.VectorTo(q)
	v = d.Dot(c.AxisDir.AsVector())
	r := d.Sub(c.AxisDir.AsVector().Scale(v))
	return wrap2pi(stdmath.Atan2(r.Dot(c.binormal)/c.MinorRadius, r.Dot(c.Ref.AsVector())/c.MajorRadius)), v
}

var _ Surface = EllipticalCylinder{}

// conjugateDegenerateTol is the relative floor on minorR/majorR below which a conjugate-diameter
// cross-section counts as degenerate (a swept profile grazing its extrusion direction, collapsing
// the perpendicular section to a line). It compares the two principal radii, so it is scale-free and
// needs no model-scale input (ADR-0042): the ratio is dimensionless.
const conjugateDegenerateTol = 1e-9

// NewEllipticalCylinderFromConjugate builds a right elliptical cylinder about the line through origin
// along axisDir whose perpendicular cross-section is the ellipse spanned by the CONJUGATE semi-
// diameters u1,u2 (both must lie in the plane perpendicular to axisDir — callers obtain them by
// projecting the swept profile there). This is the exact elementarisation of a STEP
// SURFACE_OF_LINEAR_EXTRUSION of a conic swept obliquely: an oblique cylinder over a conic IS a right
// elliptical cylinder, whose section is the profile's oblique projection (two conjugate diameters).
//
// Example:
//
//	// ellipse (a,b along m,w) swept along d; project the axes perpendicular to d:
//	u1 := m.Sub(d.Scale(m.Dot(d))).Scale(a)
//	u2 := w.Sub(d.Scale(w.Dot(d))).Scale(b)
//	cyl, err := NewEllipticalCylinderFromConjugate(center, d, u1, u2)
func NewEllipticalCylinderFromConjugate(origin math.Point3, axisDir, u1, u2 math.Vector3) (EllipticalCylinder, error) {
	majorR, minorR, majorDir, err := principalAxesFromConjugate(u1, u2)
	if err != nil {
		return EllipticalCylinder{}, err
	}
	return NewEllipticalCylinder(origin, axisDir, majorDir, majorR, minorR)
}

// principalAxesFromConjugate returns the principal semi-radii (majorR ≥ minorR) and the major-axis
// direction (length majorR, not normalized — NewEllipticalCylinder re-projects it) of the ellipse
// whose boundary is p(θ)=cosθ·u1+sinθ·u2 for conjugate semi-diameters u1,u2. Closed-form basis-free
// 2×2 symmetric eigen-solution: with A=u1·u1, B=u1·u2, C=u2·u2, the extremal angle is
// θ*=½·atan2(2B, A−C); the principal directions/radii are p(θ*) and p(θ*+π/2). atan2 (never
// atan(2B/(A−C))) keeps the branch total through A=C and the B=0 circle-section case (Rytz's
// construction would divide by zero there). Errors when the section collapses toward a line.
func principalAxesFromConjugate(u1, u2 math.Vector3) (majorR, minorR float64, majorDir math.Vector3, err error) {
	a := u1.Dot(u1)
	b := u1.Dot(u2)
	c := u2.Dot(u2)
	theta := 0.5 * stdmath.Atan2(2*b, a-c)
	cos, sin := stdmath.Cos(theta), stdmath.Sin(theta)
	vMaj := u1.Scale(cos).Add(u2.Scale(sin))
	vMin := u1.Scale(-sin).Add(u2.Scale(cos))
	majorR, minorR, majorDir = vMaj.Length(), vMin.Length(), vMaj
	if minorR > majorR { // atan2 picks an extremum, not necessarily the max — order them
		majorR, minorR, majorDir = minorR, majorR, vMin
	}
	if majorR == 0 || minorR < conjugateDegenerateTol*majorR {
		return 0, 0, math.Vector3{}, fmt.Errorf(
			"geom: degenerate elliptical-cylinder section: principal semi-radii (%g, %g) from conjugate diameters |u1|=%g |u2|=%g (extrusion grazes the profile plane)",
			majorR, minorR, stdmath.Sqrt(a), stdmath.Sqrt(c))
	}
	return majorR, minorR, majorDir, nil
}
