// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// TestLineInferenceAppliesHorizontal: a nearly horizontal line picks up a
// horizontal constraint on commit, reported as a typed record (M06-F10, #625).
func TestLineInferenceAppliesHorizontal(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0.05))
	constraints, _ := s.ApplyLineInference(l, DefaultInferenceOptions())
	if len(constraints) != 1 || constraints[0].Kind != types.InferHorizontal {
		t.Fatalf("applied = %+v, want one horizontal inference", constraints)
	}
	if s.GeometricConstraints().Count() != 1 {
		t.Fatalf("constraints = %d, want the auto-applied horizontal", s.GeometricConstraints().Count())
	}
	s.Solve()
	if dy := l.B.Position().Y - l.A.Position().Y; float64(dy) > 1e-9 || float64(dy) < -1e-9 {
		t.Errorf("solved Δy = %v, want 0 (the inferred horizontal drives the solve)", dy)
	}
}

// TestInferencePriorityPicksFamily: a segment that is both nearly horizontal
// and nearly parallel to an existing slanted-but-snapped line obeys the
// priority preference.
func TestInferencePriorityPicksFamily(t *testing.T) {
	build := func() (*Sketch, *Line) {
		s := NewSketches().Add(XYPlane())
		s.Lines().AddByTwoPoints(gmath.P2(0, 5), gmath.P2(4, 5.02)) // nearly horizontal reference
		return s, s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0.04))
	}

	s, l := build()
	opts := DefaultInferenceOptions() // horizontal/vertical preferred
	applied, _ := s.ApplyLineInference(l, opts)
	if len(applied) != 1 || applied[0].Kind != types.InferHorizontal {
		t.Errorf("H/V priority applied %+v, want horizontal", applied)
	}

	s, l = build()
	opts.Priority = types.PriorityParallelPerpendicular
	applied, _ = s.ApplyLineInference(l, opts)
	if len(applied) != 1 || applied[0].Kind != types.InferParallel {
		t.Errorf("parallel priority applied %+v, want parallel", applied)
	}
	if len(applied) == 1 && len(applied[0].Entities) != 2 {
		t.Errorf("parallel record entities = %d, want the line + its reference", len(applied[0].Entities))
	}
}

// TestInferenceSnapsEndpoints: an endpoint within snap distance of an
// existing point is snapped coincident and reported as a point inference.
func TestInferenceSnapsEndpoints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	anchor := s.Points().Add(gmath.P2(2, 1))
	l := s.Lines().AddByTwoPoints(gmath.P2(2.0005, 1.0005), gmath.P2(5, 4))
	_, points := s.ApplyLineInference(l, DefaultInferenceOptions())
	if len(points) != 1 || points[0].Kind != types.SketchInferenceOnPoint {
		t.Fatalf("point inferences = %+v, want one onPoint snap", points)
	}
	if got := l.A.Position(); float64(got.DistanceTo(anchor.Position())) > 1e-12 {
		t.Errorf("snapped endpoint = %v, want exactly on the anchor", got)
	}
}

// TestInferenceDisabledAppliesNothing: turning inference off is honored, and
// infer-without-constrain reports records without touching the sketch.
func TestInferenceDisabledAppliesNothing(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0.01))

	off := DefaultInferenceOptions()
	off.InferEnabled = false
	if c, p := s.ApplyLineInference(l, off); len(c)+len(p) != 0 {
		t.Errorf("disabled inference applied %d records", len(c)+len(p))
	}

	audit := DefaultInferenceOptions()
	audit.ConstrainEnabled = false
	c, _ := s.ApplyLineInference(l, audit)
	if len(c) != 1 {
		t.Fatalf("audit mode reported %d records, want 1", len(c))
	}
	if s.GeometricConstraints().Count() != 0 {
		t.Errorf("audit mode added %d constraints, want 0", s.GeometricConstraints().Count())
	}
}

// TestGlyphSuggestionsFeedOverlay: the headless glyph feed reports the active
// inferences for an in-progress segment.
func TestGlyphSuggestionsFeedOverlay(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Points().Add(gmath.P2(4, 0))
	got := s.GlyphSuggestions(gmath.P2(0, 0), gmath.P2(4.0005, 0.0005), DefaultInferenceOptions())
	kinds := map[SuggestionKind]bool{}
	for _, sg := range got {
		kinds[sg.Kind] = true
	}
	if !kinds[SuggestHorizontal] || !kinds[SuggestCoincident] {
		t.Errorf("glyph kinds = %v, want horizontal + coincident", kinds)
	}
	if len(s.GlyphSuggestions(gmath.P2(0, 0), gmath.P2(4, 0), InferenceOptions{})) != 0 {
		t.Error("disabled inference must feed no glyphs")
	}
}
