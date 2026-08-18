// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// sheetForRip builds a 4×4 sheet-metal wall (2 mm thick) on XY plus a rip-line sketch, and
// returns the engine and the rip sketch ready to add a rip.
func sheetForRip(t *testing.T, a, b math.Point2) (*PartFeatures, *sketch.Sketch) {
	t.Helper()
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{Sketch: squareSketch(4), ProfileIndex: 0, Operation: ops.NewBody})
	rip := sketch.NewSketches().Add(sketch.XYPlane())
	rip.Lines().AddByTwoPoints(a, b)
	return fs, rip
}

// TestRipSlitsSheet a rip cuts a slit along the line, removing material and leaving one
// watertight solid (a partial rip line keeps the sheet connected).
func TestRipSlitsSheet(t *testing.T) {
	fs, rip := sheetForRip(t, math.P2(1, 2), math.P2(3, 2)) // a line across the middle, ends inside
	fs.Recompute()
	full := sheetVolume(fs.Result()[0])

	pf := NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{Sketch: rip, LineIndex: 0, Gap: func() float64 { return 0.05 }, GapSide: SymmetricDir})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("rip sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	if v := sheetVolume(body); !(v < full) {
		t.Errorf("rip removed no material: %g vs %g", v, full)
	}
}

// rectSheetMetal builds a w×h sheet-metal wall (2 mm thick) on XY and returns the engine and body.
func rectSheetMetal(t *testing.T, w, h float64) (*PartFeatures, *topo.Body) {
	t.Helper()
	ps := param.NewParameters()
	mustParam(t, ps, "Thickness", "2 mm")
	fs := NewPartFeatures(ps)
	NewSheetMetalFaceFeatures(fs).Add(&SheetMetalFaceDefinition{
		Sketch: rectSketchOn(sketch.XYPlane(), 0, 0, w, h), ProfileIndex: 0, Operation: ops.NewBody,
	})
	fs.Recompute()
	return fs, fs.Result()[0]
}

// flatBottomFace returns the lower large flat face (normal ±Z) of a sheet body — the face a rip
// is picked on.
func flatBottomFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	var best *topo.Face
	var bestZ float64
	for _, f := range body.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || stdmath.Abs(float64(pl.Normal().Z)) < 0.9 {
			continue
		}
		if z := meanVertexZ(f); best == nil || z < bestZ {
			best, bestZ = f, z
		}
	}
	if best == nil {
		t.Fatal("no flat ±Z face on the sheet")
	}
	return best
}

// meanVertexZ is the average Z of a face's vertices, used to pick the bottom of two parallel faces.
func meanVertexZ(f *topo.Face) float64 {
	var s float64
	vs := f.Vertices()
	for _, v := range vs {
		s += float64(v.Point().Z)
	}
	return s / float64(len(vs))
}

// totalRipVolume sums the volume of every body a rip leaves — a full-extent rip splits the sheet
// into two, so the measure must not read a single body.
func totalRipVolume(bodies []*topo.Body) float64 {
	var v float64
	for _, b := range bodies {
		v += sheetVolume(b)
	}
	return v
}

// TestRipGapSidePositionsSlit gapSide places the removed material: ripping a line a hair inside
// the sheet's bottom edge, "positive" takes the whole gap into the sheet, "negative" takes it out
// (so most of the slit falls in empty space), and "symmetric" straddles — three distinct volumes.
func TestRipGapSidePositionsSlit(t *testing.T) {
	const gap, run, thick = 0.2, 2.0, 0.2 // slit ends x∈[1,3]; line at y=0.05 (gap/4 inside the edge)
	cases := map[ExtentDirection]float64{
		PositiveDir:  gap * run * thick,  // slit y∈[0.05,0.25], all inside → full gap deep
		SymmetricDir: 0.15 * run * thick, // slit y∈[-0.05,0.15] → 0.15 inside
		NegativeDir:  0.05 * run * thick, // slit y∈[-0.15,0.05] → 0.05 inside
	}
	for side, wantRemoved := range cases {
		fs, rip := sheetForRip(t, math.P2(1, 0.05), math.P2(3, 0.05))
		fs.Recompute()
		full := sheetVolume(fs.Result()[0])
		pf := NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{
			Sketch: rip, LineIndex: 0, Gap: func() float64 { return gap }, GapSide: side,
		})
		fs.Recompute()
		if !pf.Health().OK() {
			t.Fatalf("gapSide %d rip sick: %+v", side, pf.Health())
		}
		removed := full - totalRipVolume(fs.Result())
		if stdmath.Abs(removed-wantRemoved) > 5e-3 {
			t.Errorf("gapSide %d removed %.5f, want %.5f", side, removed, wantRemoved)
		}
	}
}

// TestRipFaceExtents a face-extents rip slits a face along its full long axis: on a 6×2 wall it
// removes gap×6×thickness and separates the plate into two strips.
func TestRipFaceExtents(t *testing.T) {
	fs, body := rectSheetMetal(t, 6, 2)
	full := totalRipVolume(fs.Result())
	face := flatBottomFace(t, body)
	pf := NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{
		FaceKey: face.ReferenceKey(), Type: FaceExtentsRip, GapSide: SymmetricDir,
		Gap: func() float64 { return 0.2 },
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("face-extents rip sick: %+v", pf.Health())
	}
	removed := full - totalRipVolume(fs.Result())
	if want := 0.2 * 6.0 * 0.2; stdmath.Abs(removed-want) > 5e-3 {
		t.Errorf("face-extents rip removed %.5f, want %.5f (gap×6×thickness)", removed, want)
	}
}

// TestRipSinglePoint a single-point rip runs from a picked corner across the face to the opposite
// boundary, separating the plate and removing a slit of about gap×diagonal×thickness.
func TestRipSinglePoint(t *testing.T) {
	fs, body := rectSheetMetal(t, 6, 2)
	full := totalRipVolume(fs.Result())
	face := flatBottomFace(t, body)
	pf := NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{
		FaceKey: face.ReferenceKey(), Type: SinglePointRip, PointKey: face.Vertices()[0].ReferenceKey(),
		GapSide: SymmetricDir, Gap: func() float64 { return 0.2 },
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("single-point rip sick: %+v", pf.Health())
	}
	// The rip runs the corner-to-corner diagonal (length √(6²+2²)); a slit of gap width along it
	// removes about gap·diagonal·thickness, a little less where the slit is clipped at the corners.
	removed := full - totalRipVolume(fs.Result())
	upper := 0.2 * stdmath.Hypot(6, 2) * 0.2
	if removed < 0.75*upper || removed > upper {
		t.Errorf("single-point rip removed %.5f, want (%.5f, %.5f]", removed, 0.75*upper, upper)
	}
}

// TestRipFaceRoundTrip a face-based rip persists its face/point keys, type and gap side (with no
// sketch) and restores them.
func TestRipFaceRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{
		FaceKey: []byte("f1"), PointKey: []byte("p1"), Type: SinglePointRip,
		GapSide: NegativeDir, Gap: constFloat(0.03),
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	d := data[0].SheetMetalRip
	if d == nil || d.Sketch != -1 || d.Type != int32(SinglePointRip) || d.GapSide != "negative" || d.Face == "" {
		t.Fatalf("face rip payload = %+v, want sketch=-1 / singlePoint / negative / a face key", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*SheetMetalRipFeature).Definition()
	if got.Type != SinglePointRip || got.GapSide != NegativeDir || string(got.FaceKey) != "f1" || string(got.PointKey) != "p1" {
		t.Errorf("restored rip = %+v, want singlePoint / negative / f1 / p1", got)
	}
}

// TestRipRejectsBadInput a rip with an out-of-range line index, or a non-positive gap, is sick.
func TestRipRejectsBadInput(t *testing.T) {
	fs, rip := sheetForRip(t, math.P2(1, 2), math.P2(3, 2))
	bad := NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{Sketch: rip, LineIndex: 9, Gap: func() float64 { return 0.05 }})
	fs.Recompute()
	if bad.Health().OK() {
		t.Error("rip with an out-of-range line index should be sick")
	}

	fs2, rip2 := sheetForRip(t, math.P2(1, 2), math.P2(3, 2))
	zero := NewSheetMetalRipFeatures(fs2).Add(&SheetMetalRipDefinition{Sketch: rip2, LineIndex: 0, Gap: func() float64 { return 0 }})
	fs2.Recompute()
	if zero.Health().OK() {
		t.Error("rip with a non-positive gap should be sick")
	}
}

// TestRipRoundTrip a rip persists its line + gap and restores.
func TestRipRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(1, 2), math.P2(3, 2))
	NewSheetMetalRipFeatures(fs).Add(&SheetMetalRipDefinition{Sketch: sk, LineIndex: 0, Gap: constFloat(0.05), GapSide: SymmetricDir})

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if data[0].Kind != "sheet-metal-rip" || data[0].SheetMetalRip == nil || data[0].SheetMetalRip.Gap != 0.05 {
		t.Fatalf("marshaled = %+v", data[0])
	}
	if d := data[0].SheetMetalRip; d.GapSide != "symmetric" || d.Sketch < 0 {
		t.Errorf("sketch rip payload = %+v, want symmetric side / a sketch index", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "sheet-metal-rip" {
		t.Errorf("restored %d features, want one rip", fresh.Count())
	}
}
