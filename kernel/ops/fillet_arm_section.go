// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// armSectionArc is the fillet cross-section quarter-circle at spine station `spine`: the arc on the
// fillet cylinder from its contact with plane `first` to its contact with plane `second`, through
// the 45° bisector point. The contacts are the feet of the perpendiculars from the cross-section's
// axis point to each host plane (each at distance = Radius, since the fillet is tangent to both) —
// the same Arc3dByThreePoints construction fillet_blend_faces.go uses at the corner arcs, re-stationed.
// Swapping first/second yields the arc in the reverse orientation (used by the two flank patches).
func armSectionArc(cyl geom.Cylinder, first, second geom.Plane, spine float64) (geom.Arc3d, bool) {
	axisPt := cyl.Origin.TranslateBy(cyl.AxisDir.AsVector().Scale(spine))
	c1 := projectOntoPlane(axisPt, first)
	c2 := projectOntoPlane(axisPt, second)
	bis := axisPt.VectorTo(c1).Add(axisPt.VectorTo(c2))
	l := bis.Length()
	if l < arcBisectorTiny*cyl.Radius {
		return geom.Arc3d{}, false // contacts near-antipodal: the bisector midpoint is ill-defined
	}
	mid := axisPt.TranslateBy(bis.Scale(cyl.Radius / l))
	arc, err := geom.Arc3dByThreePoints(c1, mid, c2)
	return arc, err == nil
}

// internalSeam is the free-placement decomposition curve shared by a flank setback patch and the
// central patch. It is a straight segment; Adjacent stays nil and Cont G0 (a fill-to-fill G1 seam is
// a coupled solve deferred). Built from the SAME two corner values on both loops so the union is
// watertight — the flank stores it reversed to keep its own cycle closed.
func internalSeam(from, to math.Point3) geom.LineSegment {
	return geom.NewLineSegment(from, to)
}

// filletContact is the point where the fillet cylinder touches `plane` at spine station `spine`:
// the projection of the cross-section's axis point onto the plane (the tangency foot, at radius).
func filletContact(cyl geom.Cylinder, plane geom.Plane, spine float64) math.Point3 {
	axisPt := cyl.Origin.TranslateBy(cyl.AxisDir.AsVector().Scale(spine))
	return projectOntoPlane(axisPt, plane)
}

// projectOntoPlane returns the orthogonal projection of p onto plane (its metric-nearest point).
func projectOntoPlane(p math.Point3, plane geom.Plane) math.Point3 {
	n := plane.Normal()
	d := plane.Origin.VectorTo(p).Dot(n) / n.Dot(n)
	return p.TranslateBy(n.Scale(-d))
}

// arcBisectorTiny is the dimensionless floor (scaled by circle radius at each use, ADR-0042) below
// which an arc's two boundary points are treated as near-antipodal and its bisector midpoint as
// ill-defined — the arc-construction sibling of ribbonDirTiny's direction-collapse floor. Shared by
// armSectionArc and the footprint-arc builders (fillet_setback_extract.go, fillet_setback_close.go).
const arcBisectorTiny = 1e-9
