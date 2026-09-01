// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/topo"
)

// stubVertex/stubEdge stand in only as distinct selectables for ranking tests; the resolution
// logic under test (highestPriority, nearestCandidate) depends solely on each candidate's kind.

func TestHighestPriorityDefaultOrder(t *testing.T) {
	t.Parallel()
	p := &RayPicker{}
	// Vertex outranks edge by default (the historical snap order), independent of insertion order.
	got, ok := p.highestPriority([]Selectable{EdgeHandle{}, VertexHandle{}})
	if !ok {
		t.Fatal("highestPriority returned none")
	}
	if _, isVertex := got.(VertexHandle); !isVertex {
		t.Fatalf("default order: got %T, want VertexHandle (vertex outranks edge)", got)
	}
}

func TestHighestPriorityHonoursUserRank(t *testing.T) {
	t.Parallel()
	p := &RayPicker{}
	st := NewSelectionFilterState()
	st.Move(st.Rank(SelectEdge), 0) // drag Edges above Vertices
	p.SetPriorityRank(st.Rank)
	got, _ := p.highestPriority([]Selectable{VertexHandle{}, EdgeHandle{}})
	if _, isEdge := got.(EdgeHandle); !isEdge {
		t.Fatalf("after reordering edge to top: got %T, want EdgeHandle", got)
	}
}

func TestNearestCandidateProximityWinsBeyondWindow(t *testing.T) {
	t.Parallel()
	p := &RayPicker{} // zero camera ⇒ depthTieEpsilon 0, so only the nearer hit qualifies
	cands := []pickCandidate{
		{t: 5.0, sel: ProfileHandle{}}, // nearer (in front)
		{t: 9.0, sel: FaceHandle{}},    // behind, higher default priority
	}
	got, ok := p.nearestCandidate(cands)
	if !ok {
		t.Fatal("nearestCandidate returned none")
	}
	if _, isProfile := got.(ProfileHandle); !isProfile {
		t.Fatalf("a hit genuinely in front must win regardless of priority: got %T", got)
	}
}

func TestNearestCandidateCoincidentResolvesByPriority(t *testing.T) {
	t.Parallel()
	p := &RayPicker{}
	// Exactly coincident depth (within the zero-epsilon window): default order prefers the face
	// over the profile (a solid on its sketch), matching the historical append precedence.
	cands := []pickCandidate{
		{t: 5.0, sel: ProfileHandle{}},
		{t: 5.0, sel: FaceHandle{}},
	}
	got, _ := p.nearestCandidate(cands)
	if _, isFace := got.(FaceHandle); !isFace {
		t.Fatalf("coincident default: got %T, want FaceHandle", got)
	}
}

// TestSessionPushesPriorityRankToPicker pins the SetPicker wiring: the picker resolves a
// coincident vertex+edge by the live SelectionFilterState order, and reordering then re-pushing
// flips the winner — the seam the Selection Filter window relies on (#1222).
func TestSessionPushesPriorityRankToPicker(t *testing.T) {
	t.Parallel()
	s := NewSession()
	p := NewRayPicker(s.Camera(), func() []*topo.Body { return nil })
	s.SetPicker(p) // pushes s.selectionFilterState.Rank

	if got, _ := p.highestPriority([]Selectable{EdgeHandle{}, VertexHandle{}}); !isVertexHandle(got) {
		t.Fatalf("default pushed rank: got %T, want VertexHandle", got)
	}
	st := s.SelectionFilterState()
	st.Move(st.Rank(SelectEdge), 0) // mutate the live state the closure reads
	if got, _ := p.highestPriority([]Selectable{EdgeHandle{}, VertexHandle{}}); isVertexHandle(got) {
		t.Fatal("after reordering edge above vertex, the picker must prefer the edge")
	}
}

func isVertexHandle(s Selectable) bool { _, ok := s.(VertexHandle); return ok }
