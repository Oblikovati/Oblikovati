// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// TestPickAllReturnsOccludedBodiesFrontToBack aims a ray through two stacked boxes and checks
// PickAll returns both, the nearer one first — the occluded-geometry list Select Other walks.
func TestPickAllReturnsOccludedBodiesFrontToBack(t *testing.T) {
	s := extrudedBox(t, 2, 4) // box [0,2]×[0,2]×[0,4], centre (1,1,2)
	front := partBodies(s)()[0]
	back := front
	near, err := ops.TransformBody(front, math.Translation4(math.V3(10, 0, 0)), func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	// Camera at +X looking toward −X: the +10x copy (near) is in front of the original (back).
	cam := scene.NewCamera(200, 200)
	cam.Eye, cam.Target, cam.Up = math.P3(40, 1, 2), math.P3(0, 1, 2), math.V3(0, 0, 1)
	p := NewRayPicker(cam, func() []*topo.Body { return []*topo.Body{back, near} })

	hits := p.PickAll(100, 100, NewSelectionFilter()) // screen centre → the ray through both
	if len(hits) != 2 {
		t.Fatalf("PickAll through two stacked boxes = %d hits, want 2", len(hits))
	}
	if h, ok := hits[0].(FaceHandle); !ok || h.Body != near {
		t.Errorf("front hit = %v, want a face of the nearer box", hits[0])
	}
	if h := hits[1].(FaceHandle); h.Body != back {
		t.Errorf("second hit = %v, want a face of the farther box", hits[1])
	}
}

// fakeMultiPicker returns a canned candidate list for PickAll (and the front-most for Pick), so the
// cycle state machine is testable without real geometry.
type fakeMultiPicker struct{ all []Selectable }

func (f fakeMultiPicker) Pick(_, _ float64, _ *SelectionFilter) (Selectable, bool) {
	if len(f.all) == 0 {
		return nil, false
	}
	return f.all[0], true
}
func (f fakeMultiPicker) PickAll(_, _ float64, _ *SelectionFilter) []Selectable { return f.all }

func TestSelectOtherCyclesAndCommits(t *testing.T) {
	a, b, c := FaceHandle{Face: aFace()}, EdgeHandle{}, FaceHandle{Face: aFace()}
	s := NewSession()
	s.SetPicker(fakeMultiPicker{all: []Selectable{a, b, c}})

	if !s.BeginSelectOther(0, 0) || !s.SelectOtherActive() {
		t.Fatal("BeginSelectOther should start a cycle with 3 candidates")
	}
	if pos, count := s.SelectOtherStatus(); pos != 1 || count != 3 || !s.Selection().Contains(a) {
		t.Fatalf("first candidate: pos=%d count=%d containsA=%v", pos, count, s.Selection().Contains(a))
	}
	s.CycleSelectOther(1)
	if !s.Selection().Contains(b) {
		t.Error("cycle +1 should select the second candidate")
	}
	s.CycleSelectOther(1)
	s.CycleSelectOther(1) // wrap 2 → 0
	if !s.Selection().Contains(a) {
		t.Error("cycle should wrap back to the first candidate")
	}
	s.CycleSelectOther(-1) // wrap 0 → 2
	if !s.Selection().Contains(c) {
		t.Error("cycle −1 from the first should wrap to the last")
	}
	s.CommitSelectOther()
	if s.SelectOtherActive() || !s.Selection().Contains(c) {
		t.Errorf("commit should end the cycle keeping c: active=%v containsC=%v", s.SelectOtherActive(), s.Selection().Contains(c))
	}
}

func TestBeginSelectOtherNoopWhenNothingOccluded(t *testing.T) {
	a := FaceHandle{Face: aFace()}
	// Fewer than two candidates → not worth cycling.
	one := NewSession()
	one.SetPicker(fakeMultiPicker{all: []Selectable{a}})
	if one.BeginSelectOther(0, 0) || one.SelectOtherActive() {
		t.Error("Select Other should not begin with a single candidate")
	}
	// A plain picker (no PickAll) → no Select Other.
	plain := NewSession()
	plain.SetPicker(stubPicker{sel: a})
	if plain.BeginSelectOther(0, 0) {
		t.Error("Select Other should not begin without a MultiPicker")
	}
}
