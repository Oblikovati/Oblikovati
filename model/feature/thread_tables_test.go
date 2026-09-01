// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/api/types"
)

// The thread table query surface (#325): every listed designation must parse
// back through the same tables, and the parsed spec carries the derived
// diameters the drawings/hole tables consume.

func TestThreadTablesRoundTripThroughParser(t *testing.T) {
	t.Parallel()
	for _, tt := range ThreadTypes() {
		sizes, err := ThreadNominalSizes(tt)
		if err != nil || len(sizes) == 0 {
			t.Fatalf("ThreadNominalSizes(%s): %v / %d sizes", tt, err, len(sizes))
		}
		for _, size := range sizes {
			designations, err := ThreadDesignationsOf(tt, size)
			if err != nil || len(designations) == 0 {
				t.Fatalf("ThreadDesignationsOf(%s, %s): %v", tt, size, err)
			}
			for _, d := range designations {
				spec, err := ParseThreadDesignation(d)
				if err != nil {
					t.Errorf("table designation %q does not parse: %v", d, err)
					continue
				}
				if spec.NominalSize != size {
					t.Errorf("designation %q parses to size %q, want %q", d, spec.NominalSize, size)
				}
			}
		}
	}
}

func TestThreadSpecDerivedDiameters(t *testing.T) {
	t.Parallel()
	spec, err := ParseThreadDesignation("M8x1.25")
	if err != nil {
		t.Fatal(err)
	}
	// ISO basic: d2 = d − 0.6495·P, tap drill = d − P.
	if d2 := 8 - 0.6495*1.25; spec.PitchDiameter != d2 {
		t.Errorf("pitch diameter = %g, want %g", spec.PitchDiameter, d2)
	}
	if spec.TapDrillDiameter != 8-1.25 {
		t.Errorf("tap drill = %g, want 6.75", spec.TapDrillDiameter)
	}
	if spec.ThreadType != "ISO" || spec.NominalSize != "M8" || !spec.Metric {
		t.Errorf("spec classification = %q/%q metric=%v, want ISO/M8/true", spec.ThreadType, spec.NominalSize, spec.Metric)
	}
}

func TestThreadClassesPerSide(t *testing.T) {
	t.Parallel()
	internal, _ := ThreadClasses("ISO", true)
	external, _ := ThreadClasses("ISO", false)
	if len(internal) == 0 || len(external) == 0 || internal[0] == external[0] {
		t.Errorf("ISO classes = %v / %v, want distinct internal/external", internal, external)
	}
	if _, err := ThreadClasses("BSW", true); err == nil {
		t.Error("an unknown thread type must error")
	}
}

func TestThreadDefinitionParityFields(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	// A cut tapered thread is rejected with a precise error (conical faces are a follow-up).
	pf := NewDressUpFeatures(fs).AddThreadDef(&ThreadDefinition{
		FaceKey: []byte("x"), Designation: "M8x1.25", Cut: true, Tapered: true,
	})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("a cut tapered thread must be sick")
	}
	// The model-diameter choice defaults to major on the resolved spec.
	def := pf.Definition().(*ThreadFeature).Definition()
	if def.ModelDiameter != 0 {
		t.Errorf("unset modelDiameter = %v, want zero (defaults to major at recompute)", def.ModelDiameter)
	}
	if types.ThreadMajorDiameter.String() != "major" {
		t.Error("frozen spelling drifted")
	}
}
