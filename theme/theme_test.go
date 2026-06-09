// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestDefaultsComplete guards the contract that a built-in defines every token — the
// head reads colors without nil checks, so a missing token would render the magenta
// fallback. Catches a token added to api/types without a default color.
func TestDefaultsComplete(t *testing.T) {
	for _, th := range Builtins() {
		for _, tok := range types.AllThemeTokens() {
			if _, ok := th.Palette()[tok]; !ok {
				t.Errorf("%s theme missing token %q", th.Name(), tok)
			}
		}
	}
}

func TestBuiltinsAreReadOnly(t *testing.T) {
	dark := DefaultDark()
	before := dark.Color(types.TokenChromeAccent)
	dark.SetColor(types.TokenChromeAccent, Rgba{R: 1}) // no-op on a built-in
	if dark.Color(types.TokenChromeAccent) != before {
		t.Error("SetColor mutated a built-in dark theme; built-ins must be read-only")
	}
}

func TestDuplicateIsIndependentSnapshot(t *testing.T) {
	dark := DefaultDark()
	custom := dark.Duplicate("My Dark")
	if custom.Kind() != KindCustom {
		t.Fatalf("Duplicate kind = %q, want custom", custom.Kind())
	}
	custom.SetColor(types.TokenChromeAccent, Rgba{R: 1, A: 1})
	if dark.Color(types.TokenChromeAccent) == custom.Color(types.TokenChromeAccent) {
		t.Error("editing the custom copy leaked into the source built-in")
	}
}

func TestLibrarySetActiveBumpsRevision(t *testing.T) {
	lib := NewLibrary(nil, "Dark")
	r0 := lib.Revision()
	lib.SetActive("Light")
	if lib.Active().Name() != "Light" {
		t.Fatalf("Active = %q, want Light", lib.Active().Name())
	}
	if lib.Revision() <= r0 {
		t.Errorf("revision did not advance on SetActive (%d -> %d)", r0, lib.Revision())
	}
	// Re-selecting the same theme must NOT bump (no visible change).
	r1 := lib.Revision()
	lib.SetActive("Light")
	if lib.Revision() != r1 {
		t.Errorf("revision advanced on no-op SetActive (%d -> %d)", r1, lib.Revision())
	}
}

func TestLibraryUnknownActiveFallsBackToDark(t *testing.T) {
	lib := NewLibrary(nil, "Nonexistent")
	if lib.Active().Name() != "Dark" {
		t.Errorf("Active = %q, want Dark fallback", lib.Active().Name())
	}
}

func TestLibraryDuplicateAndEdit(t *testing.T) {
	lib := NewLibrary(nil, "Dark")
	dup, err := lib.Duplicate("Dark", "My Dark")
	if err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	if lib.Active() != dup {
		t.Error("Duplicate did not activate the new custom theme")
	}
	r := lib.Revision()
	lib.EditActiveColor(types.TokenChromeAccent, Rgba{R: 1, A: 1})
	if lib.Revision() <= r {
		t.Error("EditActiveColor did not bump the revision")
	}
	if dup.Color(types.TokenChromeAccent) != (Rgba{R: 1, A: 1}) {
		t.Error("EditActiveColor did not recolor the active custom theme")
	}
}

func TestLibraryDuplicateRejectsDuplicateName(t *testing.T) {
	lib := NewLibrary(nil, "Dark")
	if _, err := lib.Duplicate("Dark", "Light"); err == nil {
		t.Error("Duplicate to an existing name should fail")
	}
}

func TestLibraryRemoveCustomReselectsDark(t *testing.T) {
	lib := NewLibrary(nil, "Dark")
	_, _ = lib.Duplicate("Dark", "My Dark") // becomes active
	if err := lib.Remove("My Dark"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if lib.Active().Name() != "Dark" {
		t.Errorf("after removing active custom, Active = %q, want Dark", lib.Active().Name())
	}
	if err := lib.Remove("Dark"); err == nil {
		t.Error("removing a built-in should fail")
	}
}
