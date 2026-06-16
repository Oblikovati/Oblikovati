// SPDX-License-Identifier: GPL-2.0-only

package colorscheme

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestNewRegistryDefaults checks the registry seeds the gallery with the first scheme active.
func TestNewRegistryDefaults(t *testing.T) {
	r := NewRegistry()
	if len(r.Schemes()) < 2 {
		t.Fatalf("expected a multi-entry gallery, got %d", len(r.Schemes()))
	}
	if r.ActiveName() != "Default" {
		t.Errorf("active = %q, want Default", r.ActiveName())
	}
	if r.BackgroundType() != types.GradientBackground {
		t.Errorf("background = %v, want Gradient (the Default scheme's)", r.BackgroundType())
	}
}

// TestSetActiveSwitchesSchemeAndBackground checks activation adopts the scheme's background.
func TestSetActiveSwitchesSchemeAndBackground(t *testing.T) {
	r := NewRegistry()
	if err := r.SetActive("High Contrast"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if r.ActiveName() != "High Contrast" {
		t.Errorf("active = %q, want High Contrast", r.ActiveName())
	}
	if r.Active().Screen != types.NewColor(0, 0, 0) {
		t.Errorf("High Contrast screen = %+v, want black", r.Active().Screen)
	}
	if r.BackgroundType() != types.OneColorBackground {
		t.Errorf("background = %v, want OneColor (High Contrast's)", r.BackgroundType())
	}
}

// TestSetActiveUnknownErrors checks an unknown name errors and names the offending value.
func TestSetActiveUnknownErrors(t *testing.T) {
	r := NewRegistry()
	err := r.SetActive("Nope")
	if err == nil {
		t.Fatal("expected an error for an unknown scheme")
	}
	if got := r.ActiveName(); got != "Default" {
		t.Errorf("active changed to %q on failure, want Default unchanged", got)
	}
}

// TestSetBackgroundTypeValidates checks an undefined background type is rejected.
func TestSetBackgroundTypeValidates(t *testing.T) {
	r := NewRegistry()
	if err := r.SetBackgroundType(types.ImageBackground); err != nil {
		t.Fatalf("SetBackgroundType(Image): %v", err)
	}
	if r.BackgroundType() != types.ImageBackground {
		t.Errorf("background = %v, want Image", r.BackgroundType())
	}
	if err := r.SetBackgroundType(types.BackgroundTypeEnum(0)); err == nil {
		t.Error("expected an error for an undefined background type")
	}
}
