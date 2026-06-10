// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// childrenKinds lists the kinds of a node's direct children, for asserting tree shape.
func childrenKinds(n BrowserNode) []string {
	kinds := make([]string, len(n.Children))
	for i, c := range n.Children {
		kinds[i] = c.Kind
	}
	return kinds
}

// topLevelKind reports whether the part root has a direct child of the given kind.
func containsLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}

func hasTopLevelKind(root BrowserNode, kind string) bool {
	for _, c := range root.Children {
		if c.Kind == kind {
			return true
		}
	}
	return false
}

// TestBrowserNestsConsumedSketchUnderFeature is the issue-#132 core: there is no top-level
// Sketches branch; a sketch consumed by one feature is nested under that feature instead.
func TestBrowserNestsConsumedSketchUnderFeature(t *testing.T) {
	s, _ := extrudedBoxPart(t)
	root := BuildBrowser(s)

	// No "sketches" folder remains at the top level.
	if hasTopLevelKind(root, "sketches") {
		t.Error("the special Sketches branch must be gone")
	}
	// The sketch must NOT be a direct child of the root (it is absorbed).
	if hasTopLevelKind(root, "sketch") {
		t.Error("a consumed sketch must not appear at the top level")
	}
	// The feature node carries the sketch as its child.
	feat := findNode(t, root, "feature")
	if kinds := childrenKinds(feat); len(kinds) != 1 || kinds[0] != "sketch" {
		t.Errorf("feature children = %v, want one nested sketch", kinds)
	}
	if _, ok := findNode(t, feat, "sketch").Select.(SketchHandle); !ok {
		t.Error("nested sketch is not selectable with a SketchHandle")
	}
}

// TestBrowserKeepsSharedSketchAtTopLevel: a Shared sketch stays at the top level even though a
// feature consumes it (Inventor's Share Sketch), so it can feed several features.
func TestBrowserKeepsSharedSketchAtTopLevel(t *testing.T) {
	s, def := extrudedBoxPart(t)
	def.Sketches().Item(0).SetShared(true)
	root := BuildBrowser(s)

	if !hasTopLevelKind(root, "sketch") {
		t.Error("a shared sketch must remain at the top level")
	}
	if kinds := childrenKinds(findNode(t, root, "feature")); len(kinds) != 0 {
		t.Errorf("feature must not absorb a shared sketch; children = %v", kinds)
	}
}

// TestBrowserOrdersTimelineByCreation: a work plane created after the extrude, and a later
// sketch on it, appear in creation order — proving the chronological interleave.
func TestBrowserOrdersTimelineByCreation(t *testing.T) {
	s, def := extrudedBoxPart(t) // sketch1 → extrude
	def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Sketches().Add(sketch.XYPlane()) // a later, unconsumed sketch
	def.Recompute()

	root := BuildBrowser(s)
	order := timelineKinds(root)
	// Expected top-level timeline order: the absorbing feature, then the work plane, then the
	// trailing unconsumed sketch (the consumed sketch is nested, not in this list).
	want := []string{"feature", "workplane", "sketch"}
	if !equalStrings(order, want) {
		t.Errorf("timeline order = %v, want %v", order, want)
	}
}

// timelineKinds lists the kinds of the root's direct children that belong to the model
// timeline (excluding the static Origin/Parameters/Solid Bodies folders).
func timelineKinds(root BrowserNode) []string {
	var out []string
	for _, c := range root.Children {
		switch c.Kind {
		case "origin", "parameters", "bodies":
			continue
		default:
			out = append(out, c.Kind)
		}
	}
	return out
}

// TestToggleSketchSharedRoundTripsThroughMenu drives the Share/Unshare action and checks the
// menu label and browser placement flip accordingly.
func TestToggleSketchSharedRoundTripsThroughMenu(t *testing.T) {
	s, def := extrudedBoxPart(t)
	sk := def.Sketches().Item(0)

	if err := s.ToggleSketchShared(sk); err != nil {
		t.Fatalf("ToggleSketchShared: %v", err)
	}
	if !sk.Shared() {
		t.Fatal("sketch should be shared after toggling")
	}
	root := BuildBrowser(s)
	if !hasTopLevelKind(root, "sketch") {
		t.Error("shared sketch should be top level after toggle")
	}
	labels := menuLabels(BrowserMenu(findNode(t, root, "sketch")))
	if !containsLabel(labels, "Unshare Sketch") {
		t.Errorf("shared sketch menu = %v, want an Unshare Sketch entry", labels)
	}

	// Unshare → the sketch is absorbed again.
	if err := s.ToggleSketchShared(sk); err != nil {
		t.Fatalf("ToggleSketchShared(unshare): %v", err)
	}
	if hasTopLevelKind(BuildBrowser(s), "sketch") {
		t.Error("unshared consumed sketch should nest under its feature again")
	}
}
