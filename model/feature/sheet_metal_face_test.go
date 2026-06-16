// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/param"
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
	fs := NewPartFeatures(ps, nil)
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
	fs := NewPartFeatures(ps, nil)
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
	fs := NewPartFeatures(param.NewParameters(), nil) // no Thickness parameter
	pf := NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	fs.Recompute()
	if pf.Health().OK() {
		t.Fatal("face without a Thickness parameter should be sick")
	}
}

// TestSheetMetalFaceSecondaryJoins a second Face with Join merges into the running sheet
// rather than starting a new body.
func TestSheetMetalFaceSecondaryJoins(t *testing.T) {
	ps := sheetMetalParams(t, "2 mm")
	fs := NewPartFeatures(ps, nil)
	faces := NewSheetMetalFaceFeatures(fs)
	faces.Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	// A larger coplanar square over the same plane: joining unions the two prisms into one sheet.
	faces.Add(&SheetMetalFaceDefinition{Sketch: squareSketch(6), ProfileIndex: 0, Operation: ops.Join})

	fs.Recompute()
	if bodies := fs.Result(); len(bodies) != 1 {
		t.Errorf("secondary Face produced %d bodies, want 1 merged sheet", len(bodies))
	}
}
