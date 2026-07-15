// SPDX-License-Identifier: GPL-2.0-only

// Package translate maps a decoded SolidWorks part onto a native Oblikovati document: each global
// variable becomes a user parameter, each decoded sketch an Oblikovati sketch. It is the offline
// twin of the SolidWorks COM API — no running SolidWorks — mirroring the Inventor translator.
package translate

import (
	"fmt"
	"math"

	m "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange/translators/solidworks/sldprt"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

// metresToCm converts a SolidWorks database coordinate (metres) to the Oblikovati database unit
// (centimetres).
const metresToCm = 100.0

// FromSolidWorks translates a .SLDPRT into a native Oblikovati part package at outPath: global
// variables become user parameters and every decoded sketch is emitted. It saves the partial
// parametric state (parameters + sketches) — features and the solid body are later work. Returns
// non-fatal warnings.
func FromSolidWorks(sldBytes []byte, outPath string) ([]string, error) {
	d, err := sldprt.Open(sldBytes)
	if err != nil {
		return nil, err
	}
	if d.Type == sldprt.Assembly {
		return nil, fmt.Errorf("solidworks: assembly translation not yet supported")
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, warns, err := buildPart(ws, outPath, d)
	if err != nil {
		return warns, err
	}
	if err := ws.Save(document); err != nil {
		return warns, err
	}
	return warns, nil
}

// buildPart adds a part document to ws at outPath and populates it from the decoded part —
// parameters, then sketches. It stops short of saving so callers control that.
func buildPart(ws *doc.Workspace, outPath string, d *sldprt.Document) (*doc.Document, []string, error) {
	document, err := compdef.AddPart(ws, outPath, true)
	if err != nil {
		return nil, nil, err
	}
	def := document.Content().(*compdef.PartComponentDefinition)
	warns := addParameters(def, d)
	warns = append(warns, addSketches(def, d)...)
	return document, warns, nil
}

// addParameters maps each global variable onto an Oblikovati user parameter, converting units to
// the database convention (length -> cm). A formula-valued variable is passed through as its
// written expression; a variable in an unhandled unit is reported and skipped.
func addParameters(def *compdef.PartComponentDefinition, d *sldprt.Document) []string {
	var warns []string
	for _, p := range d.Parameters() {
		expr, ok := parameterExpression(p)
		if !ok {
			warns = append(warns, fmt.Sprintf("parameter %q: unsupported unit in %q", p.Name, p.Expression))
			continue
		}
		if _, err := def.Parameters().AddUserParameter(p.Name, expr); err != nil {
			warns = append(warns, fmt.Sprintf("parameter %q: %v", p.Name, err))
		}
	}
	return warns
}

// parameterExpression renders a decoded global variable as an Oblikovati parameter expression. A
// numeric literal is converted to the database unit (length in cm, angle in degrees, plain number
// as-is); a formula is passed through verbatim (ok is still true so the parameter is created).
func parameterExpression(p sldprt.Parameter) (string, bool) {
	value, unit, ok := p.Number()
	if !ok {
		return p.Expression, true // a formula referencing other variables
	}
	switch unit {
	case "":
		return fmt.Sprintf("%g", value), true
	case "deg", "rad":
		return fmt.Sprintf("%g %s", value, unit), true
	default:
		cm, ok := lengthToCm(value, unit)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("%g cm", cm), true
	}
}

// lengthToCm converts a length value in a SolidWorks unit to centimetres.
func lengthToCm(value float64, unit string) (float64, bool) {
	factor := map[string]float64{"mm": 0.1, "cm": 1, "m": 100, "in": 2.54, "ft": 30.48}
	f, ok := factor[unit]
	return value * f, ok
}

// addSketches emits every decoded sketch onto the XY plane, converting metres to centimetres.
func addSketches(def *compdef.PartComponentDefinition, d *sldprt.Document) []string {
	for _, s := range d.Sketches() {
		emitSketch(def, s)
	}
	return nil
}

// emitSketch adds one decoded sketch to the document on the XY plane, in centimetres. Endpoints
// that share a coordinate share one sketch point (sharedPoints), so a closed profile stays closed
// with the same degrees of freedom as the source rather than minting free endpoints per segment.
func emitSketch(def *compdef.PartComponentDefinition, s sldprt.Sketch) *sketch.Sketch {
	if len(s.Points)+len(s.Lines)+len(s.Circles)+len(s.Arcs)+len(s.Ellipses)+len(s.Splines) == 0 {
		return nil
	}
	sk := def.Sketches().Add(sketch.XYPlane())
	pointAt := sharedPoints(sk)
	// Standalone points are emitted only for a point-only sketch; otherwise each entity creates its
	// own points (via pointAt / AddByCenterRadius), so emitting s.Points too would add free points.
	var points []*sketch.Point
	if len(s.Lines)+len(s.Circles)+len(s.Arcs)+len(s.Ellipses)+len(s.Splines) == 0 {
		for _, p := range s.Points {
			points = append(points, pointAt(p))
		}
	}
	lines := make([]*sketch.Line, len(s.Lines))
	for i, l := range s.Lines {
		lines[i] = sk.Lines().Add(pointAt(l.A), pointAt(l.B))
	}
	arcs := make([]*sketch.Arc, len(s.Arcs))
	for i, a := range s.Arcs {
		arcs[i] = sk.Arcs().Add(pointAt(a.Center), pointAt(a.Start), pointAt(a.End), minorArcCCW(a))
	}
	circles := make([]*sketch.Circle, len(s.Circles))
	for i, c := range s.Circles {
		circles[i] = sk.Circles().AddByCenterRadius(m.P2(c.Center.X*metresToCm, c.Center.Y*metresToCm), m.Scalar(c.Radius*metresToCm))
	}
	for _, e := range s.Ellipses {
		sk.Ellipses().Add(m.P2(e.Center.X*metresToCm, e.Center.Y*metresToCm), m.V2(m.Scalar(e.MajorX), m.Scalar(e.MajorY)), m.Scalar(e.MajorRadius*metresToCm), m.Scalar(e.MinorRadius*metresToCm))
	}
	for _, sp := range s.Splines {
		fit := make([]m.Point2, len(sp.FitPoints))
		for i, p := range sp.FitPoints {
			fit[i] = m.P2(p.X*metresToCm, p.Y*metresToCm)
		}
		sk.Splines().AddByPoints(fit, sp.Closed)
	}
	applyConstruction(s, lines, arcs, circles)
	applyConstraints(sk, s.Constraints, lines, arcs, circles)
	applyDimensions(sk, s.Dimensions, lines, arcs, circles, points)
	return sk
}

// applyConstruction marks the emitted entities that SolidWorks stored as construction (reference)
// geometry, so they shape constraints/dimensions but are excluded from profiles. The decoded flags
// are in draw order across all kinds; this only acts where that order maps unambiguously onto a typed
// slice: a pure-circle sketch (circles are decoded in cached = draw order) or a lone arc. For a line
// loop the vertices are re-ordered during reconstruction, so its draw order is not recoverable yet —
// attribution there waits on the entity-graph walk; the flags are still decoded on the Sketch.
func applyConstruction(s sldprt.Sketch, lines []*sketch.Line, arcs []*sketch.Arc, circles []*sketch.Circle) {
	// Lines carry their own per-entity flags from the exact reference reconstruction (index-aligned to
	// Lines), so they are marked directly regardless of the other kinds present.
	if len(s.LineConstruction) == len(lines) {
		for i, l := range lines {
			l.SetConstruction(s.LineConstruction[i])
		}
	}
	// Circles/arcs use the draw-order flags, mapped only where that order is unambiguous.
	flags := s.Construction
	if len(flags) != len(lines)+len(arcs)+len(circles) {
		return // count mismatch: cannot trust the mapping
	}
	switch {
	case len(lines) == 0 && len(arcs) == 0:
		for i, c := range circles {
			c.SetConstruction(flags[i])
		}
	case len(lines) == 0 && len(circles) == 0 && len(arcs) == 1:
		arcs[0].SetConstruction(flags[0])
	}
}

// sharedPoints returns a resolver that maps a SolidWorks coordinate (metres) to a sketch Point in
// centimetres, returning the same Point for coordinates within coincideEps so touching corners are
// coincident.
func sharedPoints(sk *sketch.Sketch) func(sldprt.Point) *sketch.Point {
	type cached struct {
		x, y float64
		pt   *sketch.Point
	}
	var cache []cached
	return func(p sldprt.Point) *sketch.Point {
		x, y := p.X*metresToCm, p.Y*metresToCm
		for _, e := range cache {
			if math.Abs(e.x-x) < coincideEps && math.Abs(e.y-y) < coincideEps {
				return e.pt
			}
		}
		pt := sk.Points().Add(m.P2(x, y))
		cache = append(cache, cached{x, y, pt})
		return pt
	}
}

// coincideEps is the coordinate tolerance (cm) below which two corners are treated as one point.
const coincideEps = 1e-6

// minorArcCCW reports whether the minor arc from Start to End sweeps counter-clockwise about the
// centre — true when the CCW span (start angle -> end angle) is the shorter (<= pi) way round.
func minorArcCCW(a sldprt.Arc) bool {
	a0 := math.Atan2(a.Start.Y-a.Center.Y, a.Start.X-a.Center.X)
	a1 := math.Atan2(a.End.Y-a.Center.Y, a.End.X-a.Center.X)
	sweep := a1 - a0
	for sweep < 0 {
		sweep += 2 * math.Pi
	}
	return sweep <= math.Pi
}
