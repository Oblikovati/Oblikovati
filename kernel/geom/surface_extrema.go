// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Interior coordinate extrema of a surface (M48/C3, Oblikovati/Oblikovati#3421). A tight box over a
// TRIMMED face cannot be read off a tessellation — a facet chord lies inside the true surface, so a
// convex bulge is under-measured by the sagitta and every culling or classification keyed on the box
// inherits that error. It is read off the analytic surface instead.
//
// The extremum of a coordinate P·e over a trimmed region is attained either on the region's boundary
// curves — which the caller bounds exactly from the edge curves — or at an INTERIOR stationary point
// of P·e, where ∇(P·e) = 0, i.e. P_u·e = P_v·e = 0. Since N = P_u × P_v, that condition is exactly
// "the surface normal is parallel to e". This file enumerates those points.
//
// For a plane, a cylinder and a cone the condition has no isolated solution: a plane's normal is
// constant, and a cylinder's or cone's normal depends on u alone, so the stationary set is a whole
// ruling along which P·e is affine — its extremes therefore sit on the region's boundary, which the
// caller already bounds. Only the sphere and the torus have genuine interior extrema.

// SurfaceAxisCriticalPoints returns every point at which s's normal is parallel to a world axis —
// the only places a trimmed face's coordinate extremum can lie off its boundary curves. ok is false
// when the surface kind can have interior extrema this cannot enumerate as isolated points, so the
// caller must fall back rather than report a box that is too small.
//
// Example: pts, ok := geom.SurfaceAxisCriticalPoints(sphere) // ok ⇒ the 6 axis poles
func SurfaceAxisCriticalPoints(s Surface) ([]math.Point3, bool) {
	switch g := s.(type) {
	case Plane, Cylinder, Cone:
		return nil, true // stationary set is a ruling; its extremes lie on the boundary
	case Sphere:
		return sphereAxisCriticalPoints(g), true
	case Torus:
		return torusAxisCriticalPoints(g)
	default:
		return nil, false
	}
}

// sphereAxisCriticalPoints returns the six points where the radial normal is parallel to a world
// axis: the axis poles c ± r·e.
func sphereAxisCriticalPoints(g Sphere) []math.Point3 {
	out := make([]math.Point3, 0, 6)
	for axis := range 3 {
		e := worldAxis(axis)
		out = append(out,
			g.Center.TranslateBy(e.Scale(math.Scalar(g.Radius))),
			g.Center.TranslateBy(e.Scale(math.Scalar(-g.Radius))))
	}
	return out
}

// torusAxisCriticalPoints returns the four points per world axis where the tube normal is parallel
// to it. It declines when the torus axis is parallel to a world axis: the stationary set for that
// axis is then a whole latitude circle (every u attains the extremum), not isolated points, and
// enumerating it would mean sampling — exactly what this replaces.
func torusAxisCriticalPoints(g Torus) ([]math.Point3, bool) {
	out := make([]math.Point3, 0, 12)
	for axis := range 3 {
		pts, ok := torusCriticalForAxis(g, axis)
		if !ok {
			return nil, false
		}
		out = append(out, pts...)
	}
	return out, true
}

// torusCriticalForAxis solves N ∥ e for one world axis. Split e = |p|·p̂ + q·a into its in-plane and
// axial parts (a the torus axis). The normal at (u, v) is cos v·d(u) + sin v·a with d(u) the radial
// direction, so it can only align with e where d(u) = ±p̂ — two meridians, u₀ and u₀+π — and there
// (±cos v)·q = sin v·|p| fixes the tube angle, giving two v roots half a turn apart on each.
func torusCriticalForAxis(g Torus, axis int) ([]math.Point3, bool) {
	e := worldAxis(axis)
	q := float64(e.Dot(g.AxisDir.AsVector()))
	p := e.Sub(g.AxisDir.AsVector().Scale(math.Scalar(q)))
	pl := float64(p.Length())
	if pl <= planeParallelTol {
		return nil, false // torus axis ∥ e: the stationary set is a latitude circle, not points
	}
	u0 := stdmath.Atan2(float64(p.Dot(g.binormal)), float64(p.Dot(g.Ref.AsVector())))
	v0 := stdmath.Atan2(q, pl)
	return []math.Point3{
		g.PointAt(u0, v0), g.PointAt(u0, v0+stdmath.Pi),
		g.PointAt(u0+stdmath.Pi, -v0), g.PointAt(u0+stdmath.Pi, -v0+stdmath.Pi),
	}, true
}

// worldAxis returns the unit vector of world axis 0, 1 or 2 (x, y, z).
func worldAxis(axis int) math.Vector3 {
	switch axis {
	case 0:
		return math.V3(1, 0, 0)
	case 1:
		return math.V3(0, 1, 0)
	default:
		return math.V3(0, 0, 1)
	}
}
