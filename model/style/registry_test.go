// SPDX-License-Identifier: GPL-2.0-only

package style

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestNewRegistrySeedsBuiltins checks the registry starts with the local built-in styles.
func TestNewRegistrySeedsBuiltins(t *testing.T) {
	m := NewRegistry()
	if _, ok := m.ByName("Default"); !ok {
		t.Error("expected a built-in Default style")
	}
	if len(m.Styles()) < 3 {
		t.Errorf("built-in gallery = %d styles, want >= 3", len(m.Styles()))
	}
}

// TestSetAddsThenUpdates checks Set adds a new style then updates it in place.
func TestSetAddsThenUpdates(t *testing.T) {
	m := NewRegistry()
	added, err := m.Set(ColorStyle{Name: "Glass", Opacity: 0.3, Location: types.LocalStyleLocation})
	if err != nil || !added {
		t.Fatalf("Set new = (added %v, err %v), want (true, nil)", added, err)
	}
	added, err = m.Set(ColorStyle{Name: "Glass", Opacity: 0.5, Location: types.LocalStyleLocation})
	if err != nil || added {
		t.Fatalf("Set existing = (added %v, err %v), want (false, nil)", added, err)
	}
	got, _ := m.ByName("Glass")
	if got.Opacity != 0.5 {
		t.Errorf("Glass opacity = %v, want 0.5 after update", got.Opacity)
	}
}

// TestSetBlankNameErrors rejects an unaddressable style.
func TestSetBlankNameErrors(t *testing.T) {
	if _, err := NewRegistry().Set(ColorStyle{}); err == nil {
		t.Error("a blank style name should error")
	}
}

// TestDeleteRemovesOrErrors checks deletion and the missing-name error.
func TestDeleteRemovesOrErrors(t *testing.T) {
	m := NewRegistry()
	if err := m.Delete("Steel"); err != nil {
		t.Fatalf("Delete Steel: %v", err)
	}
	if _, ok := m.ByName("Steel"); ok {
		t.Error("Steel should be gone after Delete")
	}
	if err := m.Delete("Steel"); err == nil {
		t.Error("deleting an absent style should error")
	}
}

// TestImportMergesLibraryStylesWithoutShadowingLocal checks a library style merges as
// Library-located, but a same-named local style wins the cascade (is not overwritten).
func TestImportMergesLibraryStylesWithoutShadowingLocal(t *testing.T) {
	m := NewRegistry()
	added := m.Import(Library{Name: "Metals", Path: "/x/metals.obkstyle", Styles: []ColorStyle{
		{Name: "Titanium", Opacity: 1},
		{Name: "Default", Opacity: 0.1}, // collides with the local built-in
	}})
	if len(added) != 1 || added[0] != "Titanium" {
		t.Fatalf("Import added = %v, want [Titanium] (Default shadowed by local)", added)
	}
	tit, _ := m.ByName("Titanium")
	if tit.Location != types.LibraryStyleLocation {
		t.Errorf("Titanium location = %v, want Library", tit.Location)
	}
	def, _ := m.ByName("Default")
	if def.Location != types.LocalStyleLocation {
		t.Errorf("Default location = %v, want Local (not shadowed)", def.Location)
	}
	if len(m.Libraries()) != 1 {
		t.Errorf("libraries = %d, want 1", len(m.Libraries()))
	}
}
