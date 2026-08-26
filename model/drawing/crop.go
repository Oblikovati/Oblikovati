// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// Drawing view crop (#1987): clip any view to a rectangular or circular fence on the sheet, with
// an optional break-mark boundary. Unlike a detail view, a crop keeps the view's scale and is not
// a new derived view — it is a post-projection clip re-applied on every recompute, so it survives
// model edits. Multiple crops intersect (each further clips the surviving curves).

// cropShape discriminates a crop fence.
type cropShape int8

const (
	cropRectangle cropShape = iota
	cropCircle
)

// cropRegion is one crop fence on the sheet (millimetres): a rectangle (x0,y0)-(x1,y1) or a circle
// (cx,cy,r), plus the break-mark boundary drawn around it.
type cropRegion struct {
	shape          cropShape
	x0, y0, x1, y1 float64 // rectangle bounds (sheet mm), normalised so x0<x1, y0<y1
	cx, cy, r      float64 // circle centre + radius (sheet mm)
	breakMark      types.CropBreakMarkLineType
}

// CropSpec is the model-facing request to crop a view (#1987): the fence shape, its rectangle
// bounds or circle (sheet mm), and the break-mark boundary. The router builds it from the wire.
type CropSpec struct {
	View                     string
	Circle                   bool
	X0, Y0, X1, Y1           float64
	CircleX, CircleY, Radius float64
	BreakMark                types.CropBreakMarkLineType
}

// AddCrop clips the named view to the given fence and re-projects it (the crop re-applies every
// recompute). It errors when no view carries that name or the fence is degenerate.
func (vs *DrawingViews) AddCrop(spec CropSpec) (*DrawingView, error) {
	v, ok := vs.ByName(spec.View)
	if !ok {
		return nil, fmt.Errorf("drawing: no view %q to crop", spec.View)
	}
	region, err := cropRegionOf(spec)
	if err != nil {
		return nil, err
	}
	v.crops = append(v.crops, region)
	vs.Recompute()
	return v, nil
}

// RemoveCrop drops every crop on the named view and re-projects it, restoring the full curve set.
func (vs *DrawingViews) RemoveCrop(name string) error {
	v, ok := vs.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no view %q to un-crop", name)
	}
	v.crops = nil
	vs.Recompute()
	return nil
}

// cropRegionOf validates a crop spec and builds its region, normalising the rectangle bounds.
func cropRegionOf(spec CropSpec) (cropRegion, error) {
	if spec.Circle {
		if spec.Radius <= 0 {
			return cropRegion{}, fmt.Errorf("drawing: circular crop needs a positive radius, got %g", spec.Radius)
		}
		return cropRegion{shape: cropCircle, cx: spec.CircleX, cy: spec.CircleY, r: spec.Radius, breakMark: spec.BreakMark}, nil
	}
	x0, x1 := minmax(spec.X0, spec.X1)
	y0, y1 := minmax(spec.Y0, spec.Y1)
	if x1-x0 <= 0 || y1-y0 <= 0 {
		return cropRegion{}, fmt.Errorf("drawing: rectangular crop is degenerate (%g×%g mm)", x1-x0, y1-y0)
	}
	return cropRegion{shape: cropRectangle, x0: x0, y0: y0, x1: x1, y1: y1, breakMark: spec.BreakMark}, nil
}

// CropCount is the number of crop fences clipping the view (#1987).
func (v *DrawingView) CropCount() int { return len(v.crops) }

// cropRecipesOf snapshots a view's crop fences for persistence (#1987).
func cropRecipesOf(crops []cropRegion) []cropRecipe {
	if len(crops) == 0 {
		return nil
	}
	out := make([]cropRecipe, len(crops))
	for i, c := range crops {
		out[i] = cropRecipe{
			Circle: c.shape == cropCircle, X0: c.x0, Y0: c.y0, X1: c.x1, Y1: c.y1,
			CircleX: c.cx, CircleY: c.cy, Radius: c.r, BreakMark: c.breakMark.String(),
		}
	}
	return out
}

// cropRegionsFrom rebuilds a view's crop fences from its recipe (#1987).
func cropRegionsFrom(recs []cropRecipe) []cropRegion {
	if len(recs) == 0 {
		return nil
	}
	out := make([]cropRegion, len(recs))
	for i, r := range recs {
		mark, _ := types.ParseCropBreakMarkLineType(r.BreakMark)
		shape := cropRectangle
		if r.Circle {
			shape = cropCircle
		}
		out[i] = cropRegion{
			shape: shape, x0: r.X0, y0: r.Y0, x1: r.X1, y1: r.Y1,
			cx: r.CircleX, cy: r.CircleY, r: r.Radius, breakMark: mark,
		}
	}
	return out
}

// applyCrops clips the view's placed curves to each crop fence in turn (intersection) and draws
// the break-mark boundary of each. A view with no crops is left untouched.
func (v *DrawingView) applyCrops() {
	for _, c := range v.crops {
		v.curves = clipCurvesToCrop(v.curves, c)
		v.curves = append(v.curves, c.breakMarkCurves()...)
	}
}

// clipCurvesToCrop returns the sub-segments of every edge/hatch/section curve that lie inside the
// fence; a curve wholly outside is dropped. Break-mark curves already added by a prior crop pass
// are preserved (they are inside by construction).
func clipCurvesToCrop(curves []DrawingCurve, c cropRegion) []DrawingCurve {
	out := make([]DrawingCurve, 0, len(curves))
	for _, cur := range curves {
		a, b, ok := c.clip(cur.A, cur.B)
		if !ok {
			continue
		}
		cur.A, cur.B = a, b
		out = append(out, cur)
	}
	return out
}

// clip restricts a segment to the crop fence, returning the inside sub-segment; ok is false when
// the segment lies wholly outside.
func (c cropRegion) clip(a, b gmath.Point2) (gmath.Point2, gmath.Point2, bool) {
	if c.shape == cropCircle {
		return clipToCircle(a, b, c.cx, c.cy, c.r)
	}
	return clipToRect(a, b, c.x0, c.y0, c.x1, c.y1)
}

// clipToRect restricts segment a→b to the axis-aligned rectangle [x0,x1]×[y0,y1] via Liang–Barsky,
// returning the inside sub-segment; ok is false when the segment lies wholly outside.
func clipToRect(a, b gmath.Point2, x0, y0, x1, y1 float64) (gmath.Point2, gmath.Point2, bool) {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	t0, t1 := 0.0, 1.0
	for _, e := range [4][2]float64{{-dx, float64(a.X) - x0}, {dx, x1 - float64(a.X)}, {-dy, float64(a.Y) - y0}, {dy, y1 - float64(a.Y)}} {
		p, q := e[0], e[1]
		if p == 0 { // parallel to this edge: outside if it starts beyond it
			if q < 0 {
				return a, b, false
			}
			continue
		}
		r := q / p
		if p < 0 {
			if r > t1 {
				return a, b, false
			}
			if r > t0 {
				t0 = r
			}
		} else {
			if r < t0 {
				return a, b, false
			}
			if r < t1 {
				t1 = r
			}
		}
	}
	return a.Lerp(b, t0), a.Lerp(b, t1), true
}

// breakMarkCurves draws the crop fence boundary as break-mark curves — a continuous outline, a
// zigzag outline (rectangle only; a circular fence falls back to continuous), or nothing.
func (c cropRegion) breakMarkCurves() []DrawingCurve {
	switch c.breakMark {
	case types.NoCropBreakMark:
		return nil
	case types.ZigzagCropBreakMark:
		if c.shape == cropRectangle {
			return c.zigzagRectCurves()
		}
		return c.outlineCurves()
	default:
		return c.outlineCurves()
	}
}

// outlineCurves draws the fence as a continuous boundary: the four rectangle edges, or a polygon
// approximation of the circle.
func (c cropRegion) outlineCurves() []DrawingCurve {
	if c.shape == cropCircle {
		return c.circleOutlineCurves()
	}
	corners := [4]gmath.Point2{
		gmath.P2(gmath.Scalar(c.x0), gmath.Scalar(c.y0)), gmath.P2(gmath.Scalar(c.x1), gmath.Scalar(c.y0)),
		gmath.P2(gmath.Scalar(c.x1), gmath.Scalar(c.y1)), gmath.P2(gmath.Scalar(c.x0), gmath.Scalar(c.y1)),
	}
	out := make([]DrawingCurve, 0, 4)
	for i := range 4 {
		out = append(out, breakCurve(corners[i], corners[(i+1)%4]))
	}
	return out
}

// cropCircleFacets is the segment count of a circular crop's polygon outline (smooth enough on a
// drawing sheet, cheap to clip/persist).
const cropCircleFacets = 48

// circleOutlineCurves approximates the circular fence as a closed polygon of break-mark segments.
func (c cropRegion) circleOutlineCurves() []DrawingCurve {
	pts := make([]gmath.Point2, cropCircleFacets)
	for i := range pts {
		th := 2 * math.Pi * float64(i) / float64(cropCircleFacets)
		pts[i] = gmath.P2(gmath.Scalar(c.cx+c.r*math.Cos(th)), gmath.Scalar(c.cy+c.r*math.Sin(th)))
	}
	out := make([]DrawingCurve, 0, cropCircleFacets)
	for i := range pts {
		out = append(out, breakCurve(pts[i], pts[(i+1)%cropCircleFacets]))
	}
	return out
}

// zigzagTeeth is the number of zigzag teeth drawn per rectangle edge; zigzagAmp is each tooth's
// perpendicular amplitude (sheet mm).
const (
	zigzagTeeth = 6
	zigzagAmp   = 2.0
)

// zigzagRectCurves draws each rectangle edge as a zigzag break line: the edge is split into
// zigzagTeeth spans whose midpoints alternate to either side of the edge by zigzagAmp.
func (c cropRegion) zigzagRectCurves() []DrawingCurve {
	corners := [4]gmath.Point2{
		gmath.P2(gmath.Scalar(c.x0), gmath.Scalar(c.y0)), gmath.P2(gmath.Scalar(c.x1), gmath.Scalar(c.y0)),
		gmath.P2(gmath.Scalar(c.x1), gmath.Scalar(c.y1)), gmath.P2(gmath.Scalar(c.x0), gmath.Scalar(c.y1)),
	}
	var out []DrawingCurve
	for i := range 4 {
		out = append(out, zigzagEdge(corners[i], corners[(i+1)%4])...)
	}
	return out
}

// zigzagEdge returns the break-mark segments of a zigzag running from a to b, its teeth deflected
// perpendicular to the edge by zigzagAmp.
func zigzagEdge(a, b gmath.Point2) []DrawingCurve {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	length := math.Hypot(dx, dy)
	if length < 1e-9 {
		return nil
	}
	nx, ny := -dy/length, dx/length // unit perpendicular
	pts := make([]gmath.Point2, 0, 2*zigzagTeeth+1)
	pts = append(pts, a)
	for i := 1; i < 2*zigzagTeeth; i++ {
		t := float64(i) / float64(2*zigzagTeeth)
		off := 0.0
		if i%2 == 1 {
			off = zigzagAmp
		}
		px := float64(a.X) + dx*t + nx*off
		py := float64(a.Y) + dy*t + ny*off
		pts = append(pts, gmath.P2(gmath.Scalar(px), gmath.Scalar(py)))
	}
	pts = append(pts, b)
	out := make([]DrawingCurve, 0, len(pts)-1)
	for i := 0; i+1 < len(pts); i++ {
		out = append(out, breakCurve(pts[i], pts[i+1]))
	}
	return out
}

// minmax returns its two arguments in ascending order.
func minmax(a, b float64) (float64, float64) {
	if a <= b {
		return a, b
	}
	return b, a
}

// breakCurve builds one visible break-mark segment between two sheet points.
func breakCurve(a, b gmath.Point2) DrawingCurve {
	return DrawingCurve{A: a, B: b, Visible: true, kind: types.DrawingBreakCurve}
}
