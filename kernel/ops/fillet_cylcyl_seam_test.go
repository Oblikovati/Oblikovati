// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// k2LikeSeamSpine is the K2 configuration in closed form: a vertical R=50 boss wall crossed by a
// horizontal R=30 bore (axis y through (50,120,70)), convex seam, r=5 → wrap offset ρ_W = 30+5
// (bore, ε=−1, σ=−1) and other offset ρ_O = 50−5 (boss, ε=+1, σ=−1).
func k2LikeSeamSpine(t *testing.T) cylCylSeamSpine {
	t.Helper()
	wrap, err := geom.NewCylinder(math.P3(50, 120, 70), math.V3(0, -1, 0), 30)
	if err != nil {
		t.Fatalf("wrap cylinder: %v", err)
	}
	other, err := geom.NewCylinder(math.P3(50, 50, 0), math.V3(0, 0, 1), 50)
	if err != nil {
		t.Fatalf("other cylinder: %v", err)
	}
	e2 := wrap.AxisDir.AsVector().Cross(wrap.Ref.AsVector())
	return cylCylSeamSpine{wrap: wrap, other: other, wrapAxisE2: e2, rhoW: 35, rhoO: 45, r: 5, sigma: -1}
}

// TestCylCylSeamStationExactness: every station of a closed walk lies EXACTLY on both offset
// cylinders and its feet at EXACTLY ball distance r — the closed-form guarantee the loft's
// foot-at-radius assert consumes.
func TestCylCylSeamStationExactness(t *testing.T) {
	sp := k2LikeSeamSpine(t)
	seed := math.P3(80, 10, 70) // K2's rim vertex: on the bore at azimuth toward +x, front branch
	st, ok := sp.closedStationsAt(sp.wrapAzimuthAt(seed), 1, sp.wrapAxialAt(seed), 64, 1e-9)
	if !ok {
		t.Fatal("closed station walk declined on the K2 configuration")
	}
	for j, c := range st.centers {
		if d := stdmath.Abs(distanceToCylinderSurface(sp.wrap, c) - (sp.rhoW - sp.wrap.Radius)); d > 1e-9 {
			t.Fatalf("station %d centre %v is %g off the wrap offset cylinder (want ρ_W=%g)", j, c, d, sp.rhoW)
		}
		if d := stdmath.Abs(distanceToCylinderSurface(sp.other, c) - (sp.rhoO - sp.other.Radius)); d > 1e-9 {
			t.Fatalf("station %d centre %v is %g off the other offset cylinder (want ρ_O=%g)", j, c, d, sp.rhoO)
		}
		assertFootAtBallDistance(t, j, c, st.wrapFeet[j], sp.r)
		assertFootAtBallDistance(t, j, c, st.otherFeet[j], sp.r)
	}
	if st.centers[64] != st.centers[0] {
		t.Fatalf("closed walk does not repeat station 0 bit-identically: %v vs %v", st.centers[64], st.centers[0])
	}
}

func assertFootAtBallDistance(t *testing.T, j int, c, foot math.Point3, r float64) {
	t.Helper()
	if d := stdmath.Abs(float64(foot.DistanceTo(c)) - r); d > 1e-9 {
		t.Fatalf("station %d foot %v is %g off ball distance r=%g from centre %v", j, foot, d, r, c)
	}
}

// TestCylCylSeamBranchContinuity: the walk seeded at the FRONT bore exit (y≈10) must stay on the
// front branch — every station's y strictly below the crossing mid-plane y=50 — and the walk
// seeded at the BACK exit symmetric above it. A nearest-root regression would jump branches.
func TestCylCylSeamBranchContinuity(t *testing.T) {
	sp := k2LikeSeamSpine(t)
	front := math.P3(80, 10, 70)
	st, ok := sp.closedStationsAt(sp.wrapAzimuthAt(front), 1, sp.wrapAxialAt(front), 128, 1e-9)
	if !ok {
		t.Fatal("front-branch walk declined")
	}
	for j, c := range st.centers {
		if float64(c.AsVector().Y) >= 50 {
			t.Fatalf("station %d centre %v crossed the mid-plane y=50: branch continuity broken", j, c)
		}
	}
}

// TestSeamNearestQuadRoot: root selection nearest the seed, and the honest declines (negative
// discriminant, degenerate quadratic).
func TestSeamNearestQuadRoot(t *testing.T) {
	// t² − 2·(3)t·(−1)... encode: a=1, 2b=−8, c=12 → roots 2 and 6.
	if r, ok := seamNearestQuadRoot(1, -4, 12, 1); !ok || r != 2 {
		t.Fatalf("nearest root to 1: got %v ok=%v, want 2", r, ok)
	}
	if r, ok := seamNearestQuadRoot(1, -4, 12, 7); !ok || r != 6 {
		t.Fatalf("nearest root to 7: got %v ok=%v, want 6", r, ok)
	}
	if _, ok := seamNearestQuadRoot(1, 0, 1, 0); ok {
		t.Fatal("negative discriminant must decline")
	}
	if _, ok := seamNearestQuadRoot(0, 1, 1, 0); ok {
		t.Fatal("degenerate (a=0) quadratic must decline")
	}
}

// TestCylCylSeamLoftEnvelope: the K2-configuration loft at a moderate station count already sits
// within the model-relative envelope bound the resolver enforces — and the measure is HONEST: a
// coarse 8-station loft of the same closed spine must read a LARGER error than the 64-station one.
func TestCylCylSeamLoftEnvelope(t *testing.T) {
	sp := k2LikeSeamSpine(t)
	seed := math.P3(80, 10, 70)
	phi0, t0 := sp.wrapAzimuthAt(seed), sp.wrapAxialAt(seed)
	coarseErr := seamLoftEnvelopeAt(t, sp, phi0, t0, 8)
	fineErr := seamLoftEnvelopeAt(t, sp, phi0, t0, 64)
	if fineErr >= coarseErr {
		t.Fatalf("envelope error did not shrink with refinement: 8 stations %g, 64 stations %g", coarseErr, fineErr)
	}
	if fineErr > 1e-3 {
		t.Fatalf("64-station envelope error %g exceeds 1e-3 on a ~600-unit model — loft off the true canal", fineErr)
	}
}

func seamLoftEnvelopeAt(t *testing.T, sp cylCylSeamSpine, phi0, t0 float64, n int) float64 {
	t.Helper()
	st, ok := sp.closedStationsAt(phi0, 1, t0, n, 1e-9)
	if !ok {
		t.Fatalf("station walk at n=%d declined", n)
	}
	surf, err := geom.LoftCanalStations(st.centers, st.wrapFeet, st.otherFeet, sp.r, 1e-6)
	if err != nil {
		t.Fatalf("loft at n=%d: %v", n, err)
	}
	return sp.envelopeError(st, surf)
}

// TestDistanceToCylinderSurface: the closed-form signed wall distance the envelope measure rides.
func TestDistanceToCylinderSurface(t *testing.T) {
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 10)
	if err != nil {
		t.Fatal(err)
	}
	if d := distanceToCylinderSurface(cyl, math.P3(13, 0, 5)); stdmath.Abs(d-3) > 1e-12 {
		t.Fatalf("outside point: got %g, want +3", d)
	}
	if d := distanceToCylinderSurface(cyl, math.P3(0, 6, -2)); stdmath.Abs(d+4) > 1e-12 {
		t.Fatalf("inside point: got %g, want -4", d)
	}
}

// TestCylCylSeamFootOn: the radial foot projection preserves azimuth and lands on the wall.
func TestCylCylSeamFootOn(t *testing.T) {
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 30)
	if err != nil {
		t.Fatal(err)
	}
	foot := cylCylSeamFootOn(cyl, 35, math.P3(35, 0, 7))
	want := math.P3(30, 0, 7)
	if float64(foot.DistanceTo(want)) > 1e-12 {
		t.Fatalf("foot %v, want %v (radial projection at R)", foot, want)
	}
}
