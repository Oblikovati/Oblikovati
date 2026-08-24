// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/app/options"
)

// TestInputPrefAccessorsPersist: the mouse-navigation preferences read the
// persisted Input group and write back through the options store, so the
// Preferences window's combos survive a restart (I-doc §4.2).
func TestInputPrefAccessorsPersist(t *testing.T) {
	store := &FakeOptionsStore{stored: options.Defaults()}
	s := NewSession()
	if err := s.UseOptionsStore(store); err != nil {
		t.Fatalf("UseOptionsStore: %v", err)
	}
	if s.MMBMode() != "pan" || s.ShiftMMBMode() != "orbit" || s.CtrlMMBMode() != "pan" {
		t.Errorf("defaults = mmb %q shift %q ctrl %q, want pan/orbit/pan",
			s.MMBMode(), s.ShiftMMBMode(), s.CtrlMMBMode())
	}
	if s.WheelInvert() || !s.ZoomToCursor() {
		t.Errorf("wheel defaults = invert %v toCursor %v, want false/true", s.WheelInvert(), s.ZoomToCursor())
	}

	if err := s.SetMMBMode("orbit"); err != nil {
		t.Fatalf("SetMMBMode: %v", err)
	}
	if err := s.SetShiftMMBMode("pan"); err != nil {
		t.Fatalf("SetShiftMMBMode: %v", err)
	}
	if err := s.SetCtrlMMBMode("zoom"); err != nil {
		t.Fatalf("SetCtrlMMBMode: %v", err)
	}
	if err := s.SetWheelInvert(true); err != nil {
		t.Fatalf("SetWheelInvert: %v", err)
	}
	if err := s.SetZoomToCursor(false); err != nil {
		t.Fatalf("SetZoomToCursor: %v", err)
	}
	if store.saves != 5 {
		t.Errorf("saves = %d, want 5 (one per setter)", store.saves)
	}
	if got := store.stored.Input; got != (options.Input{
		MMBMode: "orbit", ShiftMMBMode: "pan", CtrlMMBMode: "zoom",
		WheelInvert: true, ZoomToCursor: false,
	}) {
		t.Errorf("stored input = %+v, want the five edited values", got)
	}
}
