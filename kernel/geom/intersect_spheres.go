// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The SPHERE × SPHERE closed form — the simplest curved-versus-curved crossing there is, and the one
// the parametric×implicit bucket cannot reach: a sphere has no straight ruling to substitute into the
// other's quadric, so the ruled∩quadric form declines both role assignments and the pair fell through
// to the marcher (ADR-0061 stage 4).
//
// Subtracting the two implicit forms cancels the quadratic terms, leaving a PLANE — the radical plane,
// perpendicular to the line of centres — so the section is that plane's circle on either sphere. It is
// the same argument the equal-cylinder case makes, and it belongs in the same place: a closed form
// inside the general intersector, gated on conditioning, never a handler above it.

// sphereSphereSection returns the circle two spheres cross in, evaluated in the radical plane. ok is
// false for any pair that is not two spheres; handled reports that the pair was DECIDED — with no
// curves when the spheres are known not to cross, and declining only where the crossing is not a curve
// the stitch can carry (a tangent touch, or two concentric spheres).
func sphereSphereSection(a, b Surface, res Resolution) (curves []Curve3, handled, ok bool) {
	sa, okA := a.(Sphere)
	sb, okB := b.(Sphere)
	if !okA || !okB {
		return nil, false, false
	}
	axis := sa.Center.VectorTo(sb.Center)
	d := float64(axis.Length())
	if d <= res.Weld() {
		return nil, false, true // concentric: coincident when the radii agree, else never meeting — neither is a crossing
	}
	if d > sa.Radius+sb.Radius+res.Weld() || d < stdmath.Abs(sa.Radius-sb.Radius)-res.Weld() {
		return nil, true, true // apart, or one strictly inside the other: decided, and empty
	}
	// The radical plane sits at signed distance h from a's centre along the line of centres.
	h := (d*d + sa.Radius*sa.Radius - sb.Radius*sb.Radius) / (2 * d)
	rSq := sa.Radius*sa.Radius - h*h
	if rSq <= 0 || stdmath.Sqrt(rSq) <= res.Weld() {
		return nil, false, true // a tangent touch: a point, not a curve
	}
	unit := axis.Scale(math.Scalar(1 / d))
	centre := sa.Center.TranslateBy(unit.Scale(math.Scalar(h)))
	circle, err := NewCircle(centre, unit, stdmath.Sqrt(rSq))
	if err != nil {
		return nil, false, true
	}
	return []Curve3{circle}, true, true
}
