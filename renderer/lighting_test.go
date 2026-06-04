// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "testing"

// TestSceneLightingForIsTotal asserts every style in the gallery resolves to a deliberate
// table row, so adding a LightingStyleID without a builder row is caught — the resolver must
// never fall through to the default for a known id (ADR-0026 §1, mirrors PassSetForIsTotal).
func TestSceneLightingForIsTotal(t *testing.T) {
	for _, opt := range LightingStyleGallery() {
		found := false
		for _, sp := range lightingStyleTable {
			if sp.id == opt.Style {
				found = true
			}
		}
		if !found {
			t.Errorf("lighting style %v (%q) has no resolver row", opt.Style, opt.Name)
		}
	}
}

// TestDefaultReproducesHeadlight pins the Default rig to the pre-IBL hardcoded shader values
// (ADR-0026 §7), so a refactor that changes the un-configured Realistic look is caught.
func TestDefaultReproducesHeadlight(t *testing.T) {
	d := DefaultSceneLighting()
	if len(d.Lights) != 1 {
		t.Fatalf("default has %d lights, want 1 headlight", len(d.Lights))
	}
	l := d.Lights[0]
	if l.Kind != DirectionalLight || !l.On {
		t.Errorf("default light = {kind:%v on:%v}, want directional+on", l.Kind, l.On)
	}
	if l.Direction != [3]float32{0.4, 0.6, 0.8} || l.Intensity != 3 {
		t.Errorf("default light dir/intensity = %v/%g, want {0.4 0.6 0.8}/3", l.Direction, l.Intensity)
	}
	if d.Ambience != 0.18 {
		t.Errorf("default ambience = %g, want 0.18", d.Ambience)
	}
	if d.Environment.IsActive() {
		t.Errorf("default must have no active environment, got %v", d.Environment)
	}
}

// TestSceneLightingForUnknownFallsBackToDefault asserts a corrupt id yields the Default rig,
// not an empty (black) scene (ADR-0026 §1).
func TestSceneLightingForUnknownFallsBackToDefault(t *testing.T) {
	got := SceneLightingFor(LightingStyleID(250))
	if len(got.Lights) != 1 || got.Ambience != 0.18 {
		t.Errorf("unknown style did not fall back to Default: %+v", got)
	}
}

// TestActiveLightsClampsToMax asserts ActiveLights drops Off lights and truncates to the UBO
// array bound, so a style cannot overflow the GPU light buffer (ADR-0026 §1).
func TestActiveLightsClampsToMax(t *testing.T) {
	var s SceneLighting
	for i := 0; i < MaxSceneLights+3; i++ {
		s.Lights = append(s.Lights, SceneLight{Kind: DirectionalLight, On: true})
	}
	s.Lights = append(s.Lights, SceneLight{Kind: PointLight, On: false}) // must be dropped
	got := s.ActiveLights()
	if len(got) != MaxSceneLights {
		t.Errorf("ActiveLights() len = %d, want %d", len(got), MaxSceneLights)
	}
	for _, l := range got {
		if !l.On {
			t.Errorf("ActiveLights() returned an Off light: %+v", l)
		}
	}
}

// TestLightingStyleGalleryMatchesTable asserts the gallery and the table stay in sync (same
// length, same order) — the gallery is the table's public view.
func TestLightingStyleGalleryMatchesTable(t *testing.T) {
	g := LightingStyleGallery()
	if len(g) != len(lightingStyleTable) {
		t.Fatalf("gallery has %d entries, table has %d", len(g), len(lightingStyleTable))
	}
	for i, opt := range g {
		if opt.Style != lightingStyleTable[i].id || opt.Name != lightingStyleTable[i].name {
			t.Errorf("gallery[%d] = {%v %q}, table = {%v %q}", i, opt.Style, opt.Name,
				lightingStyleTable[i].id, lightingStyleTable[i].name)
		}
	}
}
