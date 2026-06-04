// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "testing"

// TestEnvironmentGalleryIsTotal asserts every preset in the gallery has a name (no
// "environment(?)" leaks) and that None is first, so the picker always offers "off".
func TestEnvironmentGalleryIsTotal(t *testing.T) {
	g := EnvironmentGallery()
	if len(g) == 0 || g[0].Preset != EnvNone {
		t.Fatalf("gallery must lead with EnvNone, got %+v", g)
	}
	for _, opt := range g {
		if !opt.Preset.IsValid() {
			t.Errorf("preset %v in gallery is not valid", opt.Preset)
		}
		if opt.Name != opt.Preset.String() {
			t.Errorf("gallery name %q != preset.String() %q", opt.Name, opt.Preset.String())
		}
	}
}

// TestEnvironmentIsActive pins the rule that drives the IBL/skybox path: a file or any
// non-None preset is active; the zero value (EnvNone, no file) is not (ADR-0026 §5,§7).
func TestEnvironmentIsActive(t *testing.T) {
	cases := []struct {
		env  Environment
		want bool
	}{
		{Environment{}, false},
		{Environment{Preset: EnvNone}, false},
		{Environment{Preset: EnvStudio}, true},
		{Environment{FilePath: "/tmp/sky.hdr"}, true},
		{Environment{Preset: EnvNone, FilePath: "/tmp/sky.hdr"}, true},
	}
	for _, c := range cases {
		if got := c.env.IsActive(); got != c.want {
			t.Errorf("IsActive(%+v) = %v, want %v", c.env, got, c.want)
		}
	}
}

// TestEnvironmentStringPrefersFile asserts a file environment labels as "File" regardless of
// the preset slot, while a preset-only environment uses the preset name.
func TestEnvironmentStringPrefersFile(t *testing.T) {
	if got := (Environment{Preset: EnvStudio, FilePath: "/x.hdr"}).String(); got != "File" {
		t.Errorf("file environment String() = %q, want \"File\"", got)
	}
	if got := (Environment{Preset: EnvOutdoors}).String(); got != "Outdoors" {
		t.Errorf("preset environment String() = %q, want \"Outdoors\"", got)
	}
}
