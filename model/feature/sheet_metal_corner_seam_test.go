// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
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

// TestParseSeamType every seam spelling resolves (gap by default), with an unknown one rejected.
func TestParseSeamType(t *testing.T) {
	for _, s := range []string{"", "gap"} {
		if got, ok := ParseSeamType(s); !ok || got != GapSeam {
			t.Errorf("ParseSeamType(%q) = (%d,%v), want (gap,true)", s, got, ok)
		}
	}
	for spelling, want := range map[string]SeamType{
		"overlap": OverlapSeam, "reverseOverlap": ReverseOverlapSeam, "noOverlap": NoOverlapSeam,
	} {
		if got, ok := ParseSeamType(spelling); !ok || got != want {
			t.Errorf("ParseSeamType(%q) = (%d,%v), want (%d,true)", spelling, got, ok, want)
		}
	}
	if _, ok := ParseSeamType("welded"); ok {
		t.Error("ParseSeamType(welded) should be rejected")
	}
}

// TestCornerSeamOverlapOnFlatSheetIsUnmodelled a non-gap seam on a FLAT sheet has no two flange
// walls to lap, so its lap/butt solid (#2085) cannot be built there: the feature stays healthy,
// leaves the volume untouched, and reports the "unmodelled" diagnostic honestly. (The modelled lap
// on a real two-flange corner is exercised in sheet_metal_corner_seam_lap_test.go.)
func TestCornerSeamOverlapOnFlatSheetIsUnmodelled(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edge := verticalCornerEdge(t, fs.Result()[0])
	pf := NewSheetMetalCornerSeamFeatures(fs).Add(&SheetMetalCornerSeamDefinition{
		EdgeKeys: [][]byte{edge.ReferenceKey()}, Gap: func() float64 { return 1.0 },
		Type: OverlapSeam, Overlap: 50,
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("overlap corner seam should stay healthy (recorded, geometry deferred): %+v", pf.Health())
	}
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	if got := sheetVolume(body); math.Abs(got-flatSheetVolume) > 1e-4 {
		t.Errorf("overlap seam removed material (%.5f, want the intact %.5f); only the gap style cuts", got, flatSheetVolume)
	}
	if !hasDiagCode(pf.Diagnostics(), codeCornerSeamUnmodeled) {
		t.Errorf("overlap seam must report %q so the deferral is visible, got %v", codeCornerSeamUnmodeled, pf.Diagnostics())
	}
}

// TestCornerSeamGapStillReports the gap style keeps cutting its notch and raises no unmodelled
// diagnostic — the deferral is only for the lap/butt styles.
func TestCornerSeamGapStillReports(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edge := verticalCornerEdge(t, fs.Result()[0])
	pf := NewSheetMetalCornerSeamFeatures(fs).Add(&SheetMetalCornerSeamDefinition{
		EdgeKeys: [][]byte{edge.ReferenceKey()}, Gap: func() float64 { return 1.0 }, Type: GapSeam,
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("gap corner seam sick: %+v", pf.Health())
	}
	if hasDiagCode(pf.Diagnostics(), codeCornerSeamUnmodeled) {
		t.Errorf("the gap style should not raise the unmodelled diagnostic: %v", pf.Diagnostics())
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

// TestCornerSeamRoundTrip the recipe (edges + gap + type + the lap fields) marshals and restores.
func TestCornerSeamRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewSheetMetalCornerSeamFeatures(fs).Add(&SheetMetalCornerSeamDefinition{
		EdgeKeys: [][]byte{[]byte("k1"), []byte("k2")}, Gap: func() float64 { return 0.3 },
		Type: ReverseOverlapSeam, Overlap: 75, ReliefShape: types.CornerSquare,
		ReliefSize: func() float64 { return 0.1 }, DefinitionType: types.CornerSeamFaceEdgeDistance,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalCornerSeam
	if data[0].Kind != "sheet-metal-corner-seam" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-corner-seam", data[0])
	}
	if len(d.Edges) != 2 || d.Gap != 0.3 || d.Type != int32(ReverseOverlapSeam) || d.Overlap != 75 ||
		d.ReliefShape != int32(types.CornerSquare) || d.ReliefSize != 0.1 ||
		d.DefinitionType != int32(types.CornerSeamFaceEdgeDistance) {
		t.Errorf("payload = %+v, want 2 edges / gap 0.3 / reverseOverlap / 75%% / square / 0.1 / faceEdge", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-corner-seam" {
		t.Fatalf("restored = %d features, want one corner seam", fresh.Count())
	}
	got := fresh.Item(0).Definition().(*SheetMetalCornerSeamFeature).Definition()
	if got.Type != ReverseOverlapSeam || got.Overlap != 75 || got.ReliefShape != types.CornerSquare ||
		evalFloat(got.ReliefSize) != 0.1 || got.DefinitionType != types.CornerSeamFaceEdgeDistance {
		t.Errorf("restored recipe = %+v, want reverseOverlap / 75%% / square / 0.1 / faceEdge", got)
	}
}

// TestCornerSeamMissingPayload restoring a corner-seam record with no payload errors.
func TestCornerSeamMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalCornerSeam(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreSheetMetalCornerSeam(nil) must error")
	}
}
