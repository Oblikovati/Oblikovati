// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
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
	viewName string  // CoG: the view it marks
	x, y     float64 // revision cloud: lower-left corner (sheet mm)
	w, h     float64 // revision cloud: size (sheet mm)
	tag      string  // revision cloud: revision label
	curves   []DrawingCurve
}

// Name, Kind, ViewName, Tag and Curves expose the annotation.
func (a *DrawingAnnotation) Name() string                      { return a.name }
func (a *DrawingAnnotation) Kind() types.DrawingAnnotationKind { return a.kind }
func (a *DrawingAnnotation) ViewName() string                  { return a.viewName }
func (a *DrawingAnnotation) Tag() string                       { return a.tag }
func (a *DrawingAnnotation) Curves() []DrawingCurve            { return a.curves }
func (a *DrawingAnnotation) CurveCount() int                   { return len(a.curves) }

// DrawingAnnotations is a sheet's annotation collection. It holds the body-resolution hook and
// the sheet's views so a CoG marker can find its view and the model's centre of mass.
type DrawingAnnotations struct {
	items []*DrawingAnnotation
	views *DrawingViews
	body  bodyLookup
}

func newDrawingAnnotations(views *DrawingViews, body bodyLookup) *DrawingAnnotations {
	return &DrawingAnnotations{views: views, body: body}
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

// Recompute re-derives the associative annotations (the CoG markers) against the current model.
func (as *DrawingAnnotations) Recompute() {
	for _, a := range as.items {
		if a.kind == types.CoGMarkerAnnotation {
			as.recomputeCoG(a)
		}
	}
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
