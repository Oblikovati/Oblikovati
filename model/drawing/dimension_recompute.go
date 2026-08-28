// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// Drawing dimensions — the SHARED ATTACHMENT / REFERENCE RESOLUTION (M48 #2225 split of dimension.go).
// The associativity path every family shares: Recompute walks the collection; recompute re-binds one
// dimension's geometry and dispatches to the per-family recompute*; dimensionBasis resolves the view,
// referenced body and projection frame; and the small shared measurement, text-anchor, vertex-snap and
// glyph-segment helpers each family calls. (The per-family Find*ByKey binding lives in each family file;
// guarding those seven sites through the collision-checked resolver is the phase-2 C6-c task.)

// Recompute re-measures every dimension against the current model — the associativity path.
func (ds *DrawingDimensions) Recompute() {
	for _, d := range ds.items {
		ds.recompute(d)
	}
}

// recompute re-binds the dimension's geometry, re-measures and rebuilds the glyph. With no
// resolvable view, model or geometry it clears the dimension.
func (ds *DrawingDimensions) recompute(d *DrawingDimension) {
	d.curves = nil
	view, body, basis, err := ds.dimensionBasis(d.viewName)
	if err != nil {
		return
	}
	if d.retrievedFrom != "" {
		ds.recomputeRetrieved(d, view, basis)
		return
	}
	if isRadial(d.dimType) {
		ds.recomputeRadial(d, view, body, basis)
		return
	}
	if d.dimType == types.AngularDimension {
		ds.recomputeAngular(d, view, body, basis)
		return
	}
	if d.dimType == types.OrdinateDimension {
		ds.recomputeOrdinate(d, view, body, basis)
		return
	}
	if d.dimType == types.ArcLengthDimension {
		ds.recomputeArcLength(d, view, body, basis)
		return
	}
	ds.recomputeLinear(d, view, body, basis)
}

// setTextAnchor records the dimension-line midpoint (mx, my) and the dimension line's unit
// perpendicular nx,ny (the offset axis — line-drag follows it). It lifts the default text anchor
// textGapMM off the line along liftSign*perpendicular (away from the measured geometry).
func (d *DrawingDimension) setTextAnchor(mx, my, nx, ny, liftSign float64) {
	if l := math.Hypot(nx, ny); l > 1e-9 {
		nx, ny = nx/l, ny/l
	}
	d.nx, d.ny = nx, ny
	d.anchorX, d.anchorY = mx+nx*liftSign*textGapMM, my+ny*liftSign*textGapMM
}

// dimensionBasis resolves a dimension's view (which must be a base view in this increment), the
// referenced model body and the view's projection frame.
func (ds *DrawingDimensions) dimensionBasis(viewName string) (*DrawingView, *topo.Body, hlr.View, error) {
	view, ok := ds.views.ByName(viewName)
	if !ok {
		return nil, nil, hlr.View{}, fmt.Errorf("drawing: no view %q to dimension", viewName)
	}
	if view.viewType != types.DrawingViewBase {
		return nil, nil, hlr.View{}, fmt.Errorf("drawing: %q is not a base view; dimension a base view", viewName)
	}
	if ds.body == nil {
		return nil, nil, hlr.View{}, fmt.Errorf("drawing: no referenced model to dimension")
	}
	body, ok := ds.body()
	if !ok {
		return nil, nil, hlr.View{}, fmt.Errorf("drawing: no referenced model to dimension")
	}
	return view, body, baseBasis(view.orientation, bodyCenter(body)), nil
}

// measureMM is the dimension's value in millimetres from the two view-plane points (centimetres):
// the true distance for an aligned dimension, or the view-X/Y component for horizontal/vertical.
func measureMM(t types.DrawingDimensionType, p1, p2 gmath.Point2) float64 {
	du, dv := float64(p2.X-p1.X), float64(p2.Y-p1.Y)
	switch t {
	case types.HorizontalDimension:
		return math.Abs(du) * cmToMM
	case types.VerticalDimension:
		return math.Abs(dv) * cmToMM
	default:
		return math.Hypot(du, dv) * cmToMM
	}
}

// dimensionAxis is the sheet-space direction the dimension is measured along: the line between the
// points for an aligned dimension, or the sheet X/Y axis for horizontal/vertical.
func dimensionAxis(t types.DrawingDimensionType, s1, s2 gmath.Point2) (ax, ay float64) {
	switch t {
	case types.HorizontalDimension:
		return 1, 0
	case types.VerticalDimension:
		return 0, 1
	default:
		dx, dy := float64(s2.X-s1.X), float64(s2.Y-s1.Y)
		l := math.Hypot(dx, dy)
		if l < 1e-9 {
			return 1, 0
		}
		return dx / l, dy / l
	}
}

// nearestVertexKey returns the reference key of the model vertex whose projection on the view sits
// closest to the pick (sheet mm), and false when the body has no vertices.
func nearestVertexKey(body *topo.Body, v *DrawingView, basis hlr.View, x, y float64) ([]byte, bool) {
	verts := body.Vertices()
	best := -1
	var bestD float64
	for i, vt := range verts {
		s := v.place(hlr.ProjectPoint(basis, vt.Point()))
		dx, dy := float64(s.X)-x, float64(s.Y)-y
		d := dx*dx + dy*dy
		if best < 0 || d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return nil, false
	}
	return verts[best].ReferenceKey(), true
}

// arrowheadCurves builds the two short barbs of an arrowhead whose tip is at (tipX, tipY) and
// whose shaft runs toward (fromX, fromY).
func arrowheadCurves(tipX, tipY, fromX, fromY float64) []DrawingCurve {
	const length, angle = 3.0, 0.35 // mm, radians (~20°)
	ux, uy := fromX-tipX, fromY-tipY
	l := math.Hypot(ux, uy)
	if l < 1e-9 {
		return nil
	}
	ux, uy = ux/l, uy/l
	cos, sin := math.Cos(angle), math.Sin(angle)
	b1x, b1y := tipX+length*(ux*cos-uy*sin), tipY+length*(ux*sin+uy*cos)
	b2x, b2y := tipX+length*(ux*cos+uy*sin), tipY+length*(-ux*sin+uy*cos)
	return []DrawingCurve{dimSegment(tipX, tipY, b1x, b1y), dimSegment(tipX, tipY, b2x, b2y)}
}

// dimSegment builds a visible drawing curve between two sheet-mm points.
func dimSegment(ax, ay, bx, by float64) DrawingCurve {
	return DrawingCurve{
		A: gmath.P2(gmath.Scalar(ax), gmath.Scalar(ay)), B: gmath.P2(gmath.Scalar(bx), gmath.Scalar(by)),
		Visible: true, kind: types.DrawingEdgeCurve,
	}
}
