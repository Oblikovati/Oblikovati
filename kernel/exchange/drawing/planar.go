// SPDX-License-Identifier: GPL-2.0-only

package drawing

import "math"

// Planar reports whether every entity lies in one Z=constant plane (within tol) and
// returns that elevation. It routes an import to a 2D Sketch (with the returned elevation
// as the plane offset) versus a Sketch3D. An empty drawing is treated as planar at z=0.
func (d *Drawing) Planar(tol float64) (elevation float64, planar bool) {
	first := true
	var z float64
	check := func(v float64) bool {
		if first {
			z, first = v, false
			return true
		}
		return math.Abs(v-z) <= tol
	}
	for _, e := range d.Entities {
		for _, v := range entityZ(e) {
			if !check(v) {
				return 0, false
			}
		}
	}
	return z, true
}

// entityZ returns the Z coordinates an entity contributes to the planarity test.
//
//nolint:funlen // one-case-per-entity-type dispatch returning each type's Z coordinates.
func entityZ(e Entity) []float64 {
	switch g := e.(type) {
	case *Line:
		return []float64{g.Start[2], g.End[2]}
	case *Circle:
		return []float64{g.Center[2]}
	case *Arc:
		return []float64{g.Center[2]}
	case *Point:
		return []float64{g.Position[2]}
	case *Ellipse:
		return []float64{g.Center[2]}
	case *LwPolyline:
		return []float64{g.Elevation}
	case *Spline:
		zs := make([]float64, 0, len(g.ControlPoints)+len(g.FitPoints))
		for _, c := range g.ControlPoints {
			zs = append(zs, c[2])
		}
		for _, f := range g.FitPoints {
			zs = append(zs, f[2])
		}
		return zs
	default:
		return nil
	}
}
