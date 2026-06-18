// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// Drawing annotations (M14-F02 #813): sheet markup that is not a view. A centre-of-gravity
// marker sits on a view at the referenced model's centre of mass (associative — it re-projects
// when the model changes); a revision cloud is a scalloped boundary over a sheet region. Both
// produce drawing curves (sheet millimetres) the head renders alongside the views.

// DrawingAnnotation is one annotation on a sheet: a CoG marker (on a view) or a revision cloud
// (over a region).
type DrawingAnnotation struct {
	name     string
	kind     types.DrawingAnnotationKind
	viewName string  // CoG / centre mark: the view it marks
	x, y     float64 // revision cloud: lower-left corner (sheet mm)
	w, h     float64 // revision cloud: size (sheet mm)
	tag      string  // revision cloud: revision label
	edgeKey  []byte  // centre mark: the circular edge it marks (associativity anchor)
	// feature control frame (GD&T): the geometric tolerance it states.
	characteristic types.GeometricCharacteristic
	tolerance      string
	datums         []string
	// surface texture: the material-removal variant (the roughness value reuses tag).
	materialRemoval types.MaterialRemoval
	rowCount        int           // parts list / hole table / custom table: the number of data rows
	revisions       []RevisionRow // revision table: the user-supplied change-history rows
	headers         []string      // custom table: the column headers
	tableRows       [][]string    // custom table: the data rows (cells aligned to headers)
	labels          []AnnotationLabel
	curves          []DrawingCurve
}

// RevisionRow is one row of a revision table: a revision identifier, its date, and a description
// of the change. The rows are user-supplied drawing history (not model-derived), so they persist.
type RevisionRow struct {
	Revision    string
	Date        string
	Description string
}

// AnnotationLabel is one piece of text an annotation renders (sheet millimetres) — e.g. a feature
// control frame's tolerance value or datum letter, centred in its compartment.
type AnnotationLabel struct {
	Text string
	X, Y float64
}

// Name, Kind, ViewName, Tag, Curves and Labels expose the annotation.
func (a *DrawingAnnotation) Name() string                      { return a.name }
func (a *DrawingAnnotation) Kind() types.DrawingAnnotationKind { return a.kind }
func (a *DrawingAnnotation) ViewName() string                  { return a.viewName }
func (a *DrawingAnnotation) Tag() string                       { return a.tag }
func (a *DrawingAnnotation) Curves() []DrawingCurve            { return a.curves }
func (a *DrawingAnnotation) CurveCount() int                   { return len(a.curves) }
func (a *DrawingAnnotation) Labels() []AnnotationLabel         { return a.labels }
func (a *DrawingAnnotation) RowCount() int                     { return a.rowCount }

// DrawingAnnotations is a sheet's annotation collection. It holds the body-resolution hook and
// the sheet's views so a CoG marker can find its view and the model's centre of mass.
type DrawingAnnotations struct {
	items []*DrawingAnnotation
	views *DrawingViews
	body  bodyLookup
	bom   bomLookup // parts-list BOM source
}

func newDrawingAnnotations(views *DrawingViews, body bodyLookup, bom bomLookup) *DrawingAnnotations {
	return &DrawingAnnotations{views: views, body: body, bom: bom}
}

// AddCoGMarker adds a centre-of-gravity marker on the named view, positioned at the referenced
// model's centre of mass. It errors if the view or model is missing.
func (as *DrawingAnnotations) AddCoGMarker(name, viewName string) (*DrawingAnnotation, error) {
	if _, ok := as.views.ByName(viewName); !ok {
		return nil, fmt.Errorf("drawing: no view %q for a centre-of-gravity marker", viewName)
	}
	if as.body == nil {
		return nil, fmt.Errorf("drawing: no referenced model for a centre-of-gravity marker")
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.CoGMarkerAnnotation, viewName: viewName}
	as.recomputeCoG(a)
	as.items = append(as.items, a)
	return a, nil
}

// AddRevisionCloud adds a scalloped revision cloud over the sheet rectangle (x, y, w, h mm).
func (as *DrawingAnnotations) AddRevisionCloud(name string, x, y, w, h float64, tag string) (*DrawingAnnotation, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("drawing: revision cloud needs a positive size, got %g×%g", w, h)
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.RevisionCloudAnnotation, x: x, y: y, w: w, h: h, tag: tag}
	a.curves = revisionCloudCurves(x, y, w, h)
	as.items = append(as.items, a)
	return a, nil
}

// AddCenterMarks adds a centre mark (crosshair) at the centre of every distinct circular model
// edge in the named base view — the auto centre-mark-all-holes action. Coincident projections (a
// through-hole's two rims) are marked once. Each mark attaches to its edge, so it re-projects when
// the model changes.
func (as *DrawingAnnotations) AddCenterMarks(viewName string) ([]*DrawingAnnotation, error) {
	view, body, basis, err := as.annotationBasis(viewName)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []*DrawingAnnotation
	for _, e := range body.Edges() {
		circle, ok := e.Geometry().(geom.Circle)
		if !ok {
			continue
		}
		c := view.place(hlr.ProjectPoint(basis, circle.Center))
		key := fmt.Sprintf("%.1f/%.1f", float64(c.X), float64(c.Y))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, as.addCenterMarkForEdge(viewName, e.ReferenceKey()))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("drawing: view %q has no circular edges to centre-mark", viewName)
	}
	return out, nil
}

// addCenterMarkForEdge appends a centre mark on a specific circular edge (already resolved).
func (as *DrawingAnnotations) addCenterMarkForEdge(viewName string, edgeKey []byte) *DrawingAnnotation {
	a := &DrawingAnnotation{name: as.uniqueName(""), kind: types.CenterMarkAnnotation, viewName: viewName, edgeKey: edgeKey}
	as.recomputeCenterMark(a)
	as.items = append(as.items, a)
	return a
}

// AddCenterlines adds the horizontal+vertical dash-dot symmetry centerlines through the named
// view's centre, spanning its extent. The lines re-derive from the view's bounds, so they track
// the model.
func (as *DrawingAnnotations) AddCenterlines(name, viewName string) (*DrawingAnnotation, error) {
	view, ok := as.views.ByName(viewName)
	if !ok {
		return nil, fmt.Errorf("drawing: no view %q for centerlines", viewName)
	}
	if _, _, _, _, ok := view.BoundsMM(); !ok {
		return nil, fmt.Errorf("drawing: view %q has no geometry for centerlines", viewName)
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.CenterlineAnnotation, viewName: viewName}
	as.recomputeCenterline(a)
	as.items = append(as.items, a)
	return a, nil
}

// Recompute re-derives the associative annotations (CoG markers, centre marks and centerlines)
// against the current model.
func (as *DrawingAnnotations) Recompute() {
	for _, a := range as.items {
		switch a.kind {
		case types.CoGMarkerAnnotation:
			as.recomputeCoG(a)
		case types.CenterMarkAnnotation:
			as.recomputeCenterMark(a)
		case types.CenterlineAnnotation:
			as.recomputeCenterline(a)
		case types.PartsListAnnotation:
			as.recomputePartsList(a)
		case types.HoleTableAnnotation:
			as.recomputeHoleTable(a)
		}
	}
}

// recomputeCenterline rebuilds the view's horizontal+vertical dash-dot centerlines from its current
// bounds (spanning the extent plus a small overshoot); with no view geometry it clears the glyph.
func (as *DrawingAnnotations) recomputeCenterline(a *DrawingAnnotation) {
	a.curves = nil
	view, ok := as.views.ByName(a.viewName)
	if !ok {
		return
	}
	minX, minY, maxX, maxY, ok := view.BoundsMM()
	if !ok {
		return
	}
	const overshoot = 4.0
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	a.curves = append(dashDotLine(minX-overshoot, cy, maxX+overshoot, cy), dashDotLine(cx, minY-overshoot, cx, maxY+overshoot)...)
}

// annotationBasis resolves a base view, the referenced model body and the view's projection frame
// — the inputs a centre mark needs to project a model edge onto the sheet.
func (as *DrawingAnnotations) annotationBasis(viewName string) (*DrawingView, *topo.Body, hlr.View, error) {
	view, ok := as.views.ByName(viewName)
	if !ok {
		return nil, nil, hlr.View{}, fmt.Errorf("drawing: no view %q to annotate", viewName)
	}
	if view.viewType != types.DrawingViewBase {
		return nil, nil, hlr.View{}, fmt.Errorf("drawing: %q is not a base view; centre-mark a base view", viewName)
	}
	if as.body == nil {
		return nil, nil, hlr.View{}, fmt.Errorf("drawing: no referenced model to annotate")
	}
	body, ok := as.body()
	if !ok {
		return nil, nil, hlr.View{}, fmt.Errorf("drawing: no referenced model to annotate")
	}
	return view, body, baseBasis(view.orientation, bodyCenter(body)), nil
}

// recomputeCenterMark re-binds the circular edge, projects its centre into the view and rebuilds the
// crosshair glyph (sized to the projected radius); with no resolvable edge it clears the mark.
func (as *DrawingAnnotations) recomputeCenterMark(a *DrawingAnnotation) {
	a.curves = nil
	view, body, basis, err := as.annotationBasis(a.viewName)
	if err != nil {
		return
	}
	edge, ok := body.FindEdgeByKey(a.edgeKey)
	if !ok {
		return
	}
	circle, ok := edge.Geometry().(geom.Circle)
	if !ok {
		return
	}
	c := view.place(hlr.ProjectPoint(basis, circle.Center))
	rim := view.place(hlr.ProjectPoint(basis, circle.PointAt(0)))
	r := math.Hypot(float64(rim.X-c.X), float64(rim.Y-c.Y))
	a.curves = centerMarkCurves(float64(c.X), float64(c.Y), r)
}

// recomputeCoG positions a marker at the referenced model's centre of mass projected into its
// view; with no resolvable model or view it clears the marker's curves.
func (as *DrawingAnnotations) recomputeCoG(a *DrawingAnnotation) {
	a.curves = nil
	view, ok := as.views.ByName(a.viewName)
	if !ok || as.body == nil {
		return
	}
	body, ok := as.body()
	if !ok {
		return
	}
	centre := view.SheetPointOfModelMM(ops.BodyGeometryProperties(body, ops.DefaultQuality()).Centroid, bodyCenter(body))
	a.curves = cogMarkerCurves(float64(centre.X), float64(centre.Y))
}

// Count, Item, ByName and Remove read/edit the collection.
func (as *DrawingAnnotations) Count() int { return len(as.items) }

func (as *DrawingAnnotations) Item(i int) *DrawingAnnotation {
	if i < 0 || i >= len(as.items) {
		return nil
	}
	return as.items[i]
}

func (as *DrawingAnnotations) ByName(name string) (*DrawingAnnotation, bool) {
	for _, a := range as.items {
		if a.name == name {
			return a, true
		}
	}
	return nil, false
}

func (as *DrawingAnnotations) Remove(name string) error {
	for i, a := range as.items {
		if a.name == name {
			as.items = append(as.items[:i], as.items[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("drawing: no annotation named %q", name)
}

func (as *DrawingAnnotations) uniqueName(requested string) string {
	if requested != "" {
		if _, exists := as.ByName(requested); !exists {
			return requested
		}
	}
	for n := len(as.items) + 1; ; n++ {
		name := fmt.Sprintf("NOTE:%d", n)
		if _, exists := as.ByName(name); !exists {
			return name
		}
	}
}

// cogMarkerCurves builds the centre-of-gravity glyph at (cx, cy): a small circle with the
// conventional crosshair, in sheet millimetres.
func cogMarkerCurves(cx, cy float64) []DrawingCurve {
	const r = 3.0
	out := circlePolyline(cx, cy, r)
	out = append(out,
		DrawingCurve{A: gmath.P2(gmath.Scalar(cx-r), gmath.Scalar(cy)), B: gmath.P2(gmath.Scalar(cx+r), gmath.Scalar(cy)), Visible: true},
		DrawingCurve{A: gmath.P2(gmath.Scalar(cx), gmath.Scalar(cy-r)), B: gmath.P2(gmath.Scalar(cx), gmath.Scalar(cy+r)), Visible: true},
	)
	return out
}

// centerMarkCurves builds a centre-mark crosshair at (cx, cy) sized to the projected hole radius r
// (sheet mm): a small solid cross through the centre plus four extension arms that reach just past
// the rim, with the conventional gap between the central cross and the arms.
func centerMarkCurves(cx, cy, r float64) []DrawingCurve {
	const overshoot, gap = 2.0, 1.5
	ext := r + overshoot
	seg := func(ax, ay, bx, by float64) DrawingCurve {
		return DrawingCurve{A: gmath.P2(gmath.Scalar(ax), gmath.Scalar(ay)), B: gmath.P2(gmath.Scalar(bx), gmath.Scalar(by)), Visible: true}
	}
	return []DrawingCurve{
		seg(cx-gap, cy, cx+gap, cy), seg(cx, cy-gap, cx, cy+gap), // central solid cross
		seg(cx-ext, cy, cx-gap, cy), seg(cx+gap, cy, cx+ext, cy), // horizontal extension arms
		seg(cx, cy-ext, cx, cy-gap), seg(cx, cy+gap, cx, cy+ext), // vertical extension arms
	}
}

// dashDotLine builds a centerline's dash-dot pattern (long dash · short dot · …, sheet mm) from
// (ax, ay) to (bx, by): it marches the repeating pattern along the line, emitting a drawing curve
// for each dash and dot and skipping the gaps.
func dashDotLine(ax, ay, bx, by float64) []DrawingCurve {
	// Pattern lengths (mm): long dash, gap, dot, gap — the ISO centerline rhythm.
	pattern := [4]float64{12, 2, 2, 2}
	const drawn = 0b0101 // bits 0 and 2 (dash and dot) are drawn; the gaps are not
	dx, dy := bx-ax, by-ay
	length := math.Hypot(dx, dy)
	if length < 1e-9 {
		return nil
	}
	ux, uy := dx/length, dy/length
	var out []DrawingCurve
	pos, k := 0.0, 0
	for pos < length {
		seg := math.Min(pattern[k%4], length-pos)
		if drawn&(1<<(k%4)) != 0 {
			s, e := pos, pos+seg
			out = append(out, DrawingCurve{
				A: gmath.P2(gmath.Scalar(ax+ux*s), gmath.Scalar(ay+uy*s)),
				B: gmath.P2(gmath.Scalar(ax+ux*e), gmath.Scalar(ay+uy*e)), Visible: true,
			})
		}
		pos += seg
		k++
	}
	return out
}

// circlePolyline tessellates a circle (cx, cy, r) into visible drawing curves.
func circlePolyline(cx, cy, r float64) []DrawingCurve {
	const n = 24
	pts := make([]gmath.Point2, n+1)
	for i := 0; i <= n; i++ {
		a := 2 * math.Pi * float64(i) / float64(n)
		pts[i] = gmath.P2(gmath.Scalar(cx+r*math.Cos(a)), gmath.Scalar(cy+r*math.Sin(a)))
	}
	out := make([]DrawingCurve, 0, n)
	for i := 0; i+1 < len(pts); i++ {
		out = append(out, DrawingCurve{A: pts[i], B: pts[i+1], Visible: true})
	}
	return out
}

// revisionCloudCurves builds a scalloped cloud boundary over the rectangle (x, y, w, h): outward
// semicircular bumps marched around the perimeter.
func revisionCloudCurves(x, y, w, h float64) []DrawingCurve {
	var out []DrawingCurve
	out = append(out, scallopEdge(x, y, x+w, y)...)     // bottom
	out = append(out, scallopEdge(x+w, y, x+w, y+h)...) // right
	out = append(out, scallopEdge(x+w, y+h, x, y+h)...) // top
	out = append(out, scallopEdge(x, y+h, x, y)...)     // left
	return out
}

// scallopBumpMM is the nominal scallop (cloud bump) diameter.
const scallopBumpMM = 6.0

// scallopEdge marches semicircular bumps (bulging to the edge's left, i.e. outward for a CCW
// rectangle) from a to b, returning their polyline segments.
func scallopEdge(ax, ay, bx, by float64) []DrawingCurve {
	dx, dy := bx-ax, by-ay
	length := math.Hypot(dx, dy)
	if length == 0 {
		return nil
	}
	ux, uy := dx/length, dy/length // along the edge
	nx, ny := -uy, ux              // outward normal (left of the CCW direction)
	count := int(math.Max(1, math.Round(length/scallopBumpMM)))
	step := length / float64(count)
	var out []DrawingCurve
	const seg = 6
	for k := 0; k < count; k++ {
		s0 := float64(k) * step
		for j := 0; j < seg; j++ {
			t0 := math.Pi * float64(j) / seg
			t1 := math.Pi * float64(j+1) / seg
			out = append(out, DrawingCurve{
				A:       bumpPoint(ax, ay, ux, uy, nx, ny, s0, step, t0),
				B:       bumpPoint(ax, ay, ux, uy, nx, ny, s0, step, t1),
				Visible: true,
			})
		}
	}
	return out
}

// bumpPoint is a point on one scallop arc: the arc spans [s0, s0+step] along the edge and bulges
// outward by up to step/2 along the normal.
func bumpPoint(ax, ay, ux, uy, nx, ny, s0, step, t float64) gmath.Point2 {
	rad := step / 2
	mid := s0 + rad
	along := mid - rad*math.Cos(t)
	out := rad * math.Sin(t)
	return gmath.P2(gmath.Scalar(ax+ux*along+nx*out), gmath.Scalar(ay+uy*along+ny*out))
}
