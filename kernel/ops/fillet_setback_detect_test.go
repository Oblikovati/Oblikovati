// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestDetectSetbackBands_S1TwoBosses pins the D1/D2 stations against the S1 fixture (box slab,
// top-front edge R=6; r8 top boss at a=10 crosses first/outer, r6 front boss at a=10 crosses
// second/inner) — the same DRAWEXE-verified numbers as the derivation doc
// (.superpowers/sdd/setback-patch-derivation.md, D1/D2): outer x_s=√48≈6.9282 (r8 boss),
// inner x_s=√20≈4.4721 (r6 boss). Uses the existing runoutFixtureCrossingBoss helper
// (fillet_runout_detect_test.go) — NOT a new fixture — since it already builds exactly this
// substrate off real S1 topology (edgeFillet's corner fields are interdependent invariants a
// hand-rolled fixture would risk violating silently).
func TestDetectSetbackBands_S1TwoBosses(t *testing.T) {
	ef, res := runoutFixtureCrossingBoss(t)
	b, ok := detectSetbackBands(ef, res)
	if !ok || len(b.bosses) != 2 {
		t.Fatalf("detectSetbackBands: ok=%v bosses=%d, want ok=true, 2 crossing bosses (r8,r6)", ok, len(b.bosses))
	}
	if stdmath.Abs(b.cutHi-stdmath.Sqrt(48)) > 1e-6 {
		t.Fatalf("outer setback cutHi = %v, want sqrt(48)=6.9282 (r8 top boss)", b.cutHi)
	}
	if stdmath.Abs(b.cutLo+stdmath.Sqrt(48)) > 1e-6 {
		t.Fatalf("outer setback cutLo = %v, want -sqrt(48)=-6.9282", b.cutLo)
	}
	if len(b.seams) != 2 || stdmath.Abs(stdmath.Abs(b.seams[0])-stdmath.Sqrt(20)) > 1e-6 {
		t.Fatalf("inner seam = %v, want ±sqrt(20)=4.4721 (r6 front boss)", b.seams)
	}
}

// TestDetectSetbackBands_BossesOrderedByReachDescending pins the ordering contract
// (crossingBoss's doc comment: "ordered by |xSetback| descending") and that each boss's wall is
// kept INTACT (a non-nil geom.Surface, never split) — the whole point of Task 2 over the old
// split-boss tiler.
func TestDetectSetbackBands_BossesOrderedByReachDescending(t *testing.T) {
	ef, res := runoutFixtureCrossingBoss(t)
	b, ok := detectSetbackBands(ef, res)
	if !ok || len(b.bosses) != 2 {
		t.Fatalf("detectSetbackBands: ok=%v bosses=%d, want ok=true, 2 bosses", ok, len(b.bosses))
	}
	if b.bosses[0].xSetback <= b.bosses[1].xSetback {
		t.Fatalf("bosses not ordered by reach descending: [0].xSetback=%v [1].xSetback=%v",
			b.bosses[0].xSetback, b.bosses[1].xSetback)
	}
	for i, boss := range b.bosses {
		if boss.wall == nil {
			t.Errorf("boss %d: wall surface is nil, want the intact boss wall (never split)", i)
		}
		if boss.footEdge == nil || boss.host == nil {
			t.Errorf("boss %d: footEdge=%v host=%v, want both set", i, boss.footEdge, boss.host)
		}
	}
}

// TestDetectSetbackBands_NonCrossingBossRejected mirrors TestDetectRunouts_NonCrossingBoss: when
// neither boss reaches the receded band (runoutFixtureBehindBand's shrunk radius), there is no
// interference to detect at all, so detectSetbackBands must honest-reject (ok=false) rather than
// return an empty-but-valid setbackBands.
func TestDetectSetbackBands_NonCrossingBossRejected(t *testing.T) {
	ef, res := runoutFixtureBehindBand(t)
	if _, ok := detectSetbackBands(ef, res); ok {
		t.Fatalf("detectSetbackBands: want ok=false when no boss crosses the receded band")
	}
}

// TestNonDegenerateSetback_NearTangencyClamp pins the D1 conditioning guard (derivation
// "Numerical pitfalls": x_s² < (k·res.Weld())² ⇒ no real interference — a boss just tangent to
// the contact line, never trust a barely-positive root near the tangency singularity): a reach
// at weld-noise scale must NOT read as a genuine crossing, while an S1-scale reach must.
func TestNonDegenerateSetback_NearTangencyClamp(t *testing.T) {
	res := ResolutionForSize(50)
	tiny := res.Weld() * (setbackNearZeroCoef - 1)
	if nonDegenerateSetback(tiny, res) {
		t.Fatalf("nonDegenerateSetback(%v): want false (below the (%v*Weld)^2 floor)", tiny, setbackNearZeroCoef)
	}
	real := stdmath.Sqrt(20.0) // S1's r6 inner setback — comfortably real, not a graze
	if !nonDegenerateSetback(real, res) {
		t.Fatalf("nonDegenerateSetback(%v): want true (S1-scale reach, not a graze)", real)
	}
}

// centeredTestCylinder is a synthetic R=6 fillet cylinder along +Z — the same radius as the S1
// corpus fixture (runoutFixtureCrossingBoss), so the centeredness threshold under test
// (setbackCenteredCoef*R) matches the real corpus scale without needing full topology (geom.
// Cylinder is a plain value type — hand-building it is safe here, unlike edgeFillet's
// interdependent corner fields; see runoutFixtureCrossingBoss's doc comment).
func centeredTestCylinder(t *testing.T) geom.Cylinder {
	t.Helper()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 6)
	if err != nil {
		t.Fatalf("centeredTestCylinder: geom.NewCylinder: %v", err)
	}
	return cyl
}

// TestSetbackMidpoint_S1BossesAgree pins the discovery that corrected this fix (see
// bossesShareTransverseMidplane's doc comment): S1's two independently-solved, verified-correct
// crossing bosses do NOT read c=(lo+hi)/2≈0 — cyl.Origin sits at an arbitrary edge endpoint, not
// the physical transverse midplane — but they DO agree with each other (both ≈10, matching the
// r8/r6 stations TestDetectSetbackBands_S1TwoBosses pins). A guard that rejected |c|≈0 literally
// would break this real, correct corpus case; this test is the regression that would catch that
// mistake reappearing.
func TestSetbackMidpoint_S1BossesAgree(t *testing.T) {
	ef, res := runoutFixtureCrossingBoss(t)
	var mids []float64
	for _, im := range detectRunouts(ef, res) {
		cut, ok := solveImprint(im, res)
		if !ok {
			continue
		}
		mids = append(mids, setbackMidpoint(cut, ef.cyl))
	}
	if len(mids) != 2 {
		t.Fatalf("S1 fixture: got %d solved imprints, want 2 (r8 top, r6 front)", len(mids))
	}
	if stdmath.Abs(mids[0]-mids[1]) > 1e-6 {
		t.Fatalf("S1 bosses' midpoints disagree: %v vs %v, want them to agree (D2 'same centered span')", mids[0], mids[1])
	}
	if !bossesShareTransverseMidplane(mids, ef.cyl) {
		t.Fatalf("bossesShareTransverseMidplane(%v): want true for S1's real, verified-correct bosses", mids)
	}
}

// TestBossesShareTransverseMidplane_DisagreeingBossRejected is the Task 2 review-finding
// regression test: two bosses whose axial midpoints differ by a full fillet radius (10 vs 4, not
// float noise) must fail bossesShareTransverseMidplane. This is the exact silent-mis-partition
// scenario the guard exists to catch — D2's bandsFromBosses would otherwise happily emit a
// plausible-looking symmetric cutLo/cutHi/seams straddling a plane neither boss is actually
// centered on, with no indication anything was wrong.
func TestBossesShareTransverseMidplane_DisagreeingBossRejected(t *testing.T) {
	cyl := centeredTestCylinder(t)
	mids := []float64{10, 4} // S1-scale outer midpoint vs one full R (6) off it
	if bossesShareTransverseMidplane(mids, cyl) {
		t.Fatalf("bossesShareTransverseMidplane(%v, R=%v): want false, disagreement is a full R, not noise", mids, cyl.Radius)
	}
}

// TestBossesShareTransverseMidplane_AgreeingBossesAccepted mirrors S1's real numbers (both bosses
// read c≈10, differing only by float noise) as a synthetic, fast unit test independent of the
// corpus import — the do-no-harm half of the guard.
func TestBossesShareTransverseMidplane_AgreeingBossesAccepted(t *testing.T) {
	cyl := centeredTestCylinder(t)
	mids := []float64{9.999999999999996, 10}
	if !bossesShareTransverseMidplane(mids, cyl) {
		t.Fatalf("bossesShareTransverseMidplane(%v, R=%v): want true, disagreement is float-noise scale", mids, cyl.Radius)
	}
}

// TestBossesShareTransverseMidplane_SingleBossTriviallyAccepted pins the documented limitation
// (setbackBands' doc comment): a lone boss has nothing to disagree with, so this guard cannot
// detect a single off-center boss — only a multi-boss mismatch.
func TestBossesShareTransverseMidplane_SingleBossTriviallyAccepted(t *testing.T) {
	cyl := centeredTestCylinder(t)
	if !bossesShareTransverseMidplane([]float64{123.456}, cyl) {
		t.Fatalf("bossesShareTransverseMidplane(single boss): want true, nothing to disagree with")
	}
}

// TestWithinCenteredTolerance_NoiseNeverRejected pins the do-no-harm requirement itself (Task 2
// review finding: "a centered boss must never be rejected by numerical noise"): an offset many
// orders of magnitude above spineParam's actual float64 noise floor (~1e-14 relative to model
// size) but still far below the setbackCenteredCoef*R=0.06 threshold must still read as within
// tolerance.
func TestWithinCenteredTolerance_NoiseNeverRejected(t *testing.T) {
	cyl := centeredTestCylinder(t)
	noise := 1e-9
	if !withinCenteredTolerance(noise, cyl) {
		t.Fatalf("withinCenteredTolerance(%v, R=%v): want true, this is float-noise scale, not a real offset", noise, cyl.Radius)
	}
}
