// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/renderer"
)

// TestLightingStyleCommandsSetRig checks the View-tab Lighting Style options activate the rig
// on the session, end to end through the ribbon command.
func TestLightingStyleCommandsSetRig(t *testing.T) {
	s := registeredSession(t)
	if err := s.Execute("View.Lighting.Outdoors"); err != nil {
		t.Fatalf("execute Lighting.Outdoors: %v", err)
	}
	if s.LightingStyleName() != "Outdoors" {
		t.Errorf("style = %q, want Outdoors", s.LightingStyleName())
	}
	cmd, ok := s.Commands().ByID("View.Lighting.Outdoors")
	if !ok || !cmd.IsActive(s) {
		t.Errorf("Outdoors command should report active after selection")
	}
	if other, _ := s.Commands().ByID("View.Lighting.Default"); other.IsActive(s) {
		t.Errorf("Default should not be active after selecting Outdoors")
	}
}

// TestEnvironmentCommandsSetEnvironment checks the Environment options drive the session's IBL
// environment (and that None clears it).
func TestEnvironmentCommandsSetEnvironment(t *testing.T) {
	s := registeredSession(t)
	if err := s.Execute("View.Environment.Studio"); err != nil {
		t.Fatalf("execute Environment.Studio: %v", err)
	}
	if e := s.Environment(); e.Preset != renderer.EnvStudio || !e.ShowImage {
		t.Errorf("environment = %+v, want Studio shown", e)
	}
	if err := s.Execute("View.Environment.None"); err != nil {
		t.Fatalf("execute Environment.None: %v", err)
	}
	if s.Environment().IsActive() {
		t.Errorf("None should clear the environment, got %+v", s.Environment())
	}
}

// TestShadowToggleCommands checks each Shadows toggle flips its setting and reports its checked
// state, and that enabling object shadows seeds a visible density.
func TestShadowToggleCommands(t *testing.T) {
	s := registeredSession(t)
	if s.ShadowSettings().ObjectShadows {
		t.Fatal("object shadows should start off")
	}
	if err := s.Execute("View.ObjectShadows"); err != nil {
		t.Fatalf("execute ObjectShadows: %v", err)
	}
	sh := s.ShadowSettings()
	if !sh.ObjectShadows || sh.Density == 0 {
		t.Errorf("after toggle, object shadows = %v density = %g, want on with density", sh.ObjectShadows, sh.Density)
	}
	cmd, _ := s.Commands().ByID("View.ObjectShadows")
	if !cmd.IsActive(s) {
		t.Errorf("ObjectShadows toggle should be active after enabling")
	}
	if err := s.Execute("View.ObjectShadows"); err != nil { // toggle back off
		t.Fatalf("re-toggle: %v", err)
	}
	if s.ShadowSettings().ObjectShadows {
		t.Errorf("object shadows should be off after second toggle")
	}
}

// TestLightingSettingsCommandToggles checks the "Lighting…" command opens and closes the
// settings panel and reports its open state.
func TestLightingSettingsCommandToggles(t *testing.T) {
	s := registeredSession(t)
	if s.LightingPanelOpen() {
		t.Fatal("panel should start closed")
	}
	if err := s.Execute("View.LightingSettings"); err != nil {
		t.Fatalf("execute LightingSettings: %v", err)
	}
	cmd, _ := s.Commands().ByID("View.LightingSettings")
	if !s.LightingPanelOpen() || !cmd.IsActive(s) {
		t.Errorf("panel should be open and command active after toggle")
	}
	if err := s.Execute("View.LightingSettings"); err != nil {
		t.Fatalf("re-toggle: %v", err)
	}
	if s.LightingPanelOpen() {
		t.Errorf("panel should be closed after second toggle")
	}
}

// TestLightingToneSetters checks the exposure/brightness/ambience setters edit the live rig.
func TestLightingToneSetters(t *testing.T) {
	s := NewSession()
	s.SetExposure(1.7)
	s.SetBrightness(2.2)
	s.SetAmbience(0.33)
	if got := s.SceneLighting(); got.Exposure != 1.7 || got.Brightness != 2.2 || got.Ambience != 0.33 {
		t.Errorf("tone = exp %g bri %g amb %g, want 1.7/2.2/0.33", got.Exposure, got.Brightness, got.Ambience)
	}
}

// TestLoadHDRCommandRequestsDialog checks the "Load HDR…" command raises a one-shot request the
// head consumes to open its file dialog, and that LoadEnvironmentFile sets a file environment.
func TestLoadHDRCommandRequestsDialog(t *testing.T) {
	s := registeredSession(t)
	if err := s.Execute("View.LoadHDR"); err != nil {
		t.Fatalf("execute LoadHDR: %v", err)
	}
	if !s.TakeLoadEnvironmentRequest() {
		t.Fatal("Load HDR command should raise a load-environment request")
	}
	if s.TakeLoadEnvironmentRequest() {
		t.Error("the request must be one-shot (cleared after taking)")
	}
	s.LoadEnvironmentFile("/tmp/studio.hdr")
	if e := s.Environment(); e.FilePath != "/tmp/studio.hdr" || !e.ShowImage {
		t.Errorf("LoadEnvironmentFile gave %+v, want the file shown", e)
	}
}

// TestLightingPanelsOnViewTab checks the Lighting Style, Environment and Shadows panels are all
// present on the View tab (the ribbon exposes the controls).
func TestLightingPanelsOnViewTab(t *testing.T) {
	s := registeredSession(t)
	tab, ok := BuildRibbon(s).Tab("View")
	if !ok {
		t.Fatal("no View tab")
	}
	for _, panel := range []string{"Lighting Style", "Environment", "Shadows"} {
		if _, ok := tab.Panel(panel); !ok {
			t.Errorf("View tab missing the %q panel", panel)
		}
	}
}
