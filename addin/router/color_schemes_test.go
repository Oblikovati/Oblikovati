// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestColorSchemesListAndActivate drives list -> getActive -> setActive, checking the active
// flag tracks the switch and the background type follows the activated scheme.
func TestColorSchemesListAndActivate(t *testing.T) {
	r, s := seededSession(t)

	var list wire.ColorSchemesResult
	call(t, r, s, "colorSchemes.list", `{}`, &list)
	if len(list.Schemes) < 2 {
		t.Fatalf("expected a multi-entry gallery, got %d", len(list.Schemes))
	}
	if !exactlyOneActive(list.Schemes) {
		t.Error("exactly one scheme should be flagged active")
	}

	var active wire.ColorSchemeView
	call(t, r, s, "colorSchemes.getActive", `{}`, &active)
	if active.Name != "Default" {
		t.Errorf("default active = %q, want Default", active.Name)
	}

	var now wire.ColorSchemeView
	call(t, r, s, "colorSchemes.setActive", mustJSON(t, wire.SetColorSchemeArgs{Name: "High Contrast"}), &now)
	if now.Name != "High Contrast" || !now.Active {
		t.Errorf("activated = %+v, want High Contrast active", now)
	}
	if now.BackgroundType != types.OneColorBackground {
		t.Errorf("background = %v, want OneColor (High Contrast's)", now.BackgroundType)
	}
	if now.ScreenColor != types.NewColor(0, 0, 0) {
		t.Errorf("screen = %+v, want black", now.ScreenColor)
	}
}

// TestColorSchemesSetActiveUnknownErrors checks an unknown scheme name is rejected.
func TestColorSchemesSetActiveUnknownErrors(t *testing.T) {
	r, s := seededSession(t)
	if err := tryCall(t, r, s, "colorSchemes.setActive", `{"name":"Nope"}`); err == nil {
		t.Error("activating an unknown scheme should error")
	}
}

// exactlyOneActive reports whether exactly one scheme carries the active flag.
func exactlyOneActive(schemes []wire.ColorSchemeView) bool {
	n := 0
	for _, s := range schemes {
		if s.Active {
			n++
		}
	}
	return n == 1
}
