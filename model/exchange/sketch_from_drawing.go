// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/drawing"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// This file converts the format-neutral drawing model (kernel/exchange/drawing) into 2D
// sketch geometry. It is shared by every drawing importer (DWG, DXF): a codec decodes a
// file into []drawing.Entity, and these adders place each entity on a sketch. The 3D
// adders live in sketch_from_drawing3d.go; the inverse (sketch → drawing) in
// sketch_to_drawing.go.

// planarTolerance is the Z spread under which a decoded drawing is treated as 2D and
// imported onto a single plane (rather than a 3D sketch).
const planarTolerance = 1e-6

// drawingImport is the shared result of placing a decoded drawing onto a part: which sketch
// kind it landed in, how many entities were added, and per-entity convert warnings. Each
// format wrapper maps it to its own public result type.
type drawingImport struct {
	is3D        bool
	entityCount int
	warnings    []string
}

// importSketchFormat is the one entry every drawing-format import goes through (#1631,
// audit I8): it looks up the format's registered [DrawingDecoder] (format_routes.go),
// decodes the bytes into one drawing per model/page, and places each on the part via
// importDrawing. The format wrappers (ImportDWG/ImportDXF/ImportPDF) are one-line
// delegations, so adding a drawing format is a decoder registration, not a new entry shape.
func importSketchFormat(part *compdef.PartComponentDefinition, format types.ExchangeFormat, data []byte, plane sketch.Plane) (SketchImportResult, error) {
	dec, ok := drawingDecoderFor(format)
	if !ok {
		return SketchImportResult{}, fmt.Errorf("import %s: no drawing decoder registered (want dwg|dxf|pdf)", format)
	}
	drawings, warns, err := dec.Decode(data)
	if err != nil {
		return SketchImportResult{}, fmt.Errorf("import %s: %w", format, err)
	}
	res := SketchImportResult{Warnings: warns}
	for _, dr := range drawings {
		imp := importDrawing(part, dr, plane)
		res.EntityCount += imp.entityCount
		res.Is3D = res.Is3D || imp.is3D
		res.Warnings = append(res.Warnings, imp.warnings...)
	}
	return res, nil
}

// importDrawing scales a decoded drawing into the part's database unit and adds it as a 2D
// sketch on plane (planar drawing) or a 3D sketch (non-planar). This is the format-neutral
// import path every codec shares; only the decode step is format-specific.
func importDrawing(part *compdef.PartComponentDefinition, dr *drawing.Drawing, plane sketch.Plane) drawingImport {
	dr.Entities = drawing.ScaleEntities(dr.Entities, drawingToDocumentScale(dr.Units, part.Units()))
	var recenter []string
	if shifted, offset, did := recenterFarFromOrigin(dr.Entities); did {
		dr.Entities, recenter = shifted, []string{recenterWarning(offset)}
	}
	if _, planar := dr.Planar(planarTolerance); planar {
		n, w := add2DEntities(part.Sketches().Add(plane), dr, dr.Entities)
		return drawingImport{entityCount: n, warnings: append(recenter, w...)}
	}
	n, w := add3DEntities(part.Sketches3D().Add(), dr.Entities)
	return drawingImport{is3D: true, entityCount: n, warnings: append(recenter, w...)}
}

// dbUnitMetres is the model's database length unit in metres (centimetres; see
// param.Length). Drawing coordinates are scaled into it on import.
const dbUnitMetres = 0.01

// drawingToDocumentScale returns the factor that converts a drawing coordinate (in the
// drawing's $INSUNITS unit) to the model's database unit (cm). A drawing with a known unit
// converts by physical size, independent of the document's display preference; a unitless
// drawing (INSUNITS 0) is taken to already be in the document's preferred length unit, so
// it scales by that unit's database factor.
func drawingToDocumentScale(insunits int, units param.UnitsOfMeasure) float64 {
	if m, ok := drawing.MetersPerUnit(insunits); ok {
		return m / dbUnitMetres
	}
	return units.FromPreferred(1, param.Length).Value // cm per the document's preferred unit
}

// add2DEntities maps each decoded entity onto a 2D sketch, returning the count added and
// warnings for any it could not place.
func add2DEntities(sk *sketch.Sketch, dr *drawing.Drawing, entities []drawing.Entity) (int, []string) {
	var warns []string
	added := 0
	for _, e := range entities {
		before := len(sk.Entities())
		if !add2DEntity(sk, e) {
			warns = append(warns, fmt.Sprintf("import: skipped %s handle %d (no 2D mapping)", e.Kind().String(), e.EntityHandle()))
			continue
		}
		added++
		// The dispatch above is a one-case-per-type conversion that returns no entities, so the
		// ones it created are the tail of the sketch — the same before-mark the driven-dimension
		// mode uses, and cheaper than threading a return through every case.
		applyImportedFormat(sk, dr, e, sk.Entities()[before:]) // formatting from the file (#2015)
	}
	return added, warns
}

// add2DEntity places one entity on a 2D sketch using its X/Y coordinates; it returns false
// for a type with no 2D mapping.
//
//nolint:funlen // one-case-per-entity-type conversion dispatch.
func add2DEntity(sk *sketch.Sketch, e drawing.Entity) bool {
	switch g := e.(type) {
	case *drawing.Line:
		sk.Lines().AddByTwoPoints(p2(g.Start), p2(g.End))
	case *drawing.Circle:
		sk.Circles().AddByCenterRadius(p2(g.Center), g.Radius)
	case *drawing.Arc:
		s, end := arcEndpoints(g.Center, g.Radius, g.StartAngle, g.EndAngle)
		sk.Arcs().AddByCenterStartEnd(p2(g.Center), s, end, true)
	case *drawing.Point:
		sk.Points().Add(p2(g.Position))
	case *drawing.Ellipse:
		majorR := math.Hypot(g.MajorAxis[0], g.MajorAxis[1])
		axis := gmath.V2(g.MajorAxis[0], g.MajorAxis[1])
		// A partial ellipse (start/end parametric angles) is an elliptical arc; importing
		// those as a full ellipse drew giant closed ovals over the drawing. DXF ELLIPSE start/end
		// params are eccentric-anomaly (parametric), so add them verbatim — NOT through the
		// true-angle EllipticalArcs.Add, which would re-interpret and mis-place the arc (#1829).
		if isFullEllipse(g.StartAngle, g.EndAngle) {
			sk.Ellipses().Add(p2(g.Center), axis, majorR, majorR*g.AxisRatio)
		} else {
			sk.EllipticalArcs().AddParametric(p2(g.Center), axis, majorR, majorR*g.AxisRatio, g.StartAngle, g.EndAngle)
		}
	case *drawing.Spline:
		add2DSpline(sk, g)
	case *drawing.LwPolyline:
		add2DPolyline(sk, g)
	default:
		return false
	}
	return true
}

// add2DSpline adds a spline from its control points, or its fit points when no control
// points are present.
func add2DSpline(sk *sketch.Sketch, g *drawing.Spline) {
	if len(g.ControlPoints) >= 2 {
		sk.Splines().AddByControlPoints(points2D(g.ControlPoints), g.Closed)
		return
	}
	if len(g.FitPoints) >= 2 {
		sk.Splines().AddByPoints(points2D(g.FitPoints), g.Closed)
	}
}

// add2DPolyline adds each polyline segment as a line, or an arc when the vertex carries a
// non-zero bulge. Closed polylines also join the last vertex to the first. Consecutive
// segments share their vertex points (a polyline is a connected chain) — both more
// faithful and roughly a third less data than independent segments, which matters for the
// undo snapshot and renderer on dense drawings.
func add2DPolyline(sk *sketch.Sketch, g *drawing.LwPolyline) {
	n := len(g.Points)
	if n < 2 {
		return
	}
	verts := make([]*sketch.Point, n)
	for i, p := range g.Points {
		verts[i] = sk.NewPoint(gmath.P2(p[0], p[1]))
	}
	last := n - 1
	if g.Closed {
		last = n
	}
	for i := 0; i < last; i++ {
		a, b := verts[i], verts[(i+1)%n]
		if bulge := bulgeAt(g.Bulges, i); bulge != 0 {
			center, ccw := bulgeArc(g.Points[i], g.Points[(i+1)%n], bulge)
			sk.Arcs().Add(sk.NewPoint(gmath.P2(center[0], center[1])), a, b, ccw)
		} else {
			sk.Lines().Add(a, b)
		}
	}
}

// bulgeAt returns the bulge for vertex i, or 0 when none is stored.
func bulgeAt(bulges []float64, i int) float64 {
	if i < len(bulges) {
		return bulges[i]
	}
	return 0
}

// bulgeArc derives the arc centre and orientation for a bulged polyline segment. A bulge
// is tan(theta/4) of the included angle; positive is counter-clockwise.
func bulgeArc(a, b [2]float64, bulge float64) ([2]float64, bool) {
	cot := (1/bulge - bulge) / 2
	cx := (a[0]+b[0])/2 - cot*(b[1]-a[1])/2
	cy := (a[1]+b[1])/2 + cot*(b[0]-a[0])/2
	return [2]float64{cx, cy}, bulge > 0
}

// arcEndpoints converts an arc's centre/radius/angles into its start and end points on a
// 2D sketch.
func arcEndpoints(center [3]float64, radius, start, end float64) (gmath.Point2, gmath.Point2) {
	s := gmath.P2(center[0]+radius*math.Cos(start), center[1]+radius*math.Sin(start))
	e := gmath.P2(center[0]+radius*math.Cos(end), center[1]+radius*math.Sin(end))
	return s, e
}

// isFullEllipse reports whether an ellipse's parametric angle span covers the whole
// perimeter (start≈0, end≈2π) — the only case that maps to a closed sketch Ellipse; any
// shorter span is an elliptical arc.
func isFullEllipse(start, end float64) bool {
	return math.Abs(start) < 1e-6 && math.Abs(end-2*math.Pi) < 1e-6
}

// p2 projects a 3D coordinate onto the sketch plane's X/Y.
func p2(v [3]float64) gmath.Point2 { return gmath.P2(v[0], v[1]) }

// points2D projects a list of 3D coordinates to 2D sketch points.
func points2D(src [][3]float64) []gmath.Point2 {
	out := make([]gmath.Point2, len(src))
	for i, v := range src {
		out[i] = gmath.P2(v[0], v[1])
	}
	return out
}
