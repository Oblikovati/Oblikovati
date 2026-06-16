// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// sheetForBend builds a square sheet (side cm, 2 mm thick) on XY with a Thickness parameter,
// plus a bend-line sketch crossing it at x=mid, and returns the engine, the bend-line sketch,
// and the feature engine ready to add a bend.
func sheetForBend(t *testing.T, side, mid float64) (*PartFeatures, *sketch.Sketch) {
	t.Helper()
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	mustParam(t, ps, "BendRadius", "2 mm")
	fs := NewPartFeatures(ps, nil)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(side), ProfileIndex: 0, Operation: ops.NewBody})

	line := sketch.NewSketches().Add(sketch.XYPlane())
	line.Lines().AddByTwoPoints(math.P2(mid, 0), math.P2(mid, side)) // across the sheet at x=mid
	return fs, line
}

// TestSheetMetalBendFoldsValidSolid bending a flat sheet along a line over the rule radius
// yields one valid solid that folded up.
func TestSheetMetalBendFoldsValidSolid(t *testing.T) {
	fs, line := sheetForBend(t, 4, 2)
	pf := NewSheetMetalBendFeatures(fs).Add(&SheetMetalBendDefinition{
		Sketch: line, LineIndex: 0, Angle: func() float64 { return stdmath.Pi / 2 }, // radius from rule
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("bend sick: %+v", pf.Health())
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("bend result = %d bodies, want 1", len(fs.Result()))
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("bent sheet invalid: %v", r.Issues)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("bent sheet has %d boundary edges, want watertight", len(open))
	}
	// The half-sheet beyond the line folded up out of the XY plane.
	if box := body.RangeBox(); box.Max.Z < 1.0 {
		t.Errorf("bent top Z = %g, want the folded half to rise (>1)", box.Max.Z)
	}
}

// TestSheetMetalBendUsesRuleRadius with no radius override the bend reads the rule's
// BendRadius parameter (a missing parameter ⇒ sick), and the 90° default applies.
func TestSheetMetalBendUsesRuleRadius(t *testing.T) {
	// No BendRadius parameter ⇒ radius resolves to 0 ⇒ sick.
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps, nil)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	line := sketch.NewSketches().Add(sketch.XYPlane())
	line.Lines().AddByTwoPoints(math.P2(2, 0), math.P2(2, 4))
	pf := NewSheetMetalBendFeatures(fs).Add(&SheetMetalBendDefinition{Sketch: line, LineIndex: 0})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("bend with no radius override and no BendRadius parameter should be sick")
	}
}

// TestSheetMetalBendDefinitionAndKind the accessors return the recipe.
func TestSheetMetalBendDefinitionAndKind(t *testing.T) {
	def := &SheetMetalBendDefinition{LineIndex: 1}
	f := &SheetMetalBendFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-bend" {
		t.Error("Definition/Kind mismatch")
	}
}

// TestSheetMetalBendRoundTrip the bend recipe (sketch + line + angle + radius + flip)
// marshals and restores, preserving the kind and payload.
func TestSheetMetalBendRoundTrip(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	fs := NewPartFeatures(nil, nil)
	NewSheetMetalBendFeatures(fs).Add(&SheetMetalBendDefinition{
		Sketch: sk, LineIndex: 0,
		Angle: func() float64 { return stdmath.Pi / 3 }, Radius: func() float64 { return 0.3 }, Flip: true,
	})
	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalBend
	if data[0].Kind != "sheet-metal-bend" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-bend", data[0])
	}
	if d.Line != 0 || stdmath.Abs(d.Angle-stdmath.Pi/3) > 1e-9 || d.Radius != 0.3 || !d.Flip {
		t.Errorf("payload = %+v, want line 0 / angle π/3 / radius 0.3 / flip", d)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-bend" {
		t.Errorf("restored = %d features, want one bend", fresh.Count())
	}
}

// TestSheetMetalBendSerializeUnknownSketch serializing a bend whose sketch is not in the part
// errors rather than dropping the line reference.
func TestSheetMetalBendSerializeUnknownSketch(t *testing.T) {
	def := &SheetMetalBendDefinition{Sketch: sketch.NewSketches().Add(sketch.XYPlane())}
	if _, err := serializeSheetMetalBend(def, oneSketch{s: sketch.NewSketches().Add(sketch.XYPlane())}); err == nil {
		t.Error("serializeSheetMetalBend with an unknown sketch must error")
	}
}

// TestSheetMetalBendMissingPayload restoring a bend record with no payload errors.
func TestSheetMetalBendMissingPayload(t *testing.T) {
	if _, err := restoreSheetMetalBend(NewPartFeatures(nil, nil), nil, oneSketch{}); err == nil {
		t.Error("restoreSheetMetalBend(nil) must error")
	}
}
