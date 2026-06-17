// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"
	"strconv"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
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
	keyA     []byte // attached model vertices (reference keys) — the associativity anchors
	keyB     []byte
	offset   float64 // dimension-line standoff from the measured points (signed, sheet mm)
	valueMM  float64 // measured model distance (mm), scale-independent
	text     string  // displayed text (the formatted value)
	anchorX  float64 // dimension-line midpoint (sheet mm) — where the head draws the text
	anchorY  float64
	curves   []DrawingCurve
}

var _ contract.DrawingDimension = (*DrawingDimension)(nil)

// Name, Type, ViewName, ValueMM, Text, CurveCount and Curves expose the dimension.
func (d *DrawingDimension) Name() string                     { return d.name }
func (d *DrawingDimension) Type() types.DrawingDimensionType { return d.dimType }
func (d *DrawingDimension) ViewName() string                 { return d.viewName }
func (d *DrawingDimension) ValueMM() float64                 { return d.valueMM }
func (d *DrawingDimension) Text() string                     { return d.text }
func (d *DrawingDimension) CurveCount() int                  { return len(d.curves) }
func (d *DrawingDimension) Curves() []DrawingCurve           { return d.curves }

// TextAnchorMM is the dimension line's midpoint (sheet mm) — where the value text is centred.
func (d *DrawingDimension) TextAnchorMM() (x, y float64) { return d.anchorX, d.anchorY }

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
	d := &DrawingDimension{name: ds.uniqueName(name), dimType: dimType, viewName: viewName, keyA: keyA, keyB: keyB, offset: offset}
	ds.recompute(d)
	ds.items = append(ds.items, d)
	return d, nil
}

// Recompute re-measures every dimension against the current model — the associativity path.
func (ds *DrawingDimensions) Recompute() {
	for _, d := range ds.items {
		ds.recompute(d)
	}
}

// recompute re-binds the dimension's two vertices, re-projects them through its view, re-measures
// and rebuilds the glyph. With no resolvable view, model or vertices it clears the dimension.
func (ds *DrawingDimensions) recompute(d *DrawingDimension) {
	d.curves = nil
	view, body, basis, err := ds.dimensionBasis(d.viewName)
	if err != nil {
		return
	}
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
	d.curves, d.anchorX, d.anchorY = linearDimensionCurves(s1, s2, ax, ay, d.offset)
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
