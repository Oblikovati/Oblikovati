// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// bendBarFixture builds a part of a flat bar with a bend line across it at x=5, returning the
// engine, the sketch (so a round-trip can re-supply it), and the placed bend feature.
func bendBarFixture(t *testing.T) (*PartFeatures, *sketch.Sketch, *PartFeature) {
	t.Helper()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(5, 0), math.P2(5, 2)) // bend line along +Y at x=5
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(subd.ToBody(subd.Box(10, 2, 1), "bar"))
	def := &BendPartDefinition{
		Sketch: sk, LineIndex: 0, BendType: types.RadiusAndAngleBend,
		Radius: constFloat(1), Angle: constFloat(stdmath.Pi / 2),
	}
	pf := NewBendPartFeatures(fs).Add(def)
	return fs, sk, pf
}

// TestBendPartFeatureFoldsValidSolid recomputes a bend feature and checks it yields one
// valid solid that folded up (M20-F17, #651).
func TestBendPartFeatureFoldsValidSolid(t *testing.T) {
	t.Parallel()
	fs, _, pf := bendBarFixture(t)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("bend sick: %+v", pf.Health())
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("bend result = %d bodies, want 1", len(fs.Result()))
	}
	if r := ops.Validate(fs.Result()[0]); !r.Valid {
		t.Fatalf("bent body invalid: %v", r.Issues)
	}
	if box := fs.Result()[0].RangeBox(); box.Max.Z < 4 {
		t.Errorf("bent top Z = %g, want the flange folded up (>4)", box.Max.Z)
	}
}

// TestBendPartDerivesAngleFromArcLength checks the arc-length+angle / radius+arc-length types
// derive the missing input.
func TestBendPartDerivesAngleFromArcLength(t *testing.T) {
	t.Parallel()
	def := &BendPartDefinition{BendType: types.RadiusAndArcLengthBend, Radius: constFloat(2), ArcLength: constFloat(stdmath.Pi)}
	r, a, err := bendRadiusAngle(def)
	if err != nil || r != 2 || stdmath.Abs(a-stdmath.Pi/2) > 1e-12 {
		t.Errorf("radiusAndArcLength → (r=%g, a=%g, err=%v), want (2, pi/2, nil)", r, a, err)
	}
	def = &BendPartDefinition{BendType: types.ArcLengthAndAngleBend, ArcLength: constFloat(stdmath.Pi), Angle: constFloat(stdmath.Pi / 2)}
	r, a, err = bendRadiusAngle(def)
	if err != nil || stdmath.Abs(r-2) > 1e-12 || stdmath.Abs(a-stdmath.Pi/2) > 1e-12 {
		t.Errorf("arcLengthAndAngle → (r=%g, a=%g, err=%v), want (2, pi/2, nil)", r, a, err)
	}
}

// TestBendPartRoundTrip preserves the bend line, type, and scalars across an .obk round-trip
// (extrude source + bend line in a second sketch, so the program serializes).
func TestBendPartRoundTrip(t *testing.T) {
	t.Parallel()
	profile := sketch.NewSketches().Add(sketch.XYPlane())
	bendSk := sketch.NewSketches().Add(sketch.XYPlane())
	bendSk.Lines().AddByTwoPoints(math.P2(5, 0), math.P2(5, 2))
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(profile, 0, ops.NewBody, func() float64 { return 1 })
	NewBendPartFeatures(fs).Add(&BendPartDefinition{
		Sketch: bendSk, LineIndex: 0, BendType: types.RadiusAndAngleBend,
		Radius: constFloat(1), Angle: constFloat(stdmath.Pi / 2),
	})
	idx := twoSketches{a: profile, b: bendSk}
	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(fresh.Count() - 1).Definition().(*BendPartFeature).Definition()
	if def.BendType != types.RadiusAndAngleBend || def.LineIndex != 0 {
		t.Errorf("restored bend = type %v line %d, want radiusAndAngle/0", def.BendType, def.LineIndex)
	}
	if got := def.Radius(); got != 1 {
		t.Errorf("restored radius = %g, want 1", got)
	}
	if got := def.Angle(); stdmath.Abs(got-stdmath.Pi/2) > 1e-12 {
		t.Errorf("restored angle = %g, want pi/2", got)
	}
}
