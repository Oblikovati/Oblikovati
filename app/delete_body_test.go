// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// #2046: DeleteBodyDefinition and PartFeatures.AddDeleteBody were implemented and routed over the
// API, but a full-text search for DeleteBody across app/ and head/ui/ returned nothing — a
// multi-body part accumulated bodies the user could not remove.

// twoBodyPart returns a session whose active part holds two disjoint 1×1×1 blocks, each its own
// body (the operation:"new" shape a Delete Body is for).
func twoBodyPart(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "two-body.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	for _, originX := range []float64{0, 4} {
		sk := def.Sketches().Add(sketch.XYPlane())
		c0 := sk.Points().Add(math.P2(originX, 0))
		c1 := sk.Points().Add(math.P2(originX+1, 0))
		c2 := sk.Points().Add(math.P2(originX+1, 1))
		c3 := sk.Points().Add(math.P2(originX, 1))
		sk.Lines().Add(c0, c1)
		sk.Lines().Add(c1, c2)
		sk.Lines().Add(c2, c3)
		sk.Lines().Add(c3, c0)
		feature.NewExtrudeFeatures(def.Features()).
			AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 1 })
	}
	def.Recompute()
	if n := def.SurfaceBodies().Count(); n != 2 {
		t.Fatalf("two-body setup: %d bodies, want 2", n)
	}
	return s, def
}

func TestDeleteBodyRemovesOnlyThatBody(t *testing.T) {
	s, def := twoBodyPart(t)
	survivor := def.SurfaceBodies().Item(0).ReferenceKey()
	if err := s.DeleteBody(def.SurfaceBodies().Item(1)); err != nil {
		t.Fatalf("DeleteBody: %v", err)
	}
	if n := def.SurfaceBodies().Count(); n != 1 {
		t.Fatalf("part has %d bodies after Delete, want 1", n)
	}
	if got := def.SurfaceBodies().Item(0).ReferenceKey(); string(got) != string(survivor) {
		t.Error("Delete removed the wrong body")
	}
}

// The deletion is a recipe feature, not a destructive edit — so it lands in the browser and
// suppressing it brings the body back.
func TestDeleteBodyIsASuppressibleFeature(t *testing.T) {
	s, def := twoBodyPart(t)
	before := def.Features().Count()
	if err := s.DeleteBody(def.SurfaceBodies().Item(1)); err != nil {
		t.Fatalf("DeleteBody: %v", err)
	}
	if got := def.Features().Count(); got != before+1 {
		t.Fatalf("feature count %d after Delete Body, want %d — the deletion is not in the recipe", got, before+1)
	}
	pf := def.Features().Item(def.Features().Count() - 1)
	pf.SetSuppressed(true)
	def.Recompute()
	if n := def.SurfaceBodies().Count(); n != 2 {
		t.Errorf("suppressing Delete Body left %d bodies, want the deleted one back (2)", n)
	}
}

func TestDeleteBodyRejectsNilBody(t *testing.T) {
	s, _ := twoBodyPart(t)
	if err := s.DeleteBody(nil); err == nil {
		t.Error("DeleteBody(nil) should error rather than silently do nothing")
	}
}

// The browser body node carries the Delete action, which is the surface the issue asks for and
// the one the reference application uses.
func TestBrowserBodyNodeOffersDelete(t *testing.T) {
	s, def := twoBodyPart(t)
	menu := BrowserMenu(s, findNode(t, BuildBrowser(s), "body"))
	var invoke func(*Session) error
	for _, item := range menu {
		if item.Label == "Delete" {
			invoke = item.Invoke
		}
	}
	if invoke == nil {
		t.Fatalf("the body node's menu has no Delete: %+v", menu)
	}
	if err := invoke(s); err != nil {
		t.Fatalf("browser Delete: %v", err)
	}
	if n := def.SurfaceBodies().Count(); n != 1 {
		t.Errorf("part has %d bodies after the browser Delete, want 1", n)
	}
}
