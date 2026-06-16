// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"math"

	"oblikovati.org/kernel/exchange/dwg"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

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
	if _, planar := drawing.Planar(dwgPlanarTolerance); planar {
		n, w := add2DEntities(part.Sketches().Add(plane), drawing.Entities)
		return DWGImportResult{EntityCount: n, Warnings: append(warns, w...)}, nil
	}
	n, w := add3DEntities(part.Sketches3D().Add(), drawing.Entities)
	return DWGImportResult{Is3D: true, EntityCount: n, Warnings: append(warns, w...)}, nil
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
			warns = append(warns, fmt.Sprintf("dwg: skipped %s handle %d (no 2D mapping)", e.EntityType().Name(), e.EntityHandle()))
		}
	}
	return added, warns
}

// add2DEntity places one entity on a 2D sketch using its X/Y coordinates; it
// returns false for a type with no 2D mapping.
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
		sk.Ellipses().Add(p2(g.Center), gmath.V2(g.MajorAxis[0], g.MajorAxis[1]), majorR, majorR*g.AxisRatio)
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
// first.
func add2DPolyline(sk *sketch.Sketch, g *dwg.LwPolyline) {
	n := len(g.Points)
	if n < 2 {
		return
	}
	last := n - 1
	if g.Closed {
		last = n
	}
	for i := 0; i < last; i++ {
		a := g.Points[i]
		b := g.Points[(i+1)%n]
		if bulge := bulgeAt(g.Bulges, i); bulge != 0 {
			center, ccw := bulgeArc(a, b, bulge)
			sk.Arcs().AddByCenterStartEnd(gmath.P2(center[0], center[1]), gmath.P2(a[0], a[1]), gmath.P2(b[0], b[1]), ccw)
		} else {
			sk.Lines().AddByTwoPoints(gmath.P2(a[0], a[1]), gmath.P2(b[0], b[1]))
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
