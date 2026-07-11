// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// wrapArcLengthSteps is the substep count for inverting arc length to a surface parameter by
// integrating the metric speed. Constant-speed directions (a plane axis, a cylinder's
// circumference/axis, a cone's/sphere's iso-circles) are exact at any count; directions whose
// speed varies along the walk (a general NURBS meridian) converge with the midpoint rule at
// this resolution.
const wrapArcLengthSteps = 128

// WrapFrame is a source curve's planar frame for a wrap projection: the plane Origin plus the
// in-plane axes U and V (unit, orthogonal). A source point p flattens to planar coordinates
// (a, b) = ((p−Origin)·U, (p−Origin)·V). The caller orients U toward the surface's u-parameter
// direction and V toward its v-parameter direction (e.g. U circumferential, V axial for a
// cylinder), so wrap maps a to arc length along u and b to arc length along v.
type WrapFrame struct {
	Origin math.Point3
	U, V   math.Vector3
}

// flatten returns p's coordinates (a, b) in the frame — the signed U and V components of p−Origin.
func (f WrapFrame) flatten(p math.Point3) (a, b float64) {
	d := f.Origin.VectorTo(p)
	return float64(d.Dot(f.U)), float64(d.Dot(f.V))
}

// WrapCurveOntoSurface maps the sampled source curve onto surface, preserving arc length from
// the anchor — the surface parameters where Frame.Origin projects (surface.ParamAt). Each source
// point flattens to (a, b) in frame and maps to the surface point reached by travelling arc
// length b along the v-parameter, then a along the u-parameter, so an in-plane displacement
// becomes an equal on-surface arc length along each parameter direction. This is Inventor's
// kWrapToSurfaceType / MapPointCurve (#1841): an isometry for developable surfaces (cylinder,
// cone) — e.g. a cylinder of radius R maps planar x to angle u = x/R — and the closest
// arc-length-along-parameters map otherwise. Returns nil for a nil surface/source or samples < 1.
func WrapCurveOntoSurface(surface Surface, source Curve3, frame WrapFrame, samples int) []math.Point3 {
	if surface == nil || source == nil || samples < 1 {
		return nil
	}
	u0, v0 := surface.ParamAt(frame.Origin)
	lo, hi := source.Domain()
	out := make([]math.Point3, samples+1)
	for i := range out {
		t := lo + (hi-lo)*float64(i)/float64(samples)
		a, b := frame.flatten(source.PointAt(t))
		v := arcLengthParam(v0, b, func(vv float64) float64 { return vDirSpeed(surface, u0, vv) })
		u := arcLengthParam(u0, a, func(uu float64) float64 { return uDirSpeed(surface, uu, v) })
		out[i] = surface.PointAt(u, v)
	}
	return out
}

// uDirSpeed / vDirSpeed are the surface's metric speeds |∂S/∂u| and |∂S/∂v| — the local
// arc-length gained per unit of each parameter.
func uDirSpeed(s Surface, u, v float64) float64 {
	du, _ := s.DerivativesAt(u, v)
	return float64(du.Length())
}

func vDirSpeed(s Surface, u, v float64) float64 {
	_, dv := s.DerivativesAt(u, v)
	return float64(dv.Length())
}

// arcLengthParam returns the parameter reached by travelling signed arc length target from
// param0 along a 1-D path whose metric speed |dP/dparam| is speed(param). It marches in
// wrapArcLengthSteps substeps, each sized to cover an equal share of the remaining arc length so
// curvature is resolved, integrates with the midpoint speed, and finishes the last partial step
// linearly. A constant speed yields the exact param0 + target/speed. speed must be > 0 on the
// range (a regular surface direction); a degenerate (≤0) speed stops the march.
func arcLengthParam(param0, target float64, speed func(float64) float64) float64 {
	if target == 0 {
		return param0
	}
	dir, remaining, param := stdmath.Copysign(1, target), stdmath.Abs(target), param0
	for i := 0; i < wrapArcLengthSteps && remaining > 0; i++ {
		sp := speed(param)
		if sp <= 0 {
			break
		}
		dParam := dir * (remaining / float64(wrapArcLengthSteps-i)) / sp
		mid := speed(param + dParam/2)
		if mid <= 0 {
			mid = sp
		}
		step := stdmath.Abs(dParam) * mid
		if step >= remaining {
			return param + dir*(remaining/mid) // finish within this step
		}
		remaining -= step
		param += dParam
	}
	return param
}
