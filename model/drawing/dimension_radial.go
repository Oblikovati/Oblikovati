// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// Drawing dimensions — the RADIAL / DIAMETRAL / ARC-LENGTH family (M48 #2225 split of dimension.go).
// These annotate a circular or arc model edge (by reference key): a radius/diameter callout on the
// nearest circle, an arc-length callout following the projected arc, and the auto-callout that
// dimensions every distinct circle in a view. Each re-measures the true model size on recompute.

// AddRadial adds a radius or diameter dimension on the circular model edge nearest the pick point
// (sheet mm) projected on the named base view; the value re-measures when the model changes.
func (ds *DrawingDimensions) AddRadial(name, viewName string, dimType types.DrawingDimensionType, pickX, pickY float64) (*DrawingDimension, error) {
	view, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	key, ok := nearestCircularEdgeKey(body, view, basis, pickX, pickY)
	if !ok {
		return nil, fmt.Errorf("drawing: view %q has no circular edge to dimension", viewName)
	}
	d := &DrawingDimension{name: ds.uniqueName(name), dimType: dimType, viewName: viewName, edgeKey: key}
	ds.recompute(d)
	ds.items = append(ds.items, d)
	return d, nil
}

// AddArcLength adds an arc-length dimension on the circular/arc model edge nearest the pick point
// (sheet mm) projected on the named base view; the value re-measures when the model changes.
func (ds *DrawingDimensions) AddArcLength(name, viewName string, pickX, pickY float64) (*DrawingDimension, error) {
	view, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	key, ok := nearestArcEdgeKey(body, view, basis, pickX, pickY)
	if !ok {
		return nil, fmt.Errorf("drawing: view %q has no circular or arc edge to dimension", viewName)
	}
	d := &DrawingDimension{name: ds.uniqueName(name), dimType: types.ArcLengthDimension, viewName: viewName, edgeKey: key}
	ds.recompute(d)
	ds.items = append(ds.items, d)
	return d, nil
}

// AddRadialForEachCircle adds a radius or diameter dimension for every distinct circular edge in
// the named base view — auto hole callouts. Coincident projections (a through-hole's two rims) are
// dimensioned once. Returns how many it added.
func (ds *DrawingDimensions) AddRadialForEachCircle(viewName string, dimType types.DrawingDimensionType) (int, error) {
	view, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	added := 0
	for _, e := range body.Edges() {
		circle, ok := e.Geometry().(geom.Circle)
		if !ok {
			continue
		}
		c := view.place(hlr.ProjectPoint(basis, circle.Center))
		key := fmt.Sprintf("%.1f/%.1f/%.2f", float64(c.X), float64(c.Y), circle.Radius)
		if seen[key] {
			continue
		}
		seen[key] = true
		ds.addRadialForEdge(viewName, dimType, e.ReferenceKey())
		added++
	}
	if added == 0 {
		return 0, fmt.Errorf("drawing: view %q has no circular edges to dimension", viewName)
	}
	return added, nil
}

// addRadialForEdge appends a radial dimension on a specific circular edge (already resolved).
func (ds *DrawingDimensions) addRadialForEdge(viewName string, dimType types.DrawingDimensionType, edgeKey []byte) {
	d := &DrawingDimension{name: ds.uniqueName(""), dimType: dimType, viewName: viewName, edgeKey: edgeKey}
	ds.recompute(d)
	ds.items = append(ds.items, d)
}

// isRadial reports whether a dimension type measures a circular edge (radius or diameter).
func isRadial(t types.DrawingDimensionType) bool {
	return t == types.RadiusDimension || t == types.DiameterDimension
}

// recomputeRadial re-binds the circular edge, re-measures its radius/diameter (the true model
// size) and rebuilds the leader glyph.
func (ds *DrawingDimensions) recomputeRadial(d *DrawingDimension, view *DrawingView, body *topo.Body, basis hlr.View) {
	edge, ok := body.FindEdgeByKey(d.edgeKey)
	if !ok {
		return
	}
	circle, ok := edge.Geometry().(geom.Circle)
	if !ok {
		return
	}
	radiusMM := circle.Radius * cmToMM
	if d.dimType == types.DiameterDimension {
		d.valueMM = 2 * radiusMM
		// Ø (U+00D8) is the diameter prefix: it is in the head's Latin-1 font, unlike the
		// typographic ⌀ (U+2300), which renders as a missing-glyph box.
		d.text = d.decorate("Ø" + formatDimValue(d.valueMM, ds.decimals()))
	} else {
		d.valueMM = radiusMM
		d.text = d.decorate("R" + formatDimValue(d.valueMM, ds.decimals()))
	}
	center := view.place(hlr.ProjectPoint(basis, circle.Center))
	arc := view.place(hlr.ProjectPoint(basis, circle.PointAt(0)))
	opp := view.place(hlr.ProjectPoint(basis, circle.PointAt(0.5)))
	curves, mx, my, nx, ny := radialDimensionCurves(center, arc, opp, d.dimType == types.DiameterDimension)
	d.curves = curves
	d.setTextAnchor(mx, my, nx, ny, 1)
}

// arcEdgeOf classifies an edge's geometry as a circular/arc curve and exposes what an arc-length
// dimension needs: the centre, the swept length (cm), whether it is a full circle, and a sampler
// that walks the curve as t runs 0→1. It reports ok=false for any non-circular edge.
func arcEdgeOf(curve geom.Curve3) (center gmath.Point3, lengthCM float64, full bool, sample func(float64) gmath.Point3, ok bool) {
	switch c := curve.(type) {
	case geom.Circle:
		return c.Center, c.Circumference(), true, c.PointAt, true
	case geom.Arc3d:
		return c.Center, c.Length(), false, c.PointAt, true
	default:
		return gmath.Point3{}, 0, false, nil, false
	}
}

// recomputeArcLength re-binds the circular/arc edge, re-measures its swept length (the true model
// size) and rebuilds the glyph — a dimension line following the projected arc.
func (ds *DrawingDimensions) recomputeArcLength(d *DrawingDimension, view *DrawingView, body *topo.Body, basis hlr.View) {
	edge, ok := body.FindEdgeByKey(d.edgeKey)
	if !ok {
		return
	}
	center3, lengthCM, full, sample, ok := arcEdgeOf(edge.Geometry())
	if !ok {
		return
	}
	d.valueMM = lengthCM * cmToMM
	d.text = d.decorate(formatDimValue(d.valueMM, ds.decimals()))
	const samples = 32
	pts := make([]gmath.Point2, 0, samples+1)
	for i := 0; i <= samples; i++ {
		pts = append(pts, view.place(hlr.ProjectPoint(basis, sample(float64(i)/samples))))
	}
	centerS := view.place(hlr.ProjectPoint(basis, center3))
	curves, mx, my, nx, ny := arcLengthDimensionCurves(centerS, pts, full)
	d.curves = curves
	d.setTextAnchor(mx, my, nx, ny, 1)
}

// arcLengthDimensionCurves builds an arc-length dimension's glyph (sheet mm): the projected arc
// offset radially outward from the centre into a dimension line that follows it, with an arrowhead
// at each end (a full circle has none). It returns the text anchor at the arc's middle, lifted
// radially outward.
func arcLengthDimensionCurves(center gmath.Point2, pts []gmath.Point2, full bool) (curves []DrawingCurve, anchorX, anchorY, nx, ny float64) {
	const gap = 6.0
	cx, cy := float64(center.X), float64(center.Y)
	off := make([][2]float64, len(pts))
	for i, p := range pts {
		px, py := float64(p.X), float64(p.Y)
		dx, dy := px-cx, py-cy
		if l := math.Hypot(dx, dy); l > 1e-9 {
			off[i] = [2]float64{px + dx/l*gap, py + dy/l*gap}
		} else {
			off[i] = [2]float64{px, py}
		}
	}
	for i := 1; i < len(off); i++ {
		curves = append(curves, dimSegment(off[i-1][0], off[i-1][1], off[i][0], off[i][1]))
	}
	if n := len(off); !full && n >= 2 {
		curves = append(curves, arrowheadCurves(off[0][0], off[0][1], off[1][0], off[1][1])...)
		curves = append(curves, arrowheadCurves(off[n-1][0], off[n-1][1], off[n-2][0], off[n-2][1])...)
	}
	mid := off[len(off)/2]
	return curves, mid[0], mid[1], mid[0] - cx, mid[1] - cy
}

// nearestArcEdgeKey returns the reference key of the circular/arc model edge whose centre projects
// nearest the pick (sheet mm), and false when the body has no such edges.
func nearestArcEdgeKey(body *topo.Body, v *DrawingView, basis hlr.View, x, y float64) ([]byte, bool) {
	var bestKey []byte
	bestD := -1.0
	for _, e := range body.Edges() {
		center, _, _, _, ok := arcEdgeOf(e.Geometry())
		if !ok {
			continue
		}
		s := v.place(hlr.ProjectPoint(basis, center))
		dx, dy := float64(s.X)-x, float64(s.Y)-y
		if d := dx*dx + dy*dy; bestD < 0 || d < bestD {
			bestD, bestKey = d, e.ReferenceKey()
		}
	}
	return bestKey, bestD >= 0
}

// nearestCircularEdgeKey returns the reference key of the circular model edge whose centre projects
// nearest the pick (sheet mm), and false when the body has no circular edges.
func nearestCircularEdgeKey(body *topo.Body, v *DrawingView, basis hlr.View, x, y float64) ([]byte, bool) {
	var bestKey []byte
	bestD := -1.0
	for _, e := range body.Edges() {
		circle, ok := e.Geometry().(geom.Circle)
		if !ok {
			continue
		}
		s := v.place(hlr.ProjectPoint(basis, circle.Center))
		dx, dy := float64(s.X)-x, float64(s.Y)-y
		if d := dx*dx + dy*dy; bestD < 0 || d < bestD {
			bestD, bestKey = d, e.ReferenceKey()
		}
	}
	return bestKey, bestD >= 0
}

// radialDimensionCurves builds a radial dimension's glyph (sheet mm): a radius leader from the
// centre to the arc with an arrowhead, or a diameter line across the circle with arrowheads at
// both ends. It returns the text anchor (the midpoint of the leader / the centre).
func radialDimensionCurves(center, arc, opp gmath.Point2, diameter bool) (curves []DrawingCurve, anchorX, anchorY, nx, ny float64) {
	cx, cy := float64(center.X), float64(center.Y)
	ax, ay := float64(arc.X), float64(arc.Y)
	// Lift the text perpendicular to the leader/diameter line so it clears it.
	pnx, pny := -(ay - cy), ax-cx
	if diameter {
		ox, oy := float64(opp.X), float64(opp.Y)
		out := []DrawingCurve{dimSegment(ax, ay, ox, oy)}
		out = append(out, arrowheadCurves(ax, ay, cx, cy)...)
		out = append(out, arrowheadCurves(ox, oy, cx, cy)...)
		return out, cx, cy, pnx, pny
	}
	out := []DrawingCurve{dimSegment(cx, cy, ax, ay)}
	out = append(out, arrowheadCurves(ax, ay, cx, cy)...)
	return out, (cx + ax) / 2, (cy + ay) / 2, pnx, pny
}
