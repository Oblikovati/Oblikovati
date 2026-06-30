// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// sheetMetalParams returns a parameter set carrying a Thickness parameter at the given
// expression — the minimum a sheet-metal Face needs to recompute.
func sheetMetalParams(t *testing.T, thicknessExpr string) *param.Parameters {
	t.Helper()
	ps := param.NewParameters()
	if _, err := ps.AddUserParameter("Thickness", thicknessExpr); err != nil {
		t.Fatalf("add Thickness: %v", err)
	}
	return ps
}

// TestSheetMetalFaceCreatesWall a base Face thickens a square profile into a watertight
// prism whose volume is side²·thickness at the rule's gauge.
func TestSheetMetalFaceCreatesWall(t *testing.T) {
	ps := sheetMetalParams(t, "2 mm") // 0.2 cm
	fs := NewPartFeatures(ps)
	pf := NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})

	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("face went sick: %+v", pf.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("face result is not a single solid (%d bodies)", len(bodies))
	}
	if len(bodies[0].Faces()) != 6 {
		t.Errorf("wall has %d faces, want 6 (a prism)", len(bodies[0].Faces()))
	}
	if open := ops.BoundaryEdges(bodies[0]); len(open) != 0 {
		t.Errorf("wall has %d boundary edges, want 0 (watertight)", len(open))
	}
	want := 4.0 * 4.0 * 0.2 // side²·thickness (cm³)
	if got := ops.BodyGeometryProperties(bodies[0], ops.Quality{ChordTolerance: 1e-4}).Volume; math.Abs(got-want) > 1e-6 {
		t.Errorf("wall volume = %.6f cm³, want %.6f", got, want)
	}
}

// TestSheetMetalFaceIsParameterBacked editing the Thickness parameter changes the wall's
// volume on the next recompute — the gauge follows the parameter, not a frozen value.
func TestSheetMetalFaceIsParameterBacked(t *testing.T) {
	ps := sheetMetalParams(t, "1 mm")
	thickness, _ := ps.ByName("Thickness")
	fs := NewPartFeatures(ps)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})

	fs.Recompute()
	thin := ops.BodyGeometryProperties(fs.Result()[0], ops.Quality{ChordTolerance: 1e-4}).Volume

	if err := ps.SetExpression(thickness.ID(), "3 mm"); err != nil {
		t.Fatalf("set Thickness: %v", err)
	}
	fs.MarkAllDirty() // a parameter edit invalidates the whole program (as the host does)
	fs.Recompute()
	thick := ops.BodyGeometryProperties(fs.Result()[0], ops.Quality{ChordTolerance: 1e-4}).Volume

	if !(thick > thin*2.5) { // 3× thickness ⇒ ~3× volume
		t.Errorf("thicker gauge did not grow the wall: 1mm=%.4f 3mm=%.4f", thin, thick)
	}
}

// TestSheetMetalFaceNeedsThickness a Face without a Thickness parameter goes sick with a
// clear message rather than building a degenerate wall.
func TestSheetMetalFaceNeedsThickness(t *testing.T) {
	fs := NewPartFeatures(param.NewParameters()) // no Thickness parameter
	pf := NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	fs.Recompute()
	if pf.Health().OK() {
		t.Fatal("face without a Thickness parameter should be sick")
	}
}

// TestSheetMetalFaceRoundTrip the Face recipe (profile + direction + operation) marshals and
// restores in-package, preserving the kind and payload.
func TestSheetMetalFaceRoundTrip(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: sk, ProfileIndex: 0, Direction: NegativeDir, Operation: ops.Join})

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if len(data) != 1 || data[0].Kind != "sheet-metal-face" || data[0].SheetMetalFace == nil {
		t.Fatalf("marshaled = %+v, want one sheet-metal-face with payload", data)
	}
	if data[0].SheetMetalFace.Direction != int32(NegativeDir) || data[0].SheetMetalFace.Operation != int32(ops.Join) {
		t.Errorf("payload = %+v, want negative/join", data[0].SheetMetalFace)
	}

	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-face" {
		t.Errorf("restored program = %d features, want one sheet-metal-face", fresh.Count())
	}
}

// TestSheetMetalFaceMissingPayload restoring a sheet-metal-face record with no payload errors.
func TestSheetMetalFaceMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalFace(NewPartFeatures(nil), nil, oneSketch{}); err == nil {
		t.Error("restoreSheetMetalFace(nil) must error")
	}
}

// TestSheetMetalFaceSerializeUnknownSketch serializing a Face whose sketch is not in the part
// errors rather than silently dropping the profile reference.
func TestSheetMetalFaceSerializeUnknownSketch(t *testing.T) {
	def := &SheetMetalFaceDefinition{Sketch: sketch.NewSketches().Add(sketch.XYPlane()), ProfileIndex: 0}
	// oneSketch over a *different* sketch ⇒ IndexOf returns false.
	if _, err := serializeSheetMetalFace(def, oneSketch{s: sketch.NewSketches().Add(sketch.XYPlane())}); err == nil {
		t.Error("serializeSheetMetalFace with an unknown sketch must error")
	}
}

// TestSheetThicknessEdgeCases the thickness reader rejects nil parameters and a non-positive
// gauge.
func TestSheetThicknessEdgeCases(t *testing.T) {
	if _, err := sheetThickness(nil); err == nil {
		t.Error("sheetThickness(nil) must error")
	}
	ps := param.NewParameters()
	if _, err := ps.AddUserParameter("Thickness", "0 mm"); err != nil {
		t.Fatalf("add Thickness: %v", err)
	}
	if _, err := sheetThickness(ps); err == nil {
		t.Error("sheetThickness with zero gauge must error")
	}
}

// TestSheetMetalFaceDirections each material side (positive/negative/symmetric) thickens the
// profile into a valid single solid of the same volume, on the chosen side of the plane.
func TestSheetMetalFaceDirections(t *testing.T) {
	want := 4.0 * 4.0 * 0.2 // side²·thickness
	for _, dir := range []ExtentDirection{PositiveDir, NegativeDir, SymmetricDir} {
		ps := sheetMetalParams(t, "2 mm")
		fs := NewPartFeatures(ps)
		def := &SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Direction: dir, Operation: ops.NewBody}
		pf := NewSheetMetalFaceFeatures(fs).Add(def)
		if pf.Definition() == nil {
			t.Fatal("Definition() returned nil")
		}
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("direction %d: face sick %+v", dir, pf.Health())
		}
		bodies := fs.Result()
		if len(bodies) != 1 || !bodies[0].IsSolid() {
			t.Fatalf("direction %d: not a single solid", dir)
		}
		if got := ops.BodyGeometryProperties(bodies[0], ops.Quality{ChordTolerance: 1e-4}).Volume; math.Abs(got-want) > 1e-6 {
			t.Errorf("direction %d: volume = %.6f, want %.6f", dir, got, want)
		}
	}
}

// TestSheetMetalFaceDefinitionAccessor Definition returns the stored recipe.
func TestSheetMetalFaceDefinitionAccessor(t *testing.T) {
	def := &SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody}
	f := &SheetMetalFaceFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-face" {
		t.Error("Definition/Kind mismatch")
	}
}

// TestSheetMetalFaceSecondaryJoins a second Face with Join merges into the running sheet
// rather than starting a new body.
func TestSheetMetalFaceSecondaryJoins(t *testing.T) {
	ps := sheetMetalParams(t, "2 mm")
	fs := NewPartFeatures(ps)
	faces := NewSheetMetalFaceFeatures(fs)
	faces.Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	// A larger coplanar square over the same plane: joining unions the two prisms into one sheet.
	faces.Add(&SheetMetalFaceDefinition{Sketch: squareSketch(6), ProfileIndex: 0, Operation: ops.Join})

	fs.Recompute()
	if bodies := fs.Result(); len(bodies) != 1 {
		t.Errorf("secondary Face produced %d bodies, want 1 merged sheet", len(bodies))
	}
}
