// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
)

func TestCoilSweepsValidHelix(t *testing.T) {
	t.Parallel()
	// A small square offset from the Y axis, swept 3 turns with pitch 2 → a helical
	// solid that climbs 3·2 = 6 plus the profile's own height (1).
	fs := NewPartFeatures(nil)
	pf := NewCoilFeatures(fs).Add(offsetSquareSketch(4, 1), 0, yAxis(),
		func() float64 { return 2 }, func() float64 { return 3 }, 0, ops.NewBody)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("coil went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("coil is not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v <= 0 {
		t.Errorf("coil volume = %g, want > 0", v)
	}
	// The helix climbs along Y by pitch·revolutions plus the profile height.
	if span := body.RangeBox().Max.Y - body.RangeBox().Min.Y; relErr(span, 2*3+1) > 0.05 {
		t.Errorf("coil axial span = %g, want ≈7 (pitch·revs + profile height)", span)
	}
}

func TestCoilRejectsZeroRevolutions(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	pf := NewCoilFeatures(fs).Add(offsetSquareSketch(4, 1), 0, yAxis(),
		func() float64 { return 2 }, func() float64 { return 0 }, 0, ops.NewBody)
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("coil with zero revolutions should go sick")
	}
}

func TestCoilRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	g.Recompute(nil)
	axis, err := g.axis(OriginYAxis)
	if err != nil {
		t.Fatalf("origin Y axis: %v", err)
	}
	sk := offsetSquareSketch(4, 1)
	fs := NewPartFeatures(nil)
	NewCoilFeatures(fs).Add(sk, 0, axis,
		func() float64 { return 2 }, func() float64 { return 3 }, 0.1, ops.Join)

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, g); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*CoilFeature).Definition()
	if def.Pitch() != 2 || def.Revolutions() != 3 || def.Taper != 0.1 {
		t.Errorf("coil def = pitch %g revs %g taper %g, want 2 3 0.1", def.Pitch(), def.Revolutions(), def.Taper)
	}
	if def.Operation != ops.Join {
		t.Errorf("coil operation = %v, want Join", def.Operation)
	}
	if def.Axis == nil || def.Axis.Key() != OriginYAxis {
		t.Errorf("coil axis not rebound to origin Y axis: %v", def.Axis)
	}
}
