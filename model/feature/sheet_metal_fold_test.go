// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// foldedSheet builds a square sheet, draws a fold line across it at x=mid, and folds it 90°
// over the rule radius at the given bend location. Returns the result body.
func foldedSheet(t *testing.T, side, mid float64, loc BendLocation) *topo.Body {
	t.Helper()
	fs, _ := seedSheetMetalSheet(t, side, map[string]string{"BendRadius": "2 mm"})
	line := sketch.NewSketches().Add(sketch.XYPlane())
	line.Lines().AddByTwoPoints(math.P2(mid, 0), math.P2(mid, side))
	pf := NewSheetMetalFoldFeatures(fs).Add(&SheetMetalFoldDefinition{
		Sketch: line, LineIndex: 0, Angle: func() float64 { return stdmath.Pi / 2 }, Location: loc,
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("fold (loc %d) sick: %+v", loc, pf.Health())
	}
	return fs.Result()[0]
}

// TestFoldFoldsValidSolid a fold along a line over the rule radius yields one valid watertight
// solid that rises out of the plane.
func TestFoldFoldsValidSolid(t *testing.T) {
	t.Parallel()
	body := foldedSheet(t, 4, 2, CenterlineOfBend)
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("folded sheet invalid: %v", r.Issues)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("folded sheet has %d boundary edges, want watertight", len(open))
	}
	if box := body.RangeBox(); box.Max.Z < 1.0 {
		t.Errorf("folded top Z = %g, want the folded half to rise (>1)", box.Max.Z)
	}
}

// TestFoldBendLocationShiftsTheFold the three bend locations place the fold at different
// positions along the sheet (the folded half's extent in +X differs), proving the location
// actually moves the bend.
func TestFoldBendLocationShiftsTheFold(t *testing.T) {
	t.Parallel()
	// Measure where the folded (raised) material begins in X for each location: the min X of
	// any vertex above the flat sheet.
	raisedMinX := func(loc BendLocation) float64 {
		body := foldedSheet(t, 4, 2, loc)
		minX := stdmath.Inf(1)
		for _, v := range body.Vertices() {
			p := v.Point()
			if p.Z > 0.3 && p.X < minX { // above the 2 mm sheet ⇒ part of the fold
				minX = p.X
			}
		}
		return minX
	}
	start := raisedMinX(StartOfBend)
	end := raisedMinX(EndOfBend)
	// End-of-bend shifts the bend toward the fixed (−X) side, so its raised material begins
	// further in −X than start-of-bend.
	if !(end < start) {
		t.Errorf("bend location did not shift the fold: start raised@%.3f, end raised@%.3f (want end < start)", start, end)
	}
}

// TestParseBendLocation the wire spellings resolve, with an unknown one rejected.
func TestParseBendLocation(t *testing.T) {
	t.Parallel()
	for s, want := range map[string]BendLocation{"": CenterlineOfBend, "centerline": CenterlineOfBend, "start": StartOfBend, "end": EndOfBend} {
		if got, ok := ParseBendLocation(s); !ok || got != want {
			t.Errorf("ParseBendLocation(%q) = (%d,%v), want (%d,true)", s, got, ok, want)
		}
	}
	if _, ok := ParseBendLocation("middle"); ok {
		t.Error("ParseBendLocation(middle) should be rejected")
	}
}

// TestFoldNeedsRadius a fold with no radius override and no BendRadius parameter goes sick.
func TestFoldNeedsRadius(t *testing.T) {
	t.Parallel()
	fs, _ := seedSheetMetalSheet(t, 4, nil) // Thickness only, no BendRadius
	line := sketch.NewSketches().Add(sketch.XYPlane())
	line.Lines().AddByTwoPoints(math.P2(2, 0), math.P2(2, 4))
	pf := NewSheetMetalFoldFeatures(fs).Add(&SheetMetalFoldDefinition{Sketch: line, LineIndex: 0})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("fold with no radius should be sick")
	}
}

// TestFoldDefinitionAndKind the accessors return the recipe.
func TestFoldDefinitionAndKind(t *testing.T) {
	t.Parallel()
	def := &SheetMetalFoldDefinition{LineIndex: 2, Location: EndOfBend}
	f := &SheetMetalFoldFeature{def: def}
	if f.Definition() != def || f.Kind() != "sheet-metal-fold" {
		t.Error("Definition/Kind mismatch")
	}
}

// TestFoldRoundTrip the fold recipe (sketch + line + angle + radius + location + flip)
// marshals and restores, preserving the kind and payload.
func TestFoldRoundTrip(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	fs := NewPartFeatures(nil)
	NewSheetMetalFoldFeatures(fs).Add(&SheetMetalFoldDefinition{
		Sketch: sk, LineIndex: 0,
		Angle: func() float64 { return stdmath.Pi / 3 }, Radius: func() float64 { return 0.3 },
		Location: EndOfBend, Flip: true,
	})
	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalFold
	if data[0].Kind != "sheet-metal-fold" || d == nil {
		t.Fatalf("marshaled = %+v, want sheet-metal-fold", data[0])
	}
	if d.Location != int32(EndOfBend) || stdmath.Abs(d.Angle-stdmath.Pi/3) > 1e-9 || d.Radius != 0.3 || !d.Flip {
		t.Errorf("payload = %+v, want end / angle π/3 / radius 0.3 / flip", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-fold" {
		t.Errorf("restored = %d features, want one fold", fresh.Count())
	}
}

// TestFoldMissingPayload restoring a fold record with no payload errors.
func TestFoldMissingPayload(t *testing.T) {
	t.Parallel()
	if _, err := restoreSheetMetalFold(NewPartFeatures(nil), nil, oneSketch{}); err == nil {
		t.Error("restoreSheetMetalFold(nil) must error")
	}
}

// TestFoldSerializeUnknownSketch serializing a fold whose sketch is not in the part errors.
func TestFoldSerializeUnknownSketch(t *testing.T) {
	t.Parallel()
	def := &SheetMetalFoldDefinition{Sketch: sketch.NewSketches().Add(sketch.XYPlane())}
	if _, err := serializeSheetMetalFold(def, oneSketch{s: sketch.NewSketches().Add(sketch.XYPlane())}); err == nil {
		t.Error("serializeSheetMetalFold with an unknown sketch must error")
	}
}
