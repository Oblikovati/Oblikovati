// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati/math"
)

func TestInferHorizontalAndVertical(t *testing.T) {
	in := NewInference()
	// Nearly horizontal: 5 units across, 0.05 up → ~0.57°, within tolerance.
	sug := in.InferSegment(math.P2(0, 0), math.P2(5, 0.05))
	if len(sug) != 1 || sug[0].Kind != SuggestHorizontal {
		t.Fatalf("expected a horizontal suggestion, got %+v", sug)
	}
	if sug[0].Priority <= 0 {
		t.Error("suggestion priority should be positive within tolerance")
	}
	// Nearly vertical.
	if sv := in.InferSegment(math.P2(0, 0), math.P2(0.05, 5)); len(sv) != 1 || sv[0].Kind != SuggestVertical {
		t.Fatalf("expected a vertical suggestion, got %+v", sv)
	}
	// A 45° segment infers neither.
	if d := in.InferSegment(math.P2(0, 0), math.P2(5, 5)); len(d) != 0 {
		t.Errorf("diagonal should infer nothing, got %+v", d)
	}
	// Degenerate zero-length segment infers nothing.
	if z := in.InferSegment(math.P2(1, 1), math.P2(1, 1)); z != nil {
		t.Errorf("zero-length segment inferred %+v", z)
	}
}

func TestInferSnapToNearestPoint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	near := s.Points().Add(math.P2(2, 2))
	far := s.Points().Add(math.P2(9, 9))
	in := NewInference()

	sug := in.InferSnap(math.P2(2.0005, 2), []*Point{far, near})
	if len(sug) != 1 || sug[0].Kind != SuggestCoincident || sug[0].Target != near {
		t.Fatalf("expected coincidence snap to the near point, got %+v", sug)
	}
	// Nothing within snap distance → no suggestion.
	if none := in.InferSnap(math.P2(5, 5), []*Point{far, near}); none != nil {
		t.Errorf("far point should not snap, got %+v", none)
	}
}

// Applying the inferred constraint produces the right relation (apply-on-commit).
func TestInferenceAppliesHorizontalConstraint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(5, 0.05))
	sug := NewInference().InferSegment(a.Position(), b.Position())
	if len(sug) == 0 || sug[0].Kind != SuggestHorizontal {
		t.Fatal("no horizontal inferred")
	}
	// Commit: apply a horizontal constraint between the endpoints.
	c := s.GeometricConstraints().AddHorizontal(a, b)
	b.SetPosition(math.P2(5, 0)) // solver would do this; here we set it
	if !satisfied(c) {
		t.Error("applied horizontal constraint not satisfied after leveling")
	}
}
