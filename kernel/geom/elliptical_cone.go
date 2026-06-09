// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// EllipticalCone is a cone of elliptical cross-section (contract: EllipticalCone) with its
// Apex on the axis along AxisDir. Ref is the in-plane major-axis direction (perpendicular
// to AxisDir); MajorAngle and MinorAngle are the half-angles (radians) between the axis and
// the surface along the major and minor directions. At axial distance v the semi-axes are
// v·tan(MajorAngle) and v·tan(MinorAngle). Parameters are (u = angle in [0,2π], v ≥ 0):
// P(u,v) = Apex + v·AxisDir + v·tan(MajorAngle)·cos u·Ref + v·tan(MinorAngle)·sin u·(AxisDir×Ref).
type EllipticalCone struct {
	Apex       math.Point3
	AxisDir    math.UnitVector3
	Ref        math.UnitVector3
	MajorAngle float64
	MinorAngle float64
	binormal   math.Vector3
}

// NewEllipticalCone builds an elliptical cone from an apex, axis direction, the major-axis
// direction (projected onto the plane perpendicular to the axis), and the two half-angles.
// Errors on a zero axis or a half angle outside the open interval (0, π/2).
func NewEllipticalCone(apex math.Point3, axisDir, majorAxis math.Vector3, majorAngle, minorAngle float64) (EllipticalCone, error) {
	for _, ang := range [2]float64{majorAngle, minorAngle} {
		if ang <= 0 || ang >= stdmath.Pi/2 {
			return EllipticalCone{}, fmt.Errorf("geom: elliptical cone half angle %g out of range (0, π/2)", ang)
		}
	}
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return EllipticalCone{}, err
	}
	ref := planarRef(a, majorAxis)
	return EllipticalCone{Apex: apex, AxisDir: a, Ref: ref, MajorAngle: majorAngle, MinorAngle: minorAngle, binormal: a.Cross(ref)}, nil
}

// PointAt returns the point at (u, v).
func (c EllipticalCone) PointAt(u, v float64) math.Point3 {
	cos, sin := cosSin(u)
	rMaj, rMin := v*stdmath.Tan(c.MajorAngle), v*stdmath.Tan(c.MinorAngle)
	radial := c.Ref.AsVector().Scale(rMaj * cos).Add(c.binormal.Scale(rMin * sin))
	return c.Apex.TranslateBy(c.AxisDir.AsVector().Scale(v)).TranslateBy(radial)
}

// DerivativesAt returns ∂P/∂u (around the ellipse) and ∂P/∂v (along the slant).
func (c EllipticalCone) DerivativesAt(u, v float64) (du, dv math.Vector3) {
	cos, sin := cosSin(u)
	tMaj, tMin := stdmath.Tan(c.MajorAngle), stdmath.Tan(c.MinorAngle)
	du = c.Ref.AsVector().Scale(-v * tMaj * sin).Add(c.binormal.Scale(v * tMin * cos))
	dv = c.AxisDir.AsVector().Add(c.Ref.AsVector().Scale(tMaj * cos)).Add(c.binormal.Scale(tMin * sin))
	return du, dv
}

// NormalAt returns the outward unit normal (du×dv normalized); it is the zero vector at the
// apex, where the partials degenerate.
func (c EllipticalCone) NormalAt(u, v float64) math.Vector3 {
	du, dv := c.DerivativesAt(u, v)
	return normalFromPartials(du, dv)
}

// UDomain returns the periodic angular range [0, 2π].
func (c EllipticalCone) UDomain() (lo, hi float64) { return fullCircleDomain() }

// VDomain returns the half-open axial range [0, +Inf).
func (c EllipticalCone) VDomain() (lo, hi float64) { return 0, stdmath.Inf(1) }

// ParamAt inverts PointAt: the angle of the ellipse and the distance from the apex along
// the axis. Dividing the projected components by the per-direction tangents (the v factor
// cancels in the ratio) reproduces an on-surface point's u exactly.
func (c EllipticalCone) ParamAt(q math.Point3) (u, v float64) {
	d := c.Apex.VectorTo(q)
	v = d.Dot(c.AxisDir.AsVector())
	r := d.Sub(c.AxisDir.AsVector().Scale(v))
	uMaj := r.Dot(c.Ref.AsVector()) / stdmath.Tan(c.MajorAngle)
	uMin := r.Dot(c.binormal) / stdmath.Tan(c.MinorAngle)
	return wrap2pi(stdmath.Atan2(uMin, uMaj)), v
}

var _ Surface = EllipticalCone{}
