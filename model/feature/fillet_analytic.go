// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// analyticCylinderFillet rounds the rim(s) of a SIMPLE analytic cylinder (an extruded circle /
// revolved disc) with a TRUE toroidal fillet: it rebuilds the body as a surface of revolution whose
// meridian gains a quarter-circle ARC at each selected rim, so the fillet is one analytic geom.Torus
// face — not the faceted rolling-ball blend the re-faceted path leaves (Oblikovati/Oblikovati#127).
//
// It applies only when the body is exactly one cylinder + 2 caps, every selected edge is one of its
// two circular rims, and the radius leaves a wall and a cap (radius < cylinder radius, and the rims
// do not overrun the height). Otherwise ok is false and the caller keeps the general blend.
func analyticCylinderFillet(body *topo.Body, edges []*topo.Edge, radius float64, feat string) (*topo.Body, bool) {
	cyl, base, height, ok := simpleCylinderParams(body)
	if !ok || radius <= 0 || radius >= cyl.Radius {
		return nil, false
	}
	axis := cyl.AxisDir
	filletBottom, filletTop := false, false
	for _, e := range edges {
		switch cylinderRim(e, base, axis, height) {
		case rimBottom:
			filletBottom = true
		case rimTop:
			filletTop = true
		default:
			return nil, false
		}
	}
	used := radius
	if filletBottom && filletTop {
		used *= 2 // both rims consume the wall from each end
	}
	if used >= height {
		return nil, false
	}
	verts := filletedCylinderMeridian(cyl.Radius, height, radius, filletBottom, filletTop)
	out, err := brep.SolidOfRevolutionMeridian(base, axis.AsVector(), verts, originalFeature(body, feat))
	if err != nil || out == nil {
		return nil, false
	}
	return out, true
}

// filletedCylinderMeridian returns the (radius,height) meridian vertices of a cylinder (radius r,
// height h) with a quarter-round of radius f on the selected rims: the rim vertex is split into a
// cap point (radius r−f) and a wall point (offset f in z), joined by a 90° arc about the inner
// corner (r−f, …) — a geom.Torus when revolved.
func filletedCylinderMeridian(r, h, f float64, bottom, top bool) []brep.RevolveVertex {
	verts := []brep.RevolveVertex{{P: math.P2(0, 0)}}
	if bottom {
		c := math.P2(r-f, f)
		verts = append(verts, brep.RevolveVertex{P: math.P2(r-f, 0)}, brep.RevolveVertex{P: math.P2(r, f), ArcCenter: &c})
	} else {
		verts = append(verts, brep.RevolveVertex{P: math.P2(r, 0)})
	}
	if top {
		c := math.P2(r-f, h-f)
		verts = append(verts, brep.RevolveVertex{P: math.P2(r, h-f)}, brep.RevolveVertex{P: math.P2(r-f, h), ArcCenter: &c})
	} else {
		verts = append(verts, brep.RevolveVertex{P: math.P2(r, h)})
	}
	return append(verts, brep.RevolveVertex{P: math.P2(0, h)})
}
