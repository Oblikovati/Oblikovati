// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"math"

	"oblikovati.org/kernel/exchange/dwg"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// add3DEntities maps decoded entities onto a 3D sketch (used when the drawing is
// non-planar). Line/Circle/Arc/Spline/Point map directly; ellipses and bulged
// polylines have no 3D adder yet and are reported as warnings.
func add3DEntities(sk *sketch.Sketch3D, entities []dwg.Entity) (int, []string) {
	var warns []string
	added := 0
	for _, e := range entities {
		if add3DEntity(sk, e) {
			added++
		} else {
			warns = append(warns, fmt.Sprintf("dwg: skipped %s handle %d (no 3D mapping)", e.EntityType().Name(), e.EntityHandle()))
		}
	}
	return added, warns
}

// add3DEntity places one entity on a 3D sketch; it returns false for a type with
// no 3D mapping.
func add3DEntity(sk *sketch.Sketch3D, e dwg.Entity) bool {
	switch g := e.(type) {
	case *dwg.Line:
		sk.AddLine3D(p3(g.Start), p3(g.End))
	case *dwg.Circle:
		sk.AddCircle3D(p3(g.Center), normal3(g.Normal), g.Radius)
	case *dwg.Arc:
		s, end := arc3DEndpoints(g.Center, g.Radius, g.StartAngle, g.EndAngle)
		sk.AddArc3D(p3(g.Center), s, end, true)
	case *dwg.Point:
		sk.AddPoint3D(p3(g.Position))
	case *dwg.Spline:
		return add3DSpline(sk, g)
	default:
		return false
	}
	return true
}

// add3DSpline adds a 3D spline from control points, or fit points when present.
func add3DSpline(sk *sketch.Sketch3D, g *dwg.Spline) bool {
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

// arc3DEndpoints converts an arc's centre/radius/angles into its 3D start and end
// points in the arc's own XY plane (extrusion assumed +Z, the planar case).
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

// normal3 makes a unit normal from a decoded extrusion vector, defaulting to +Z
// for a degenerate (zero) vector.
func normal3(v [3]float64) gmath.UnitVector3 {
	u, err := gmath.UnitVector3FromVector(gmath.V3(v[0], v[1], v[2]))
	if err != nil {
		return gmath.V3(0, 0, 1).AsUnit()
	}
	return u
}
