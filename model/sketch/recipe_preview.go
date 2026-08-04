// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Turning a [Recipe] into drawable polylines. This is what the live preview paints while a
// shape is being placed, and it reads the same Recipe the commit applies — so what the user
// sees while dragging and what they get on release cannot disagree (#2014).
//
// Nothing is materialised: the recipe is sampled directly from its own parameters, so a preview
// frame touches neither the sketch, the solver, nor the undo stream.

// previewSegments is the sampling resolution for a full circle or ellipse; arcs use a
// proportional share of it. It matches the committed wireframe's resolution so the rubber band
// and the finished geometry look the same.
const previewSegments = 64

// PreviewCurve is one drawable curve from a recipe: its sampled points, whether it closes, and
// whether it is construction geometry (drawn dashed rather than solid).
type PreviewCurve struct {
	Points       []math.Point2
	Closed       bool
	Construction bool
}

// RecipeCurves samples every entity in the recipe into a drawable curve, in recipe order.
//
//	for _, c := range sketch.RecipeCurves(r) { draw(c.Points, c.Closed, c.Construction) }
func RecipeCurves(r Recipe) []PreviewCurve {
	out := make([]PreviewCurve, 0, len(r.Entities))
	for _, e := range r.Entities {
		pts, closed, ok := sampleRecipeEntity(e, r.Points)
		if !ok {
			continue
		}
		out = append(out, PreviewCurve{Points: pts, Closed: closed, Construction: e.Construction})
	}
	return out
}

// sampleRecipeEntity samples one recipe entity. A standalone point has no curve to draw.
func sampleRecipeEntity(e RecipeEntity, pts []math.Point2) ([]math.Point2, bool, bool) {
	if !recipeIndicesInRange(e.Points, len(pts)) {
		return nil, false, false
	}
	switch e.Kind {
	case RecipeLine:
		return []math.Point2{pts[e.Points[0]], pts[e.Points[1]]}, false, true
	case RecipeArc:
		return sampleRecipeArc(pts[e.Points[0]], pts[e.Points[1]], pts[e.Points[2]], e.CounterClockwise), false, true
	case RecipeCircle:
		return sampleRecipeCircle(pts[e.Points[0]], float64(e.Radius)), true, true
	case RecipeEllipse:
		return sampleRecipeEllipse(pts[e.Points[0]], e.MajorAxis, float64(e.Radius), float64(e.MinorRadius)), true, true
	case RecipeSpline:
		return recipePolyline(e.Points, pts), e.Closed, true
	default: // RecipePoint draws no curve
		return nil, false, false
	}
}

// recipeIndicesInRange reports whether every index addresses a point the recipe defines.
func recipeIndicesInRange(idx []int, n int) bool {
	for _, i := range idx {
		if i < 0 || i >= n {
			return false
		}
	}
	return len(idx) > 0
}

// recipePolyline gathers the positions an entity names, in order.
func recipePolyline(idx []int, pts []math.Point2) []math.Point2 {
	out := make([]math.Point2, len(idx))
	for i, j := range idx {
		out[i] = pts[j]
	}
	return out
}

// sampleRecipeCircle returns a closed ring around center.
func sampleRecipeCircle(center math.Point2, r float64) []math.Point2 {
	pts := make([]math.Point2, previewSegments)
	for i := range pts {
		a := 2 * stdmath.Pi * float64(i) / previewSegments
		pts[i] = math.P2(center.X+math.Scalar(r*stdmath.Cos(a)), center.Y+math.Scalar(r*stdmath.Sin(a)))
	}
	return pts
}

// sampleRecipeArc walks the arc from start to end about center, in the given direction, with a
// resolution proportional to the swept angle.
func sampleRecipeArc(center, start, end math.Point2, ccw bool) []math.Point2 {
	r := center.DistanceTo(start)
	a0 := stdmath.Atan2(float64(start.Y-center.Y), float64(start.X-center.X))
	sweep := directedSweep(a0, stdmath.Atan2(float64(end.Y-center.Y), float64(end.X-center.X)), ccw)
	n := arcSampleCount(sweep)
	pts := make([]math.Point2, n+1)
	for i := range pts {
		a := a0 + sweep*float64(i)/float64(n)
		pts[i] = math.P2(center.X+math.Scalar(r*stdmath.Cos(a)), center.Y+math.Scalar(r*stdmath.Sin(a)))
	}
	return pts
}

// directedSweep is the signed angle from a0 to a1 in the requested direction, in (0, 2π].
func directedSweep(a0, a1 float64, ccw bool) float64 {
	d := a1 - a0
	for d <= 0 {
		d += 2 * stdmath.Pi
	}
	for d > 2*stdmath.Pi {
		d -= 2 * stdmath.Pi
	}
	if !ccw {
		return d - 2*stdmath.Pi
	}
	return d
}

// arcSampleCount scales the sample count with the swept angle, so a short arc is not
// over-sampled and a near-full one is not under-sampled. At least two segments always.
func arcSampleCount(sweep float64) int {
	n := int(previewSegments * stdmath.Abs(sweep) / (2 * stdmath.Pi))
	if n < 2 {
		return 2
	}
	return n
}

// sampleRecipeEllipse returns a closed ring around center, oriented by majorAxis.
func sampleRecipeEllipse(center math.Point2, majorAxis math.Vector2, majorR, minorR float64) []math.Point2 {
	rot := stdmath.Atan2(float64(majorAxis.Y), float64(majorAxis.X))
	cos, sin := stdmath.Cos(rot), stdmath.Sin(rot)
	pts := make([]math.Point2, previewSegments)
	for i := range pts {
		a := 2 * stdmath.Pi * float64(i) / previewSegments
		x, y := majorR*stdmath.Cos(a), minorR*stdmath.Sin(a)
		pts[i] = math.P2(center.X+math.Scalar(x*cos-y*sin), center.Y+math.Scalar(x*sin+y*cos))
	}
	return pts
}

// RecipeOutline flattens a recipe's real (non-construction) geometry into one polyline, which
// is what the inference-glyph overlay reads to find the segment being rubber-banded. A ring of
// lines becomes its corner ring; anything else is its curves' samples end to end.
func RecipeOutline(r Recipe) (pts []math.Point2, closed bool) {
	if ring, ok := lineRingOutline(r); ok {
		return ring, len(ring) > 2
	}
	for _, c := range RecipeCurves(r) {
		if c.Construction {
			continue
		}
		pts = append(pts, c.Points...)
		closed = closed || c.Closed
	}
	return pts, closed
}

// lineRingOutline chains a recipe made only of lines into its corner ring, de-duplicating the
// endpoints consecutive segments share. ok is false when the recipe holds any other curve.
func lineRingOutline(r Recipe) ([]math.Point2, bool) {
	var idx []int
	for _, e := range r.Entities {
		if e.Construction {
			continue
		}
		if e.Kind != RecipeLine {
			return nil, false
		}
		idx = appendRingIndices(idx, e.Points)
	}
	if len(idx) == 0 {
		return nil, false
	}
	return recipePolyline(trimClosingIndex(idx), r.Points), true
}

// appendRingIndices chains a segment onto the running ring, skipping its start when it repeats
// the previous segment's end.
func appendRingIndices(idx, seg []int) []int {
	if len(idx) > 0 && idx[len(idx)-1] == seg[0] {
		return append(idx, seg[1])
	}
	return append(idx, seg...)
}

// trimClosingIndex drops a trailing index that repeats the first, so a closed ring is reported
// by its distinct corners rather than with the first corner twice.
func trimClosingIndex(idx []int) []int {
	if len(idx) > 1 && idx[0] == idx[len(idx)-1] {
		return idx[:len(idx)-1]
	}
	return idx
}
