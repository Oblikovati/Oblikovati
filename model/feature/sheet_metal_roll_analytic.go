// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Analytic contour roll (#2019 follow-up). A contour roll is a partial (or full) revolve of a
// thickened straight-line band about an axis, so it shares the analytic revolve builders: a full turn
// is a true surface-of-revolution shell (cylinder / cone walls), a partial roll an analytic sector
// (partial cylinder / cone walls + two planar caps). This keeps a rolled tube's walls TRUE cylinders
// — a projected face reads real arcs, not the ~64-facet-per-turn swept prism. A band that does not
// revolve to a valid analytic solid falls back to the faceted section sweep.

// analyticContourRoll builds the analytic rolled shell from the thickened band, or nil to keep the
// faceted section sweep. A full turn uses the periodic surface-of-revolution builder, a partial angle
// the sector builder; either result is validated and rejected (→ nil) if it is not a watertight solid.
func analyticContourRoll(band []math.Point3, axis *WorkAxis, angle float64, feat string) *topo.Body {
	verts, ok := bandMeridianVerts(band, axis)
	if !ok {
		return nil
	}
	o, a := axis.Origin(), axis.Direction().AsVector()
	var body *topo.Body
	var err error
	if angle >= 2*stdmath.Pi-1e-9 {
		body, err = brep.SolidOfRevolutionMeridian(o, a, verts, feat)
	} else {
		ref0, hasRadial := bandStartRadial(band, axis)
		if !hasRadial {
			return nil
		}
		body, err = brep.SolidOfRevolutionSector(o, a, ref0, angle, verts, feat)
	}
	if err != nil || body == nil || !body.IsSolid() || !ops.Validate(body).Valid {
		return nil
	}
	return body
}

// bandMeridianVerts maps a thickened band's 3D points to (r, z) meridian vertices about the axis
// (r = perpendicular distance from the axis, z = distance along it), all straight edges. ok is false
// for fewer than three points.
func bandMeridianVerts(band []math.Point3, axis *WorkAxis) ([]brep.RevolveVertex, bool) {
	if len(band) < 3 {
		return nil, false
	}
	o, a := axis.Origin(), axis.Direction().AsVector()
	verts := make([]brep.RevolveVertex, len(band))
	for i, p := range band {
		v := o.VectorTo(p)
		z := v.Dot(a)
		r := float64(v.Sub(a.Scale(z)).Length())
		verts[i] = brep.RevolveVertex{P: math.P2(math.Scalar(r), math.Scalar(z))}
	}
	return verts, true
}

// bandStartRadial returns the band's radial direction — the point farthest from the axis — which is
// the sector's start half-plane (a contour roll starts at the profile, offset 0). ok is false when
// the whole band lies on the axis.
func bandStartRadial(band []math.Point3, axis *WorkAxis) (math.Vector3, bool) {
	o, a := axis.Origin(), axis.Direction().AsVector()
	best, bestR := math.Vector3{}, 0.0
	for _, p := range band {
		v := o.VectorTo(p)
		radial := v.Sub(a.Scale(v.Dot(a)))
		if r := float64(radial.Length()); r > bestR {
			bestR, best = r, radial
		}
	}
	if bestR == 0 {
		return math.Vector3{}, false
	}
	return best, true
}
