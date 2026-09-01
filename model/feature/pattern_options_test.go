// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestRectStepSpacing checks the fitted spacing divides the step across the gaps while the
// default keeps it as the per-occurrence offset (M20-F18).
func TestRectStepSpacing(t *testing.T) {
	t.Parallel()
	step := math.V3(6, 0, 0)
	if got := (PatternOptions{}).rectStep(step, 3); got.X != 6 {
		t.Errorf("default rectStep.X = %g, want 6 (step is the gap)", got.X)
	}
	if got := (PatternOptions{Spacing: types.SpacingFitted}).rectStep(step, 3); got.X != 3 {
		t.Errorf("fitted rectStep.X = %g, want 3 (span 6 over 2 gaps)", got.X)
	}
}

// TestCircIncrementSpacing checks the three circular spacing interpretations.
func TestCircIncrementSpacing(t *testing.T) {
	t.Parallel()
	full := 2 * stdmath.Pi
	if got := (PatternOptions{}).circIncrement(full, 4); stdmath.Abs(got-full/4) > 1e-12 {
		t.Errorf("default circ inc = %g, want angle/count", got)
	}
	if got := (PatternOptions{Spacing: types.SpacingBetween}).circIncrement(0.5, 4); got != 0.5 {
		t.Errorf("between circ inc = %g, want the angle itself (0.5)", got)
	}
	if got := (PatternOptions{Spacing: types.SpacingFitted}).circIncrement(full, 3); stdmath.Abs(got-stdmath.Pi) > 1e-12 {
		t.Errorf("fitted circ inc = %g, want angle/(count-1) = pi", got)
	}
}

// TestPatternBoundaryClipsOutsideOccurrences drops the occurrences whose centre falls
// outside the boundary loop (M20-F18).
func TestPatternBoundaryClipsOutsideOccurrences(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	src := NewBaseFeatures(fs).AddBase(prismBody()) // unit cube [0,1]^3, centre (0.5,0.5,0.5)
	rect := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()},
		func() int { return 3 }, func() int { return 1 }, math.V3(2, 0, 0), math.Vector3{})
	fs.Recompute()
	if n := len(fs.Result()); n != 3 {
		t.Fatalf("unclipped pattern = %d bodies, want 3", n)
	}

	// A boundary covering x∈[-1,3.5] keeps occurrences 0 (x=0.5) and 1 (x=2.5) but clips
	// occurrence 2 (centre x=4.5).
	boundary, err := NewPatternBoundary(math.P3(0, 0, 0), math.V3(0, 0, 1),
		[]math.Point3{{X: -1, Y: -1}, {X: 3.5, Y: -1}, {X: 3.5, Y: 2}, {X: -1, Y: 2}}, types.IncludeByCentroid)
	if err != nil {
		t.Fatalf("NewPatternBoundary: %v", err)
	}
	rect.Definition().Options = PatternOptions{Boundary: boundary}
	fs.MarkDirty(fs.Item(1)) // the pattern is feature 1 (base is 0)
	fs.Recompute()
	if n := len(fs.Result()); n != 2 {
		t.Errorf("clipped pattern = %d bodies, want 2 (one occurrence outside the boundary)", n)
	}
}

// TestPatternOptionsRoundTrip preserves the spacing type and boundary across an .obk
// round-trip (M20-F18).
func TestPatternOptionsRoundTrip(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil)
	src := NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	rect := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()},
		func() int { return 2 }, func() int { return 1 }, math.V3(4, 0, 0), math.Vector3{})
	boundary, _ := NewPatternBoundary(math.P3(0, 0, 0), math.V3(0, 0, 1),
		[]math.Point3{{X: -1, Y: -1}, {X: 9, Y: -1}, {X: 9, Y: 2}, {X: -1, Y: 2}}, types.IncludeByCentroid)
	rect.Definition().Options = PatternOptions{Spacing: types.SpacingFitted, Compute: types.ComputeOptimized, Boundary: boundary}

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	opts := fresh.Item(1).Definition().(*RectangularPatternFeature).Definition().Options
	if opts.Spacing != types.SpacingFitted || opts.Compute != types.ComputeOptimized {
		t.Errorf("restored options = spacing %v compute %v, want fitted/optimized", opts.Spacing, opts.Compute)
	}
	if opts.Boundary == nil || len(opts.Boundary.Polygon) != 4 {
		t.Errorf("restored boundary = %+v, want a 4-vertex loop", opts.Boundary)
	}
}
