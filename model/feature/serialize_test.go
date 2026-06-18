// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// oneSketch is a SketchIndexer over a single sketch at index 0 — enough to round-trip
// a feature program that consumes one sketch.
type oneSketch struct{ s *sketch.Sketch }

func (o oneSketch) IndexOf(s *sketch.Sketch) (int, bool) { return 0, s == o.s }
func (o oneSketch) At(i int) (*sketch.Sketch, bool) {
	if i == 0 {
		return o.s, true
	}
	return nil, false
}

func TestExtrudeFeatureRoundTrip(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if len(data) != 1 || data[0].Kind != "extrude" || data[0].Extrude == nil {
		t.Fatalf("marshaled = %+v, want one extrude with payload", data)
	}
	if data[0].Extrude.Distance != 5 {
		t.Errorf("distance = %v, want 5", data[0].Extrude.Distance)
	}

	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 || fresh.Item(0).Kind() != "extrude" {
		t.Errorf("restored program = %d features, want one extrude", fresh.Count())
	}
}

// fakeFeature is a feature kind with no codec, used to prove Save errors rather than
// dropping it.
type fakeFeature struct{}

func (fakeFeature) Kind() string                    { return "fake" }
func (fakeFeature) Recompute(Input) (Output, error) { return Output{}, nil }

func TestDressUpFeaturesRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	du := NewDressUpFeatures(fs)
	du.AddFillet([][]byte{[]byte("edge-a"), []byte("edge-b")}, func() float64 { return 0.5 })
	du.AddChamfer([][]byte{[]byte("edge-c")}, func() float64 { return 0.3 })
	du.AddShell([][]byte{[]byte("face-a")}, func() float64 { return 2 })
	du.AddDraft([][]byte{[]byte("face-b")}, func() float64 { return 0.1 })
	du.AddThread([]byte("face-c"), "M6x1", false)

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 5 {
		t.Fatalf("feature count after round trip = %d, want 5", fresh.Count())
	}

	// The fillet's edge keys and radius must survive byte-for-byte / value-for-value.
	fillet := fresh.Item(0).Definition().(*FilletFeature).Definition()
	if len(fillet.EdgeKeys) != 2 || string(fillet.EdgeKeys[0]) != "edge-a" || string(fillet.EdgeKeys[1]) != "edge-b" {
		t.Errorf("fillet edge keys = %v, want [edge-a edge-b]", fillet.EdgeKeys)
	}
	if fillet.Radius() != 0.5 {
		t.Errorf("fillet radius = %v, want 0.5", fillet.Radius())
	}
	thread := fresh.Item(4).Definition().(*ThreadFeature).Definition()
	if string(thread.FaceKey) != "face-c" || thread.Designation != "M6x1" {
		t.Errorf("thread = key %q designation %q, want face-c M6x1", thread.FaceKey, thread.Designation)
	}
}

// TestFaceFilletRoundTrip checks the face-fillet feature's two face sets + radius survive a recipe
// round trip (#694).
func TestFaceFilletRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewDressUpFeatures(fs).AddFaceFillet(
		[][]byte{[]byte("face-a"), []byte("face-b")}, [][]byte{[]byte("face-c")}, func() float64 { return 0.7 })
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*FaceFilletFeature).Definition()
	if len(d.FaceKeysA) != 2 || string(d.FaceKeysA[0]) != "face-a" || string(d.FaceKeysA[1]) != "face-b" {
		t.Errorf("faceA keys = %v, want [face-a face-b]", d.FaceKeysA)
	}
	if len(d.FaceKeysB) != 1 || string(d.FaceKeysB[0]) != "face-c" {
		t.Errorf("faceB keys = %v, want [face-c]", d.FaceKeysB)
	}
	if d.Radius() != 0.7 {
		t.Errorf("radius = %v, want 0.7", d.Radius())
	}
}

// TestRuleFilletRoundTrip checks the rule fillet's rule + radius survive a recipe round trip (#486).
func TestRuleFilletRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewDressUpFeatures(fs).AddRuleFillet(RuleFilletAllFillets, func() float64 { return 1.5 })
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	d := fresh.Item(0).Definition().(*RuleFilletFeature).Definition()
	if d.Rule != RuleFilletAllFillets {
		t.Errorf("rule = %v, want %v", d.Rule, RuleFilletAllFillets)
	}
	if d.Radius() != 1.5 {
		t.Errorf("radius = %v, want 1.5", d.Radius())
	}
}

// TestChamferFlatCornersRoundTrip checks the chamfer corner treatment survives a recipe
// round trip in both states, and that an older recipe with no flag restores as flat (the
// default, matching a freshly created chamfer).
func TestChamferFlatCornersRoundTrip(t *testing.T) {
	for _, flat := range []bool{true, false} {
		fs := NewPartFeatures(nil, nil)
		NewDressUpFeatures(fs).AddChamferCorners([][]byte{[]byte("edge")}, func() float64 { return 0.3 }, flat)
		data, err := fs.MarshalRecipe(oneSketch{})
		if err != nil {
			t.Fatalf("MarshalRecipe(flat=%v): %v", flat, err)
		}
		if data[0].Chamfer.FlatCorners == nil || *data[0].Chamfer.FlatCorners != flat {
			t.Fatalf("serialized FlatCorners = %v, want %v", data[0].Chamfer.FlatCorners, flat)
		}
		fresh := NewPartFeatures(nil, nil)
		if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
			t.Fatalf("ApplyRecipe(flat=%v): %v", flat, err)
		}
		if got := fresh.Item(0).Definition().(*ChamferFeature).Definition().FlatCorners; got != flat {
			t.Errorf("restored FlatCorners = %v, want %v", got, flat)
		}
	}

	// An older recipe without the flag restores as flat (the default).
	fresh := NewPartFeatures(nil, nil)
	legacy := []FeatureData{{Kind: "chamfer", Chamfer: &EdgeDressData{Edges: []string{}, Value: 0.3}}}
	if err := fresh.ApplyRecipe(legacy, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe(legacy): %v", err)
	}
	if !fresh.Item(0).Definition().(*ChamferFeature).Definition().FlatCorners {
		t.Error("legacy chamfer without flatCorners should restore as flat")
	}
}

// TestFilletCornerTypeRoundTrip checks the fillet corner treatment survives a recipe round trip,
// and that an older recipe with no field restores as miter (the zero-value default).
func TestFilletCornerTypeRoundTrip(t *testing.T) {
	for _, corner := range []types.FilletCornerType{types.FilletCornerMiter, types.FilletCornerSetback, types.FilletCornerRound} {
		fs := NewPartFeatures(nil, nil)
		NewDressUpFeatures(fs).AddFilletCorner([][]byte{[]byte("edge")}, func() float64 { return 0.3 }, corner)
		data, err := fs.MarshalRecipe(oneSketch{})
		if err != nil {
			t.Fatalf("MarshalRecipe(%v): %v", corner, err)
		}
		if got := types.FilletCornerType(data[0].Fillet.CornerType); got != corner {
			t.Fatalf("serialized CornerType = %v, want %v", got, corner)
		}
		fresh := NewPartFeatures(nil, nil)
		if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
			t.Fatalf("ApplyRecipe(%v): %v", corner, err)
		}
		if got := fresh.Item(0).Definition().(*FilletFeature).Definition().CornerType; got != corner {
			t.Errorf("restored CornerType = %v, want %v", got, corner)
		}
	}
	// An older recipe without the field restores as miter (the zero default).
	fresh := NewPartFeatures(nil, nil)
	legacy := []FeatureData{{Kind: "fillet", Fillet: &EdgeDressData{Edges: []string{}, Value: 0.3}}}
	if err := fresh.ApplyRecipe(legacy, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe(legacy): %v", err)
	}
	if got := fresh.Item(0).Definition().(*FilletFeature).Definition().CornerType; got != types.FilletCornerMiter {
		t.Errorf("legacy fillet without cornerType should restore as miter, got %v", got)
	}
}

func TestSolidFeaturesRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewHoleFeatures(fs).AddTapped([]byte("face-1"), func() float64 { return 6 }, func() float64 { return 10 }, "M6x1")
	NewBossFeatures(fs).Add([]byte("face-2"), func() float64 { return 8 }, func() float64 { return 4 })
	NewModifyFeatures(fs).AddCombine(0, 1, ops.Cut)

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 3 {
		t.Fatalf("feature count = %d, want 3", fresh.Count())
	}

	hole := fresh.Item(0).Definition().(*HoleFeature).Definition()
	if string(hole.PlacementFaceKey) != "face-1" || hole.Diameter() != 6 || hole.Depth() != 10 {
		t.Errorf("hole = face %q d %v depth %v, want face-1 6 10", hole.PlacementFaceKey, hole.Diameter(), hole.Depth())
	}
	if !hole.Tap.Tapped || hole.Tap.Designation != "M6x1" {
		t.Errorf("hole tap = %+v, want tapped M6x1", hole.Tap)
	}
	boss := fresh.Item(1).Definition().(*BossFeature).Definition()
	if string(boss.PlacementFaceKey) != "face-2" || boss.Height() != 4 {
		t.Errorf("boss = face %q height %v, want face-2 4", boss.PlacementFaceKey, boss.Height())
	}
	combine := fresh.Item(2).Definition().(*CombineFeature).Definition()
	if combine.TargetIndex != 0 || combine.ToolIndex != 1 || combine.Operation != ops.Cut {
		t.Errorf("combine = %+v, want target 0 tool 1 Cut", combine)
	}
}

func TestThroughAllHoleRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewHoleFeatures(fs).AddDrilledThrough([]byte("face-9"), func() float64 { return 3 })

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	hole := fresh.Item(0).Definition().(*HoleFeature).Definition()
	if !hole.ThroughAll || string(hole.PlacementFaceKey) != "face-9" || hole.Diameter() != 3 {
		t.Errorf("restored hole = throughAll %v face %q d %v, want true face-9 3", hole.ThroughAll, hole.PlacementFaceKey, hole.Diameter())
	}
}

func TestCounterboreHoleRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	cb := NewHoleFeatures(fs).AddCounterbore([]byte("face-3"),
		func() float64 { return 2 }, func() float64 { return 6 },
		func() float64 { return 4 }, func() float64 { return 1 })
	cb.Definition().(*HoleFeature).Definition().ThroughAll = true

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	h := fresh.Item(0).Definition().(*HoleFeature).Definition()
	if h.Type != CounterboreHole || !h.ThroughAll || h.CounterDiameter() != 4 || h.CounterDepth() != 1 {
		t.Errorf("restored counterbore = type %d through %v cØ %v cDepth %v, want counterbore true 4 1",
			h.Type, h.ThroughAll, h.CounterDiameter(), h.CounterDepth())
	}
}

func TestCountersinkHoleRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewHoleFeatures(fs).AddCountersink([]byte("face-4"),
		func() float64 { return 2 }, func() float64 { return 5 },
		func() float64 { return 4 }, func() float64 { return 1.5708 })

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	h := fresh.Item(0).Definition().(*HoleFeature).Definition()
	if h.Type != CountersinkHole || h.CounterDiameter() != 4 || h.CounterAngle() != 1.5708 {
		t.Errorf("restored countersink = type %d sinkØ %v angle %v, want countersink 4 1.5708",
			h.Type, h.CounterDiameter(), h.CounterAngle())
	}
}

func TestDrilledHolePointAngleRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewHoleFeatures(fs).AddDrilled([]byte("face-7"), func() float64 { return 2 }, func() float64 { return 4 })
	pf.Definition().(*HoleFeature).Definition().PointAngle = func() float64 { return 2.0594 } // ~118°

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	h := fresh.Item(0).Definition().(*HoleFeature).Definition()
	if h.PointAngle == nil {
		t.Fatal("restored hole lost its point angle")
	}
	if got := h.PointAngle(); got != 2.0594 {
		t.Errorf("restored point angle = %v, want 2.0594", got)
	}
}

func TestRibRoundTrip(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	fs := NewPartFeatures(nil, nil)
	NewRibFeatures(fs).Add(sk, 0, func() float64 { return 1.5 }, func() float64 { return 3 }, ops.Join)

	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	rib := fresh.Item(0).Definition().(*RibFeature).Definition()
	if rib.ProfileIndex != 0 || rib.Thickness() != 1.5 || rib.Depth() != 3 || rib.Operation != ops.Join {
		t.Errorf("restored rib = profile %d thickness %v depth %v op %v, want 0 1.5 3 Join",
			rib.ProfileIndex, rib.Thickness(), rib.Depth(), rib.Operation)
	}
}

func TestEmbossRoundTrip(t *testing.T) {
	sk := squareSketch(4)
	fs := NewPartFeatures(nil, nil)
	NewEmbossFeatures(fs).Add(sk, []int{0}, func() float64 { return 0.8 }, true, 0.1)

	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	e := fresh.Item(0).Definition().(*EmbossFeature).Definition()
	if !e.Engrave || e.Depth() != 0.8 || len(e.ProfileIndices) != 1 || e.Taper != 0.1 {
		t.Errorf("restored emboss = engrave %v depth %v profiles %v taper %v, want true 0.8 [0] 0.1",
			e.Engrave, e.Depth(), e.ProfileIndices, e.Taper)
	}
}

func TestRevolveAboutCenterlineRoundTrip(t *testing.T) {
	sk := offsetSquareSketch(2, 2)
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true)
	fs := NewPartFeatures(nil, nil)
	NewRevolveFeatures(fs).AddAboutCenterline(sk, 0, func() float64 { return 0 }, ops.NewBody)

	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	rv := fresh.Item(0).Definition().(*RevolveFeature).Definition()
	if rv.Axis != nil {
		t.Error("a centerline revolve must restore with a nil axis (centerline mode)")
	}
}

func TestPatternFeaturesRebindSourceByIndex(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil, nil)
	src := NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	NewPatternFeatures(fs).AddRectangular([]ID{src.ID()}, func() int { return 3 }, func() int { return 2 }, math.V3(2, 0, 0), math.V3(0, 2, 0))
	NewPatternFeatures(fs).AddMirror([]ID{src.ID()}, []byte("plane-key"), math.P3(0, 0, 0), math.V3(1, 0, 0))

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 3 {
		t.Fatalf("feature count = %d, want 3 (extrude + rect pattern + mirror)", fresh.Count())
	}

	// The pattern's source must re-bind to the RESTORED extrude's id, not the old one.
	wantSource := fresh.Item(0).ID()
	rect := fresh.Item(1).Definition().(*RectangularPatternFeature).Definition()
	if len(rect.SourceFeatures) != 1 || rect.SourceFeatures[0] != wantSource {
		t.Errorf("rect pattern source = %v, want [%d]", rect.SourceFeatures, wantSource)
	}
	if rect.CountX() != 3 || rect.CountY() != 2 {
		t.Errorf("rect counts = %dx%d, want 3x2", rect.CountX(), rect.CountY())
	}
	if rect.StepX != math.V3(2, 0, 0) || rect.StepY != math.V3(0, 2, 0) {
		t.Errorf("rect steps = %v / %v, want (2,0,0) / (0,2,0)", rect.StepX, rect.StepY)
	}
	mirror := fresh.Item(2).Definition().(*MirrorFeature).Definition()
	if len(mirror.SourceFeatures) != 1 || mirror.SourceFeatures[0] != wantSource {
		t.Errorf("mirror source = %v, want [%d]", mirror.SourceFeatures, wantSource)
	}
	if string(mirror.MirrorPlaneKey) != "plane-key" {
		t.Errorf("mirror plane key = %q, want plane-key", mirror.MirrorPlaneKey)
	}
}

func TestSurfaceFeaturesRoundTrip(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil, nil)
	NewBoundaryPatchFeatures(fs).Add(sk, 0, PatchFree)
	NewRuledSurfaceFeatures(fs).AddByDistance(sk, 0, RuledNormal, func() float64 { return 2 })

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 2 {
		t.Fatalf("feature count = %d, want 2", fresh.Count())
	}

	bp := fresh.Item(0).Definition().(*BoundaryPatchFeature).Definition()
	if bp.Loops.Count() != 1 || bp.Loops.Item(0).ProfileIndex != 0 || bp.Loops.Item(0).Condition != PatchFree {
		t.Errorf("boundary patch loop = %+v, want one free loop on profile 0", bp.Loops.Item(0))
	}
	rs := fresh.Item(1).Definition().(*RuledSurfaceFeature).Definition()
	if rs.Type != RuledNormal || rs.Distance() != 2 {
		t.Errorf("ruled surface = type %v dist %v, want normal 2", rs.Type, rs.Distance())
	}
}

func TestFaceEditFeaturesRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	m := NewModifyFeatures(fs)
	m.AddSplit([][]byte{[]byte("f-split")})
	m.AddMoveFace([][]byte{[]byte("f-move")}, math.V3(1, 2, 3))
	m.AddFaceOffset([][]byte{[]byte("f-off")}, 0.5)
	m.AddDeleteFace([][]byte{[]byte("f-del")})
	m.AddReplaceFace([][]byte{[]byte("f-rep")}, []byte("f-target"))
	m.AddThicken(0.5)

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}

	want := []string{"split", "move-face", "face-offset", "delete-face", "replace-face", "thicken"}
	if fresh.Count() != len(want) {
		t.Fatalf("feature count = %d, want %d", fresh.Count(), len(want))
	}
	for i, kind := range want {
		if got := fresh.Item(i).Kind(); got != kind {
			t.Errorf("feature %d kind = %q, want %q", i, got, kind)
		}
	}
	// The split's face key must survive byte-for-byte.
	split := fresh.Item(0).Definition().(faceEditor)
	if len(split.FaceKeys()) != 1 || string(split.FaceKeys()[0]) != "f-split" {
		t.Errorf("split face keys = %v, want [f-split]", split.FaceKeys())
	}
}

func TestUncodedFeatureErrorsRatherThanDrops(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	fs.Add(fakeFeature{})
	if _, err := fs.MarshalRecipe(oneSketch{}); err == nil {
		t.Error("MarshalRecipe silently accepted a feature with no codec; it must error")
	}
}

// TestThreadParityFieldsRoundTrip: the #325 thread fields (class / tapered /
// modelDiameter) survive the recipe codec, and a legacy thread without them
// restores with the defaults.
func TestThreadParityFieldsRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewDressUpFeatures(fs).AddThreadDef(&ThreadDefinition{
		FaceKey: []byte("face-1"), Designation: "M8x1.25", Cut: false,
		Class: "6H", Tapered: true, ModelDiameter: types.ThreadTapDrillDiameter,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[0].Thread; d.Class != "6H" || !d.Tapered || d.ModelDiameter != "tapDrill" {
		t.Fatalf("serialized thread = %+v, want class/tapered/modelDiameter carried", d)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*ThreadFeature).Definition()
	if def.Class != "6H" || !def.Tapered || def.ModelDiameter != types.ThreadTapDrillDiameter {
		t.Errorf("restored thread = %+v, want the parity fields back", def)
	}

	legacy := []FeatureData{{Kind: "thread", Thread: &ThreadData{Face: "ZmFjZQ==", Designation: "M6x1"}}}
	old := NewPartFeatures(nil, nil)
	if err := old.ApplyRecipe(legacy, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe(legacy): %v", err)
	}
	if d := old.Item(0).Definition().(*ThreadFeature).Definition(); d.Class != "" || d.Tapered || d.ModelDiameter != 0 {
		t.Errorf("legacy thread restored with non-defaults: %+v", d)
	}
}
