// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"math"
	"os"

	"oblikovati.org/kernel/exchange/dwg"
	"oblikovati.org/model/sketch"
)

// ExportDWG encodes a 2D sketch's geometry as an R2000 DWG file: the inverse of ImportDWG.
// Each sketch curve maps to its DWG entity (line/circle/arc/ellipse/spline/polyline) in
// the sketch plane's X/Y; coordinates are written in database units (cm). It is the
// counterpart that lets a sketch round-trip out to DWG and back.
//
//	data, err := exchange.ExportDWG(sk)
func ExportDWG(sk *sketch.Sketch) ([]byte, error) {
	entities := sketchToDWG(sk)
	return dwg.Write(&dwg.Drawing{Entities: entities, Units: insunitsCentimetre})
}

// ExportDWGFile writes ExportDWG's output to path.
func ExportDWGFile(sk *sketch.Sketch, path string) error {
	data, err := ExportDWG(sk)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("export dwg: write %q: %w", path, err)
	}
	return nil
}

// insunitsCentimetre is the $INSUNITS code for centimetres — the model's database unit, so
// exported coordinates need no scaling.
const insunitsCentimetre = 5

// sketchToDWG converts a sketch's 2D entities to the neutral DWG entity model.
func sketchToDWG(sk *sketch.Sketch) []dwg.Entity {
	var out []dwg.Entity
	for i := 0; i < sk.Lines().Count(); i++ {
		l := sk.Lines().Item(i)
		out = append(out, &dwg.Line{Start: sketchPt(l.StartPoint()), End: sketchPt(l.EndPoint())})
	}
	for i := 0; i < sk.Circles().Count(); i++ {
		c := sk.Circles().Item(i)
		out = append(out, &dwg.Circle{Center: sketchPt(c.CenterPoint()), Radius: float64(c.Radius), Normal: [3]float64{0, 0, 1}})
	}
	for i := 0; i < sk.Arcs().Count(); i++ {
		out = append(out, arcToDWG(sk.Arcs().Item(i)))
	}
	for i := 0; i < sk.Splines().Count(); i++ {
		out = append(out, splineToDWG(sk.Splines().Item(i)))
	}
	return out
}

// arcToDWG converts a sketch arc to a DWG arc. DWG arcs always sweep counter-clockwise
// from start to end, so a clockwise sketch arc is emitted with its endpoints swapped
// (CCW end→start traces the same geometry).
func arcToDWG(a *sketch.Arc) *dwg.Arc {
	c := a.Center.Position()
	start, end := a.Start.Position(), a.End.Position()
	if !a.CounterClockwise {
		start, end = end, start
	}
	angle := func(p [2]float64) float64 { return math.Atan2(p[1]-c.Y, p[0]-c.X) }
	return &dwg.Arc{
		Center:     [3]float64{c.X, c.Y, 0},
		Radius:     float64(a.Radius()),
		StartAngle: angle([2]float64{start.X, start.Y}),
		EndAngle:   angle([2]float64{end.X, end.Y}),
		Normal:     [3]float64{0, 0, 1},
	}
}

// splineToDWG converts a sketch spline to a DWG spline, emitting its defining points as
// control points (ImportDWG rebuilds the curve from them, so the points round-trip).
func splineToDWG(s *sketch.Spline) *dwg.Spline {
	ctrl := make([][3]float64, s.PointCount())
	for i, p := range s.Points {
		ctrl[i] = sketchPt(p)
	}
	return &dwg.Spline{Degree: 3, ControlPoints: ctrl, Closed: s.Closed}
}

// sketchPt lifts a sketch point's 2D position into a Z=0 DWG coordinate.
func sketchPt(p *sketch.Point) [3]float64 {
	pos := p.Position()
	return [3]float64{pos.X, pos.Y, 0}
}
