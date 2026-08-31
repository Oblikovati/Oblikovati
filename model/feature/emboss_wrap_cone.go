// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Wrapping an emboss onto a CONE (Inventor's Wrap to Face on a conical face). A cone is developable,
// so — like the cylinder — arc length on the face equals distance in the sketch when the sketch
// plane is TANGENT to it. But a cone's development is a circular SECTOR about the apex, not a
// rectangle: every tangent plane of a cone passes through the apex, distance from the apex in the
// plane IS the slant distance on the cone, and a full turn around the cone (Δu = 2π) unrolls to a
// sector of only 2π·sin(α). So the correspondence is POLAR about the apex — a sketch point at polar
// (ρ, φ) from the apex lands at slant s = ρ and turn (u − u0) = φ / sin(α), i.e. the angle a unit
// arc subtends SHRINKS with slant. That is exactly what wrapping a cone as if it were a cylinder
// would get wrong. The offset an emboss is raised by is measured along the cone's surface NORMAL
// (cos α·radial − sin α·axis), which tilts back toward the apex — not radially as on a cylinder.

// coneWrapFrame is the sketch→cone correspondence, built once the plane is known tangent. gDir is
// the unit generator direction in the plane (φ = 0) and circ0 the in-plane circumferential
// direction (φ = +90°, the +u sense); radial0 = radial(u0) is the tangency generator's outward
// radial. refRadius is the local surface radius at the sketch origin, sizing the overlap pad.
type coneWrapFrame struct {
	cone       geom.Cone
	apex       math.Point3
	axis       math.Vector3
	sinA, cosA float64
	tanA       float64
	radial0    math.Vector3 // radial(u0): outward, ⟂ axis, in the sketch plane
	circ0      math.Vector3 // axis × radial0: circumferential at u0, in the plane, +u sense
	gDir       math.Vector3 // unit generator at u0 = cosA·axis + sinA·radial0 (lies in the plane)
	refRadius  float64      // local radius (slant·sinA) at the sketch origin — the pad reference
}

// wrapConeTangencyTol mirrors wrapTangencyTol: a loose, length-relative admissibility test on an
// authored work plane, not a fit — so it survives the plane having been built from the face itself.
const wrapConeTangencyTol = 1e-6

// coneWrapFrameFor builds the frame for a cone, checking the two things the wrap needs and deriving
// the tangency generator from the plane's own normal. A cone's tangent plane (a) passes through the
// apex and (b) has a normal tilted so n·axis = −sin(α) (the outward normal leans back toward the
// apex). Both together fix a single generator, whose outward radial is (n + sin α·axis)/cos α.
func coneWrapFrameFor(cone geom.Cone, plane sketch.Plane) (coneWrapFrame, error) {
	axis := cone.AxisDir.AsVector()
	sinA, cosA := stdmath.Sin(cone.HalfAngle), stdmath.Cos(cone.HalfAngle)
	ref := float64(cone.Apex.DistanceTo(plane.Origin()))
	if ref <= 0 {
		return coneWrapFrame{}, fmt.Errorf("emboss: wrapToFace: the sketch origin sits on the cone apex; sketch away from the apex, where the surface has a defined tangent")
	}
	n := orientedOutwardNormal(cone, plane, axis)
	if apexOff := stdmath.Abs(float64(cone.Apex.VectorTo(plane.Origin()).Dot(n))); apexOff > wrapConeTangencyTol*ref {
		return coneWrapFrame{}, fmt.Errorf("emboss: wrapToFace needs the sketch plane through the cone's APEX (every tangent plane of a cone does); the apex stands %g off it — sketch on a work plane tangent to the cone", apexOff)
	}
	if along := float64(n.Dot(axis)) + sinA; stdmath.Abs(along) > wrapConeTangencyTol {
		return coneWrapFrame{}, fmt.Errorf("emboss: wrapToFace needs the sketch plane TANGENT to the cone (its normal·axis = −sin(halfAngle) = %g), but the normal·axis is %g; sketch on a work plane tangent to the face", -sinA, float64(n.Dot(axis)))
	}
	radial0, err := math.UnitVector3FromVector(n.Add(axis.Scale(math.Scalar(sinA))).Scale(math.Scalar(1 / cosA)))
	if err != nil {
		return coneWrapFrame{}, fmt.Errorf("emboss: wrapToFace: degenerate cone tangency frame")
	}
	fr := coneWrapFrame{
		cone: cone, apex: cone.Apex, axis: axis, sinA: sinA, cosA: cosA, tanA: stdmath.Tan(cone.HalfAngle),
		radial0: radial0.AsVector(), circ0: axis.Cross(radial0.AsVector()), gDir: axis.Scale(cosA).Add(radial0.AsVector().Scale(sinA)),
	}
	fr.refRadius = fr.slantOf(plane.Origin()) * sinA
	return fr, nil
}

// orientedOutwardNormal returns the plane normal flipped to point AWAY from the axis (the outward
// side of the cone), so the tangency algebra derives the outward radial rather than its opposite.
func orientedOutwardNormal(cone geom.Cone, plane sketch.Plane, axis math.Vector3) math.Vector3 {
	n := plane.Normal().AsVector()
	w := cone.Apex.VectorTo(plane.Origin())
	radialAtOrigin := w.Sub(axis.Scale(w.Dot(axis))) // origin's outward radial from the axis
	if n.Dot(radialAtOrigin) < 0 {
		return n.Scale(-1)
	}
	return n
}

// slantOf is the distance from the apex to a model point measured IN the sketch plane (the point's
// slant distance on the cone, since the tangent plane's radial-from-apex equals the cone's slant).
func (fr coneWrapFrame) slantOf(p math.Point3) float64 {
	w := fr.apex.VectorTo(p)
	along, perp := float64(w.Dot(fr.gDir)), float64(w.Dot(fr.circ0))
	return stdmath.Hypot(along, perp)
}

// polar decomposes a sketch point into its polar coordinates about the apex in the sketch plane:
// ρ the slant distance, φ the turn from the tangency generator (gDir at φ = 0, circ0 at +90°).
func (fr coneWrapFrame) polar(p math.Point2, plane sketch.Plane) (rho, phi float64) {
	w := fr.apex.VectorTo(plane.ToModel(p))
	along, perp := float64(w.Dot(fr.gDir)), float64(w.Dot(fr.circ0))
	return stdmath.Hypot(along, perp), stdmath.Atan2(perp, along)
}

// radialAt returns radial(u) for u = u0 + theta, by rotating radial0 toward circ0 by theta.
func (fr coneWrapFrame) radialAt(theta float64) math.Vector3 {
	cos, sin := stdmath.Cos(theta), stdmath.Sin(theta)
	return fr.radial0.Scale(math.Scalar(cos)).Add(fr.circ0.Scale(math.Scalar(sin)))
}

// at maps a sketch point onto the cone, offset by `level` along the surface normal (level = 0 lands
// on the face; +depth is raised out). The polar (ρ, φ) about the apex becomes slant s = ρ (axial
// v = s·cos α, local radius s·sin α) and turn theta = φ / sin α; the point is then pushed `level`
// along the cone normal cos α·radial(u) − sin α·axis.
func (fr coneWrapFrame) at(p math.Point2, plane sketch.Plane, level float64) math.Point3 {
	rho, phi := fr.polar(p, plane)
	theta := phi / fr.sinA
	radial := fr.radialAt(theta)
	v := rho * fr.cosA
	surface := fr.apex.
		TranslateBy(fr.axis.Scale(math.Scalar(v))).
		TranslateBy(radial.Scale(math.Scalar(rho * fr.sinA)))
	normal := radial.Scale(math.Scalar(fr.cosA)).Sub(fr.axis.Scale(math.Scalar(fr.sinA)))
	return surface.TranslateBy(normal.Scale(math.Scalar(level)))
}

// angleSpan is the turn (in u) a sketch segment subtends on the cone — how the resampling step is
// chosen, so the wrapped loop follows the cone's circumferential curvature. The u-turn is the
// apex-subtended angle φ divided by sin α (Δu = Δφ / sin α).
func (fr coneWrapFrame) angleSpan(a, b math.Point2, plane sketch.Plane) float64 {
	_, pa := fr.polar(a, plane)
	_, pb := fr.polar(b, plane)
	return stdmath.Abs(pb-pa) / fr.sinA
}

// offsets returns the inner and outer NORMAL offset the cone pad is built between (0 on the face).
// A raise sinks the inner loop a pad into the material and lifts the outer to +depth; an engrave
// cuts to −depth and lifts the outer a pad clear of the skin. The pad is the cylinder's sagitta rule
// at the local radius, doubled to comfortably cover a profile reaching past the sketch origin —
// over-padding only buries the hidden loop deeper, never touching the visible +depth/−depth face.
func (fr coneWrapFrame) offsets(depth float64, engrave bool) (inner, outer float64, err error) {
	pad := 4 * fr.refRadius * (1 - stdmath.Cos(wrapAngularStep/2))
	if !engrave {
		return -pad, depth, nil
	}
	return -depth, pad, nil
}

// capSurface returns the cone the pad's inner/outer face lies on at normal offset `level`: offsetting a
// cone by d along its outward normal is a COAXIAL cone of the SAME half-angle whose apex shifts by
// −d/sin α along the axis (the parallel surface of a cone is a cone). So the wrapped pad gets true cone
// cap faces, not a flat cap over a curved loop.
func (fr coneWrapFrame) capSurface(level float64) (geom.Surface, bool) {
	apex := fr.apex.TranslateBy(fr.axis.Scale(math.Scalar(-level / fr.sinA)))
	cone, err := geom.NewConeWithRef(apex, fr.axis, fr.radial0, fr.cone.HalfAngle)
	if err != nil {
		return nil, false
	}
	return cone, true
}
