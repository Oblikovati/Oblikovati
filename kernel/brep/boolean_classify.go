// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// insideSolid reports whether p is inside the solid b, by casting a skewed ray from p and
// counting crossings of b's planar faces (odd ⇒ inside). The direction is off-axis to
// avoid grazing a face plane or hitting an edge/vertex.
func insideSolid(b *topo.Body, p math.Point3) bool {
	faces, ok := facesOf(b)
	if !ok {
		return false
	}
	dir := math.V3(0.5773, 0.5774, 0.5775)
	crossings := 0
	for _, f := range faces {
		if rayHitsFace(p, dir, f) {
			crossings++
		}
	}
	return crossings%2 == 1
}

// rayHitsFace reports whether the ray p+t·dir (t>0) passes through the face's polygon.
func rayHitsFace(p math.Point3, dir math.Vector3, f planarFace) bool {
	n := f.normal
	den := dir.Dot(n)
	if stdmath.Abs(den) < 1e-12 {
		return false // ray parallel to the face plane
	}
	t := f.plane.Origin.VectorTo(p).Dot(n) / -den // n·(p+t dir) = n·origin ⇒ t = n·(origin−p)/(n·dir)
	if t <= 1e-9 {
		return false
	}
	hit := p.TranslateBy(dir.Scale(t))
	return pointInFace2D(to2D(f.plane, hit), f)
}
