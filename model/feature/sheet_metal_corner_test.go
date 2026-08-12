// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// verticalCornerEdges returns all through-thickness (Z-aligned) corner edges of the sheet.
func verticalCornerEdges(t *testing.T, body *topo.Body) []*topo.Edge {
	t.Helper()
	var out []*topo.Edge
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if math.Abs(a.X-b.X) < 1e-6 && math.Abs(a.Y-b.Y) < 1e-6 && math.Abs(a.Z-b.Z) > 1e-6 {
			out = append(out, e)
		}
	}
	if len(out) < 2 {
		t.Fatalf("need ≥2 vertical corner edges, found %d", len(out))
	}
	return out
}

// verticalCornerEdge returns a through-thickness (Z-aligned) corner edge of the sheet — the
// edge a sheet-metal corner treatment finishes.
func verticalCornerEdge(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if math.Abs(a.X-b.X) < 1e-6 && math.Abs(a.Y-b.Y) < 1e-6 && math.Abs(a.Z-b.Z) > 1e-6 {
			return e
		}
	}
	t.Fatal("no vertical corner edge found on the sheet")
	return nil
}

// flatSheetVolume is the volume of the un-cornered 4×4×0.2 sheet.
const flatSheetVolume = 4.0 * 4.0 * 0.2

func sheetVolume(body *topo.Body) float64 {
	return ops.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-4}).Volume
}

// TestCornerChamferRemovesCorner a corner chamfer cuts a triangular notch off the sheet
// corner: the result is a valid watertight solid whose volume drops by setback²/2 · thickness.
func TestCornerChamferRemovesCorner(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edge := verticalCornerEdge(t, fs.Result()[0])
	pf := NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		EdgeKeys: [][]byte{edge.ReferenceKey()}, Treatment: CornerChamfer, Size: func() float64 { return 1.0 },
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("corner chamfer sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("chamfered corner invalid: %v", r.Issues)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("chamfered corner not watertight: %d boundary edges", len(open))
	}
	want := flatSheetVolume - (1.0*1.0/2)*0.2 // triangular prism removed
	if got := sheetVolume(body); math.Abs(got-want) > 1e-4 {
		t.Errorf("chamfered volume = %.5f, want %.5f", got, want)
	}
}

// TestCornerRoundRemovesCorner a corner round rolls a fillet off the corner: a valid solid
// whose volume drops by the corner sliver (r² − πr²/4) · thickness.
func TestCornerRoundRemovesCorner(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edge := verticalCornerEdge(t, fs.Result()[0])
	const r = 1.0
	pf := NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		EdgeKeys: [][]byte{edge.ReferenceKey()}, Treatment: CornerRound, Size: func() float64 { return r },
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("corner round sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("rounded corner invalid: %v", r.Issues)
	}
	want := flatSheetVolume - (r*r-math.Pi*r*r/4)*0.2 // corner sliver removed
	if got := sheetVolume(body); math.Abs(got-want)/want > 0.02 {
		t.Errorf("rounded volume = %.5f, want ~%.5f (±2%%)", got, want)
	}
}

// TestCornerChamferTwoDistances a two-distance chamfer cuts an ASYMMETRIC triangle (d1 on one
// face, d2 on the other): the volume drops by d1·d2/2·thickness.
func TestCornerChamferTwoDistances(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edge := verticalCornerEdge(t, fs.Result()[0])
	const d1, d2 = 1.0, 0.5
	pf := NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		EdgeKeys: [][]byte{edge.ReferenceKey()}, Treatment: CornerChamfer, ChamferType: types.ChamferTwoDistances,
		Size: func() float64 { return d1 }, Distance2: func() float64 { return d2 },
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("two-distance corner chamfer sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("two-distance chamfer invalid: %v", r.Issues)
	}
	want := flatSheetVolume - (d1*d2/2)*0.2
	if got := sheetVolume(body); math.Abs(got-want) > 1e-4 {
		t.Errorf("two-distance chamfer volume = %.5f, want %.5f", got, want)
	}
}

// TestCornerChamferDistanceAndAngle a distance-and-angle chamfer sets the second setback from the
// angle (d2 = d1·tan θ), so the volume drops by d1·(d1·tanθ)/2·thickness.
func TestCornerChamferDistanceAndAngle(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edge := verticalCornerEdge(t, fs.Result()[0])
	const d1 = 1.0
	angle := math.Pi / 6 // 30°, tan = 0.5773…
	pf := NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		EdgeKeys: [][]byte{edge.ReferenceKey()}, Treatment: CornerChamfer, ChamferType: types.ChamferDistanceAndAngle,
		Size: func() float64 { return d1 }, Angle: func() float64 { return angle },
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("distance-and-angle corner chamfer sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("distance-and-angle chamfer invalid: %v", r.Issues)
	}
	want := flatSheetVolume - (d1*(d1*math.Tan(angle))/2)*0.2
	if got := sheetVolume(body); math.Abs(got-want) > 1e-4 {
		t.Errorf("distance-and-angle chamfer volume = %.5f, want %.5f", got, want)
	}
}

// TestCornerRoundMultipleRadii a corner round with two edge sets rolls a DIFFERENT radius on each
// corner in one feature: the volume drops by both corner slivers.
func TestCornerRoundMultipleRadii(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edges := verticalCornerEdges(t, fs.Result()[0])
	const r1, r2 = 1.0, 0.5
	pf := NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		Treatment: CornerRound,
		RoundSets: []CornerRoundSet{
			{EdgeKeys: [][]byte{edges[0].ReferenceKey()}, Radius: func() float64 { return r1 }},
			{EdgeKeys: [][]byte{edges[1].ReferenceKey()}, Radius: func() float64 { return r2 }},
		},
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("multi-radius corner round sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("multi-radius round invalid: %v", r.Issues)
	}
	sliver := func(r float64) float64 { return (r*r - math.Pi*r*r/4) * 0.2 }
	want := flatSheetVolume - sliver(r1) - sliver(r2)
	if got := sheetVolume(body); math.Abs(got-want)/want > 0.02 {
		t.Errorf("multi-radius round volume = %.5f, want ~%.5f (±2%%)", got, want)
	}
}

// TestCornerRejectsBadInput a corner with no edges or a non-positive size goes sick.
func TestCornerRejectsBadInput(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	edge := verticalCornerEdge(t, fs.Result()[0])
	noSize := NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		EdgeKeys: [][]byte{edge.ReferenceKey()}, Treatment: CornerChamfer, Size: func() float64 { return 0 },
	})
	fs.Recompute()
	if noSize.Health().OK() {
		t.Error("zero-size corner should be sick")
	}
}

// TestParseCornerTreatment the wire spellings resolve, with an unknown one rejected.
func TestParseCornerTreatment(t *testing.T) {
	for s, want := range map[string]CornerTreatment{"chamfer": CornerChamfer, "round": CornerRound} {
		if got, ok := ParseCornerTreatment(s); !ok || got != want {
			t.Errorf("ParseCornerTreatment(%q) = (%d,%v), want (%d,true)", s, got, ok, want)
		}
	}
	if _, ok := ParseCornerTreatment("seam"); ok {
		t.Error("ParseCornerTreatment(seam) should be rejected here")
	}
}

// TestCornerDefinitionAndKind the accessors return the recipe.
func TestCornerDefinitionAndKind(t *testing.T) {
	def := &SheetMetalCornerDefinition{Treatment: CornerRound}
	f := &SheetMetalCornerFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-corner" {
		t.Error("Definition/Kind mismatch")
	}
}

// TestCornerRoundTrip the corner recipe (edges + treatment + size) marshals and restores.
func TestCornerRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		EdgeKeys: [][]byte{[]byte("k1"), []byte("k2")}, Treatment: CornerRound, Size: func() float64 { return 0.4 },
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalCorner
	if data[0].Kind != "sheet-metal-corner" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-corner", data[0])
	}
	if len(d.Edges) != 2 || d.Treatment != int32(CornerRound) || d.Size != 0.4 {
		t.Errorf("payload = %+v, want 2 edges / round / size 0.4", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-corner" {
		t.Errorf("restored = %d features, want one corner", fresh.Count())
	}
}

// TestCornerVariantRoundTrip a two-distance chamfer and a multi-set round both persist their
// variant fields and restore them.
func TestCornerVariantRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		EdgeKeys: [][]byte{[]byte("k1")}, Treatment: CornerChamfer, ChamferType: types.ChamferTwoDistances,
		Size: func() float64 { return 1 }, Distance2: func() float64 { return 0.5 }, FaceKey: []byte("f1"),
	})
	NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		Treatment: CornerRound,
		RoundSets: []CornerRoundSet{
			{EdgeKeys: [][]byte{[]byte("a")}, Radius: func() float64 { return 1 }},
			{EdgeKeys: [][]byte{[]byte("b")}, Radius: func() float64 { return 0.5 }},
		},
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	ch := data[0].SheetMetalCorner
	if ch.ChamferType != int32(types.ChamferTwoDistances) || ch.Distance2 != 0.5 || ch.FaceKey == "" {
		t.Errorf("chamfer payload = %+v, want twoDistances / 0.5 / a face key", ch)
	}
	if rd := data[1].SheetMetalCorner; len(rd.RoundSets) != 2 || rd.RoundSets[1].Radius != 0.5 {
		t.Errorf("round payload = %+v, want 2 sets / second radius 0.5", rd)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*SheetMetalCornerFeature).Definition()
	if got.ChamferType != types.ChamferTwoDistances || evalFloat(got.Distance2) != 0.5 || string(got.FaceKey) != "f1" {
		t.Errorf("restored chamfer = %+v, want twoDistances / 0.5 / f1", got)
	}
	round := fresh.Item(1).Definition().(*SheetMetalCornerFeature).Definition()
	if len(round.RoundSets) != 2 || evalFloat(round.RoundSets[1].Radius) != 0.5 || string(round.RoundSets[0].EdgeKeys[0]) != "a" {
		t.Errorf("restored round = %+v, want 2 sets / second radius 0.5 / first edge a", round)
	}
}

// TestCornerMissingPayload restoring a corner record with no payload errors.
func TestCornerMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalCorner(NewPartFeatures(nil), nil); err == nil {
		t.Error("restoreSheetMetalCorner(nil) must error")
	}
}

// TestCornerNoEdges a corner with no edge keys goes sick.
func TestCornerNoEdges(t *testing.T) {
	fs, _ := seedSheetMetalSheet(t, 4, nil)
	pf := NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{Treatment: CornerChamfer, Size: func() float64 { return 1 }})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("corner with no edges should be sick")
	}
}
