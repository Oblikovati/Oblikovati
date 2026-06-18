// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"
	"strconv"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// Drawing dimensions (M14-F03 PBI-141, #388): a linear dimension annotates the distance between
// two points on a drawing view. It attaches to two model vertices (by reference key), so it is
// associative — on recompute it re-resolves the vertices against the current model, re-projects
// them through the view and re-measures, and the value updates when the model size changes.

// DrawingDimension is one linear dimension on a sheet: its measured value (true model size), the
// view it annotates, the two attached model vertices, and the drawing curves of its glyph
// (extension lines, dimension line, arrowheads).
type DrawingDimension struct {
	name     string
	dimType  types.DrawingDimensionType
	viewName string
	keyA     []byte // linear: attached model vertices (reference keys) — the associativity anchors
	keyB     []byte
	edgeKey  []byte  // radial: the attached circular edge; angular: the first straight edge
	edgeKeyB []byte  // angular: the second straight edge
	offset   float64 // dimension-line standoff from the measured points (signed, sheet mm)
	valueMM  float64 // measured model distance (mm), scale-independent (0 for angular)
	valueDeg float64 // measured angle (degrees) for an angular dimension
	text     string  // displayed text (the formatted value)
	anchorX  float64 // text anchor (sheet mm), lifted off the dimension line by textGapMM
	anchorY  float64
	textDX   float64 // user text nudge from the anchor (sheet mm) — drag the text to set it
	textDY   float64
	nx       float64 // unit perpendicular of the dimension line — text-lift + line-drag direction
	ny       float64
	curves   []DrawingCurve
}

// textGapMM lifts the value text off the dimension line so it stays readable by default.
const textGapMM = 5.0

var _ contract.DrawingDimension = (*DrawingDimension)(nil)

// Name, Type, ViewName, ValueMM, Text, CurveCount and Curves expose the dimension.
func (d *DrawingDimension) Name() string                     { return d.name }
func (d *DrawingDimension) Type() types.DrawingDimensionType { return d.dimType }
func (d *DrawingDimension) ViewName() string                 { return d.viewName }
func (d *DrawingDimension) ValueMM() float64                 { return d.valueMM }
func (d *DrawingDimension) ValueDeg() float64                { return d.valueDeg }
func (d *DrawingDimension) Text() string                     { return d.text }
func (d *DrawingDimension) CurveCount() int                  { return len(d.curves) }
func (d *DrawingDimension) Curves() []DrawingCurve           { return d.curves }

// TextAnchorMM is the dimension line's midpoint (sheet mm) — where the value text is centred.
func (d *DrawingDimension) TextAnchorMM() (x, y float64) {
	return d.anchorX + d.textDX, d.anchorY + d.textDY
}

// DrawingDimensions is a sheet's dimension collection. It holds the view collection (to resolve a
// dimension's view and project through it) and the body-resolution hook (to re-bind the attached
// vertices on recompute).
type DrawingDimensions struct {
	items []*DrawingDimension
	views *DrawingViews
	body  bodyLookup
}

func newDrawingDimensions(views *DrawingViews, body bodyLookup) *DrawingDimensions {
	return &DrawingDimensions{views: views, body: body}
}

// AddLinear adds a linear dimension on the named base view between two pick points (sheet mm),
// each snapped to the nearest projected model vertex so the value tracks the model. dimType
// selects the measured component; offset stands the dimension line off the points (sheet mm).
func (ds *DrawingDimensions) AddLinear(name, viewName string, dimType types.DrawingDimensionType, x1, y1, x2, y2, offset float64) (*DrawingDimension, error) {
	view, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	keyA, okA := nearestVertexKey(body, view, basis, x1, y1)
	keyB, okB := nearestVertexKey(body, view, basis, x2, y2)
	if !okA || !okB {
		return nil, fmt.Errorf("drawing: view %q has no model vertices to dimension", viewName)
	}
	return ds.addLinearFromKeys(name, viewName, dimType, keyA, keyB, offset), nil
}

// addLinearFromKeys creates a linear dimension between two already-resolved vertex keys.
func (ds *DrawingDimensions) addLinearFromKeys(name, viewName string, dimType types.DrawingDimensionType, keyA, keyB []byte, offset float64) *DrawingDimension {
	d := &DrawingDimension{name: ds.uniqueName(name), dimType: dimType, viewName: viewName, keyA: keyA, keyB: keyB, offset: offset}
	ds.recompute(d)
	ds.items = append(ds.items, d)
	return d
}

// AddBaselineSet adds a baseline set: a linear dimension from the first pick point to each of the
// others, stacked at increasing offsets (the datum-referenced common case). Each point is snapped
// to the nearest projected model vertex, so every dimension stays associative.
func (ds *DrawingDimensions) AddBaselineSet(viewName string, dimType types.DrawingDimensionType, points [][2]float64) ([]*DrawingDimension, error) {
	keys, err := ds.snapPoints(viewName, points)
	if err != nil {
		return nil, err
	}
	const baseGap = 14.0
	out := make([]*DrawingDimension, 0, len(keys)-1)
	for i := 1; i < len(keys); i++ {
		out = append(out, ds.addLinearFromKeys("", viewName, dimType, keys[0], keys[i], -baseGap*float64(i)))
	}
	return out, nil
}

// AddChainSet adds a chain set: a linear dimension between each consecutive pair of pick points,
// all on one line (running dimensions). Each point is snapped to the nearest model vertex.
func (ds *DrawingDimensions) AddChainSet(viewName string, dimType types.DrawingDimensionType, points [][2]float64) ([]*DrawingDimension, error) {
	keys, err := ds.snapPoints(viewName, points)
	if err != nil {
		return nil, err
	}
	const chainOffset = -14.0
	out := make([]*DrawingDimension, 0, len(keys)-1)
	for i := 0; i+1 < len(keys); i++ {
		out = append(out, ds.addLinearFromKeys("", viewName, dimType, keys[i], keys[i+1], chainOffset))
	}
	return out, nil
}

// AddSetForViewCorners dimensions a base view's bounding-box corners as a baseline or chain set —
// the single-action dimension set (a full multi-pick set is a follow-up). Aligned measurement
// gives non-degenerate values (width / diagonal / height).
func (ds *DrawingDimensions) AddSetForViewCorners(viewName string, dimType types.DrawingDimensionType, baseline bool) ([]*DrawingDimension, error) {
	view, ok := ds.views.ByName(viewName)
	if !ok {
		return nil, fmt.Errorf("drawing: no view %q to dimension", viewName)
	}
	minX, minY, maxX, maxY, ok := view.BoundsMM()
	if !ok {
		return nil, fmt.Errorf("drawing: view %q has no geometry to dimension", viewName)
	}
	pts := [][2]float64{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
	if baseline {
		return ds.AddBaselineSet(viewName, dimType, pts)
	}
	return ds.AddChainSet(viewName, dimType, pts)
}

// snapPoints resolves each pick point to a model-vertex key on the view; it needs at least two
// points and a resolvable model.
func (ds *DrawingDimensions) snapPoints(viewName string, points [][2]float64) ([][]byte, error) {
	if len(points) < 2 {
		return nil, fmt.Errorf("drawing: a dimension set needs at least two points, got %d", len(points))
	}
	view, body, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	keys := make([][]byte, len(points))
	for i, p := range points {
		k, ok := nearestVertexKey(body, view, basis, p[0], p[1])
		if !ok {
			return nil, fmt.Errorf("drawing: view %q has no model vertices to dimension", viewName)
		}
		keys[i] = k
	}
	return keys, nil
}

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

// MoveText nudges the named dimension's value text by (dx, dy) sheet millimetres — the drag-the-
// text gesture, so overlapping dimensions can be separated for readability.
func (ds *DrawingDimensions) MoveText(name string, dx, dy float64) {
	if d, ok := ds.ByName(name); ok {
		d.textDX += dx
		d.textDY += dy
	}
}

// MoveLine shifts the named dimension's dimension line perpendicular to itself by the drag
// (dx, dy) sheet millimetres and re-derives the glyph — the drag-the-line gesture. Linear only in
// this increment (radial/angular line position is fixed; nudge their text instead).
func (ds *DrawingDimensions) MoveLine(name string, dx, dy float64) {
	d, ok := ds.ByName(name)
	if !ok || isRadial(d.dimType) || d.dimType == types.AngularDimension {
		return
	}
	d.offset += dx*d.nx + dy*d.ny
	ds.recompute(d)
}

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
	if isRadial(d.dimType) {
		ds.recomputeRadial(d, view, body, basis)
		return
	}
	if d.dimType == types.AngularDimension {
		ds.recomputeAngular(d, view, body, basis)
		return
	}
	ds.recomputeLinear(d, view, body, basis)
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
	d.text = strconv.FormatFloat(d.valueDeg, 'g', 4, 64) + "°"
	curves, mx, my, nx, ny := angularDimensionCurves(a, b)
	d.curves = curves
	d.setTextAnchor(mx, my, nx, ny, 1)
}

// recomputeLinear re-binds the two vertices, re-projects them and rebuilds the linear glyph.
func (ds *DrawingDimensions) recomputeLinear(d *DrawingDimension, view *DrawingView, body *topo.Body, basis hlr.View) {
	va, okA := body.FindVertexByKey(d.keyA)
	vb, okB := body.FindVertexByKey(d.keyB)
	if !okA || !okB {
		return
	}
	p1 := hlr.ProjectPoint(basis, va.Point())
	p2 := hlr.ProjectPoint(basis, vb.Point())
	d.valueMM = measureMM(d.dimType, p1, p2)
	d.text = strconv.FormatFloat(d.valueMM, 'g', 4, 64)
	s1, s2 := view.place(p1), view.place(p2)
	ax, ay := dimensionAxis(d.dimType, s1, s2)
	var mx, my float64
	d.curves, mx, my = linearDimensionCurves(s1, s2, ax, ay, d.offset)
	// Lift the text off the dimension line, on the side away from the measured geometry.
	d.setTextAnchor(mx, my, -ay, ax, sign(d.offset))
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
		d.text = "Ø" + strconv.FormatFloat(d.valueMM, 'g', 4, 64)
	} else {
		d.valueMM = radiusMM
		d.text = "R" + strconv.FormatFloat(d.valueMM, 'g', 4, 64)
	}
	center := view.place(hlr.ProjectPoint(basis, circle.Center))
	arc := view.place(hlr.ProjectPoint(basis, circle.PointAt(0)))
	opp := view.place(hlr.ProjectPoint(basis, circle.PointAt(0.5)))
	curves, mx, my, nx, ny := radialDimensionCurves(center, arc, opp, d.dimType == types.DiameterDimension)
	d.curves = curves
	d.setTextAnchor(mx, my, nx, ny, 1)
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

// Count, Item, ByName and Remove read/edit the collection.
func (ds *DrawingDimensions) Count() int { return len(ds.items) }

func (ds *DrawingDimensions) Item(i int) *DrawingDimension {
	if i < 0 || i >= len(ds.items) {
		return nil
	}
	return ds.items[i]
}

func (ds *DrawingDimensions) ByName(name string) (*DrawingDimension, bool) {
	for _, d := range ds.items {
		if d.name == name {
			return d, true
		}
	}
	return nil, false
}

func (ds *DrawingDimensions) Remove(name string) error {
	for i, d := range ds.items {
		if d.name == name {
			ds.items = append(ds.items[:i], ds.items[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("drawing: no dimension named %q", name)
}

func (ds *DrawingDimensions) uniqueName(requested string) string {
	if requested != "" {
		if _, exists := ds.ByName(requested); !exists {
			return requested
		}
	}
	for n := len(ds.items) + 1; ; n++ {
		name := fmt.Sprintf("DIM:%d", n)
		if _, exists := ds.ByName(name); !exists {
			return name
		}
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

// linearDimensionCurves builds a linear dimension's glyph (sheet mm): an extension line from each
// measured point out to the dimension line, the dimension line itself, and an arrowhead at each
// end. The dimension line is parallel to (ax, ay) and offset along its perpendicular by offset.
// It also returns the dimension line's midpoint (the text anchor).
func linearDimensionCurves(s1, s2 gmath.Point2, ax, ay, offset float64) (curves []DrawingCurve, anchorX, anchorY float64) {
	nx, ny := -ay, ax // perpendicular to the measurement axis
	s1x, s1y := float64(s1.X), float64(s1.Y)
	s2x, s2y := float64(s2.X), float64(s2.Y)
	lvl := s1x*nx + s1y*ny + offset // n-coordinate of the dimension line
	e1x, e1y := s1x+offset*nx, s1y+offset*ny
	d2 := lvl - (s2x*nx + s2y*ny)
	e2x, e2y := s2x+d2*nx, s2y+d2*ny
	out := []DrawingCurve{
		dimSegment(s1x, s1y, e1x, e1y),
		dimSegment(s2x, s2y, e2x, e2y),
		dimSegment(e1x, e1y, e2x, e2y),
	}
	out = append(out, arrowheadCurves(e1x, e1y, e2x, e2y)...)
	out = append(out, arrowheadCurves(e2x, e2y, e1x, e1y)...)
	return out, (e1x + e2x) / 2, (e1y + e2y) / 2
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
