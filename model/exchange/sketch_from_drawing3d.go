// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"math"

	"oblikovati.org/kernel/exchange/drawing"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// This file converts the format-neutral drawing model into 3D sketch geometry, used when a
// decoded drawing is non-planar. Shared by every drawing importer (DWG, DXF). The 2D
// adders live in sketch_from_drawing.go.

// add3DEntities maps decoded entities onto a 3D sketch (used when the drawing is
// non-planar). Line/Circle/Arc/Spline/Point/Ellipse/LwPolyline all map directly; an entity
// with no 3D adder is reported as a warning rather than silently dropped.
func add3DEntities(sk *sketch.Sketch3D, entities []drawing.Entity) (int, []string) {
	var warns []string
	added := 0
	for _, e := range entities {
		if add3DEntity(sk, e) {
			added++
		} else {
			warns = append(warns, fmt.Sprintf("import: skipped %s handle %d (no 3D mapping)", e.Kind().String(), e.EntityHandle()))
		}
	}
	return added, warns
}

// add3DEntity places one entity on a 3D sketch; it returns false for a type with no 3D
// mapping.
func add3DEntity(sk *sketch.Sketch3D, e drawing.Entity) bool {
	switch g := e.(type) {
	case *drawing.Line:
		sk.AddLine3D(p3(g.Start), p3(g.End))
	case *drawing.Circle:
		sk.AddCircle3D(p3(g.Center), normal3(g.Normal), g.Radius)
	case *drawing.Arc:
		s, end := arc3DEndpoints(g.Center, g.Radius, g.StartAngle, g.EndAngle)
		sk.AddArc3D(p3(g.Center), s, end, true)
	case *drawing.Point:
		sk.AddPoint3D(p3(g.Position))
	case *drawing.Spline:
		return add3DSpline(sk, g)
	case *drawing.Ellipse:
		add3DEllipse(sk, g)
	case *drawing.LwPolyline:
		return add3DPolyline(sk, g)
	default:
		return false
	}
	return true
}

// add3DEllipse places a DWG ellipse on a 3D sketch in its own plane (center + extrusion
// normal + major axis). A partial parametric span becomes an elliptical arc, a full span a
// closed ellipse — mirroring the 2D mapping (which split arcs out to stop giant ovals).
func add3DEllipse(sk *sketch.Sketch3D, g *drawing.Ellipse) {
	majorR := math.Hypot(math.Hypot(g.MajorAxis[0], g.MajorAxis[1]), g.MajorAxis[2])
	center, normal, axis := p3(g.Center), normal3(g.Normal), axisUnit(g.MajorAxis)
	if isFullEllipse(g.StartAngle, g.EndAngle) {
		sk.AddEllipse3D(center, normal, axis, majorR, majorR*g.AxisRatio)
		return
	}
	sweep := g.EndAngle - g.StartAngle
	if sweep <= 0 {
		sweep += 2 * math.Pi
	}
	sk.AddEllipticalArc3D(center, normal, axis, majorR, majorR*g.AxisRatio, g.StartAngle, sweep)
}

// add3DPolyline places each LWPOLYLINE segment on a 3D sketch at the polyline's elevation: a
// straight run as a 3D line, a bulged run as a 3D arc in the elevation plane. Mirrors
// add2DPolyline; returns false only for a degenerate (<2 vertex) polyline.
func add3DPolyline(sk *sketch.Sketch3D, g *drawing.LwPolyline) bool {
	n := len(g.Points)
	if n < 2 {
		return false
	}
	z := g.Elevation
	last := n - 1
	if g.Closed {
		last = n
	}
	for i := 0; i < last; i++ {
		a, b := g.Points[i], g.Points[(i+1)%n]
		if bulge := bulgeAt(g.Bulges, i); bulge != 0 {
			c, ccw := bulgeArc(a, b, bulge)
			sk.AddArc3D(gmath.P3(c[0], c[1], z), gmath.P3(a[0], a[1], z), gmath.P3(b[0], b[1], z), ccw)
		} else {
			sk.AddLine3D(gmath.P3(a[0], a[1], z), gmath.P3(b[0], b[1], z))
		}
	}
	return true
}

// axisUnit makes a unit vector from a decoded ellipse major-axis vector, defaulting to +X for
// a degenerate (zero) vector.
func axisUnit(v [3]float64) gmath.UnitVector3 {
	u, err := gmath.UnitVector3FromVector(gmath.V3(v[0], v[1], v[2]))
	if err != nil {
		return gmath.V3(1, 0, 0).AsUnit()
	}
	return u
}

// add3DSpline adds a 3D spline from control points, or fit points when present.
func add3DSpline(sk *sketch.Sketch3D, g *drawing.Spline) bool {
	if len(g.ControlPoints) >= 2 {
		sk.AddSpline3D(points3D(g.ControlPoints), g.Closed, false)
		return true
	}
	if len(g.FitPoints) >= 2 {
		sk.AddSpline3D(points3D(g.FitPoints), g.Closed, true)
		return true
	}
	return false
}

// arc3DEndpoints converts an arc's centre/radius/angles into its 3D start and end points
// in the arc's own XY plane (extrusion assumed +Z, the planar case).
func arc3DEndpoints(center [3]float64, radius, start, end float64) (gmath.Point3, gmath.Point3) {
	s := gmath.P3(center[0]+radius*math.Cos(start), center[1]+radius*math.Sin(start), center[2])
	e := gmath.P3(center[0]+radius*math.Cos(end), center[1]+radius*math.Sin(end), center[2])
	return s, e
}

func p3(v [3]float64) gmath.Point3 { return gmath.P3(v[0], v[1], v[2]) }

func points3D(src [][3]float64) []gmath.Point3 {
	out := make([]gmath.Point3, len(src))
	for i, v := range src {
		out[i] = gmath.P3(v[0], v[1], v[2])
	}
	return out
}

// normal3 makes a unit normal from a decoded extrusion vector, defaulting to +Z for a
// degenerate (zero) vector.
func normal3(v [3]float64) gmath.UnitVector3 {
	u, err := gmath.UnitVector3FromVector(gmath.V3(v[0], v[1], v[2]))
	if err != nil {
		return gmath.V3(0, 0, 1).AsUnit()
	}
	return u
}
