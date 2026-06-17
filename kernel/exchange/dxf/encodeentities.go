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
	case *drawing.LwPolyline:
		entityHead(w, "LWPOLYLINE", handle, owner, "AcDbPolyline")
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
	case *drawing.Spline:
		encodeSpline(w, g, handle, owner)
	}
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
func encodeSpline(w *tagWriter, s *drawing.Spline, handle, owner uint64) {
	entityHead(w, "SPLINE", handle, owner, "AcDbSpline")
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
// (always "0" — the geometry carries no layer), and the type-specific subclass marker.
func entityHead(w *tagWriter, typ string, handle, owner uint64, subclass string) {
	w.tag(0, typ)
	w.handle(5, handle)
	w.handle(330, owner)
	w.tag(100, "AcDbEntity")
	w.tag(8, "0")
	w.tag(100, subclass)
}
