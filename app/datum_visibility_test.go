// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/feature"
)

// The origin Center Point could not be shown at all: its browser node had no menu (it is not
// renameable either, so right-clicking it produced nothing), and the V shortcut reached only
// work planes. Nothing drew it, and yet it was clickable. These lock the whole path (#2016).

// originCenterOf returns the active document's origin centre point.
func originCenterOf(t *testing.T, s *Session) *feature.WorkPoint {
	t.Helper()
	wg, ok := s.ActiveWorkGeometry()
	if !ok {
		t.Fatal("no active work geometry")
	}
	c, ok := wg.WorkPointByRef(feature.OriginCenter)
	if !ok {
		t.Fatal("no origin centre point")
	}
	return c
}

// browserNodeOfKind finds the first node of a kind in the tree.
func browserNodeOfKind(n BrowserNode, kind string) (BrowserNode, bool) {
	if n.Kind == kind {
		return n, true
	}
	for _, c := range n.Children {
		if got, ok := browserNodeOfKind(c, kind); ok {
			return got, true
		}
	}
	return BrowserNode{}, false
}

func TestCenterPointBrowserMenuTogglesVisibility(t *testing.T) {
	s := newSessionWithPart(t)
	node, ok := browserNodeOfKind(BuildBrowser(s), "workpoint")
	if !ok {
		t.Fatal("the browser has no work-point node")
	}

	items := BrowserMenu(s, node)
	if len(items) == 0 {
		t.Fatal("the Center Point's menu is empty; a work plane and axis both offer Visibility")
	}
	visibility, ok := findMenuItem(items, "Visibility")
	if !ok {
		t.Fatalf("menu = %v, want a Visibility item", menuLabels(items))
	}

	center := originCenterOf(t, s)
	if center.Visible() {
		t.Fatal("precondition: the origin centre starts hidden")
	}
	if err := visibility.Invoke(s); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if !center.Visible() {
		t.Error("the Visibility item did not show the Center Point")
	}
	if err := visibility.Invoke(s); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if center.Visible() {
		t.Error("the Visibility item does not toggle back off")
	}
}

// The V shortcut reached work planes only, so pressing it on a selected axis or point did
// nothing. It now flips whichever datums are selected.
func TestToggleVisibilityShortcutReachesEveryDatum(t *testing.T) {
	s := newSessionWithPart(t)
	wg, _ := s.ActiveWorkGeometry()
	center := originCenterOf(t, s)
	axis := wg.WorkAxes().Item(0)
	plane := wg.OriginPlanes()[0]

	// Select replaces the selection, so the other two go in the way a user adds them: Shift+click.
	s.Select(WorkPointHandle{Point: center})
	s.applyPickToSelection(WorkAxisHandle{Axis: axis}, ShiftMod)
	s.applyPickToSelection(WorkPlaneHandle{Plane: plane}, ShiftMod)
	before := []bool{center.Visible(), axis.Visible(), plane.Visible()}

	if err := dispatchToggleVisibility(s); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	for i, got := range []bool{center.Visible(), axis.Visible(), plane.Visible()} {
		if got == before[i] {
			t.Errorf("datum %d visibility unchanged (%v); V must flip every selected datum", i, got)
		}
	}
}

// A datum nothing draws must not be clickable — the rule planes and axes already follow.
func TestHiddenCenterPointIsNotPickable(t *testing.T) {
	s := newSessionWithPart(t)
	center := originCenterOf(t, s)

	if got := len(s.PickableWorkPoints()); got != 0 {
		t.Errorf("PickableWorkPoints = %d while the Center Point is hidden, want 0", got)
	}
	center.SetVisible(true)
	if got := len(s.PickableWorkPoints()); got != 1 {
		t.Errorf("PickableWorkPoints = %d once shown, want 1", got)
	}
}

// ToggleSelectedDatumVisibility reports how many datums it changed, so an empty selection is
// distinguishable from a real toggle.
func TestToggleSelectedDatumVisibilityCountsWhatItChanged(t *testing.T) {
	s := newSessionWithPart(t)
	if got := s.ToggleSelectedDatumVisibility(); got != 0 {
		t.Errorf("toggled %d datums with nothing selected, want 0", got)
	}
	s.Select(WorkPointHandle{Point: originCenterOf(t, s)})
	if got := s.ToggleSelectedDatumVisibility(); got != 1 {
		t.Errorf("toggled %d datums, want 1", got)
	}
}
