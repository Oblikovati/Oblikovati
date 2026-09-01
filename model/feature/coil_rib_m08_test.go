// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/sketch"
)

// Coil taper, the two-of-three shape spec, and the to-next rib
// (M08 PBI-096, #316).

// coilProfileSketch is a small square at radius 5 from the Z axis.
func coilProfileSketch() *sketch.Sketch {
	return offsetSquareSketch(5, 1)
}

func zWorkAxis() *WorkAxis {
	dir, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	return &WorkAxis{origin: math.P3(0, 0, 0), dir: dir}
}

func coilRecompute(t *testing.T, def *CoilDefinition) float64 {
	t.Helper()
	fs := NewPartFeatures(nil)
	pf := NewCoilFeatures(fs).AddDefinition(def)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("coil sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("coil not a valid solid: %+v", r)
	}
	return ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
}

// TestCoilHeightModes: pitch+height and revolutions+height match the
// equivalent pitch+revolutions coil.
func TestCoilHeightModes(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~11s): `make test-corpus`")
	}
	t.Parallel()
	pr := coilRecompute(t, &CoilDefinition{
		Sketch: coilProfileSketch(), Axis: zWorkAxis(),
		Pitch: angleConst(2), Revolutions: angleConst(3), Operation: ops.NewBody,
	})
	ph := coilRecompute(t, &CoilDefinition{
		Sketch: coilProfileSketch(), Axis: zWorkAxis(),
		Pitch: angleConst(2), Height: angleConst(6), Operation: ops.NewBody,
	})
	rh := coilRecompute(t, &CoilDefinition{
		Sketch: coilProfileSketch(), Axis: zWorkAxis(),
		Revolutions: angleConst(3), Height: angleConst(6), Operation: ops.NewBody,
	})
	if relErr(ph, pr) > 1e-9 || relErr(rh, pr) > 1e-9 {
		t.Errorf("height-mode volumes = %g / %g, want both exactly the pitch+revs %g", ph, rh, pr)
	}
}

// TestCoilShapeSpecValidation: all three (overdetermined) and fewer than two
// are precise errors.
func TestCoilShapeSpecValidation(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	pf := NewCoilFeatures(fs).AddDefinition(&CoilDefinition{
		Sketch: coilProfileSketch(), Axis: zWorkAxis(),
		Pitch: angleConst(2), Revolutions: angleConst(3), Height: angleConst(6),
		Operation: ops.NewBody,
	})
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Error("pitch+revolutions+height (overdetermined) must be sick")
	}
	fs2 := NewPartFeatures(nil)
	pf2 := NewCoilFeatures(fs2).AddDefinition(&CoilDefinition{
		Sketch: coilProfileSketch(), Axis: zWorkAxis(),
		Pitch: angleConst(2), Operation: ops.NewBody,
	})
	fs2.Recompute()
	if pf2.Health().Status != health.Sick {
		t.Error("pitch alone must be sick")
	}
}

// TestCoilTaperGrowsRadius: a tapered coil's top turn sits farther from the
// axis — the range box grows by ≈ tan(taper)·height over the untapered coil.
func TestCoilTaperGrowsRadius(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~4s): `make test-corpus`")
	}
	t.Parallel()
	const taper, pitch, revs = 0.2, 2.0, 3.0
	mk := func(tp float64) *CoilDefinition {
		return &CoilDefinition{
			Sketch: coilProfileSketch(), Axis: zWorkAxis(),
			Pitch: angleConst(pitch), Revolutions: angleConst(revs),
			Taper: tp, Operation: ops.NewBody,
		}
	}
	fs := NewPartFeatures(nil)
	NewCoilFeatures(fs).AddDefinition(mk(0))
	fs.Recompute()
	plain := fs.Result()[0].RangeBox()
	fs2 := NewPartFeatures(nil)
	NewCoilFeatures(fs2).AddDefinition(mk(taper))
	fs2.Recompute()
	tapered := fs2.Result()[0].RangeBox()
	wantGrowth := stdmath.Tan(taper) * pitch * revs // radius gain at the top
	growth := float64(tapered.Max.X - plain.Max.X)
	if stdmath.Abs(growth-wantGrowth) > 0.1 {
		t.Errorf("taper radius growth = %g, want ≈%g", growth, wantGrowth)
	}
	if v := ops.Validate(fs2.Result()[0]); !v.Valid {
		t.Errorf("tapered coil invalid: %+v", v.Issues)
	}
}

// TestRibToNextReachesPlate: a rib sketched above a plate with toNext extends
// exactly down to the plate's top face and joins it.
func TestRibToNextReachesPlate(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	// Plate: z ∈ [0, 1], spanning xy ∈ [-5, 5].
	plate := centeredSquareOn(sketch.XYPlane(), 5)
	NewExtrudeFeatures(fs).AddByDistanceExtent(plate, 0, ops.NewBody, angleConst(1))
	// Rib path on the plane z=3: a line across the plate.
	ribSk := lineSketchOn(planeAtZ(3))
	def := &RibDefinition{
		Sketch: ribSk, ProfileIndex: 0,
		Thickness: angleConst(0.5),
		Depth:     angleConst(-1), // sign only: grow downward
		ToNext:    true,
		Operation: ops.Join,
	}
	pf := NewRibFeatures(fs).AddDefinition(def)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("to-next rib sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("rib-joined body invalid: %+v", r.Issues)
	}
	// Plate 100 + wall: thickness 0.5 × length 8 × the 2 drop from z=3 to z=1.
	want := 100 + 0.5*8*2
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.02 {
		t.Errorf("to-next rib volume = %g, want ≈%g", got, want)
	}
}

// TestRibToNextNoMaterialSick: a to-next rib with nothing to land on is sick
// with a precise message.
func TestRibToNextNoMaterialSick(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	def := &RibDefinition{
		Sketch: lineSketchOn(planeAtZ(3)), ProfileIndex: 0,
		Thickness: angleConst(0.5), ToNext: true, Operation: ops.NewBody,
	}
	pf := NewRibFeatures(fs).AddDefinition(def)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Error("a to-next rib with no material must be sick")
	}
}

// lineSketchOn is a sketch holding one open line x ∈ [-4, 4] at y = 0.
func lineSketchOn(plane sketch.Plane) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	a := s.Points().Add(math.P2(-4, 0))
	b := s.Points().Add(math.P2(4, 0))
	s.Lines().Add(a, b)
	return s
}
