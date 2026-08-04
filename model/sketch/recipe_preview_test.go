// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// A recipe's curves are what the live preview paints, so each entity kind must sample into
// something drawable and carry its construction flag through (#2014).
func TestRecipeCurvesSamplesEveryKind(t *testing.T) {
	cases := []struct {
		name         string
		r            Recipe
		curves       int
		closed       int
		minPts       int
		construction int
	}{
		{"line", LineRecipe(math.P2(0, 0), math.P2(10, 0)), 1, 0, 2, 0},
		{"circle", CircleRecipe(math.P2(0, 0), 5), 1, 1, 8, 0},
		{"arc", ArcRecipe(math.P2(0, 0), math.P2(5, 0), math.P2(0, 5), true), 1, 0, 3, 0},
		{"ellipse", EllipseRecipe(math.P2(0, 0), math.V2(1, 0), 5, 3), 1, 1, 8, 0},
		{"rectangle", RectangleRecipe(math.P2(0, 0), math.P2(10, 8)), 4, 0, 2, 0},
		{"centre rectangle", CenterRectangleRecipe(math.P2(0, 0), math.P2(5, 4)), 6, 0, 2, 2},
		{"polygon", PolygonRecipe(math.P2(0, 0), math.P2(5, 0), 6, true), 7, 1, 2, 1},
		{"spline", SplineRecipe([]math.Point2{math.P2(0, 0), math.P2(3, 4), math.P2(7, 1)}, true), 1, 0, 3, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			curves := RecipeCurves(c.r)
			if len(curves) != c.curves {
				t.Fatalf("curves = %d, want %d", len(curves), c.curves)
			}
			closed, construction := 0, 0
			for i, cv := range curves {
				if len(cv.Points) < c.minPts {
					t.Errorf("curve %d has %d points, want at least %d", i, len(cv.Points), c.minPts)
				}
				if cv.Closed {
					closed++
				}
				if cv.Construction {
					construction++
				}
			}
			if closed != c.closed {
				t.Errorf("closed curves = %d, want %d", closed, c.closed)
			}
			if construction != c.construction {
				t.Errorf("construction curves = %d, want %d", construction, c.construction)
			}
		})
	}
}

// A standalone point has no curve to draw.
func TestRecipeCurvesSkipsPoints(t *testing.T) {
	if curves := RecipeCurves(PointRecipe(math.P2(1, 2))); len(curves) != 0 {
		t.Errorf("curves = %d, want 0 — a point draws no curve", len(curves))
	}
}

// A sampled arc must start at its start point and end at its end point, whichever way it sweeps.
func TestRecipeArcSampleEndpoints(t *testing.T) {
	for _, ccw := range []bool{true, false} {
		start, end := math.P2(5, 0), math.P2(0, 5)
		pts := sampleRecipeArc(math.P2(0, 0), start, end, ccw)
		if !pts[0].IsEqualTo(start, 1e-9) {
			t.Errorf("ccw=%v: first sample = %v, want the start %v", ccw, pts[0], start)
		}
		if last := pts[len(pts)-1]; !last.IsEqualTo(end, 1e-9) {
			t.Errorf("ccw=%v: last sample = %v, want the end %v", ccw, last, end)
		}
	}
}

// The sweep is positive counter-clockwise and negative clockwise, and never zero for distinct
// endpoints — a zero sweep would collapse the arc to a point.
func TestDirectedSweepSigns(t *testing.T) {
	quarter := stdmath.Pi / 2
	if got := directedSweep(0, quarter, true); stdmath.Abs(got-quarter) > 1e-9 {
		t.Errorf("ccw sweep = %v, want %v", got, quarter)
	}
	if got := directedSweep(0, quarter, false); got >= 0 {
		t.Errorf("cw sweep = %v, want negative", got)
	}
}

// A short arc is not over-sampled and a long one is not under-sampled, with a floor of two
// segments so even a sliver has a shape.
func TestArcSampleCountScalesWithSweep(t *testing.T) {
	full := arcSampleCount(2 * stdmath.Pi)
	half := arcSampleCount(stdmath.Pi)
	if half >= full {
		t.Errorf("half-turn samples %d, full turn %d — the count must scale with the sweep", half, full)
	}
	if n := arcSampleCount(1e-6); n < 2 {
		t.Errorf("sliver samples = %d, want at least 2", n)
	}
}

// The outline joins a ring of lines into its distinct corners, which is what the inference-glyph
// overlay reads to find the segment being rubber-banded.
func TestRecipeOutlineJoinsLineRing(t *testing.T) {
	pts, closed := RecipeOutline(RectangleRecipe(math.P2(0, 0), math.P2(10, 8)))
	if len(pts) != 4 {
		t.Errorf("outline = %d points, want the 4 distinct corners", len(pts))
	}
	if !closed {
		t.Error("a rectangle's outline must report closed")
	}
}

// Construction geometry is not part of the outline: the centre rectangle's diagonals must not
// appear as edges.
func TestRecipeOutlineExcludesConstruction(t *testing.T) {
	pts, _ := RecipeOutline(CenterRectangleRecipe(math.P2(0, 0), math.P2(5, 4)))
	if len(pts) != 4 {
		t.Errorf("outline = %d points, want 4 — the diagonals are construction, not edges", len(pts))
	}
}

// A recipe whose geometry is not a line ring falls back to its curves' samples.
func TestRecipeOutlineFallsBackToSamples(t *testing.T) {
	pts, closed := RecipeOutline(CircleRecipe(math.P2(0, 0), 5))
	if len(pts) < 8 || !closed {
		t.Errorf("circle outline = %d points closed=%v, want a closed ring", len(pts), closed)
	}
}
