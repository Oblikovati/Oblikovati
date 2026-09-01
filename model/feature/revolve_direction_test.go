// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// Revolve direction (#2019). Default / Flipped / Symmetric all sweep the SAME angle, so they all
// produce the same volume — the difference is only ever WHERE that material lands. These tests
// therefore assert the centroid relative to the profile plane (z=0, the sweep's zero angle), which
// is the one thing that separates them and does not depend on the rotation's handedness.

// revolveCentroid builds a partial revolve of the 2×2 square at x∈[2,4] about Y and returns its
// volume and centroid.
func revolveCentroid(t *testing.T, angle float64, dir ExtentDirection, angle2 func() float64) ops.GeometryProperties {
	t.Helper()
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).Add(offsetSquareSketch(2, 2), 0, yWorkAxis(), angleConst(angle), ops.NewBody)
	def := pf.Definition().(*RevolveFeature).Definition()
	def.Direction, def.Angle2 = dir, angle2
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("revolve sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("revolve not a valid solid: %+v", r)
	}
	return query.BodyGeometryProperties(body, ops.DefaultQuality())
}

// quarterWasher is a 90° sweep of the 2×2 square at x∈[2,4]: a quarter of the 24π washer.
const quarterWasher = stdmath.Pi * (4*4 - 2*2) * 2 / 4

// TestFlippedRevolveMirrorsTheDefaultSweep is the point of the feature: Flipped sweeps the same
// angle the other way, so its material is the default's mirrored across the profile plane. Before
// #2019 the second direction could only be ADDED to the first, never used on its own, so [-90°,0]
// was unreachable — and on a Cut or Join that is a different feature, not a different view.
func TestFlippedRevolveMirrorsTheDefaultSweep(t *testing.T) {
	t.Parallel()
	fwd := revolveCentroid(t, stdmath.Pi/2, PositiveDir, nil)
	back := revolveCentroid(t, stdmath.Pi/2, NegativeDir, nil)

	if relErr(fwd.Volume, quarterWasher) > 0.01 || relErr(back.Volume, quarterWasher) > 0.01 {
		t.Fatalf("volumes %g / %g, want both ≈%g — a flip must not change how much is swept", fwd.Volume, back.Volume, quarterWasher)
	}
	if stdmath.Abs(float64(fwd.Centroid.Z)) < 0.5 {
		t.Fatalf("the default sweep's centroid sits at z=%v; it must be off the profile plane for the mirror to mean anything", fwd.Centroid.Z)
	}
	if stdmath.Abs(float64(back.Centroid.Z+fwd.Centroid.Z)) > 0.02 {
		t.Errorf("flipped centroid z=%v, want %v — the mirror of the default sweep", back.Centroid.Z, -fwd.Centroid.Z)
	}
	if stdmath.Abs(float64(back.Centroid.X-fwd.Centroid.X)) > 0.02 {
		t.Errorf("flipped centroid x=%v, want %v — a flip mirrors across the profile plane, nothing else", back.Centroid.X, fwd.Centroid.X)
	}
}

// TestSymmetricRevolveStraddlesTheProfilePlane: Symmetric sweeps half the angle each way, so the
// solid is balanced about the profile plane while covering the same total angle.
func TestSymmetricRevolveStraddlesTheProfilePlane(t *testing.T) {
	t.Parallel()
	sym := revolveCentroid(t, stdmath.Pi/2, SymmetricDir, nil)

	if relErr(sym.Volume, quarterWasher) > 0.01 {
		t.Errorf("symmetric volume %g, want ≈%g — the angle is the TOTAL, split half each way", sym.Volume, quarterWasher)
	}
	if stdmath.Abs(float64(sym.Centroid.Z)) > 0.02 {
		t.Errorf("symmetric centroid z=%v, want ≈0 — half the sweep must land each side of the profile", sym.Centroid.Z)
	}
}

// TestAsymmetricRevolveOutranksTheDirection: with a second angle set the revolve names both sides
// itself, so a stale Direction must not shift the span.
func TestAsymmetricRevolveOutranksTheDirection(t *testing.T) {
	t.Parallel()
	asym := revolveCentroid(t, stdmath.Pi/2, NegativeDir, angleConst(stdmath.Pi/2))

	if relErr(asym.Volume, 2*quarterWasher) > 0.01 {
		t.Errorf("asymmetric volume %g, want ≈%g (90° each way)", asym.Volume, 2*quarterWasher)
	}
	if stdmath.Abs(float64(asym.Centroid.Z)) > 0.02 {
		t.Errorf("asymmetric 90°/90° centroid z=%v, want ≈0", asym.Centroid.Z)
	}
}

// TestDirectionVanishesOnAFullRevolution: a complete turn is rotationally closed, so no direction
// is observable and none may disturb the solid.
func TestDirectionVanishesOnAFullRevolution(t *testing.T) {
	t.Parallel()
	full := stdmath.Pi * (4*4 - 2*2) * 2
	for _, dir := range []ExtentDirection{PositiveDir, NegativeDir, SymmetricDir} {
		got := revolveCentroid(t, 2*stdmath.Pi, dir, nil)
		if relErr(got.Volume, full) > 0.01 {
			t.Errorf("direction %v on a full turn gave %g, want ≈%g", dir, got.Volume, full)
		}
	}
}

// TestRevolveSpanResolvesEachDirection pins the span arithmetic itself, which the geometry tests
// can only observe indirectly.
func TestRevolveSpanResolvesEachDirection(t *testing.T) {
	t.Parallel()
	quarter := stdmath.Pi / 2
	cases := []struct {
		name         string
		dir          ExtentDirection
		angle2       func() float64
		total, start float64
	}{
		{"default", PositiveDir, nil, quarter, 0},
		{"flipped", NegativeDir, nil, quarter, -quarter},
		{"symmetric", SymmetricDir, nil, quarter, -quarter / 2},
		{"asymmetric", PositiveDir, angleConst(quarter / 2), quarter * 1.5, -quarter / 2},
	}
	for _, c := range cases {
		total, start := revolveSpan(&RevolveDefinition{Angle: angleConst(quarter), Angle2: c.angle2, Direction: c.dir})
		if stdmath.Abs(total-c.total) > 1e-9 || stdmath.Abs(start-c.start) > 1e-9 {
			t.Errorf("%s span = (%g, %g), want (%g, %g)", c.name, total, start, c.total, c.start)
		}
	}
}

// TestRevolveDirectionSurvivesTheRecipe: the direction is part of the feature, not of the session,
// so a flipped revolve must reload flipped. It is applied on every axis path, so the centerline
// case — which restores through its own branch — is the one worth pinning.
func TestRevolveDirectionSurvivesTheRecipe(t *testing.T) {
	t.Parallel()
	sk := offsetSquareSketch(2, 2)
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true)
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).AddAboutCenterline(sk, 0, angleConst(stdmath.Pi/2), ops.NewBody)
	pf.Definition().(*RevolveFeature).Definition().Direction = NegativeDir

	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if got := fresh.Item(0).Definition().(*RevolveFeature).Definition().Direction; got != NegativeDir {
		t.Errorf("restored direction = %v, want NegativeDir — a reloaded revolve swept the other way", got)
	}
}
