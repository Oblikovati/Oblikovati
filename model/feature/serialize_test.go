// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
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
	du.AddThread([]byte("face-c"), "M6x1")

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
