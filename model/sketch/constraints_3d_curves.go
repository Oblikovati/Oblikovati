// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// The curve-attachment 3D constraints of issue #142 (M22 F04/F05): SplineFitPoints3D
// ties external geometry to a fit spline's interpolation points, and Helical3D ties a
// helix to the circle it starts on. Both express intent the solver and the browser
// can name (like Concentric3D vs a plain coincidence).

// SplineFitPoints3D ties a 3D point to one fit point of a fit-type spline, so geometry
// chained to the spline stays attached when the spline re-solves — Inventor's
// spline-fit-point coincidence. The fit-point index is chosen at creation (nearest to
// the attached point) and is stable thereafter.
type SplineFitPoints3D struct {
	constraintBase
	Spline   *Spline3D
	P        *Point3D
	FitIndex int
}

// NewSplineFitPoints3D attaches p to the nearest fit point of sp. It rejects control
// splines (their points are not on the curve) and splines without points.
func NewSplineFitPoints3D(sp *Spline3D, p *Point3D) (*SplineFitPoints3D, error) {
	if !sp.IsFitType() {
		return nil, fmt.Errorf("sketch: splineFitPoints needs a fit (interpolating) spline, entity %d is a control spline", sp.EntityID())
	}
	if len(sp.Points) == 0 {
		return nil, fmt.Errorf("sketch: splineFitPoints: spline %d has no fit points", sp.EntityID())
	}
	return &SplineFitPoints3D{
		constraintBase: newConstraint(), Spline: sp, P: p, FitIndex: nearestFitIndex(sp, p),
	}, nil
}

// NewSplineFitPoints3DAt attaches p to the fit point at the given index — the
// deserializer's constructor, which must rebind the saved index, not re-derive it.
func NewSplineFitPoints3DAt(sp *Spline3D, p *Point3D, index int) (*SplineFitPoints3D, error) {
	if !sp.IsFitType() {
		return nil, fmt.Errorf("sketch: splineFitPoints needs a fit (interpolating) spline, entity %d is a control spline", sp.EntityID())
	}
	if index < 0 || index >= len(sp.Points) {
		return nil, fmt.Errorf("sketch: splineFitPoints index %d out of range (%d fit points)", index, len(sp.Points))
	}
	return &SplineFitPoints3D{constraintBase: newConstraint(), Spline: sp, P: p, FitIndex: index}, nil
}

// nearestFitIndex returns the index of the spline fit point closest to p.
func nearestFitIndex(sp *Spline3D, p *Point3D) int {
	best, bestD := 0, stdmath.Inf(1)
	for i, fp := range sp.Points {
		if d := float64(fp.Position().DistanceTo(p.Position())); d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// residualAD: v = [fitPoint.xyz, P.xyz]; the fit point and the attached point coincide.
func (c *SplineFitPoints3D) residualAD(v []ad.Number) []ad.Number {
	d := adV3(v, 0).Sub(adV3(v, 3))
	return []ad.Number{d.X, d.Y, d.Z}
}
func (c *SplineFitPoints3D) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *SplineFitPoints3D) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *SplineFitPoints3D) Variables() []*math.Scalar {
	fp := c.Spline.Points[c.FitIndex]
	return []*math.Scalar{&fp.X, &fp.Y, &fp.Z, &c.P.X, &c.P.Y, &c.P.Z}
}

// Helical3D ties a helix to the circle it starts on: the helix origin coincides with
// the circle's center and the start radius equals the circle's radius — the
// thread-on-cylinder-rim relation a coil path needs (M22-F04). The circle's axis is
// definitional on both sides (neither is a solver DOF), so orientation is not a
// residual; the constructor is where an axis mismatch surfaces.
type Helical3D struct {
	constraintBase
	H *HelicalCurve3D
	C *Circle3D
}

// NewHelical3D constrains h to start on circle c. It rejects a helix whose winding
// axis is not parallel to the circle's plane normal — the relation would be
// unsatisfiable by solving (neither axis is a DOF).
func NewHelical3D(h *HelicalCurve3D, c *Circle3D) (*Helical3D, error) {
	if !h.Axis.AsVector().IsParallelTo(c.Axis.AsVector(), 0) {
		return nil, fmt.Errorf("sketch: helical: helix axis %v is not parallel to circle axis %v", h.Axis, c.Axis)
	}
	return &Helical3D{constraintBase: newConstraint(), H: h, C: c}, nil
}

// residualAD: v = [H.Origin.xyz, H.StartRadius, C.Center.xyz, C.Radius]. The helix origin
// coincides with the circle center and the start radius equals the circle radius.
func (c *Helical3D) residualAD(v []ad.Number) []ad.Number {
	d := adV3(v, 0).Sub(adV3(v, 4))
	return []ad.Number{d.X, d.Y, d.Z, v[3].Sub(v[7])}
}
func (c *Helical3D) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *Helical3D) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *Helical3D) Variables() []*math.Scalar {
	return []*math.Scalar{
		&c.H.Origin.X, &c.H.Origin.Y, &c.H.Origin.Z, &c.H.StartRadius,
		&c.C.Center.X, &c.C.Center.Y, &c.C.Center.Z, &c.C.Radius,
	}
}
