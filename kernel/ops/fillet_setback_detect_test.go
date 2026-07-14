// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"
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
	if math.Abs(b.cutHi-math.Sqrt(48)) > 1e-6 {
		t.Fatalf("outer setback cutHi = %v, want sqrt(48)=6.9282 (r8 top boss)", b.cutHi)
	}
	if math.Abs(b.cutLo+math.Sqrt(48)) > 1e-6 {
		t.Fatalf("outer setback cutLo = %v, want -sqrt(48)=-6.9282", b.cutLo)
	}
	if len(b.seams) != 2 || math.Abs(math.Abs(b.seams[0])-math.Sqrt(20)) > 1e-6 {
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
	real := math.Sqrt(20.0) // S1's r6 inner setback — comfortably real, not a graze
	if !nonDegenerateSetback(real, res) {
		t.Fatalf("nonDegenerateSetback(%v): want true (S1-scale reach, not a graze)", real)
	}
}
