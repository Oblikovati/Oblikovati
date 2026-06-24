// SPDX-License-Identifier: GPL-2.0-only

package brep

import "oblikovati.org/kernel/geom"

// Curved boolean imprint (M2 Phase 1, Oblikovati/Oblikovati#1334). The imprint stage of the
// curved boolean asks, for a pair of faces, "where do their surfaces meet?". For the planar
// boolean that is plane∩plane → a line segment (imprint, boolean.go). Lifted to curved faces
// it is surface∩surface → analytic conic(s): plane∩cylinder ⟂ axis → a circle, plane∩sphere →
// a circle, plane∩cone ⟂ axis → a circle, oblique plane∩cylinder → an ellipse, plane∩plane → a
// line. Phase 1 covers exactly the pairs geom.IntersectSurfacesAnalytic solves in closed form;
// curved∩curved (cylinder∩cylinder) is Phase 2's SSI tracer. Clipping the returned curve to the
// two faces' trimmed regions is the split stage (next slice) — these are the full surface
// intersection curves.

// curvedImprint returns the analytic intersection curve(s) of two faces' surfaces and whether
// the pair was handled in closed form. handled == false means no analytic solver applies (a
// curved∩curved pair, or a NURBS face) — the caller defers that pair to the numeric tracer in
// Phase 2. handled == true with an empty slice means the surfaces provably do not cross
// (parallel planes, a sphere clear of the plane, a tangent touch).
//
// Example — a cylinder side cut by a plane perpendicular to its axis yields one circle:
//
//	cyl, _ := brep.SolidCylinder(math.P3(0,0,0), math.V3(0,0,1), 3, 4)
//	box, _ := brep.SolidBlock(math.P3(-5,-5,2), math.P3(5,5,3), "box")
//	curves, ok := curvedImprint(facesOfAny(cyl)[2], facesOfAny(box)[0]) // ok, one geom.Circle
func curvedImprint(a, b curvedFace) (curves []geom.Curve3, handled bool) {
	return geom.IntersectSurfacesAnalytic(a.surface, b.surface)
}
