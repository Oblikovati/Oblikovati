// SPDX-License-Identifier: GPL-2.0-only

package dxf

import "oblikovati.org/kernel/exchange/drawing"

// encodeEntity writes one entity's ENTITIES record. A type with no encoder is skipped
// silently (it never reaches here from sketchToDrawing, but block expansion may produce a
// broader set later).
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
	}
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
