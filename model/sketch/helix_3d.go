// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// HelicalCurve3D is a 3D-sketch helical curve (M22-F04): the canonical sweep path for
// threads, springs and coils. It is a thin model wrapper over kernel/geom.Helix3d,
// anchored at a constrainable Origin point on its axis. The origin's three coordinates
// and the start radius are solver DOFs; the pitch, turn count, radial growth, axis and
// handedness are definitional (set from the coil dialog / parameters, not solved).
type HelicalCurve3D struct {
	entityBase
	Origin        *Point3D
	Axis          math.UnitVector3
	StartRadius   math.Scalar
	AxialPerTurn  float64 // pitch: axial rise per revolution
	RadialPerTurn float64 // radial growth per revolution (0 ⇒ constant-radius helix)
	Turns         float64 // number of revolutions (> 0)
	Clockwise     bool
}

// scalarDOFs makes HelicalCurve3D a [scalar3DContributor]: its start radius is a free DOF
// beyond the origin point's three.
func (h *HelicalCurve3D) scalarDOFs() []*math.Scalar { return []*math.Scalar{&h.StartRadius} }

// Curve builds the kernel helix for this entity (RefDir is an arbitrary in-plane axis
// perpendicular to the helix axis), or an error if the axis/turns are degenerate.
func (h *HelicalCurve3D) Curve() (geom.Helix3d, error) {
	refDir := perpendicularTo(h.Axis)
	return geom.NewHelix3d(
		h.Origin.Position(), h.Axis.AsVector(), refDir,
		float64(h.StartRadius), h.AxialPerTurn, h.RadialPerTurn, h.Turns, h.Clockwise,
	)
}

// Height returns the helix's total axial rise (pitch × turns).
func (h *HelicalCurve3D) Height() float64 { return h.AxialPerTurn * h.Turns }

// AddHelix3D adds a helical curve anchored at origin, winding about axis. axialPerTurn is
// the pitch, radialPerTurn the per-revolution radial growth (0 for a cylindrical helix,
// nonzero for a tapered helix or — with axialPerTurn 0 — a flat spiral), over turns
// revolutions.
func (s *Sketch3D) AddHelix3D(origin math.Point3, axis math.UnitVector3, startRadius, axialPerTurn, radialPerTurn, turns float64, clockwise bool) *HelicalCurve3D {
	h := &HelicalCurve3D{
		entityBase: newEntity(), Origin: s.newPoint3D(origin), Axis: axis,
		StartRadius: math.Scalar(startRadius), AxialPerTurn: axialPerTurn,
		RadialPerTurn: radialPerTurn, Turns: turns, Clockwise: clockwise,
	}
	s.addEntity3D(h)
	return h
}

// addHelix3DPt builds a helix over an existing origin point (the restore seam).
func (s *Sketch3D) addHelix3DPt(origin *Point3D, axis math.UnitVector3, startRadius, axialPerTurn, radialPerTurn, turns float64, clockwise bool) *HelicalCurve3D {
	h := &HelicalCurve3D{
		entityBase: newEntity(), Origin: origin, Axis: axis,
		StartRadius: math.Scalar(startRadius), AxialPerTurn: axialPerTurn,
		RadialPerTurn: radialPerTurn, Turns: turns, Clockwise: clockwise,
	}
	s.addEntity3D(h)
	return h
}

// perpendicularTo returns an arbitrary unit vector perpendicular to u (the helix's angle-0
// reference direction). It picks the world axis least aligned with u to stay well-conditioned.
func perpendicularTo(u math.UnitVector3) math.Vector3 {
	seed := math.V3(1, 0, 0)
	if absScalar(u.X()) > 0.9 {
		seed = math.V3(0, 1, 0)
	}
	perp := seed.Sub(u.AsVector().Scale(seed.Dot(u.AsVector())))
	return perp
}

// absScalar returns the absolute value of a scalar.
func absScalar(s math.Scalar) math.Scalar {
	if s < 0 {
		return -s
	}
	return s
}
