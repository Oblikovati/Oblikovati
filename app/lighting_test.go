// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/renderer"
)

// TestNewSessionStartsOnThreePointLighting asserts a fresh session has the Three Point
// studio rig — the out-of-the-box lighting for every visual style now that the whole rig
// lights every shaded mode (ADR-0026 §8).
func TestNewSessionStartsOnThreePointLighting(t *testing.T) {
	s := NewSession()
	if s.LightingStyleName() != "Three Point" {
		t.Errorf("new session style = %q, want Three Point", s.LightingStyleName())
	}
	if len(s.SceneLighting().Lights) != 3 {
		t.Errorf("default rig has %d lights, want the key/fill/back trio", len(s.SceneLighting().Lights))
	}
}

// TestSetLightingStyleResolvesRig checks switching the style replaces the live rig with that
// preset's lights, and that an unknown name errors.
func TestSetLightingStyleResolvesRig(t *testing.T) {
	s := NewSession()
	if err := s.SetLightingStyle("Outdoors"); err != nil {
		t.Fatalf("SetLightingStyle(Outdoors): %v", err)
	}
	if s.LightingStyleName() != "Outdoors" {
		t.Errorf("style = %q, want Outdoors", s.LightingStyleName())
	}
	if !s.Environment().IsActive() {
		t.Error("Outdoors should bring an active environment")
	}
	if err := s.SetLightingStyle("Nope"); err == nil {
		t.Error("expected error for unknown style")
	}
}

// TestAddLightClampsToMax checks AddLight refuses to exceed the UBO light array bound.
func TestAddLightClampsToMax(t *testing.T) {
	s := NewSession()
	// Default starts with one light; fill to the max, then expect an error.
	for len(s.Lights()) < renderer.MaxSceneLights {
		if _, err := s.AddLight(renderer.PointLight); err != nil {
			t.Fatalf("AddLight before max: %v", err)
		}
	}
	if _, err := s.AddLight(renderer.PointLight); err == nil {
		t.Errorf("AddLight past max (%d) should error", renderer.MaxSceneLights)
	}
}

// TestSetLightRejectsOutOfRange checks SetLight validates the index instead of panicking.
func TestSetLightRejectsOutOfRange(t *testing.T) {
	s := NewSession()
	if err := s.SetLight(99, renderer.SceneLight{}); err == nil {
		t.Error("SetLight(99) should error on an out-of-range index")
	}
}

// TestGroundShadowMappingRoundTrips pins the public GroundShadowEnum ⇄ renderer-flag bijection
// (including the X-ray distinction) at the app layer.
func TestGroundShadowMappingRoundTrips(t *testing.T) {
	for _, g := range types.AllGroundShadows() {
		var sh renderer.ShadowSettings
		ApplyGroundShadow(&sh, g)
		if got := GroundShadowForSettings(sh); got != g {
			t.Errorf("ground shadow %v round-tripped to %v", g, got)
		}
	}
	// Sanity: None clears the flag; X-ray sets both.
	var sh renderer.ShadowSettings
	ApplyGroundShadow(&sh, types.XRayGroundShadow)
	if !sh.GroundShadows || !sh.GroundXRay {
		t.Errorf("X-ray should set GroundShadows and GroundXRay, got %+v", sh)
	}
}

// TestLightKindDefinitionBijection checks the renderer LightKind ⇄ public definition mapping is
// total and stable over the public enum.
func TestLightKindDefinitionBijection(t *testing.T) {
	for _, d := range types.AllLightDefinitionTypes() {
		if got := DefinitionForLightKind(LightKindForDefinition(d)); got != d {
			t.Errorf("definition %v round-tripped to %v", d, got)
		}
	}
}
