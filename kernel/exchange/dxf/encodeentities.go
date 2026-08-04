// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"strconv"

	"oblikovati.org/kernel/exchange/drawing"
)

// encodeEntity writes one entity's ENTITIES record. A type with no encoder is skipped
// silently (it never reaches here from sketchToDrawing, but block expansion may produce a
// broader set later).
//
//nolint:funlen // one-case-per-entity-type encode dispatch (inverse of decodeEntity).
func encodeEntity(w *tagWriter, dr *drawing.Drawing, e drawing.Entity, handle, owner uint64) {
	layer := layerOf(e)
	style := dr.Styles[e.EntityHandle()]
	switch g := e.(type) {
	case *drawing.Line:
		entityHead(w, "LINE", handle, owner, "AcDbLine", layer, style)
		w.coord(10, g.Start)
		w.coord(11, g.End)
	case *drawing.Circle:
		entityHead(w, "CIRCLE", handle, owner, "AcDbCircle", layer, style)
		w.coord(10, g.Center)
		w.real(40, g.Radius)
	case *drawing.Point:
		entityHead(w, "POINT", handle, owner, "AcDbPoint", layer, style)
		w.coord(10, g.Position)
	case *drawing.Arc:
		// ARC carries two subclass markers (AcDbCircle then AcDbArc); angles are DEGREES.
		entityHead(w, "ARC", handle, owner, "AcDbCircle", layer, style)
		w.coord(10, g.Center)
		w.real(40, g.Radius)
		w.tag(100, "AcDbArc")
		w.real(50, radToDeg(g.StartAngle))
		w.real(51, radToDeg(g.EndAngle))
	case *drawing.Ellipse:
		// ELLIPSE start/end parameters are RADIANS — written unconverted, unlike ARC.
		entityHead(w, "ELLIPSE", handle, owner, "AcDbEllipse", layer, style)
		w.coord(10, g.Center)
		w.coord(11, g.MajorAxis)
		w.coord(210, normalOrZ(g.Normal))
		w.real(40, g.AxisRatio)
		w.real(41, g.StartAngle)
		w.real(42, g.EndAngle)
	case *drawing.LwPolyline:
		encodeLwPolyline(w, g, handle, owner, layer, style)
	case *drawing.Spline:
		encodeSpline(w, g, handle, owner, layer, style)
	case *drawing.Text:
		encodeText(w, g, handle, owner, layer, style)
	}
}

// encodeLwPolyline writes an LWPOLYLINE: vertex count, closed flag, optional elevation, then
// each vertex with its trailing bulge (0 = straight to the next).
func encodeLwPolyline(w *tagWriter, g *drawing.LwPolyline, handle, owner uint64, layer string, style drawing.Style) {
	entityHead(w, "LWPOLYLINE", handle, owner, "AcDbPolyline", layer, style)
	w.integer(90, len(g.Points))
	w.integer(70, closedFlag(g.Closed))
	if g.Elevation != 0 {
		w.real(38, g.Elevation)
	}
	for i, pt := range g.Points {
		w.real(10, pt[0])
		w.real(20, pt[1])
		if i < len(g.Bulges) && g.Bulges[i] != 0 {
			w.real(42, g.Bulges[i]) // bulge attaches to the vertex it follows
		}
	}
}

// encodeText writes a single-line TEXT label. It repeats the AcDbText subclass after the
// geometry (the second marker is the alignment block); a minimal label needs only the
// insertion point, height and string.
func encodeText(w *tagWriter, g *drawing.Text, handle, owner uint64, layer string, style drawing.Style) {
	entityHead(w, "TEXT", handle, owner, "AcDbText", layer, style)
	w.coord(10, g.Position)
	w.real(40, g.Height)
	w.tag(1, g.Value)
	if g.Rotation != 0 {
		w.real(50, radToDeg(g.Rotation))
	}
	w.tag(100, "AcDbText")
}

// layerOf returns an entity's target layer name (empty for a geometry that carries none),
// which entityHead maps to the default "0" layer.
func layerOf(e drawing.Entity) string {
	switch g := e.(type) {
	case *drawing.Line:
		return g.Layer
	case *drawing.Circle:
		return g.Layer
	case *drawing.Arc:
		return g.Layer
	case *drawing.Point:
		return g.Layer
	case *drawing.Ellipse:
		return g.Layer
	case *drawing.LwPolyline:
		return g.Layer
	case *drawing.Spline:
		return g.Layer
	case *drawing.Text:
		return g.Layer
	}
	return ""
}

// closedFlag is the LWPOLYLINE 70 flag value for an open/closed polyline.
func closedFlag(closed bool) int {
	if closed {
		return 1
	}
	return 0
}

// encodeSpline writes a SPLINE. Knots and weights interleave with the control points per the
// DXF layout. When the model carries control points but no knot vector (the common case from
// a sketch export), a clamped uniform knot vector is generated so the curve is a valid NURBS
// AutoCAD accepts rather than a knot-less spline.
//
//nolint:funlen // sequential SPLINE field writes across header, knots, control and fit points.
func encodeSpline(w *tagWriter, s *drawing.Spline, handle, owner uint64, layer string, style drawing.Style) {
	entityHead(w, "SPLINE", handle, owner, "AcDbSpline", layer, style)
	degree := splineDegree(s)
	knots := s.Knots
	if len(knots) == 0 && len(s.ControlPoints) >= 2 {
		knots = clampedKnots(len(s.ControlPoints), degree)
	}
	flag := 8 // planar
	if s.Closed {
		flag |= 1
	}
	if s.Rational {
		flag |= 4
	}
	w.integer(70, flag)
	w.integer(71, degree)
	w.integer(72, len(knots))
	w.integer(73, len(s.ControlPoints))
	w.integer(74, len(s.FitPoints))
	for _, k := range knots {
		w.real(40, k)
	}
	for i, cp := range s.ControlPoints {
		w.coord(10, cp)
		if i < len(s.Weights) {
			w.real(41, s.Weights[i])
		}
	}
	for _, fp := range s.FitPoints {
		w.coord(11, fp)
	}
}

// splineDegree returns the spline's degree, defaulting to 3 and clamping to one less than
// the control-point count (a degree cannot exceed that for a valid B-spline).
func splineDegree(s *drawing.Spline) int {
	d := s.Degree
	if d < 1 {
		d = 3
	}
	if n := len(s.ControlPoints); n >= 2 && d > n-1 {
		d = n - 1
	}
	return d
}

// clampedKnots builds a clamped uniform knot vector for n control points of the given
// degree: the first and last (degree+1) knots are repeated (0 and 1), the interior knots
// evenly spaced — the standard vector for a curve that passes through its end control points.
func clampedKnots(n, degree int) []float64 {
	total := n + degree + 1
	interior := n - degree - 1
	knots := make([]float64, total)
	for i := range knots {
		switch {
		case i <= degree:
			knots[i] = 0
		case i >= total-degree-1:
			knots[i] = 1
		default:
			knots[i] = float64(i-degree) / float64(interior+1)
		}
	}
	return knots
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
// (the entity's named layer, or "0" when it carries none), and the type-specific subclass.
func entityHead(w *tagWriter, typ string, handle, owner uint64, subclass, layer string, style drawing.Style) {
	w.tag(0, typ)
	w.handle(5, handle)
	w.handle(330, owner)
	w.tag(100, "AcDbEntity")
	w.tag(8, layerOr0(layer))
	writeEntityStyle(w, style)
	w.tag(100, subclass)
}

// writeEntityStyle emits the entity's formatting overrides inside the AcDbEntity block, where the
// DXF reference places them (#2015). A field left at its inherit sentinel is omitted rather than
// written as BYLAYER: omission is what an unformatted entity looks like in every file AutoCAD
// writes, so an unstyled export stays byte-identical to what it produced before.
func writeEntityStyle(w *tagWriter, style drawing.Style) {
	if style.Color != drawing.ColorByLayer && style.Color != drawing.ColorByBlock {
		w.tag(62, strconv.Itoa(style.Color))
	}
	if style.LineType != "" {
		w.tag(6, style.LineType)
	}
	if style.LineWeight != drawing.LineWeightByLayer && style.LineWeight != 0 {
		w.tag(370, strconv.Itoa(style.LineWeight))
	}
}

// layerOr0 returns the layer name, or the default "0" layer when empty.
func layerOr0(layer string) string {
	if layer == "" {
		return "0"
	}
	return layer
}
