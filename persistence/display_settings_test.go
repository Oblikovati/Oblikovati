// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/display"
)

// customSettings is a non-default display-settings value used to prove a faithful round-trip.
func customSettings() display.Settings {
	s := display.DefaultSettings()
	s.BackgroundType = types.OneColorBackground
	s.EdgeColor = types.NewColor(255, 40, 40)
	s.HiddenLineDimmingPercent = 33
	s.GroundPlane.Color = types.NewColor(40, 90, 200)
	s.GroundPlane.Visible = false
	s.GroundPlane.MinorLinesPerMajorGridLine = 8
	s.ShowObjectShadows = false
	s.TexturesOn = false
	return s
}

// TestDisplaySettingsConverterRoundTrip checks model→record→model preserves every field
// (M16-F07 #643).
func TestDisplaySettingsConverterRoundTrip(t *testing.T) {
	set := customSettings()
	got := fromCodecDisplaySettings(toCodecDisplaySettings(set))
	if got != set {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, set)
	}
}

// TestPackageDisplaySettingsSurvivesMarshal checks the per-document display settings round-trip
// through the .obk YAML (marshal → decode), and are written as readable YAML (not base64).
func TestPackageDisplaySettingsSurvivesMarshal(t *testing.T) {
	p := NewPackage()
	if err := p.SetManifest(Manifest{SchemaVersion: CurrentSchemaVersion, DocumentType: 1, DisplayName: "P"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	p.SetDisplaySettings(toCodecDisplaySettings(customSettings()))

	data, err := p.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), "displaySettings:") {
		t.Error("display settings should be written as a readable YAML block")
	}

	p2, err := decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p2.DisplaySettings() == nil {
		t.Fatal("display settings lost in the marshal/decode round-trip")
	}
	if got := fromCodecDisplaySettings(p2.DisplaySettings()); got != customSettings() {
		t.Errorf("decoded settings = %+v, want %+v", got, customSettings())
	}
}
