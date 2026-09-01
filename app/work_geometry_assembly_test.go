// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// activeAssemblySession returns a session with a fresh, empty assembly active — its origin frame
// (origin planes/axes/point) is seeded on creation, the fixture for the assembly work-geometry tests.
func activeAssemblySession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(asmDoc); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	return s
}

// TestActiveWorkGeometryResolvesAssembly: the active-work-geometry accessor resolves an assembly's
// origin frame, not only a part's — both own a feature.WorkGeometry, so datum rendering and picking
// can source from it model-agnostically.
func TestActiveWorkGeometryResolvesAssembly(t *testing.T) {
	t.Parallel()
	s := activeAssemblySession(t)
	wg, ok := s.ActiveWorkGeometry()
	if !ok || wg == nil {
		t.Fatalf("ActiveWorkGeometry(assembly) = (%v, %v), want the assembly's seeded work geometry", wg, ok)
	}
	if got := wg.WorkPlanes().Count(); got < 3 {
		t.Errorf("assembly origin planes = %d, want >= 3 (XY/XZ/YZ)", got)
	}
}

// TestPickableWorkPlanesFollowsVisibilityInAssembly: an assembly's origin planes, like a part's,
// are hidden by default (issue #1520) and become mouse-pickable only once shown — and the assembly
// path is no longer part-gated (PickableWorkPlanes returned nil for an assembly before). A hidden
// origin plane is reachable solely through its browser node, never a viewport click.
func TestPickableWorkPlanesFollowsVisibilityInAssembly(t *testing.T) {
	t.Parallel()
	s := activeAssemblySession(t)
	if got := len(s.PickableWorkPlanes()); got != 0 {
		t.Fatalf("assembly PickableWorkPlanes (default) = %d, want 0 (origin planes hidden until shown)", got)
	}
	wg, _ := s.ActiveWorkGeometry()
	wg.WorkPlanes().Item(0).SetVisible(true)
	if got := len(s.PickableWorkPlanes()); got != 1 {
		t.Errorf("a shown assembly origin plane should be pickable: got %d, want 1", got)
	}
}

// TestPickableWorkAxesFollowsVisibilityInAssembly: an assembly's origin axes (X/Y/Z), like a
// part's, are hidden by default (the Origin folder) and become pickable AND drawn once made
// visible. Before, an assembly's axes were never offered (the source was part-gated); now toggling
// them on shows them in the assembly exactly as in a part.
func TestPickableWorkAxesFollowsVisibilityInAssembly(t *testing.T) {
	t.Parallel()
	s := activeAssemblySession(t)
	if got := len(s.PickableWorkAxes()); got != 0 {
		t.Fatalf("assembly PickableWorkAxes (default) = %d, want 0 (origin axes hidden until shown, as in a part)", got)
	}
	wg, _ := s.ActiveWorkGeometry()
	axes := wg.WorkAxes()
	for i := 0; i < axes.Count(); i++ {
		axes.Item(i).SetVisible(true) // the user shows the Origin folder's axes
	}
	if got := len(s.PickableWorkAxes()); got != 3 {
		t.Errorf("assembly PickableWorkAxes (shown) = %d, want 3 origin axes (X/Y/Z)", got)
	}
}

// TestPickableWorkPointsIncludesAssemblyOrigin: an assembly's origin centre point is a snap target.
func TestPickableWorkPointsIncludesAssemblyOrigin(t *testing.T) {
	t.Parallel()
	s := activeAssemblySession(t)
	// The origin centre starts hidden, and what the overlays do not draw must not be clickable —
	// otherwise a click near the origin snaps to a point nothing shows (#2016). Showing it is
	// what makes it a pick target, so the assembly's origin point is reachable either way.
	if got := len(s.PickableWorkPoints()); got != 0 {
		t.Errorf("assembly PickableWorkPoints = %d while the origin centre is hidden, want 0", got)
	}
	wg, ok := s.ActiveWorkGeometry()
	if !ok {
		t.Fatal("assembly has no work geometry")
	}
	center, ok := wg.WorkPointByRef(feature.OriginCenter)
	if !ok {
		t.Fatal("assembly has no origin centre point")
	}
	center.SetVisible(true)
	if got := len(s.PickableWorkPoints()); got != 1 {
		t.Errorf("assembly PickableWorkPoints = %d once shown, want 1 origin centre point", got)
	}
}

// TestAssemblyBrowserHasOriginFolder: the assembly browser shows an Origin folder with the seven
// coordinate-system elements (3 planes + 3 axes + 1 point), like a part's — before, an assembly's
// tree had Parameters and occurrences but no Origin folder at all.
func TestAssemblyBrowserHasOriginFolder(t *testing.T) {
	t.Parallel()
	s := activeAssemblySession(t)
	root := BuildBrowser(s)
	if !hasTopLevelKind(root, "origin") {
		t.Fatal("assembly browser has no Origin folder; a part's does")
	}
	var origin BrowserNode
	for _, c := range root.Children {
		if c.Kind == "origin" {
			origin = c
		}
	}
	if got := len(origin.Children); got != 7 {
		t.Errorf("assembly Origin folder has %d entries, want 7 (3 planes + 3 axes + 1 point)", got)
	}
}
