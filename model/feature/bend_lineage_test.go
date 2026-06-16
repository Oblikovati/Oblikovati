// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"
)

// constClosure makes a parameter-backed-style constant closure for a test definition.
func constClosure(v float64) func() float64 { return func() float64 { return v } }

// TestFlangeBendSpecsDefaultsTo90 a flange with no overrides reports one 90° bend deferring
// its radius to the rule (signalled by a non-positive radius).
func TestFlangeBendSpecsDefaultsTo90(t *testing.T) {
	f := &SheetMetalFlangeFeature{def: &SheetMetalFlangeDefinition{Height: constClosure(1)}}
	specs := f.BendSpecs(0.1)
	if len(specs) != 1 {
		t.Fatalf("flange BendSpecs len = %d, want 1", len(specs))
	}
	if math.Abs(specs[0].Angle-math.Pi/2) > 1e-12 {
		t.Errorf("angle = %g, want π/2", specs[0].Angle)
	}
	if specs[0].Radius > 0 {
		t.Errorf("radius = %g, want <= 0 (defer to rule)", specs[0].Radius)
	}
}

// TestFlangeBendSpecsHonorsOverrides a flange with explicit angle/radius reports them.
func TestFlangeBendSpecsHonorsOverrides(t *testing.T) {
	f := &SheetMetalFlangeFeature{def: &SheetMetalFlangeDefinition{
		Height: constClosure(1), Angle: constClosure(math.Pi / 3), Radius: constClosure(0.5),
	}}
	specs := f.BendSpecs(0.1)
	if math.Abs(specs[0].Angle-math.Pi/3) > 1e-12 || specs[0].Radius != 0.5 {
		t.Errorf("specs = %+v, want angle π/3 radius 0.5", specs[0])
	}
}

// TestBendAndFoldBendSpecs the bend and fold features each report one 90° bend by default.
func TestBendAndFoldBendSpecs(t *testing.T) {
	bend := &SheetMetalBendFeature{def: &SheetMetalBendDefinition{}}
	fold := &SheetMetalFoldFeature{def: &SheetMetalFoldDefinition{}}
	for name, specs := range map[string][]BendSpec{"bend": bend.BendSpecs(0.1), "fold": fold.BendSpecs(0.1)} {
		if len(specs) != 1 || math.Abs(specs[0].Angle-math.Pi/2) > 1e-12 {
			t.Errorf("%s BendSpecs = %+v, want one π/2 bend", name, specs)
		}
	}
}

// TestHemBendSpecsGaugeDerivedRadius a hem reports a 180° fold whose radius is derived from
// the gauge — a closed hem at half the thickness, an open hem at half its gap.
func TestHemBendSpecsGaugeDerivedRadius(t *testing.T) {
	closed := &SheetMetalHemFeature{def: &SheetMetalHemDefinition{Type: ClosedHem}}
	specs := closed.BendSpecs(0.2)
	if len(specs) != 1 || math.Abs(specs[0].Angle-math.Pi) > 1e-12 {
		t.Fatalf("closed hem specs = %+v, want one π fold", specs)
	}
	if specs[0].Radius != 0.1 {
		t.Errorf("closed hem radius = %g, want 0.1 (half the 0.2 gauge)", specs[0].Radius)
	}
	open := &SheetMetalHemFeature{def: &SheetMetalHemDefinition{Type: OpenHem, Gap: constClosure(0.4)}}
	if r := open.BendSpecs(0.2)[0].Radius; r != 0.2 {
		t.Errorf("open hem radius = %g, want 0.2 (half the 0.4 gap)", r)
	}
}
