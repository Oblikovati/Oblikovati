// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Where a flange's wall lands (#1957). Neither control changes the height NUMBER, which is exactly
// why they are easy to get wrong and worth measuring: two flanges with the same height and angle
// but a different position or datum are different parts.

// flangeBody folds the given flange onto a 4 cm square sheet of 2 mm gauge.
func flangeBody(t *testing.T, def *SheetMetalFlangeDefinition) *topo.Body {
	t.Helper()
	pf, fs := flangeFeature(t, def)
	if !pf.Health().OK() {
		t.Fatalf("flange sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("flanged sheet is not a valid solid: %+v", r)
	}
	return body
}

// flangeFeature adds the flange and recomputes, without demanding success.
func flangeFeature(t *testing.T, def *SheetMetalFlangeDefinition) (*PartFeature, *PartFeatures) {
	t.Helper()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	def.EdgeKey = edge.ReferenceKey()
	pf := NewSheetMetalFlangeFeatures(fs).Add(def)
	fs.Recompute()
	return pf, fs
}

// overhang returns how far a body reaches past the seed sheet's flanged edge. The seed's picked
// edge lies at y=0 and the flange folds outward in −Y, so the overhang is how far below y=0 the
// body goes.
func overhang(body *topo.Body) float64 {
	most := 0.0
	for _, v := range body.Vertices() {
		if d := -float64(v.Point().Y); d > most {
			most = d
		}
	}
	return most
}

// TestBendPositionSetsTheWallBack: the default starts the bend AT the edge, so a 90° wall's outer
// face overhangs the part by radius+thickness. The flush positions set the section back by exactly
// that much (or by the radius alone), which is the whole point — the wall stops overhanging.
func TestBendPositionSetsTheWallBack(t *testing.T) {
	const radius, thickness = 0.3, 0.2
	base := func(p BendPosition) *SheetMetalFlangeDefinition {
		return &SheetMetalFlangeDefinition{
			Height: constClosure(1.0), Radius: constClosure(radius), Position: p,
		}
	}
	adjacent := overhang(flangeBody(t, base(BendAtAdjacentFace)))
	outside := overhang(flangeBody(t, base(BendOutsideBaseFace)))
	inside := overhang(flangeBody(t, base(BendInsideBendFace)))

	if want := radius + thickness; stdmath.Abs(adjacent-want) > 1e-6 {
		t.Errorf("adjacent-face flange overhangs by %.4f, want %.4f (r+t)", adjacent, want)
	}
	if stdmath.Abs(outside) > 1e-6 {
		t.Errorf("outside-base-face flange overhangs by %.4f, want 0 — it should finish flush", outside)
	}
	if stdmath.Abs(inside-thickness) > 1e-6 {
		t.Errorf("inside-bend-face flange overhangs by %.4f, want %.4f (the thickness)", inside, thickness)
	}
}

// TestBendPositionOffsetMovesItFurther: the two offset positions are the flush ones plus an
// explicit distance, so the wall moves INTO the part by that much.
func TestBendPositionOffsetMovesItFurther(t *testing.T) {
	flush := flangeBody(t, &SheetMetalFlangeDefinition{
		Height: constClosure(1.0), Radius: constClosure(0.3), Position: BendOutsideBaseFace,
	})
	offset := flangeBody(t, &SheetMetalFlangeDefinition{
		Height: constClosure(1.0), Radius: constClosure(0.3),
		Position: BendOuterEdgeOffset, PositionOffset: constClosure(0.5),
	})
	// Flush already stops at the edge; the offset takes the wall a further 0.5 INTO the part, so
	// the outer face is no longer at the edge and the part loses the material the wall displaces.
	if got := overhang(offset); got > 1e-6 {
		t.Errorf("offset flange overhangs by %.4f, want none — it sits inside the part", got)
	}
	if wallY := wallOuterFaceY(offset); stdmath.Abs(wallY-0.5) > 1e-6 {
		t.Errorf("offset flange's wall stands at y=%.4f, want 0.5 (flush plus the 0.5 offset)", wallY)
	}
	if wallY := wallOuterFaceY(flush); stdmath.Abs(wallY) > 1e-6 {
		t.Errorf("flush flange's wall stands at y=%.4f, want the edge at 0", wallY)
	}
}

// wallOuterFaceY returns where the raised wall stands: the smallest y among the body's vertices
// that are well above the sheet, which is the wall's outer face.
func wallOuterFaceY(body *topo.Body) float64 {
	least := stdmath.Inf(1)
	for _, v := range body.Vertices() {
		if float64(v.Point().Z) > 0.5 && float64(v.Point().Y) < least {
			least = float64(v.Point().Y)
		}
	}
	return least
}

// TestHeightDatumChangesTheWallLength: the height is the same number in every case; what changes
// is where it starts. The outer datum spends the most inside the bend, so it builds the shortest
// wall — a flange that ignored the datum would come out the tangent length every time.
func TestHeightDatumChangesTheWallLength(t *testing.T) {
	const radius, thickness, height = 0.3, 0.2, 1.0
	top := func(d HeightDatum) float64 {
		return topZOf(flangeBody(t, &SheetMetalFlangeDefinition{
			Height: constClosure(height), Radius: constClosure(radius), HeightDatum: d,
		}))
	}
	// On a 90° bend the section finishes the turn lying flat, so the bend leaves the wall starting
	// at the sheet face (z=0.2) plus the radius; the top is that plus the straight run. The
	// setbacks at tan(45°)=1 are r+t for the outer datum and r for the inner.
	tangent, outer, inner := top(HeightFromTangent), top(HeightFromOuterFace), top(HeightFromInnerFace)
	base := 0.2 + radius
	if want := base + height; stdmath.Abs(tangent-want) > 1e-6 {
		t.Errorf("tangent-datum flange top = %.4f, want %.4f", tangent, want)
	}
	if want := base + height - (radius + thickness); stdmath.Abs(outer-want) > 1e-6 {
		t.Errorf("outer-datum flange top = %.4f, want %.4f (r+t of the height is inside the bend)", outer, want)
	}
	if want := base + height - radius; stdmath.Abs(inner-want) > 1e-6 {
		t.Errorf("inner-datum flange top = %.4f, want %.4f", inner, want)
	}
}

// TestOrthoHeightDatumMeasuresPerpendicular: on a right-angle bend the ortho datums agree with
// their along-wall twins, and on any other angle they do not — which is the only reason Inventor
// carries both. A 45° bend is where an implementation that ignored the distinction shows up.
func TestOrthoHeightDatumMeasuresPerpendicular(t *testing.T) {
	const radius, thickness, height = 0.3, 0.2, 1.0
	at := func(d HeightDatum, angle float64) float64 {
		return topZOf(flangeBody(t, &SheetMetalFlangeDefinition{
			Height: constClosure(height), Radius: constClosure(radius),
			Angle: constClosure(angle), HeightDatum: d,
		}))
	}
	right := stdmath.Pi / 2
	if a, b := at(HeightFromOuterFace, right), at(HeightFromOuterFaceOrtho, right); stdmath.Abs(a-b) > 1e-6 {
		t.Errorf("on a right-angle bend the outer and outer-ortho datums differ: %.4f vs %.4f", a, b)
	}
	// At 45° the wall climbs at sin(45°), so an orthogonal height needs a wall √2 times as long.
	const eighth = stdmath.Pi / 4
	along, ortho := at(HeightFromOuterFace, eighth), at(HeightFromOuterFaceOrtho, eighth)
	if !(ortho > along+0.2) {
		t.Errorf("at 45° the ortho datum (%.4f) should build a much taller wall than the along-wall one (%.4f)", ortho, along)
	}
	// The outer corner is where the base's OUTER face (the sheet's underside, z=0) meets the wall's
	// outer face, so an orthogonal height of 1.0 puts the wall's outer end at z=1.0. topZOf reads
	// the INNER end, which on a 45° wall stands a further t·cos(45°) above it.
	if want := height + thickness*stdmath.Cos(eighth); stdmath.Abs(ortho-want) > 1e-6 {
		t.Errorf("outer-ortho flange top = %.4f, want %.4f (1.0 above the outer corner at z=0)", ortho, want)
	}
}

// TestHeightSwallowedByTheBendIsRefused: measured from the outer corner, a height smaller than the
// bend's own setback leaves no wall at all. Clamping it to zero would build a bare bend and call it
// a flange of that height.
func TestHeightSwallowedByTheBendIsRefused(t *testing.T) {
	pf, _ := flangeFeature(t, &SheetMetalFlangeDefinition{
		Height: constClosure(0.2), Radius: constClosure(0.5), HeightDatum: HeightFromOuterFace,
	})
	if pf.Health().OK() {
		t.Error("a height entirely inside the bend should be refused, not clamped")
	}
}

// TestOrthoHeightOnAFlatFoldIsRefused: a 180° fold's wall runs back along the base face, so it
// never rises — an orthogonal height has nothing to measure and the division would blow up.
func TestOrthoHeightOnAFlatFoldIsRefused(t *testing.T) {
	pf, _ := flangeFeature(t, &SheetMetalFlangeDefinition{
		Height: constClosure(1.0), Radius: constClosure(0.3),
		Angle: constClosure(stdmath.Pi), HeightDatum: HeightFromInnerFaceOrtho,
	})
	if pf.Health().OK() {
		t.Error("an orthogonal height on a flat fold should be refused")
	}
}

// TestFlangePlacementRoundTrips: the position and datum decide the part's dimensions, so losing
// them on reopen would rebuild a different part at the same stated height.
func TestFlangePlacementRoundTrips(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: []byte("edge"), Height: constClosure(1.0),
		Position: BendInnerEdgeOffset, PositionOffset: constClosure(0.25),
		HeightDatum: HeightFromOuterFaceOrtho,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalFlange
	if d.BendPosition != "innerEdgeOffset" || d.PositionOffset != 0.25 || d.HeightDatum != "outerOrtho" {
		t.Fatalf("serialized flange = %+v, want the offset position and ortho datum", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*SheetMetalFlangeFeature).Definition()
	if def.Position != BendInnerEdgeOffset || def.HeightDatum != HeightFromOuterFaceOrtho {
		t.Errorf("restored flange = position %d datum %d, want the offset position and ortho datum",
			def.Position, def.HeightDatum)
	}
	if got := evalFloat(def.PositionOffset); got != 0.25 {
		t.Errorf("restored position offset = %g, want 0.25", got)
	}
}

// TestUnknownPlacementNamesAreRefused: a misspelled position or datum must not fall back to the
// default and quietly build the part at different dimensions.
func TestUnknownPlacementNamesAreRefused(t *testing.T) {
	if _, ok := ParseBendPosition("tangentToSideFace"); ok {
		t.Error("a position this build does not implement should not resolve")
	}
	if _, ok := ParseHeightDatum("outside"); ok {
		t.Error(`ParseHeightDatum("outside") should not resolve`)
	}
	if p, ok := ParseBendPosition(""); !ok || p != BendAtAdjacentFace {
		t.Errorf("the empty position should be the adjacent face, got %v/%v", p, ok)
	}
	if d, ok := ParseHeightDatum(""); !ok || d != HeightFromTangent {
		t.Errorf("the empty datum should be the tangent, got %v/%v", d, ok)
	}
}
