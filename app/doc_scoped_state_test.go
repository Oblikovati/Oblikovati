// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// addBoxPart adds an extruded-box part document (activated) to the session and returns its
// document and first body — the geometry the doc-scoped view-state tests switch between.
func addBoxPart(t *testing.T, s *Session, name string) (*doc.Document, *topo.Body) {
	t.Helper()
	d, err := s.Workspace().Add(doc.Part, name, true)
	if err != nil {
		t.Fatalf("add part %s: %v", name, err)
	}
	def := compdef.NewPartComponentDefinition()
	d.SetContent(def)
	sk := def.Sketches().Add(sketch.XYPlane())
	c := []*sketch.Point{
		sk.Points().Add(math.P2(-2, -2)), sk.Points().Add(math.P2(2, -2)),
		sk.Points().Add(math.P2(2, 2)), sk.Points().Add(math.P2(-2, 2)),
	}
	for i := range c {
		sk.Lines().Add(c[i], c[(i+1)%len(c)])
	}
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	bodies := s.VisibleBodies()
	if len(bodies) == 0 {
		t.Fatalf("part %s produced no body", name)
	}
	return d, bodies[0]
}

// TestHiddenBodiesAreDocumentScoped: hiding a body in one document must not hide (or leak into)
// another document's bodies, and switching back restores the first document's hidden state (#1105).
func TestHiddenBodiesAreDocumentScoped(t *testing.T) {
	s := NewSession()
	docA, bodyA := addBoxPart(t, s, "A.obk")
	docB, bodyB := addBoxPart(t, s, "B.obk") // B is now active

	// Hide A's body while A is active.
	if err := s.Workspace().SetActiveDocument(docA); err != nil {
		t.Fatalf("activate A: %v", err)
	}
	s.SetBodyVisible(bodyA, false)
	if s.BodyVisible(bodyA) {
		t.Fatal("body A should be hidden after SetBodyVisible(false)")
	}

	// Switching to B must NOT carry A's hidden state — B's body stays visible.
	if err := s.Workspace().SetActiveDocument(docB); err != nil {
		t.Fatalf("activate B: %v", err)
	}
	if !s.BodyVisible(bodyB) {
		t.Error("body B is hidden after switching from A — hidden state leaked across documents")
	}

	// Switching back to A must restore A's hidden state (per-document, not cleared).
	if err := s.Workspace().SetActiveDocument(docA); err != nil {
		t.Fatalf("re-activate A: %v", err)
	}
	if s.BodyVisible(bodyA) {
		t.Error("body A became visible after a round-trip — its per-document hidden state was lost")
	}
}

// TestClosingDocumentReleasesHiddenBodies: a closed document's hidden-body set is dropped, so it
// cannot leak into a later document and does not accumulate (#1105).
func TestClosingDocumentReleasesHiddenBodies(t *testing.T) {
	s := NewSession()
	docA, bodyA := addBoxPart(t, s, "A.obk")
	if err := s.Workspace().SetActiveDocument(docA); err != nil {
		t.Fatalf("activate A: %v", err)
	}
	s.SetBodyVisible(bodyA, false)

	if err := s.Workspace().Close(docA, true); err != nil {
		t.Fatalf("close A: %v", err)
	}
	if _, ok := s.hiddenBodyKeysByDoc[docA.ID()]; ok {
		t.Error("closing document A left its hidden-body set behind (leak)")
	}
}

// TestSelectionClearsOnDocumentSwitch: the selection holds refs into the active document, so
// activating a different document must clear it rather than leak stale picks into the new view
// (#1105). Re-activating the same document leaves the selection intact.
func TestSelectionClearsOnDocumentSwitch(t *testing.T) {
	s := NewSession()
	docA, bodyA := addBoxPart(t, s, "A.obk")
	docB, _ := addBoxPart(t, s, "B.obk")

	if err := s.Workspace().SetActiveDocument(docA); err != nil {
		t.Fatalf("activate A: %v", err)
	}
	s.Selection().Add(BodyHandle{Body: bodyA})
	if s.Selection().Count() == 0 {
		t.Fatal("selection should hold body A")
	}

	// Re-activating the SAME document keeps the selection.
	if err := s.Workspace().SetActiveDocument(docA); err != nil {
		t.Fatalf("re-activate A: %v", err)
	}
	if s.Selection().Count() == 0 {
		t.Error("re-activating the same document cleared the selection")
	}

	// Switching to a DIFFERENT document clears the stale selection.
	if err := s.Workspace().SetActiveDocument(docB); err != nil {
		t.Fatalf("activate B: %v", err)
	}
	if s.Selection().Count() != 0 {
		t.Error("switching documents left document A's selection in place (stale picks leak)")
	}
}
