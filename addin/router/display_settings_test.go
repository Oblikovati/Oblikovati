// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestDisplayOptionsRoundTrip reads the app display options, edits a few fields, writes them
// back, and checks the reply reflects the change.
func TestDisplayOptionsRoundTrip(t *testing.T) {
	r, s := seededSession(t)

	var got wire.DisplayModeOptionsView
	call(t, r, s, "display.getOptions", `{}`, &got)
	if !got.DisplayQuality.IsValid() {
		t.Fatalf("default display quality %v is not valid", got.DisplayQuality)
	}

	got.DisplayQuality = types.RoughDisplayQuality
	got.UseRayTracing = true
	got.RayTracingQuality = types.BestRayTracingQuality
	var back wire.DisplayModeOptionsView
	call(t, r, s, "display.setOptions", mustJSON(t, got), &back)
	if back.DisplayQuality != types.RoughDisplayQuality || !back.UseRayTracing {
		t.Errorf("set options = %+v, want rough quality + ray tracing on", back)
	}

	var reread wire.DisplayModeOptionsView
	call(t, r, s, "display.getOptions", `{}`, &reread)
	if reread.RayTracingQuality != types.BestRayTracingQuality {
		t.Errorf("reread ray-tracing quality = %v, want Best", reread.RayTracingQuality)
	}
}

// TestDisplaySettingsRoundTrip reads the active document's display settings, flips background
// and ground state, writes them back, and checks they persist.
func TestDisplaySettingsRoundTrip(t *testing.T) {
	r, s := seededSession(t)

	var got wire.DisplaySettingsView
	call(t, r, s, "document.getDisplaySettings", `{}`, &got)
	if !got.GroundShadow.IsValid() {
		t.Fatalf("default ground shadow %v is not valid", got.GroundShadow)
	}

	got.BackgroundType = types.OneColorBackground
	got.GroundPlane.Visible = false
	got.ShowObjectShadows = false
	call(t, r, s, "document.setDisplaySettings", mustJSON(t, wire.SetDisplaySettingsArgs{Settings: got}), &wire.DisplaySettingsView{})

	var reread wire.DisplaySettingsView
	call(t, r, s, "document.getDisplaySettings", `{}`, &reread)
	if reread.BackgroundType != types.OneColorBackground || reread.GroundPlane.Visible || reread.ShowObjectShadows {
		t.Errorf("reread = %+v, want OneColor bg, hidden ground, no object shadows", reread)
	}
}
