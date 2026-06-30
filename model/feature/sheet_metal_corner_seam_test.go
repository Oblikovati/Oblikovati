// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"
)

// TestCornerSeamCutsGap a gap corner seam removes a square notch (gap²·thickness) at the
// corner edge: the result is a valid watertight solid with that much material removed.
func TestCornerSeamCutsGap(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edge := verticalCornerEdge(t, fs.Result()[0])
	pf := NewSheetMetalCornerSeamFeatures(fs).Add(&SheetMetalCornerSeamDefinition{
		EdgeKeys: [][]byte{edge.ReferenceKey()}, Gap: func() float64 { return 1.0 },
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("corner seam sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	want := flatSheetVolume - (1.0*1.0)*0.2 // square notch gap²·thickness
	if got := sheetVolume(body); math.Abs(got-want) > 1e-4 {
		t.Errorf("seamed volume = %.5f, want %.5f", got, want)
	}
}

// TestCornerSeamRejectsBadInput a seam with no edges or a non-positive gap goes sick.
func TestCornerSeamRejectsBadInput(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edge := verticalCornerEdge(t, fs.Result()[0])
	noGap := NewSheetMetalCornerSeamFeatures(fs).Add(&SheetMetalCornerSeamDefinition{
		EdgeKeys: [][]byte{edge.ReferenceKey()}, Gap: func() float64 { return 0 },
	})
	fs.Recompute()
	if noGap.Health().OK() {
		t.Error("zero-gap corner seam should be sick")
	}

	fs2, _ := seedSheetMetalSheet(t, 4, nil)
	noEdges := NewSheetMetalCornerSeamFeatures(fs2).Add(&SheetMetalCornerSeamDefinition{Gap: func() float64 { return 1 }})
	fs2.Recompute()
	if noEdges.Health().OK() {
		t.Error("corner seam with no edges should be sick")
	}
}

// TestParseSeamType the gap spelling resolves, with an unknown one rejected.
func TestParseSeamType(t *testing.T) {
	for _, s := range []string{"", "gap"} {
		if got, ok := ParseSeamType(s); !ok || got != GapSeam {
			t.Errorf("ParseSeamType(%q) = (%d,%v), want (gap,true)", s, got, ok)
		}
	}
	if _, ok := ParseSeamType("overlap"); ok {
		t.Error("ParseSeamType(overlap) should be unsupported for now")
	}
}

// TestCornerSeamDefinitionAndKind the accessors return the recipe.
func TestCornerSeamDefinitionAndKind(t *testing.T) {
	def := &SheetMetalCornerSeamDefinition{Type: GapSeam}
	f := &SheetMetalCornerSeamFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-corner-seam" {
		t.Error("Definition/Kind mismatch")
	}
}

// TestCornerSeamRoundTrip the recipe (edges + gap + type) marshals and restores.
func TestCornerSeamRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewSheetMetalCornerSeamFeatures(fs).Add(&SheetMetalCornerSeamDefinition{
		EdgeKeys: [][]byte{[]byte("k1"), []byte("k2")}, Gap: func() float64 { return 0.3 }, Type: GapSeam,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalCornerSeam
	if data[0].Kind != "sheet-metal-corner-seam" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-corner-seam", data[0])
	}
	if len(d.Edges) != 2 || d.Gap != 0.3 {
		t.Errorf("payload = %+v, want 2 edges / gap 0.3", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-corner-seam" {
		t.Errorf("restored = %d features, want one corner seam", fresh.Count())
	}
}

// TestCornerSeamMissingPayload restoring a corner-seam record with no payload errors.
func TestCornerSeamMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalCornerSeam(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreSheetMetalCornerSeam(nil) must error")
	}
}
