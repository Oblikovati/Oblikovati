// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
)

// Drawing dimensions — the ANGULAR family (M48 #2225 split of dimension.go). An angular dimension
// measures the angle between two straight model edges (by reference key) projected onto a base view,
// and draws an arc glyph between them centred on their intersection. Both edges re-resolve on
// recompute (associativity). This file also carries the straight-edge projection and the arc glyph.

// AddAngular adds an angular dimension between the two straight model edges nearest the pick
// points (sheet mm) on the named base view; the measured angle re-derives when the model changes.
func (ds *DrawingDimensions) AddAngular(name, viewName string, x1, y1, x2, y2 float64) (*DrawingDimension, error) {
	view, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	keyA, okA := nearestStraightEdgeKey(body, view, basis, x1, y1)
	keyB, okB := nearestStraightEdgeKey(body, view, basis, x2, y2)
	if !okA || !okB || string(keyA) == string(keyB) {
		return nil, fmt.Errorf("drawing: view %q needs two distinct straight edges for an angular dimension", viewName)
	}
	d := &DrawingDimension{name: ds.uniqueName(name), dimType: types.AngularDimension, viewName: viewName, edgeKey: keyA, edgeKeyB: keyB}
	ds.recompute(d)
	ds.items = append(ds.items, d)
	return d, nil
}

// AddAngularForFirstCorner dimensions the angle between the first two non-parallel straight edges
// in the named base view — a single-action corner/bevel-angle callout.
func (ds *DrawingDimensions) AddAngularForFirstCorner(viewName string) (*DrawingDimension, error) {
	_, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	view, _ := ds.views.ByName(viewName)
	type seg struct {
		key []byte
		dir [2]float64
	}
	var segs []seg
	for _, e := range body.Edges() {
		if s, ok := straightEdgeOnSheet(body, view, basis, e.ReferenceKey()); ok {
			segs = append(segs, seg{e.ReferenceKey(), s.dir})
		}
	}
	for i := 0; i < len(segs); i++ {
		for j := i + 1; j < len(segs); j++ {
			if a := angleBetweenDeg(segs[i].dir, segs[j].dir); a > 1 && a < 179 {
				return ds.addAngularEdges(viewName, segs[i].key, segs[j].key), nil
			}
		}
	}
	return nil, fmt.Errorf("drawing: view %q has no two non-parallel straight edges to dimension", viewName)
}

// addAngularEdges appends an angular dimension between two already-resolved straight edges.
func (ds *DrawingDimensions) addAngularEdges(viewName string, keyA, keyB []byte) *DrawingDimension {
	d := &DrawingDimension{name: ds.uniqueName(""), dimType: types.AngularDimension, viewName: viewName, edgeKey: keyA, edgeKeyB: keyB}
	ds.recompute(d)
	ds.items = append(ds.items, d)
	return d
}

// recomputeAngular re-binds the two straight edges, projects them and measures the angle between
// them (degrees), then builds the arc glyph.
func (ds *DrawingDimensions) recomputeAngular(d *DrawingDimension, view *DrawingView, body *topo.Body, basis hlr.View) {
	a, okA := straightEdgeOnSheet(body, view, basis, d.edgeKey)
	b, okB := straightEdgeOnSheet(body, view, basis, d.edgeKeyB)
	if !okA || !okB {
		return
	}
	d.valueDeg = angleBetweenDeg(a.dir, b.dir)
	d.text = d.decorate(formatDimValue(d.valueDeg, ds.decimals()) + "°")
	curves, mx, my, nx, ny := angularDimensionCurves(a, b)
	d.curves = curves
	d.setTextAnchor(mx, my, nx, ny, 1)
}

// sheetSegment is a straight model edge projected onto the sheet: its endpoints, midpoint and unit
// direction (sheet mm).
type sheetSegment struct {
	ax, ay, bx, by float64
	mx, my         float64
	dir            [2]float64
}

// straightEdgeOnSheet resolves a straight edge by key and projects it onto the view's plane.
func straightEdgeOnSheet(body *topo.Body, v *DrawingView, basis hlr.View, key []byte) (sheetSegment, bool) {
	edge, ok := body.FindEdgeByKey(key)
	if !ok {
		return sheetSegment{}, false
	}
	line, ok := edge.Geometry().(geom.LineSegment)
	if !ok {
		return sheetSegment{}, false
	}
	a := v.place(hlr.ProjectPoint(basis, line.StartPoint))
	b := v.place(hlr.ProjectPoint(basis, line.EndPoint))
	ax, ay, bx, by := float64(a.X), float64(a.Y), float64(b.X), float64(b.Y)
	dx, dy := bx-ax, by-ay
	l := math.Hypot(dx, dy)
	if l < 1e-9 {
		return sheetSegment{}, false
	}
	return sheetSegment{ax, ay, bx, by, (ax + bx) / 2, (ay + by) / 2, [2]float64{dx / l, dy / l}}, true
}

// nearestStraightEdgeKey returns the reference key of the straight model edge whose midpoint
// projects nearest the pick (sheet mm), and false when the body has no straight edges.
func nearestStraightEdgeKey(body *topo.Body, v *DrawingView, basis hlr.View, x, y float64) ([]byte, bool) {
	var bestKey []byte
	bestD := -1.0
	for _, e := range body.Edges() {
		line, ok := e.Geometry().(geom.LineSegment)
		if !ok {
			continue
		}
		mid := line.StartPoint.Midpoint(line.EndPoint)
		s := v.place(hlr.ProjectPoint(basis, mid))
		dx, dy := float64(s.X)-x, float64(s.Y)-y
		if d := dx*dx + dy*dy; bestD < 0 || d < bestD {
			bestD, bestKey = d, e.ReferenceKey()
		}
	}
	return bestKey, bestD >= 0
}

// angleBetweenDeg is the angle between two unit directions, in degrees ∈ [0, 180].
func angleBetweenDeg(d1, d2 [2]float64) float64 {
	dot := math.Max(-1, math.Min(1, d1[0]*d2[0]+d1[1]*d2[1]))
	return math.Acos(dot) * 180 / math.Pi
}

// angularDimensionCurves builds an angular dimension's arc glyph (sheet mm) spanning between the
// two edges, centred on their intersection, with arrowheads at the arc ends. It returns the text
// anchor on the arc's bisector.
func angularDimensionCurves(a, b sheetSegment) (curves []DrawingCurve, anchorX, anchorY, nx, ny float64) {
	cx, cy, ok := lineIntersection(a, b)
	if !ok {
		cx, cy = (a.mx+b.mx)/2, (a.my+b.my)/2
	}
	const radius = 22.0
	a0 := math.Atan2(a.my-cy, a.mx-cx)
	sweep := normalizeAngle(math.Atan2(b.my-cy, b.mx-cx) - a0)
	out := arcPolyline(cx, cy, radius, a0, a0+sweep)
	out = append(out, arcArrowheads(cx, cy, radius, a0, a0+sweep)...)
	mid := a0 + sweep/2
	// The text sits on the bisector, lifted radially outward (away from the corner).
	return out, cx + radius*math.Cos(mid), cy + radius*math.Sin(mid), math.Cos(mid), math.Sin(mid)
}

// lineIntersection returns the intersection of the two segments' infinite lines, ok=false when
// they are parallel.
func lineIntersection(a, b sheetSegment) (x, y float64, ok bool) {
	denom := a.dir[0]*b.dir[1] - a.dir[1]*b.dir[0]
	if math.Abs(denom) < 1e-9 {
		return 0, 0, false
	}
	dx, dy := b.mx-a.mx, b.my-a.my
	t := (dx*b.dir[1] - dy*b.dir[0]) / denom
	return a.mx + t*a.dir[0], a.my + t*a.dir[1], true
}

// arcPolyline samples a circular arc (centre, radius, from angle a0 to a1) into drawing curves.
func arcPolyline(cx, cy, r, a0, a1 float64) []DrawingCurve {
	const n = 24
	out := make([]DrawingCurve, 0, n)
	px, py := cx+r*math.Cos(a0), cy+r*math.Sin(a0)
	for i := 1; i <= n; i++ {
		t := a0 + (a1-a0)*float64(i)/n
		x, y := cx+r*math.Cos(t), cy+r*math.Sin(t)
		out = append(out, dimSegment(px, py, x, y))
		px, py = x, y
	}
	return out
}

// arcArrowheads builds an arrowhead at each end of the arc, pointing tangentially into the sweep.
func arcArrowheads(cx, cy, r, a0, a1 float64) []DrawingCurve {
	const eps = 0.12 // radians inward, to set the barb direction
	tip0x, tip0y := cx+r*math.Cos(a0), cy+r*math.Sin(a0)
	from0x, from0y := cx+r*math.Cos(a0+sign(a1-a0)*eps), cy+r*math.Sin(a0+sign(a1-a0)*eps)
	tip1x, tip1y := cx+r*math.Cos(a1), cy+r*math.Sin(a1)
	from1x, from1y := cx+r*math.Cos(a1-sign(a1-a0)*eps), cy+r*math.Sin(a1-sign(a1-a0)*eps)
	out := arrowheadCurves(tip0x, tip0y, from0x, from0y)
	return append(out, arrowheadCurves(tip1x, tip1y, from1x, from1y)...)
}

// normalizeAngle wraps an angle into (-π, π], so an arc sweeps the short way between two edges.
func normalizeAngle(d float64) float64 {
	for d > math.Pi {
		d -= 2 * math.Pi
	}
	for d <= -math.Pi {
		d += 2 * math.Pi
	}
	return d
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}
