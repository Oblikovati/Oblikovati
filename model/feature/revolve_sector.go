// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Analytic PARTIAL revolve (#2019 / #2164 class). A full 360° revolve already builds true surfaces of
// revolution (SolidOfRevolutionMeridian); a partial-angle revolve fell back to the faceted section
// sweep, so its walls facet and a projected face reads chorded arcs. This routes a partial revolve of
// a straight-edged (line-only) meridian through the analytic sector builder instead — partial
// cylinder / cone / disk-sector walls plus the two planar caps at the start and end angles. A curved
// meridian (a fillet → torus/sphere sector) still facets; the sector builder declines it.

// buildPartialRevolveSolid builds the analytic partial-revolve sector, or (nil, false) to keep the
// faceted revolve — a curved meridian, a profile that degenerates onto the axis, or a build failure.
func buildPartialRevolveSolid(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, angle, start float64, feat string) (*topo.Body, bool) {
	verts, ok := meridianVertsFromProfile(prof, plane, axis)
	if !ok {
		return nil, false
	}
	ref0, ok := revolveStartRadial(prof, plane, axis, start)
	if !ok {
		return nil, false
	}
	body, err := brep.SolidOfRevolutionSector(axis.Origin(), axis.Direction().AsVector(), ref0, angle, verts, feat)
	if err != nil || body == nil {
		return nil, false
	}
	return body, true
}

// revolveStartRadial returns the radial direction (perpendicular to the axis, in the profile's
// half-plane) at the revolve START angle: the profile's own radial direction rotated by start — so
// the sector's start cap sits on the swept profile exactly where the faceted revolve places it. It
// picks the profile point farthest from the axis for a well-conditioned direction. ok is false when
// the whole profile sits on the axis (no radial direction — a degenerate revolve).
func revolveStartRadial(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, start float64) (math.Vector3, bool) {
	o, a := axis.Origin(), axis.Direction().AsVector()
	best, bestR := math.Vector3{}, 0.0
	for _, p2 := range prof.OuterLoop().Polygon() {
		v := o.VectorTo(plane.ToModel(p2))
		radial := v.Sub(a.Scale(v.Dot(a)))
		if r := float64(radial.Length()); r > bestR {
			bestR, best = r, radial
		}
	}
	if bestR == 0 {
		return math.Vector3{}, false
	}
	return math.Rotation4(start, axis.Direction(), o).TransformVector(best), true
}
