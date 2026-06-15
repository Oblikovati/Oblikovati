// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/topo"
)

// TestWorkSurfacesSyncGathersSheetBodies keeps only the open (non-solid) bodies of a
// result and auto-names them in order (M20-F16).
func TestWorkSurfacesSyncGathersSheetBodies(t *testing.T) {
	c := NewWorkSurfaces()
	c.Sync([]*topo.Body{prismBody(), patchSurfaceBody(2, 3), prismBody(), patchSurfaceBody(1, 1)})

	if c.Count() != 2 {
		t.Fatalf("work surface count = %d, want 2 (only the sheet bodies)", c.Count())
	}
	if c.Item(0).Name() != "Surface1" || c.Item(1).Name() != "Surface2" {
		t.Errorf("names = %q,%q, want Surface1,Surface2", c.Item(0).Name(), c.Item(1).Name())
	}
	if !c.Item(0).Visible() || c.Item(0).Translucent() {
		t.Errorf("new surface defaults = visible %v translucent %v, want true/false", c.Item(0).Visible(), c.Item(0).Translucent())
	}
}

// TestWorkSurfacesSyncPreservesDisplayState keeps a renamed/hidden surface's state across
// a resync that rebuilds the underlying body objects, and drops a surface that disappears.
func TestWorkSurfacesSyncPreservesDisplayState(t *testing.T) {
	c := NewWorkSurfaces()
	c.Sync([]*topo.Body{patchSurfaceBody(2, 3), patchSurfaceBody(1, 1)})
	if err := c.Item(0).SetName("Parting"); err != nil {
		t.Fatalf("SetName: %v", err)
	}
	c.Item(0).SetVisible(false)
	c.Item(0).SetTranslucent(true)

	// A fresh recompute hands back new body objects for the same surfaces.
	fresh := patchSurfaceBody(2, 3)
	c.Sync([]*topo.Body{fresh, patchSurfaceBody(1, 1)})
	if got := c.Item(0); got.Name() != "Parting" || got.Visible() || !got.Translucent() {
		t.Errorf("display state lost on resync: name %q visible %v translucent %v", got.Name(), got.Visible(), got.Translucent())
	}
	if c.Item(0).Body() != fresh {
		t.Error("surface body not refreshed to the new object on resync")
	}

	// Dropping the second surface trims the collection.
	c.Sync([]*topo.Body{patchSurfaceBody(2, 3)})
	if c.Count() != 1 {
		t.Errorf("count after a surface disappears = %d, want 1", c.Count())
	}
}

// TestWorkSurfacesRenameGuards rejects an empty name and reports a duplicate (excluding self).
func TestWorkSurfacesRenameGuards(t *testing.T) {
	c := NewWorkSurfaces()
	c.Sync([]*topo.Body{patchSurfaceBody(2, 3), patchSurfaceBody(1, 1)})
	if err := c.Item(0).SetName(""); err == nil {
		t.Error("empty name must be rejected")
	}
	if c.HasName("Surface1", 0) {
		t.Error("HasName must exclude the surface at the excluded index")
	}
	if !c.HasName("Surface2", 0) {
		t.Error("HasName must report a name used by another surface")
	}
}

// TestWorkSurfacesItemOutOfRange returns nil rather than panicking.
func TestWorkSurfacesItemOutOfRange(t *testing.T) {
	c := NewWorkSurfaces()
	if c.Item(0) != nil || c.Item(-1) != nil {
		t.Error("Item out of range must be nil")
	}
}
