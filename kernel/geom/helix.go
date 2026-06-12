// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Helix3d is a 3D helical curve (the canonical sweep path for threads, springs and
// coils). It winds about an axis through Origin in direction Axis, starting on the
// in-plane ray RefDir at StartRadius, advancing AxialPerTurn along the axis and growing
// RadialPerTurn outward per revolution, for Turns revolutions.
// every reference coil mode: a cylindrical helix (RadialPerTurn = 0), a conical/tapered
// helix (both nonzero), and a flat spiral (AxialPerTurn = 0). Clockwise flips the
// handedness about Axis. Parameterized t ∈ [0, 1] over the whole curve.
type Helix3d struct {
	Origin        math.Point3
	Axis          math.UnitVector3
	RefDir        math.UnitVector3 // unit, in the plane ⟂ Axis; marks angle 0
	StartRadius   float64
	AxialPerTurn  float64 // pitch: axial rise per revolution
	RadialPerTurn float64 // radial growth per revolution (0 ⇒ constant-radius helix)
	Turns         float64 // number of revolutions (> 0)
	Clockwise     bool    // handedness of the winding about Axis
}

// NewHelix3d builds a helix, projecting refDir onto the plane ⟂ axis and normalizing it
// (so RefDir is exactly perpendicular to Axis). It errors on a zero axis or a
// non-positive turn count (a helix with no revolutions is degenerate).
//
// Example — a 5-turn, 10 mm-pitch, 8 mm-radius right-handed spring about +Z:
//
//	z, _ := math.NewUnitVector3(0, 0, 1)
//	x, _ := math.NewUnitVector3(1, 0, 0)
//	h, _ := geom.NewHelix3d(math.P3(0,0,0), z.AsVector(), x.AsVector(), 0.8, 1.0, 0, 5, false)
func NewHelix3d(origin math.Point3, axis, refDir math.Vector3, startRadius, axialPerTurn, radialPerTurn, turns float64, clockwise bool) (Helix3d, error) {
	n, err := math.UnitVector3FromVector(axis)
	if err != nil {
		return Helix3d{}, err
	}
	if turns <= 0 {
		return Helix3d{}, &InvalidHelixError{Turns: turns}
	}
	return Helix3d{
		Origin: origin, Axis: n, RefDir: planarRef(n, refDir),
		StartRadius: startRadius, AxialPerTurn: axialPerTurn, RadialPerTurn: radialPerTurn,
		Turns: turns, Clockwise: clockwise,
	}, nil
}

// binormal returns Axis × RefDir, the in-plane unit vector at angle +π/2.
func (h Helix3d) binormal() math.Vector3 { return h.Axis.Cross(h.RefDir) }

// angleAt returns the signed winding angle at parameter t (negative when clockwise).
func (h Helix3d) angleAt(t float64) float64 {
	a := twoPi * h.Turns * t
	if h.Clockwise {
		return -a
	}
	return a
}

// radiusAt returns the radius at parameter t (StartRadius plus the per-turn growth).
func (h Helix3d) radiusAt(t float64) float64 {
	return h.StartRadius + h.RadialPerTurn*h.Turns*t
}

// heightAt returns the axial advance at parameter t.
func (h Helix3d) heightAt(t float64) float64 { return h.AxialPerTurn * h.Turns * t }

// PointAt returns the position at parameter t.
func (h Helix3d) PointAt(t float64) math.Point3 {
	p := pointOnCircle(h.Origin, h.RefDir.AsVector(), h.binormal(), h.radiusAt(t), h.angleAt(t))
	return p.TranslateBy(h.Axis.AsVector().Scale(math.Scalar(h.heightAt(t))))
}

// TangentAt returns the derivative dP/dt (the full chain: radial growth + winding +
// axial advance), unnormalized.
func (h Helix3d) TangentAt(t float64) math.Vector3 {
	ang := h.angleAt(t)
	r := h.radiusAt(t)
	cos, sin := cosSin(ang)
	ref, bin := h.RefDir.AsVector(), h.binormal()

	dRadius := h.RadialPerTurn * h.Turns
	dAngle := twoPi * h.Turns
	if h.Clockwise {
		dAngle = -dAngle
	}
	dHeight := h.AxialPerTurn * h.Turns

	radialUnit := ref.Scale(math.Scalar(cos)).Add(bin.Scale(math.Scalar(sin)))
	radialTangent := ref.Scale(math.Scalar(-sin)).Add(bin.Scale(math.Scalar(cos)))
	out := radialUnit.Scale(math.Scalar(dRadius))
	out = out.Add(radialTangent.Scale(math.Scalar(r * dAngle)))
	return out.Add(h.Axis.AsVector().Scale(math.Scalar(dHeight)))
}

// Domain returns [0, 1].
func (h Helix3d) Domain() (lo, hi float64) { return 0, 1 }

// StartPoint and EndPoint return the helix endpoints.
func (h Helix3d) StartPoint() math.Point3 { return h.PointAt(0) }
func (h Helix3d) EndPoint() math.Point3   { return h.PointAt(1) }

// Length returns the helix arc length. A constant-radius helix has the closed form
// √((2πr)² + pitch²)·turns; the general (tapered/spiral) case integrates the speed by
// composite Simpson, which is exact for the constant-speed cylindrical case and accurate
// to ~1e-9 otherwise.
func (h Helix3d) Length() float64 {
	if h.RadialPerTurn == 0 {
		circumference := twoPi * h.StartRadius
		perTurn := stdmath.Hypot(circumference, h.AxialPerTurn)
		return perTurn * h.Turns
	}
	return simpsonLength(h.speedAt, 0, 1, helixLengthIntervals)
}

// speedAt returns |dP/dt| at parameter t (the integrand for arc length).
func (h Helix3d) speedAt(t float64) float64 { return float64(h.TangentAt(t).Length()) }

// helixLengthIntervals is the Simpson subdivision count for the general arc-length
// integral — even, and high enough that the tapered/spiral length is accurate to ~1e-9.
const helixLengthIntervals = 256

// simpsonLength integrates f over [a, b] by composite Simpson with n (even) intervals.
func simpsonLength(f func(float64) float64, a, b float64, n int) float64 {
	step := (b - a) / float64(n)
	sum := f(a) + f(b)
	for i := 1; i < n; i++ {
		x := a + float64(i)*step
		if i%2 == 1 {
			sum += 4 * f(x)
		} else {
			sum += 2 * f(x)
		}
	}
	return sum * step / 3
}
