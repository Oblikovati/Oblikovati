// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"
)

// TestSheetMetalCutThroughAll a through-all cut of a 2×2 square removes that column from the
// 4×4 sheet: the result is a valid watertight solid lighter by 2²·thickness.
func TestSheetMetalCutThroughAll(t *testing.T) {
	t.Parallel()
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	pf := NewSheetMetalCutFeatures(fs).Add(&SheetMetalCutDefinition{Sketch: squareSketch(2), ProfileIndex: 0})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("cut sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	want := flatSheetVolume - 2.0*2.0*0.2 // the 2×2 column through the 2 mm sheet
	if got := smSolidVolume(body); math.Abs(got-want) > 1e-4 {
		t.Errorf("cut volume = %.5f, want %.5f", got, want)
	}
}

// TestSheetMetalCutDistance a depth-limited cut removes less than through-all (a partial pocket).
func TestSheetMetalCutDistance(t *testing.T) {
	t.Parallel()
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	pf := NewSheetMetalCutFeatures(fs).Add(&SheetMetalCutDefinition{
		Sketch: squareSketch(2), ProfileIndex: 0, Distance: func() float64 { return 0.1 }, // 1 mm of the 2 mm sheet
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("pocket cut sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	want := flatSheetVolume - 2.0*2.0*0.1 // half-depth pocket
	if got := smSolidVolume(body); math.Abs(got-want) > 1e-4 {
		t.Errorf("pocket volume = %.5f, want %.5f", got, want)
	}
}

// TestSheetMetalCutAcrossBendRejected the across-bend option is reserved until the flat
// pattern (F04) and must error rather than silently do a normal cut.
func TestSheetMetalCutAcrossBendRejected(t *testing.T) {
	t.Parallel()
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	pf := NewSheetMetalCutFeatures(fs).Add(&SheetMetalCutDefinition{Sketch: squareSketch(2), ProfileIndex: 0, AcrossBend: true})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("across-bend cut should be sick (not supported until F04)")
	}
}

// TestSheetMetalCutDefinitionAndKind the accessors return the recipe.
func TestSheetMetalCutDefinitionAndKind(t *testing.T) {
	t.Parallel()
	def := &SheetMetalCutDefinition{ProfileIndex: 1}
	f := &SheetMetalCutFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-cut" {
		t.Error("Definition/Kind mismatch")
	}
}

// TestSheetMetalCutRoundTrip the recipe (sketch + profile + direction + distance + acrossBend)
// marshals and restores; a 0 distance restores as nil (through all).
func TestSheetMetalCutRoundTrip(t *testing.T) {
	t.Parallel()
	sk := squareSketch(2)
	fs := NewPartFeatures(nil)
	NewSheetMetalCutFeatures(fs).Add(&SheetMetalCutDefinition{
		Sketch: sk, ProfileIndex: 0, Direction: NegativeDir, Distance: func() float64 { return 0.5 },
	})
	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalCut
	if data[0].Kind != "sheet-metal-cut" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-cut", data[0])
	}
	if d.Direction != int32(NegativeDir) || d.Distance != 0.5 {
		t.Errorf("payload = %+v, want negative / distance 0.5", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-cut" {
		t.Errorf("restored = %d features, want one cut", fresh.Count())
	}
}

// TestSheetMetalCutMissingPayload / unknown sketch restore errors.
func TestSheetMetalCutMissingPayload(t *testing.T) {
	t.Parallel()
	if _, err := restoreSheetMetalCut(NewPartFeatures(nil), nil, oneSketch{}); err == nil {
		t.Error("restoreSheetMetalCut(nil) must error")
	}
	if _, err := serializeSheetMetalCut(&SheetMetalCutDefinition{Sketch: squareSketch(1)}, oneSketch{s: squareSketch(2)}); err == nil {
		t.Error("serialize with an unknown sketch must error")
	}
}
