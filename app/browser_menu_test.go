// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// extrudedBoxPart builds a part with one sketch profile extruded into one solid, so the
// browser has a sketch node, a feature node and a Solid Bodies folder to exercise.
func extrudedBoxPart(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-2, -2))
	c1 := sk.Points().Add(math.P2(2, -2))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(-2, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	return s, def
}

// findNode returns the first descendant (depth-first) with the given kind, or fails.
func findNode(t *testing.T, n BrowserNode, kind string) BrowserNode {
	t.Helper()
	if got, ok := searchNode(n, kind); ok {
		return got
	}
	t.Fatalf("no browser node of kind %q under %q", kind, n.Label)
	return BrowserNode{}
}

func searchNode(n BrowserNode, kind string) (BrowserNode, bool) {
	for _, c := range n.Children {
		if c.Kind == kind {
			return c, true
		}
		if got, ok := searchNode(c, kind); ok {
			return got, true
		}
	}
	return BrowserNode{}, false
}

func TestBuildBrowserMakesSketchFeatureBodySelectable(t *testing.T) {
	s, _ := extrudedBoxPart(t)
	root := BuildBrowser(s)

	if _, ok := findNode(t, root, "sketch").Select.(SketchHandle); !ok {
		t.Error("sketch node is not selectable with a SketchHandle")
	}
	if _, ok := findNode(t, root, "feature").Select.(FeatureHandle); !ok {
		t.Error("feature node is not selectable with a FeatureHandle")
	}
	body := findNode(t, root, "body")
	if _, ok := body.Select.(BodyHandle); !ok {
		t.Errorf("body node is not selectable with a BodyHandle (label %q)", body.Label)
	}
	if body.Label != "Solid1" {
		t.Errorf("first body label = %q, want Solid1", body.Label)
	}
}

func TestBrowserMenuByKind(t *testing.T) {
	s, _ := extrudedBoxPart(t)
	root := BuildBrowser(s)

	if labels := menuLabels(BrowserMenu(findNode(t, root, "sketch"))); !equalStrings(labels, []string{"Edit Sketch", "Visibility", "Delete"}) {
		t.Errorf("sketch menu = %v", labels)
	}
	if labels := menuLabels(BrowserMenu(findNode(t, root, "feature"))); !equalStrings(labels, []string{"Suppress", "Delete"}) {
		t.Errorf("feature menu = %v", labels)
	}
	if labels := menuLabels(BrowserMenu(findNode(t, root, "workplane"))); !equalStrings(labels, []string{"New Sketch", "Visibility"}) {
		t.Errorf("workplane menu = %v", labels)
	}
	if m := BrowserMenu(findNode(t, root, "parameters")); m != nil {
		t.Errorf("parameters folder should have no menu, got %v", menuLabels(m))
	}
}

func TestBrowserMenuDeleteSketchRemovesIt(t *testing.T) {
	s, def := extrudedBoxPart(t)
	node := findNode(t, BuildBrowser(s), "sketch")
	del := menuItem(t, BrowserMenu(node), "Delete")
	if err := del.Invoke(s); err != nil {
		t.Fatalf("Delete sketch: %v", err)
	}
	if def.Sketches().Count() != 0 {
		t.Errorf("sketch count after Delete = %d, want 0", def.Sketches().Count())
	}
}

func TestBrowserMenuDeleteFeatureRemovesAndRecomputes(t *testing.T) {
	s, def := extrudedBoxPart(t)
	node := findNode(t, BuildBrowser(s), "feature")
	del := menuItem(t, BrowserMenu(node), "Delete")
	if err := del.Invoke(s); err != nil {
		t.Fatalf("Delete feature: %v", err)
	}
	if def.Features().Count() != 0 {
		t.Errorf("feature count after Delete = %d, want 0", def.Features().Count())
	}
	if len(def.SurfaceBodies().All()) != 0 {
		t.Errorf("body remains after deleting its only feature: %d bodies", len(def.SurfaceBodies().All()))
	}
}

func TestBrowserMenuSuppressTogglesLabel(t *testing.T) {
	s, _ := extrudedBoxPart(t)
	node := findNode(t, BuildBrowser(s), "feature")
	if err := menuItem(t, BrowserMenu(node), "Suppress").Invoke(s); err != nil {
		t.Fatalf("Suppress: %v", err)
	}
	// Rebuild: the same feature node should now offer Unsuppress.
	node = findNode(t, BuildBrowser(s), "feature")
	if _, ok := findMenuItem(BrowserMenu(node), "Unsuppress"); !ok {
		t.Errorf("after Suppress, menu = %v, want an Unsuppress entry", menuLabels(BrowserMenu(node)))
	}
}

func TestBrowserMenuEditSketchEntersIt(t *testing.T) {
	s, def := extrudedBoxPart(t)
	node := findNode(t, BuildBrowser(s), "sketch")
	if err := menuItem(t, BrowserMenu(node), "Edit Sketch").Invoke(s); err != nil {
		t.Fatalf("Edit Sketch: %v", err)
	}
	if s.ActiveSketch() != def.Sketches().Item(0) {
		t.Error("Edit Sketch did not enter the sketch")
	}
}

func menuLabels(items []BrowserMenuItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Label
	}
	return out
}

func findMenuItem(items []BrowserMenuItem, label string) (BrowserMenuItem, bool) {
	for _, it := range items {
		if it.Label == label {
			return it, true
		}
	}
	return BrowserMenuItem{}, false
}

func menuItem(t *testing.T, items []BrowserMenuItem, label string) BrowserMenuItem {
	t.Helper()
	if it, ok := findMenuItem(items, label); ok {
		return it
	}
	t.Fatalf("menu has no %q item; labels = %v", label, menuLabels(items))
	return BrowserMenuItem{}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
