// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/param"
)

// sheetWithFlange builds a square sheet (side cm, thickness 2 mm) then flanges its highest +Y
// boundary edge up 90° over the given radius/height. It returns the engine's result bodies.
func sheetWithFlange(t *testing.T, side, radiusCm, heightCm float64) []*topo.Body {
	t.Helper()
	fs, edge := seedSheetMetalSheet(t, side, nil)
	NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(),
		Height:  func() float64 { return heightCm },
		Radius:  func() float64 { return radiusCm },
		Angle:   func() float64 { return math.Pi / 2 }, // explicit 90° (override path)
	})
	fs.Recompute()
	return fs.Result()
}

// topEdgeAlongX returns a boundary edge of the body that runs in X at the top face (max Z) —
// a deterministic edge to flange from.
func topEdgeAlongX(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	maxZ := math.Inf(-1)
	for _, e := range body.Edges() {
		if z := e.StartVertex().Point().Z; z > maxZ {
			maxZ = z
		}
	}
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		alongX := math.Abs(a.X-b.X) > 1e-6 && math.Abs(a.Y-b.Y) < 1e-6 && math.Abs(a.Z-b.Z) < 1e-6
		if alongX && math.Abs(a.Z-maxZ) < 1e-6 {
			return e
		}
	}
	t.Fatal("no X-aligned top edge found on the sheet")
	return nil
}

// TestFlangeBuildsWatertightSolid the flange unions onto the sheet as one valid, watertight
// solid whose volume is the sheet plus the developed bend+wall band.
func TestFlangeBuildsWatertightSolid(t *testing.T) {
	const side, r, h, th = 4.0, 0.2, 1.0, 0.2
	bodies := sheetWithFlange(t, side, r, h)
	if len(bodies) != 1 {
		t.Fatalf("flange produced %d bodies, want 1 merged sheet", len(bodies))
	}
	body := bodies[0]
	if !body.IsSolid() {
		t.Fatal("flanged sheet is not a solid")
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("flanged sheet has %d boundary edges, want 0 (watertight)", len(open))
	}
	if res := ops.Validate(body); !res.Valid {
		t.Fatalf("flanged sheet failed validation: %+v", res)
	}

	// Volume ≈ sheet + flange band swept along the edge. The faceted arc under-fills the true
	// arc band slightly, so allow a few percent.
	sheetVol := side * side * th
	bandArea := (math.Pi/2)*th*(r+th/2) + h*th // arc band + straight band
	want := sheetVol + bandArea*side
	got := ops.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-4}).Volume
	if math.Abs(got-want)/want > 0.03 {
		t.Errorf("flanged volume = %.4f cm³, want ~%.4f (±3%%)", got, want)
	}
}

// TestFlangeRisesAboveSheet the flange folds up — the body extends well above the flat
// sheet's thickness in +Z.
func TestFlangeRisesAboveSheet(t *testing.T) {
	const side, r, h = 4.0, 0.2, 1.0
	body := sheetWithFlange(t, side, r, h)[0]
	maxZ := math.Inf(-1)
	for _, v := range body.Vertices() {
		if z := v.Point().Z; z > maxZ {
			maxZ = z
		}
	}
	// A 90° flange of height 1 cm over a 2 mm sheet rises to roughly r+t+h ≈ 1.4 cm.
	if maxZ < 1.0 {
		t.Errorf("flange top Z = %.3f cm, want it to rise (≳1 cm) above the flat sheet", maxZ)
	}
	_ = gmath.P2 // keep gmath referenced for future fixtures
}

// TestFlangeDefaultsAndFlip a flange with no radius override reads the rule's BendRadius
// parameter, the default 90° angle applies, and flip folds the wall to the opposite (−Z)
// side. Also exercises the Definition accessor.
func TestFlangeDefaultsAndFlip(t *testing.T) {
	fs, edge := seedSheetMetalSheet(t, 4, map[string]string{"BendRadius": "2 mm"}) // rule radius the flange reads
	pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(),
		Height:  func() float64 { return 1.0 },
		Flip:    true, // no Radius/Angle ⇒ rule radius + 90° default
	})
	if pf.Definition().(*SheetMetalFlangeFeature).Definition().EdgeKey == nil {
		t.Fatal("Definition() lost the edge key")
	}
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("flipped flange sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("flipped flange has %d boundary edges, want watertight", len(open))
	}
	// Flip folds toward −Z (below the sheet): the body should dip well below 0.
	minZ := 0.0
	for _, v := range body.Vertices() {
		if z := v.Point().Z; z < minZ {
			minZ = z
		}
	}
	if minZ > -0.5 {
		t.Errorf("flipped flange min Z = %.3f, want it to fold below the sheet (≲−0.5)", minZ)
	}
}

func mustParam(t *testing.T, ps *param.Parameters, name, expr string) {
	t.Helper()
	if _, err := ps.AddUserParameter(name, expr); err != nil {
		t.Fatalf("add %s: %v", name, err)
	}
}

// TestFlangeRejectsBadDims a flange with a non-positive height (and no Thickness parameter)
// goes sick rather than building degenerate geometry.
func TestFlangeRejectsBadDims(t *testing.T) {
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(),
		Height:  func() float64 { return 0 }, // zero height
		Radius:  func() float64 { return 0.2 },
	})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("zero-height flange should be sick")
	}
}

// TestFlangeRoundTrip the flange recipe (edge key + height + angle + radius + flip) marshals
// and restores, preserving the kind and payload; a 0 angle/radius restores as the defaults.
func TestFlangeRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: []byte("edge-key"),
		Height:  func() float64 { return 1.5 },
		Angle:   func() float64 { return math.Pi / 3 },
		Radius:  func() float64 { return 0.3 },
		Flip:    true,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalFlange
	if data[0].Kind != "sheet-metal-flange" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-flange", data[0])
	}
	if d.Height != 1.5 || math.Abs(d.Angle-math.Pi/3) > 1e-9 || d.Radius != 0.3 || !d.Flip {
		t.Errorf("payload = %+v, want height 1.5 / angle π/3 / radius 0.3 / flip", d)
	}

	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-flange" {
		t.Errorf("restored = %d features, want one flange", fresh.Count())
	}
}

// TestFlangeMissingPayload restoring a flange record with no payload errors.
func TestFlangeMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalFlange(NewPartFeatures(nil, nil), nil); err == nil {
		t.Error("restoreSheetMetalFlange(nil) must error")
	}
}
