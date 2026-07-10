// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
)

// customSketchSettings is a non-default sketch settings value for the round-trip tests (#147).
func customSketchSettings() types.SketchSettings {
	return types.SketchSettings{
		InferConstraints:     true,
		AutoApplyConstraints: false,
		ConstraintPriority:   types.PriorityParallelPerpendicular,
		// #1877 grid/snap + constraint-display + relax fields, all non-default so the round-trip
		// proves each survives the .obk.
		XSnapSpacing:                 0.25,
		YSnapSpacing:                 0.5,
		SnapsPerMinorGrid:            4,
		MinorLinesPerMajorGridLine:   8,
		PersistInferredConstraints:   true,
		DisplayConstraintsOnCreation: true,
		EditDimensionsWhenCreated:    false,
		OverConstrainedBehavior:      types.OverConstrainedApplyDriving,
		EnableRelaxMode:              true,
	}
}

// TestSketchSettingsConverterRoundTrip pins the on-disk record conversion (#147).
func TestSketchSettingsConverterRoundTrip(t *testing.T) {
	set := customSketchSettings()
	if got := fromCodecSketchSettings(toCodecSketchSettings(set)); got != set {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", got, set)
	}
}

// TestPackageSketchSettingsSurvivesMarshal checks the per-document sketch settings round-trip through
// the .obk YAML (marshal → decode), written as a readable YAML block (#147).
func TestPackageSketchSettingsSurvivesMarshal(t *testing.T) {
	p := NewPackage()
	if err := p.SetManifest(Manifest{SchemaVersion: CurrentSchemaVersion, DocumentType: 1, DisplayName: "P"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	p.SetSketchSettings(toCodecSketchSettings(customSketchSettings()))

	data, err := p.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), "sketchSettings:") {
		t.Error("sketch settings should be written as a readable YAML block")
	}

	p2, err := decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if p2.SketchSettings() == nil {
		t.Fatal("sketch settings lost in the marshal/decode round-trip")
	}
	if got := fromCodecSketchSettings(p2.SketchSettings()); got != customSketchSettings() {
		t.Errorf("decoded settings = %+v, want %+v", got, customSketchSettings())
	}
}
