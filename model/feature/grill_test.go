// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/model/sketch"
)

// grillSketch returns a sketch with a 6×6 boundary vent (at 2..8) bridged by two 0.5-wide ribs
// drawn as inner rectangles — the finder resolves it to one profile (area 31) whose holes are
// the ribs.
func grillSketch() *sketch.Sketch {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(sk, 2, 2, 8, 8)           // boundary 6×6 (area 36)
	addRect(sk, 3.75, 2.5, 4.25, 7.5) // rib 1 (0.5 × 5)
	addRect(sk, 5.75, 2.5, 6.25, 7.5) // rib 2 (0.5 × 5)
	return sk
}

// TestGrillCutsVentLeavingRibs builds a 10×10×1 wall (vol 100) and a grill: the vent removes
// the boundary-minus-ribs area (31), leaving the ribs bridging — a validated solid of vol 69,
// which is more than the 64 a plain 6×6 window would leave (so the ribs stayed).
func TestGrillCutsVentLeavingRibs(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	wall := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(wall, 0, 0, 10, 10)
	NewExtrudeFeatures(fs).AddByDistanceExtent(wall, 0, ops.NewBody, func() float64 { return 1 })

	grill := NewGrillFeatures(fs).Add(&GrillDefinition{Sketch: grillSketch(), Boundaries: []int{0}})
	fs.Recompute()

	if !grill.Health().OK() {
		t.Fatalf("grill sick: %+v", grill.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("grill body not a valid solid: %+v", r.Issues)
	}
	if v := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-69) > 1e-6 {
		t.Errorf("grill volume = %g, want 69 (100 − 31 vent), so >64 ⇒ ribs remain", v)
	}
}

// TestGrillCrossingBarsVolume: a grid of crossing ribs and spars (overlapping rectangles) cuts
// the correct vent — boundary 49 minus the bars' union 13.92 = 35.08 removed from a 100 wall,
// leaving 64.92. Built as boundary − union(bars), robust to the crossings (#863).
func TestGrillCrossingBarsVolume(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	wall := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(wall, 0, 0, 10, 10)
	NewExtrudeFeatures(fs).AddByDistanceExtent(wall, 0, ops.NewBody, func() float64 { return 1 })

	g := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(g, 1.5, 1.5, 8.5, 8.5) // boundary 7×7 = 49
	for _, x := range []float64{3, 5, 7} {
		addRect(g, x-0.2, 1.8, x+0.2, 8.2) // vertical ribs
	}
	for _, y := range []float64{3, 5, 7} {
		addRect(g, 1.8, y-0.2, 8.2, y+0.2) // horizontal spars (cross the ribs)
	}
	grill := NewGrillFeatures(fs).Add(&GrillDefinition{Sketch: g, Boundaries: []int{0}})
	fs.Recompute()

	if !grill.Health().OK() {
		t.Fatalf("crossing-bar grill sick: %+v", grill.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("crossing-bar grill body not a valid solid: %+v", r.Issues)
	}
	if v := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-64.92) > 1e-3 {
		t.Errorf("crossing-bar grill volume = %g, want 64.92 (100 − vent 35.08)", v)
	}
}

// TestGrillBoundaryOnlyIsAWindow: a boundary with no inner structure cuts a plain window
// (vol 100 − 36 = 64).
func TestGrillBoundaryOnlyIsAWindow(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	wall := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(wall, 0, 0, 10, 10)
	NewExtrudeFeatures(fs).AddByDistanceExtent(wall, 0, ops.NewBody, func() float64 { return 1 })

	plain := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(plain, 2, 2, 8, 8)
	NewGrillFeatures(fs).Add(&GrillDefinition{Sketch: plain, Boundaries: []int{0}})
	fs.Recompute()

	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("windowed body not a valid solid: %+v", r.Issues)
	}
	if v := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-64) > 1e-6 {
		t.Errorf("window volume = %g, want 64 (100 − 36)", v)
	}
}

// TestGrillNoBoundaryRejected: an empty boundary set is an error.
func TestGrillNoBoundaryRejected(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	wall := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(wall, 0, 0, 10, 10)
	NewExtrudeFeatures(fs).AddByDistanceExtent(wall, 0, ops.NewBody, func() float64 { return 1 })
	g := NewGrillFeatures(fs).Add(&GrillDefinition{Sketch: grillSketch(), Boundaries: nil})
	fs.Recompute()
	if g.Health().OK() {
		t.Error("grill with no boundary should be sick")
	}
}

// TestGrillRoundTrip checks the grill survives the recipe codec (sketch index + boundaries).
func TestGrillRoundTrip(t *testing.T) {
	t.Parallel()
	wallSk := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(wallSk, 0, 0, 10, 10)
	gSk := grillSketch()
	idx := sketchList{sks: []*sketch.Sketch{wallSk, gSk}}

	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(wallSk, 0, ops.NewBody, func() float64 { return 1 })
	NewGrillFeatures(fs).Add(&GrillDefinition{Sketch: gSk, Boundaries: []int{0}, Draft: 0.1})

	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(1).Definition().(*GrillFeature).Definition()
	if len(def.Boundaries) != 1 || def.Boundaries[0] != 0 || def.Sketch != gSk || stdmath.Abs(def.Draft-0.1) > 1e-9 {
		t.Errorf("restored grill = %+v, want boundary [0] draft 0.1 on the grill sketch", def)
	}
}
