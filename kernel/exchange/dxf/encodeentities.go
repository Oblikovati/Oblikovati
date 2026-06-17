// SPDX-License-Identifier: GPL-2.0-only

package dxf

import "oblikovati.org/kernel/exchange/drawing"

// encodeEntity writes one entity's ENTITIES record. A type with no encoder is skipped
// silently (it never reaches here from sketchToDrawing, but block expansion may produce a
// broader set later).
//
//nolint:funlen // one-case-per-entity-type encode dispatch (inverse of decodeEntity).
func encodeEntity(w *tagWriter, e drawing.Entity, handle, owner uint64) {
	switch g := e.(type) {
	case *drawing.Line:
		entityHead(w, "LINE", handle, owner, "AcDbLine")
		w.coord(10, g.Start)
		w.coord(11, g.End)
	case *drawing.Circle:
		entityHead(w, "CIRCLE", handle, owner, "AcDbCircle")
		w.coord(10, g.Center)
		w.real(40, g.Radius)
	case *drawing.Point:
		entityHead(w, "POINT", handle, owner, "AcDbPoint")
		w.coord(10, g.Position)
	case *drawing.Arc:
		// ARC carries two subclass markers (AcDbCircle then AcDbArc); angles are DEGREES.
		entityHead(w, "ARC", handle, owner, "AcDbCircle")
		w.coord(10, g.Center)
		w.real(40, g.Radius)
		w.tag(100, "AcDbArc")
		w.real(50, radToDeg(g.StartAngle))
		w.real(51, radToDeg(g.EndAngle))
	case *drawing.Ellipse:
		// ELLIPSE start/end parameters are RADIANS — written unconverted, unlike ARC.
		entityHead(w, "ELLIPSE", handle, owner, "AcDbEllipse")
		w.coord(10, g.Center)
		w.coord(11, g.MajorAxis)
		w.coord(210, normalOrZ(g.Normal))
		w.real(40, g.AxisRatio)
		w.real(41, g.StartAngle)
		w.real(42, g.EndAngle)
	}
}

// normalOrZ defaults a zero extrusion vector to +Z (a 2D entity), so a model entity that
// never set its normal still writes a valid AcDbEllipse extrusion.
func normalOrZ(n [3]float64) [3]float64 {
	if n == [3]float64{} {
		return [3]float64{0, 0, 1}
	}
	return n
}

// entityHead writes the common ENTITIES preamble shared by every entity: the type marker,
// the handle, the owner (the *Model_Space block record), the AcDbEntity subclass, the layer
// (always "0" — the geometry carries no layer), and the type-specific subclass marker.
func entityHead(w *tagWriter, typ string, handle, owner uint64, subclass string) {
	w.tag(0, typ)
	w.handle(5, handle)
	w.handle(330, owner)
	w.tag(100, "AcDbEntity")
	w.tag(8, "0")
	w.tag(100, subclass)
}
