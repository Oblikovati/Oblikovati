// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// #2019: the Revolve panel had no Direction row, so every revolve swept forward from the profile.
// The model has carried a second-direction angle since #313, but nothing named the side the FIRST
// angle grows on, so Flipped and Symmetric were unreachable and Asymmetric could not be asked for
// from the UI at all. These drive the tool the way the panel does.

// revolveWith runs the tool over the offset square, applies the panel's direction options, commits,
// and returns the resulting body's properties.
func revolveWith(t *testing.T, apply func(rv *RevolveTool)) ops.GeometryProperties {
	t.Helper()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	s.SetPicker(stubPicker{sel: profile})
	rv := NewRevolveTool()
	s.StartTool(rv)
	s.Click(1, 1)
	rv.SetAngle(stdmath.Pi / 2)
	apply(rv)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("revolved body not a valid solid: %+v", r)
	}
	return ops.BodyGeometryProperties(body, ops.DefaultQuality())
}

// quarterWasherApp is the volume of a 90° sweep of the 2×2 square at x∈[2,4].
const quarterWasherApp = stdmath.Pi * (4*4 - 2*2) * 2 / 4

// TestFlippedDirectionSweepsTheOtherWay: the panel's Flipped toggle must move the material to the
// other side of the profile, not merely relabel it. Volume alone cannot tell the two apart, so the
// assertion is on the centroid's side of the profile plane.
func TestFlippedDirectionSweepsTheOtherWay(t *testing.T) {
	t.Parallel()
	fwd := revolveWith(t, func(*RevolveTool) {})
	back := revolveWith(t, func(rv *RevolveTool) { rv.SetDirection(feature.NegativeDir) })

	if relErr := stdmath.Abs(back.Volume-quarterWasherApp) / quarterWasherApp; relErr > 0.01 {
		t.Fatalf("flipped volume %g, want ≈%g — a flip must not change how much is swept", back.Volume, quarterWasherApp)
	}
	if stdmath.Abs(float64(fwd.Centroid.Z)) < 0.5 {
		t.Fatalf("the default sweep sits at z=%v; it must be off the profile plane for a flip to be visible", fwd.Centroid.Z)
	}
	if stdmath.Abs(float64(back.Centroid.Z+fwd.Centroid.Z)) > 0.02 {
		t.Errorf("flipped centroid z=%v, want %v — the default sweep mirrored", back.Centroid.Z, -fwd.Centroid.Z)
	}
}

// TestSymmetricDirectionSplitsTheAngle: Symmetric sweeps half of Angle A each way, so the solid
// balances on the profile plane while covering the same total angle.
func TestSymmetricDirectionSplitsTheAngle(t *testing.T) {
	t.Parallel()
	sym := revolveWith(t, func(rv *RevolveTool) { rv.SetDirection(feature.SymmetricDir) })

	if relErr := stdmath.Abs(sym.Volume-quarterWasherApp) / quarterWasherApp; relErr > 0.01 {
		t.Errorf("symmetric volume %g, want ≈%g — Angle A is the total, halved each way", sym.Volume, quarterWasherApp)
	}
	if stdmath.Abs(float64(sym.Centroid.Z)) > 0.02 {
		t.Errorf("symmetric centroid z=%v, want ≈0", sym.Centroid.Z)
	}
}

// TestAsymmetricDirectionAddsTheSecondAngle: the Asymmetric toggle plus Angle B sweeps both ways
// with separate angles.
func TestAsymmetricDirectionAddsTheSecondAngle(t *testing.T) {
	t.Parallel()
	asym := revolveWith(t, func(rv *RevolveTool) {
		rv.SetAsymmetric(true)
		rv.SetSecondAngle(stdmath.Pi / 2)
	})

	want := 2 * quarterWasherApp
	if relErr := stdmath.Abs(asym.Volume-want) / want; relErr > 0.01 {
		t.Errorf("asymmetric volume %g, want ≈%g (90° each way)", asym.Volume, want)
	}
	if stdmath.Abs(float64(asym.Centroid.Z)) > 0.02 {
		t.Errorf("asymmetric 90°/90° centroid z=%v, want ≈0", asym.Centroid.Z)
	}
}

// TestLeavingAsymmetricDropsTheSecondAngle guards the EDIT path specifically. On the create path a
// fresh definition has no second angle to go stale, so switching back to a single direction is
// harmless; an edit reuses the committed definition, and leaving its Angle2 in place would keep
// sweeping the extra material after the user chose Default again.
func TestLeavingAsymmetricDropsTheSecondAngle(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	part := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	axis, ok := part.WorkGeometry().AxisByRef(feature.OriginYAxis)
	if !ok {
		t.Fatal("part has no Y origin axis")
	}
	pf := feature.NewRevolveFeatures(part.Features()).AddTwoDirectional(profile.Sketch, 0, axis,
		func() float64 { return stdmath.Pi / 2 }, func() float64 { return stdmath.Pi / 2 }, ops.NewBody)
	part.Recompute()

	s.BeginEditFeature(FeatureHandle{Feature: pf})
	rv, isRevolve := s.ActiveTool().Tool().(*RevolveTool)
	if !isRevolve {
		t.Fatal("editing a revolve did not re-open the revolve tool")
	}
	rv.SetAsymmetric(false) // the user picks Default again
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	body := part.SurfaceBodies().Item(0)
	got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if stdmath.Abs(got-quarterWasherApp)/quarterWasherApp > 0.01 {
		t.Errorf("volume %g, want ≈%g — the abandoned Angle B kept widening the sweep", got, quarterWasherApp)
	}
}

// TestEditingARevolveKeepsItsDirection: re-opening a flipped revolve must show it flipped, or the
// next OK would silently sweep it forward again.
func TestEditingARevolveKeepsItsDirection(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	part := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	axis, ok := part.WorkGeometry().AxisByRef(feature.OriginYAxis)
	if !ok {
		t.Fatal("part has no Y origin axis")
	}
	pf := feature.NewRevolveFeatures(part.Features()).Add(profile.Sketch, 0, axis,
		func() float64 { return stdmath.Pi / 2 }, ops.NewBody)
	def := pf.Definition().(*feature.RevolveFeature).Definition()
	def.Direction = feature.NegativeDir
	def.Angle2 = func() float64 { return stdmath.Pi / 4 }

	edited := editRevolveTool(pf, pf.Definition().(*feature.RevolveFeature))

	if edited.Direction() != feature.NegativeDir {
		t.Errorf("re-opened direction = %v, want NegativeDir", edited.Direction())
	}
	if !edited.Asymmetric() {
		t.Error("a revolve with a second angle must re-open asymmetric")
	}
	if got := edited.SecondAngle(); stdmath.Abs(got-stdmath.Pi/4) > 1e-9 {
		t.Errorf("re-opened Angle B = %v rad, want %v", got, stdmath.Pi/4)
	}
}
