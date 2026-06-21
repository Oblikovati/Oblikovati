// SPDX-License-Identifier: GPL-2.0-only

package drawing

// EntityAnchor returns a single representative point for an entity — its centre, start, or
// first vertex — summarising where the entity sits without sampling its whole geometry. The
// importer averages/medians these to find a drawing's centre for recentering (see
// model/exchange recenter). ok is false for a type that carries no positional anchor.
//
// Example:
//
//	if p, ok := drawing.EntityAnchor(e); ok { /* p[0],p[1],p[2] */ }
//
//nolint:funlen // one-case-per-entity-type dispatch returning each type's anchor point.
func EntityAnchor(e Entity) (p [3]float64, ok bool) {
	switch g := e.(type) {
	case *Line:
		return g.Start, true
	case *Circle:
		return g.Center, true
	case *Arc:
		return g.Center, true
	case *Point:
		return g.Position, true
	case *Ellipse:
		return g.Center, true
	case *LwPolyline:
		if len(g.Points) == 0 {
			return p, false
		}
		return [3]float64{g.Points[0][0], g.Points[0][1], g.Elevation}, true
	case *Spline:
		if len(g.ControlPoints) > 0 {
			return g.ControlPoints[0], true
		}
		if len(g.FitPoints) > 0 {
			return g.FitPoints[0], true
		}
		return p, false
	default:
		return p, false
	}
}
