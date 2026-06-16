// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// flangedSheetForUnfold builds a square sheet with one 90° flange and returns the engine and
// the bend transform that flattens it (base normal +Z; bend line = the flange's recorded edge).
func flangedSheetForUnfold(t *testing.T) (*PartFeatures, BendTransform) {
	t.Helper()
	fs, edge := seedSheetMetalSheet(t, 4, map[string]string{"BendRadius": "2 mm"})
	pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(), Height: func() float64 { return 2 },
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("flange unhealthy: %s", pf.Health().Reason)
	}
	p, _ := pf.Definition().(*SheetMetalFlangeFeature).Placement()
	return fs, BendTransform{
		LinePoint: p.AxisStart, LineDir: p.AxisStart.VectorTo(p.AxisEnd),
		Up: p.Up.AsVector(), Out: p.Outward.AsVector(), Angle: p.Angle,
		Radius: p.Radius, Thickness: p.Thickness, Neutral: p.Radius + 0.44*p.Thickness,
	}
}

// topZ returns the body's highest vertex Z.
func topZ(b *topo.Body) float64 {
	m := stdmath.Inf(-1)
	for _, v := range b.Vertices() {
		if v.Point().Z > m {
			m = v.Point().Z
		}
	}
	return m
}

// TestUnfoldFeatureFlattens the unfold feature flattens the flange to a watertight solid lying
// in the base plane (the wall no longer rises above the gauge).
func TestUnfoldFeatureFlattens(t *testing.T) {
	fs, bt := flangedSheetForUnfold(t)
	if folded := topZ(fs.Result()[0]); folded < 1.5 {
		t.Fatalf("folded flange should rise; maxZ=%.3f", folded)
	}
	NewSheetMetalUnfoldFeatures(fs).Add(&SheetMetalUnfoldDefinition{Bends: []BendTransform{bt}})
	fs.Recompute()
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	if mz := topZ(body); mz > 0.5 {
		t.Errorf("unfolded flange not flat: maxZ=%.3f, want ~gauge (0.2)", mz)
	}
}

// TestRefoldFeatureRestores refold after unfold restores the fold (the wall rises again) as a
// watertight solid.
func TestRefoldFeatureRestores(t *testing.T) {
	fs, bt := flangedSheetForUnfold(t)
	z0 := topZ(fs.Result()[0])
	NewSheetMetalUnfoldFeatures(fs).Add(&SheetMetalUnfoldDefinition{Bends: []BendTransform{bt}})
	fs.Recompute()
	NewSheetMetalRefoldFeatures(fs).Add(&SheetMetalRefoldDefinition{Bends: []BendTransform{bt}})
	fs.Recompute()
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	if mz := topZ(body); stdmath.Abs(mz-z0) > 0.05 {
		t.Errorf("refold did not restore the fold: maxZ=%.3f, want %.3f", mz, z0)
	}
}

// TestUnfoldRefoldRoundTrip an unfold and a refold feature persist their bend transforms and
// restore (they reference no sketch, so the recipe is self-contained).
func TestUnfoldRefoldRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	bend := BendTransform{LinePoint: math.P3(0, 0, 0.2), LineDir: math.V3(4, 0, 0), Up: math.V3(0, 0, 1), Out: math.V3(0, 1, 0), Angle: stdmath.Pi / 2, Radius: 0.2, Thickness: 0.1, Neutral: 0.244}
	NewSheetMetalUnfoldFeatures(fs).Add(&SheetMetalUnfoldDefinition{Bends: []BendTransform{bend}})
	NewSheetMetalRefoldFeatures(fs).Add(&SheetMetalRefoldDefinition{Bends: []BendTransform{bend}})

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if data[0].Kind != "sheet-metal-unfold" || data[0].SheetMetalUnfold == nil || len(data[0].SheetMetalUnfold.Bends) != 1 {
		t.Fatalf("unfold marshaled = %+v", data[0])
	}
	if data[1].Kind != "sheet-metal-refold" || data[1].SheetMetalRefold == nil {
		t.Fatalf("refold marshaled = %+v", data[1])
	}

	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 2 || fresh.Item(0).Kind() != "sheet-metal-unfold" || fresh.Item(1).Kind() != "sheet-metal-refold" {
		t.Errorf("restored %d features (%q, …), want unfold+refold", fresh.Count(), fresh.Item(0).Kind())
	}
	restored := fresh.Item(0).Definition().(*SheetMetalUnfoldFeature).Definition().Bends
	if len(restored) != 1 || stdmath.Abs(restored[0].Angle-stdmath.Pi/2) > 1e-12 {
		t.Errorf("restored unfold bends = %+v, want one π/2 bend", restored)
	}
}

// TestUnfoldRefoldMissingPayload restoring a nil payload errors.
func TestUnfoldRefoldMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalUnfold(NewPartFeatures(nil, nil), nil); err == nil {
		t.Error("restoreSheetMetalUnfold(nil) must error")
	}
	if _, err := restoreSheetMetalRefold(NewPartFeatures(nil, nil), nil); err == nil {
		t.Error("restoreSheetMetalRefold(nil) must error")
	}
}

// TestUnfoldBendRejectsDegenerateFrame a bend transform with a degenerate fold normal or a
// non-positive developed length cannot define a development and errors with the offending input.
func TestUnfoldBendRejectsDegenerateFrame(t *testing.T) {
	fs, bt := flangedSheetForUnfold(t)
	zeroUp := bt
	zeroUp.Up = math.V3(0, 0, 0)
	if _, err := unfoldBend(fs.Result()[0], zeroUp, +1, "unfold"); err == nil {
		t.Error("unfoldBend with a degenerate fold normal must error")
	}
	badNeutral := bt
	badNeutral.Neutral = 0
	if _, err := unfoldBend(fs.Result()[0], badNeutral, +1, "unfold"); err == nil {
		t.Error("unfoldBend with a non-positive neutral radius must error")
	}
}
