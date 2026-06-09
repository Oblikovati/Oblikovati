// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// bodyZRange returns the min and max Z of a body's vertices — the extrude reach.
func bodyZRange(b *topo.Body) (lo, hi float64) {
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range b.Vertices() {
		z := float64(v.Point().Z)
		lo, hi = stdmath.Min(lo, z), stdmath.Max(hi, z)
	}
	return lo, hi
}

// extrudeWith builds a fresh part, extrudes a 2×2 square with the given extent/taper, and
// returns the resulting body.
func extrudeWith(t *testing.T, ext Extent, taper float64) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(param.NewParameters(), nil)
	pf := NewExtrudeFeatures(fs).AddExtrude(squareSketch(2), []int{0}, ops.NewBody, ext, taper)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("extrude sick: %+v", pf.Health())
	}
	return fs.Result()[0]
}

func TestExtrudeSymmetricSpansBothSides(t *testing.T) {
	body := extrudeWith(t, Extent{Type: DistanceExtent, Direction: SymmetricDir, Distance: func() float64 { return 6 }}, 0)
	lo, hi := bodyZRange(body)
	if !approxEq(lo, -3) || !approxEq(hi, 3) {
		t.Errorf("symmetric extrude z-range = [%g,%g], want [-3,3]", lo, hi)
	}
}

func TestExtrudeNegativeDirection(t *testing.T) {
	body := extrudeWith(t, Extent{Type: DistanceExtent, Direction: NegativeDir, Distance: func() float64 { return 4 }}, 0)
	lo, hi := bodyZRange(body)
	if !approxEq(lo, -4) || !approxEq(hi, 0) {
		t.Errorf("negative extrude z-range = [%g,%g], want [-4,0]", lo, hi)
	}
}

func TestExtrudeAsymmetricTwoDirection(t *testing.T) {
	body := extrudeWith(t, Extent{
		Type:     DistanceExtent,
		Distance: func() float64 { return 4 }, Distance2: func() float64 { return 2 },
	}, 0)
	lo, hi := bodyZRange(body)
	if !approxEq(lo, -2) || !approxEq(hi, 4) {
		t.Errorf("asymmetric extrude z-range = [%g,%g], want [-2,4]", lo, hi)
	}
}

func TestExtrudeTaperWidensTopAndStaysValid(t *testing.T) {
	// A positive taper drafts the walls outward: the far cap is larger than the 2×2 base.
	body := extrudeWith(t, Extent{Type: DistanceExtent, Distance: func() float64 { return 5 }}, 0.2)
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("tapered extrude failed validation: %+v", r)
	}
	box := body.RangeBox()
	if d := box.Diagonal(); d.X <= 2.0001 || d.Y <= 2.0001 {
		t.Errorf("tapered extrude XY extent = (%g,%g), want > 2 (drafted outward)", d.X, d.Y)
	}
}

func TestExtrudeToWorkPlane(t *testing.T) {
	g := NewWorkGeometry()
	g.Recompute(nil)
	target := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 7 }) // z=7
	g.Recompute(nil)
	body := extrudeWith(t, Extent{Type: ToFaceExtent, ToPlane: target}, 0)
	lo, hi := bodyZRange(body)
	if !approxEq(lo, 0) || !approxEq(hi, 7) {
		t.Errorf("to-plane extrude z-range = [%g,%g], want [0,7]", lo, hi)
	}
}

func TestExtrudeModeRoundTrip(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddExtrude(sk, []int{0}, ops.Cut,
		Extent{Type: ThroughAllExtent, Direction: SymmetricDir, Distance: func() float64 { return 5 }}, 0.15)
	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	ed := data[0].Extrude
	if ed == nil || ed.Extent != "through-all" || ed.Direction != "symmetric" || ed.Taper != 0.15 || ed.Operation != "cut" {
		t.Fatalf("marshaled extrude = %+v, want through-all/symmetric/cut/taper 0.15", ed)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	ef := fresh.Item(0).feature.(*ExtrudeFeature)
	if ef.Extent().Type != ThroughAllExtent || ef.Extent().Direction != SymmetricDir || ef.Taper() != 0.15 {
		t.Errorf("restored extent = %+v taper=%v, want through-all/symmetric/0.15", ef.Extent(), ef.Taper())
	}
}

func TestExtrudeThroughAllSpansExistingMaterial(t *testing.T) {
	// Through-all measures the existing material's reach along the normal and spans it
	// (plus a margin). Tested with a new-body operation so it exercises the span without
	// the kernel's intersecting boolean (overlapping cut/join is phase B).
	fs := NewPartFeatures(param.NewParameters(), nil)
	ex := NewExtrudeFeatures(fs)
	ex.AddByDistanceExtent(squareSketch(4), 0, ops.NewBody, func() float64 { return 3 }) // block z0..3
	tool := ex.AddExtrude(squareSketch(2), []int{0}, ops.NewBody,
		Extent{Type: ThroughAllExtent, Direction: PositiveDir}, 0)
	fs.Recompute()
	if !tool.Health().OK() {
		t.Fatalf("through-all sick: %+v", tool.Health())
	}
	// The block reaches z=3; the through-all prism spans to 3 + margin.
	lo, hi := bodyZRange(fs.Result()[1])
	if !approxEq(lo, 0) || !approxEq(hi, 3+throughAllMargin) {
		t.Errorf("through-all z-range = [%g,%g], want [0,%g]", lo, hi, 3+throughAllMargin)
	}
}
