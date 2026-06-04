// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"
)

// argJSON marshals a request value to the JSON string the router test harness passes as args.
func argJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return string(b)
}

// TestLightingStyleRoundTrips drives lighting.setStyle then lighting.getStyle for every
// gallery style and asserts the name survives — the dogfood proof the lighting-style contract
// reaches the session and back.
func TestLightingStyleRoundTrips(t *testing.T) {
	r, s := seededSession(t)
	var list wire.LightingStyleListResult
	call(t, r, s, "lighting.listStyles", "{}", &list)
	if len(list.Styles) == 0 {
		t.Fatal("listStyles returned no styles")
	}
	for _, st := range list.Styles {
		var set wire.LightingStyleView
		call(t, r, s, "lighting.setStyle", argJSON(t, wire.SetLightingStyleArgs{Name: st.Name}), &set)
		if set.Name != st.Name {
			t.Errorf("setStyle(%q) = %q", st.Name, set.Name)
		}
		var got wire.LightingStyleView
		call(t, r, s, "lighting.getStyle", "{}", &got)
		if got.Name != st.Name {
			t.Errorf("after set %q, getStyle = %q", st.Name, got.Name)
		}
	}
}

// TestLightingSetStyleRejectsUnknown checks an unknown style name is an error, not a silent
// fallback.
func TestLightingSetStyleRejectsUnknown(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "lighting.setStyle", []byte(`{"name":"Nope"}`)); err == nil {
		t.Fatal("expected error for unknown lighting style")
	}
}

// TestLightRoundTrips adds a point light, edits it, and reads it back — proving the light
// list/add/set path preserves the full light state through the wire DTO.
func TestLightRoundTrips(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "lighting.setStyle", `{"name":"Default"}`, nil)
	var before wire.LightListResult
	call(t, r, s, "lighting.listLights", "{}", &before)

	var added wire.LightInfo
	call(t, r, s, "lighting.addLight", argJSON(t, wire.AddLightArgs{DefinitionType: types.PointLight}), &added)
	if added.LightDefinitionType != types.PointLight {
		t.Fatalf("addLight definition = %v, want Point", added.LightDefinitionType)
	}

	added.Intensity = 2.5
	added.Color = types.Rgba{R: 0.2, G: 0.4, B: 0.8, A: 1}
	added.Position = [3]float64{1, 2, 3}
	var set wire.LightInfo
	call(t, r, s, "lighting.setLight", argJSON(t, wire.SetLightArgs{Index: added.Index, Light: added}), &set)
	if set.Intensity != 2.5 || set.Color.B != 0.8 || set.Position != [3]float64{1, 2, 3} {
		t.Errorf("setLight did not round-trip: %+v", set)
	}

	var after wire.LightListResult
	call(t, r, s, "lighting.listLights", "{}", &after)
	if len(after.Lights) != len(before.Lights)+1 {
		t.Errorf("light count = %d, want %d", len(after.Lights), len(before.Lights)+1)
	}
}

// TestShadowSettingsRoundTrip pins the GroundShadowEnum ⇄ renderer-flag mapping through the
// view.setShadows / view.getShadows path, including the X-ray distinction.
func TestShadowSettingsRoundTrip(t *testing.T) {
	r, s := seededSession(t)
	for _, g := range types.AllGroundShadows() {
		in := wire.ShadowSettings{
			GroundShadow: g, ObjectShadows: true, AmbientShadows: g == types.GroundShadow,
			Density: 0.6, Softness: 0.3,
		}
		var set wire.ShadowSettings
		call(t, r, s, "view.setShadows", argJSON(t, in), &set)
		if set.GroundShadow != g {
			t.Errorf("setShadows ground %v round-tripped to %v", g, set.GroundShadow)
		}
		var got wire.ShadowSettings
		call(t, r, s, "view.getShadows", "{}", &got)
		if got.GroundShadow != g || got.ObjectShadows != in.ObjectShadows {
			t.Errorf("getShadows = %+v, want ground %v objects %v", got, g, in.ObjectShadows)
		}
	}
}

// TestEnvironmentRoundTrips switches to a preset, loads a file, and reads back — proving the
// preset/file environment contract reaches the session.
func TestEnvironmentRoundTrips(t *testing.T) {
	r, s := seededSession(t)
	var set wire.EnvironmentView
	call(t, r, s, "environment.set",
		argJSON(t, wire.SetEnvironmentArgs{Preset: "Studio", Intensity: 1, ShowImage: true}), &set)
	if set.Preset != "Studio" || !set.ShowImage {
		t.Errorf("environment.set = %+v, want Studio shown", set)
	}
	var presets wire.EnvironmentPresetListResult
	call(t, r, s, "environment.listPresets", "{}", &presets)
	active := ""
	for _, p := range presets.Presets {
		if p.Active {
			active = p.Name
		}
	}
	if active != "Studio" {
		t.Errorf("active preset = %q, want Studio", active)
	}

	var loaded wire.EnvironmentView
	call(t, r, s, "environment.loadImage", `{"filePath":"/tmp/sky.hdr"}`, &loaded)
	if loaded.FilePath != "/tmp/sky.hdr" || loaded.Preset != "" {
		t.Errorf("loadImage = %+v, want file path set and no preset", loaded)
	}
}

// TestEnvironmentSetRejectsUnknownPreset checks an unknown preset name is an error.
func TestEnvironmentSetRejectsUnknownPreset(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "environment.set", []byte(`{"preset":"Mars"}`)); err == nil {
		t.Fatal("expected error for unknown environment preset")
	}
}
