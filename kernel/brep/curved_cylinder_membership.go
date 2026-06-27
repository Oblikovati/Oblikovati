// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cylinder solid-membership oracle for the general curved∩curved boolean (#1403). curvedSolidMembership
// (curved_general_boolean.go) classifies a point as inside/outside a primitive curved solid; the cylinder
// case lives here. (These helpers outlived the bespoke crossing-cylinder handlers the general pipeline
// replaced — the general path is now their only caller.)

// crossAxis is the axis of a crossing operand (a cylinder or a cone) — a point on it, the unit direction,
// and the angle-zero reference — enough to measure where a point sits about that axis without knowing the
// surface type.
type crossAxis struct {
	point math.Point3
	dir   math.Vector3
	ref   math.Vector3
}

// cylAxis reads the crossing axis of a cylinder (its origin and axis direction).
func cylAxis(c geom.Cylinder) crossAxis {
	return crossAxis{c.Origin, c.AxisDir.AsVector(), c.Ref.AsVector()}
}

// radialOf returns the spoke from the axis out to p — the component of (axis point → p) perpendicular to the
// axis, whose length is the radial distance.
func radialOf(p math.Point3, ax crossAxis) math.Vector3 {
	v := ax.point.VectorTo(p)
	return v.Sub(ax.dir.Scale(v.Dot(ax.dir)))
}

// pointInsideCylinderSolid reports whether p is strictly inside the finite cylinder solid — within the
// radius and between the caps, by a small model-relative margin so a point on the surface counts as outside.
func pointInsideCylinderSolid(c geom.Cylinder, base math.Point3, height float64, p math.Point3) bool {
	margin := geom.ResolutionForSize(c.Radius + height).Plane() // model-relative inside-solid margin (#1399)
	if float64(radialOf(p, cylAxis(c)).Length()) > c.Radius-margin {
		return false
	}
	axis := c.AxisDir.AsVector()
	vBase := float64(c.Origin.VectorTo(base).Dot(axis))
	v := float64(c.Origin.VectorTo(p).Dot(axis))
	return v >= vBase+margin && v <= vBase+height-margin
}
