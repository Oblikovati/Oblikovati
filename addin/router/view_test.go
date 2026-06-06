// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strconv"
	"testing"

	"oblikovati/api/types"
	"oblikovati/api/wire"
)

// TestViewDisplayModeRoundTrips drives view.setDisplayMode then view.getDisplayMode for every
// DisplayModeEnum member and asserts the mode + label survive the round trip — the dogfood
// proof that the public display-mode contract reaches the session and back.
func TestViewDisplayModeRoundTrips(t *testing.T) {
	r, s := seededSession(t)
	for _, m := range types.AllDisplayModes() {
		var set wire.DisplayModeView
		call(t, r, s, "view.setDisplayMode", `{"mode":`+strconv.Itoa(int(m))+`}`, &set)
		if set.Mode != m || set.Name != m.String() {
			t.Errorf("setDisplayMode(%d) = (%d,%q), want (%d,%q)", m, set.Mode, set.Name, m, m.String())
		}
		var got wire.DisplayModeView
		call(t, r, s, "view.getDisplayMode", "{}", &got)
		if got.Mode != m {
			t.Errorf("after set %s, getDisplayMode = %d, want %d", m, got.Mode, m)
		}
	}
}

// TestViewSetDisplayModeRejectsUnknown checks an undefined mode id is an error, not a silent
// fallback.
func TestViewSetDisplayModeRejectsUnknown(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "view.setDisplayMode", []byte(`{"mode":1}`)); err == nil {
		t.Fatal("expected error for unknown display mode id 1")
	}
}

// TestViewListDisplayModesFlagsActive checks the enumeration returns all 11 modes with exactly
// the active one flagged.
func TestViewListDisplayModesFlagsActive(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "view.setDisplayMode", `{"mode":`+strconv.Itoa(int(types.RealisticRendering))+`}`, nil)
	var list wire.ListDisplayModesResult
	call(t, r, s, "view.listDisplayModes", "{}", &list)
	if len(list.Modes) != 11 {
		t.Fatalf("listDisplayModes = %d modes, want 11", len(list.Modes))
	}
	active := 0
	for _, m := range list.Modes {
		if m.Active {
			active++
			if m.Mode != types.RealisticRendering {
				t.Errorf("active mode = %s, want Realistic", m.Mode)
			}
		}
	}
	if active != 1 {
		t.Errorf("active count = %d, want exactly 1", active)
	}
}
