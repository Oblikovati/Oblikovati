// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"math"

	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/model/sketch"
)

// This file converts a 2D sketch's geometry into the format-neutral drawing model, the
// inverse of sketch_from_drawing.go. It is shared by every drawing exporter (DWG, DXF): a
// codec encodes the resulting []drawing.Entity to its file format. Coordinates are emitted
// in the sketch plane's X/Y, in database units (cm).

// sketchToDrawing converts a sketch's 2D entities to the neutral drawing entity model.
func sketchToDrawing(sk *sketch.Sketch) []drawing.Entity {
	var out []drawing.Entity
	for i := 0; i < sk.Lines().Count(); i++ {
		l := sk.Lines().Item(i)
		out = append(out, &drawing.Line{Start: sketchPt(l.StartPoint()), End: sketchPt(l.EndPoint())})
	}
	for i := 0; i < sk.Circles().Count(); i++ {
		c := sk.Circles().Item(i)
		out = append(out, &drawing.Circle{Center: sketchPt(c.CenterPoint()), Radius: float64(c.Radius), Normal: [3]float64{0, 0, 1}})
	}
	for i := 0; i < sk.Arcs().Count(); i++ {
		out = append(out, arcToDrawing(sk.Arcs().Item(i)))
	}
	for i := 0; i < sk.Splines().Count(); i++ {
		out = append(out, splineToDrawing(sk.Splines().Item(i)))
	}
	return out
}

// arcToDrawing converts a sketch arc to a drawing arc. Drawing arcs always sweep
// counter-clockwise from start to end, so a clockwise sketch arc is emitted with its
// endpoints swapped (CCW end→start traces the same geometry).
func arcToDrawing(a *sketch.Arc) *drawing.Arc {
	c := a.Center.Position()
	start, end := a.Start.Position(), a.End.Position()
	if !a.CounterClockwise {
		start, end = end, start
	}
	angle := func(p [2]float64) float64 { return math.Atan2(p[1]-c.Y, p[0]-c.X) }
	return &drawing.Arc{
		Center:     [3]float64{c.X, c.Y, 0},
		Radius:     float64(a.Radius()),
		StartAngle: angle([2]float64{start.X, start.Y}),
		EndAngle:   angle([2]float64{end.X, end.Y}),
		Normal:     [3]float64{0, 0, 1},
	}
}

// splineToDrawing converts a sketch spline to a drawing spline, emitting its defining
// points as control points (the importer rebuilds the curve from them, so the points
// round-trip).
func splineToDrawing(s *sketch.Spline) *drawing.Spline {
	ctrl := make([][3]float64, s.PointCount())
	for i, p := range s.Points {
		ctrl[i] = sketchPt(p)
	}
	return &drawing.Spline{Degree: 3, ControlPoints: ctrl, Closed: s.Closed}
}

// sketchPt lifts a sketch point's 2D position into a Z=0 drawing coordinate.
func sketchPt(p *sketch.Point) [3]float64 {
	pos := p.Position()
	return [3]float64{pos.X, pos.Y, 0}
}
