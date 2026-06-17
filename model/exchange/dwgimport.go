// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"math"
	"os"

	"oblikovati.org/kernel/exchange/dwg"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// ImportDWGFile reads a .dwg file and imports it into part on the chosen plane (2D)
// or as a Sketch3D, the path-based companion to ImportDWG. The caller resolves the
// plane from the work plane the user picked.
func ImportDWGFile(part *compdef.PartComponentDefinition, path string, plane sketch.Plane) (DWGImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DWGImportResult{}, fmt.Errorf("import dwg: read %q: %w", path, err)
	}
	return ImportDWG(part, data, plane)
}

// dwgPlanarTolerance is the Z spread under which a decoded drawing is treated as
// 2D and imported onto a single plane.
const dwgPlanarTolerance = 1e-6

// DWGImportResult summarises a DWG import: whether it landed in a 3D sketch (vs a
// 2D sketch on the chosen plane), how many entities were added, and any
// per-entity decode/convert warnings.
type DWGImportResult struct {
	Is3D        bool
	EntityCount int
	Warnings    []string
}

// ImportDWG decodes DWG bytes and adds their geometry to part. A planar drawing
// becomes a 2D Sketch on plane (the DWG world origin mapping to the plane origin,
// per the caller's chosen work plane); a non-planar drawing becomes a Sketch3D and
// plane is ignored. Entities of a type with no sketch mapping yet are skipped with
// a warning rather than failing the import.
//
// Example:
//
//	res, err := exchange.ImportDWG(part, data, workPlane.Plane())
func ImportDWG(part *compdef.PartComponentDefinition, data []byte, plane sketch.Plane) (DWGImportResult, error) {
	drawing, warns, err := dwg.Decode(data)
	if err != nil {
		return DWGImportResult{}, fmt.Errorf("import dwg: %w", err)
	}
	drawing.Entities = dwg.ScaleEntities(drawing.Entities, dwgToDocumentScale(drawing.Units, part.Units()))
	if _, planar := drawing.Planar(dwgPlanarTolerance); planar {
		n, w := add2DEntities(part.Sketches().Add(plane), drawing.Entities)
		return DWGImportResult{EntityCount: n, Warnings: append(warns, w...)}, nil
	}
	n, w := add3DEntities(part.Sketches3D().Add(), drawing.Entities)
	return DWGImportResult{Is3D: true, EntityCount: n, Warnings: append(warns, w...)}, nil
}

// dbUnitMetres is the model's database length unit in metres (centimetres; see
// param.Length). DWG coordinates are scaled into it on import.
const dbUnitMetres = 0.01

// dwgToDocumentScale returns the factor that converts a DWG coordinate (in the drawing's
// $INSUNITS unit) to the model's database unit (cm). A drawing with a known unit converts
// by physical size, independent of the document's display preference; a unitless drawing
// (INSUNITS 0) is taken to already be in the document's preferred length unit, so it scales
// by that unit's database factor.
func dwgToDocumentScale(insunits int, units param.UnitsOfMeasure) float64 {
	if m, ok := dwg.MetersPerUnit(insunits); ok {
		return m / dbUnitMetres
	}
	return units.FromPreferred(1, param.Length).Value // cm per the document's preferred unit
}

// add2DEntities maps each decoded entity onto a 2D sketch, returning the count
// added and warnings for any it could not place.
func add2DEntities(sk *sketch.Sketch, entities []dwg.Entity) (int, []string) {
	var warns []string
	added := 0
	for _, e := range entities {
		if add2DEntity(sk, e) {
			added++
		} else {
			warns = append(warns, fmt.Sprintf("dwg: skipped %s handle %d (no 2D mapping)", e.Kind().String(), e.EntityHandle()))
		}
	}
	return added, warns
}

// add2DEntity places one entity on a 2D sketch using its X/Y coordinates; it
// returns false for a type with no 2D mapping.
//
//nolint:funlen // one-case-per-entity-type conversion dispatch.
func add2DEntity(sk *sketch.Sketch, e dwg.Entity) bool {
	switch g := e.(type) {
	case *dwg.Line:
		sk.Lines().AddByTwoPoints(p2(g.Start), p2(g.End))
	case *dwg.Circle:
		sk.Circles().AddByCenterRadius(p2(g.Center), g.Radius)
	case *dwg.Arc:
		s, end := arcEndpoints(g.Center, g.Radius, g.StartAngle, g.EndAngle)
		sk.Arcs().AddByCenterStartEnd(p2(g.Center), s, end, true)
	case *dwg.Point:
		sk.Points().Add(p2(g.Position))
	case *dwg.Ellipse:
		majorR := math.Hypot(g.MajorAxis[0], g.MajorAxis[1])
		axis := gmath.V2(g.MajorAxis[0], g.MajorAxis[1])
		// AutoCAD stores most arcs/elliptical arcs as a partial ELLIPSE entity
		// (start/end parametric angles). Importing those as a full ellipse drew giant
		// closed ovals over the drawing — emit a bounded EllipticalArc when partial.
		if isFullEllipse(g.StartAngle, g.EndAngle) {
			sk.Ellipses().Add(p2(g.Center), axis, majorR, majorR*g.AxisRatio)
		} else {
			sk.EllipticalArcs().Add(p2(g.Center), axis, majorR, majorR*g.AxisRatio, g.StartAngle, g.EndAngle)
		}
	case *dwg.Spline:
		add2DSpline(sk, g)
	case *dwg.LwPolyline:
		add2DPolyline(sk, g)
	default:
		return false
	}
	return true
}

// add2DSpline adds a spline from its control points, or its fit points when no
// control points are present.
func add2DSpline(sk *sketch.Sketch, g *dwg.Spline) {
	if len(g.ControlPoints) >= 2 {
		sk.Splines().AddByControlPoints(points2D(g.ControlPoints), g.Closed)
		return
	}
	if len(g.FitPoints) >= 2 {
		sk.Splines().AddByPoints(points2D(g.FitPoints), g.Closed)
	}
}

// add2DPolyline adds each polyline segment as a line, or an arc when the vertex
// carries a non-zero bulge. Closed polylines also join the last vertex to the
// first. Consecutive segments share their vertex points (a polyline is a connected
// chain) — both more faithful and roughly a third less data than independent
// segments, which matters for the undo snapshot and renderer on dense drawings.
func add2DPolyline(sk *sketch.Sketch, g *dwg.LwPolyline) {
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

// bulgeArc derives the arc centre and orientation for a bulged polyline segment.
// A bulge is tan(theta/4) of the included angle; positive is counter-clockwise.
func bulgeArc(a, b [2]float64, bulge float64) ([2]float64, bool) {
	cot := (1/bulge - bulge) / 2
	cx := (a[0]+b[0])/2 - cot*(b[1]-a[1])/2
	cy := (a[1]+b[1])/2 + cot*(b[0]-a[0])/2
	return [2]float64{cx, cy}, bulge > 0
}

// arcEndpoints converts an arc's centre/radius/angles into its start and end
// points on a 2D sketch.
func arcEndpoints(center [3]float64, radius, start, end float64) (gmath.Point2, gmath.Point2) {
	s := gmath.P2(center[0]+radius*math.Cos(start), center[1]+radius*math.Sin(start))
	e := gmath.P2(center[0]+radius*math.Cos(end), center[1]+radius*math.Sin(end))
	return s, e
}

// isFullEllipse reports whether a DWG ellipse's parametric angle span covers the whole
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
