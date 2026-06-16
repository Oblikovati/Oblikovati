// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/param"
)

// sheetWithHem builds a square sheet then hems its highest +X top edge with the given type.
func sheetWithHem(t *testing.T, hemType HemType, lengthCm, gapCm float64) *topo.Body {
	t.Helper()
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps, nil)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	fs.Recompute()

	edge := topEdgeAlongX(t, fs.Result()[0])
	def := &SheetMetalHemDefinition{EdgeKey: edge.ReferenceKey(), Length: func() float64 { return lengthCm }, Type: hemType}
	if gapCm > 0 {
		def.Gap = func() float64 { return gapCm }
	}
	pf := NewSheetMetalHemFeatures(fs).Add(def)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("hem sick: %+v", pf.Health())
	}
	return fs.Result()[0]
}

// TestClosedHemBuildsWatertightSolid a closed hem folds the edge back as one valid watertight
// solid, adding material above the sheet.
func TestClosedHemBuildsWatertightSolid(t *testing.T) {
	body := sheetWithHem(t, ClosedHem, 0.8, 0)
	if !body.IsSolid() {
		t.Fatal("hemmed sheet is not a solid")
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("hemmed sheet has %d boundary edges, want 0 (watertight)", len(open))
	}
	if res := ops.Validate(body); !res.Valid {
		t.Fatalf("hemmed sheet failed validation: %+v", res)
	}
	// The fold curls back over the parent, so the body rises above the flat sheet's 0.2 cm.
	maxZ := math.Inf(-1)
	for _, v := range body.Vertices() {
		if z := v.Point().Z; z > maxZ {
			maxZ = z
		}
	}
	if maxZ < 0.25 {
		t.Errorf("closed hem top Z = %.3f, want it to rise above the sheet", maxZ)
	}
}

// TestOpenHemLeavesLargerLoop an open hem with a gap folds over a larger radius than a closed
// hem, so it rises higher.
func TestOpenHemLeavesLargerLoop(t *testing.T) {
	closedTop := topZOf(sheetWithHem(t, ClosedHem, 0.8, 0))
	openTop := topZOf(sheetWithHem(t, OpenHem, 0.8, 0.6)) // gap 6 mm ⇒ radius 3 mm
	if !(openTop > closedTop) {
		t.Errorf("open hem (top %.3f) should loop higher than closed hem (top %.3f)", openTop, closedTop)
	}
}

func topZOf(body *topo.Body) float64 {
	maxZ := math.Inf(-1)
	for _, v := range body.Vertices() {
		if z := v.Point().Z; z > maxZ {
			maxZ = z
		}
	}
	return maxZ
}

// TestHemTypeParse the wire spellings resolve, and an unknown one is rejected.
func TestHemTypeParse(t *testing.T) {
	for s, want := range map[string]HemType{"": ClosedHem, "closed": ClosedHem, "open": OpenHem} {
		if got, ok := ParseHemType(s); !ok || got != want {
			t.Errorf("ParseHemType(%q) = (%d,%v), want (%d,true)", s, got, ok, want)
		}
	}
	if _, ok := ParseHemType("rolled"); ok {
		t.Error("ParseHemType(rolled) should be unsupported for now")
	}
}

// TestHemRoundTrip the hem recipe (edge + length + type + gap + flip) marshals and restores.
func TestHemRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewSheetMetalHemFeatures(fs).Add(&SheetMetalHemDefinition{
		EdgeKey: []byte("edge"),
		Length:  func() float64 { return 0.6 },
		Type:    OpenHem,
		Gap:     func() float64 { return 0.4 },
		Flip:    true,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalHem
	if data[0].Kind != "sheet-metal-hem" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-hem", data[0])
	}
	if d.Length != 0.6 || d.Type != int32(OpenHem) || d.Gap != 0.4 || !d.Flip {
		t.Errorf("payload = %+v, want length 0.6 / open / gap 0.4 / flip", d)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-hem" {
		t.Errorf("restored = %d features, want one hem", fresh.Count())
	}
}

// TestHemDefinitionAccessor Definition/Kind return the stored recipe.
func TestHemDefinitionAccessor(t *testing.T) {
	def := &SheetMetalHemDefinition{EdgeKey: []byte("k"), Length: func() float64 { return 1 }, Type: OpenHem}
	f := &SheetMetalHemFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-hem" {
		t.Error("Definition/Kind mismatch")
	}
}

// TestHemMissingPayload restoring a hem record with no payload errors.
func TestHemMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalHem(NewPartFeatures(nil, nil), nil); err == nil {
		t.Error("restoreSheetMetalHem(nil) must error")
	}
}

// TestHemRejectsBadDims a zero-length hem goes sick.
func TestHemRejectsBadDims(t *testing.T) {
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps, nil)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	fs.Recompute()
	edge := topEdgeAlongX(t, fs.Result()[0])
	pf := NewSheetMetalHemFeatures(fs).Add(&SheetMetalHemDefinition{EdgeKey: edge.ReferenceKey(), Length: func() float64 { return 0 }})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("zero-length hem should be sick")
	}
}
