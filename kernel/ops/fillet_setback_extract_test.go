// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestExtractSetbackPatches_S1IntactFootprintRibbon is the crux test: the S1 fixture (r8 top boss +
// r6 front boss, filleted R=6) tiles into exactly three setback RailLoops (left flank / central /
// right flank), each a closed valence-4 loop whose footprint rail carries the INTACT boss wall (a
// full geom.Cylinder) as its Adjacent — the D3 correction over the boss's whole, unsplit footprint
// rail (never the split sub-arcs of the deleted boss-splitting path).
//
// The footprint side is G0, not G1. It was declared G1 while a Coons fill needed a tangency ribbon
// there; the rolling-ball derivation shows the run-out ball passes THROUGH the footprint EDGE and is
// NOT tangent to the boss wall (checked on S6: the surface normal at the mid-band footprint point is
// (0,−1,2.828)/3 against the sphere's (0,−1,0)), so declaring G1 asserted a tangency OCCT's own blend
// does not have. See .superpowers/sdd/runout-envelope-report.md.
func TestExtractSetbackPatches_S1IntactFootprintRibbon(t *testing.T) {
	t.Parallel()
	ef, res := runoutFixtureCrossingBoss(t)
	b, ok := detectSetbackBands(ef, res)
	if !ok {
		t.Fatalf("detectSetbackBands: ok=false, want the S1 two-boss bands")
	}
	loops, ok := extractSetbackPatches(b, ef, res)
	if !ok || len(loops) != 3 {
		t.Fatalf("extractSetbackPatches: ok=%v loops=%d, want 3 (flank/central/flank)", ok, len(loops))
	}
	for i, lp := range loops {
		if lp.Valence() != 4 {
			t.Errorf("loop %d valence=%d, want 4", i, lp.Valence())
		}
		if !lp.Closed(res.Weld()) {
			t.Fatalf("loop %d not closed within weld=%v", i, res.Weld())
		}
		assertFootprintOnIntactWall(t, lp, ef)
	}
	assertBothBossRadiiPresent(t, loops, ef)
}

// TestExtractSetbackPatches_LoopsResolveToValidPatches is Step 4: every emitted loop must be one the
// blend engine accepts — resolveBlend returns a certificate-Valid patch (coons4 over the intact-boss
// rails), proving the rails are well-formed, not just closed.
func TestExtractSetbackPatches_LoopsResolveToValidPatches(t *testing.T) {
	t.Parallel()
	ef, res := runoutFixtureCrossingBoss(t)
	b, _ := detectSetbackBands(ef, res)
	loops, ok := extractSetbackPatches(b, ef, res)
	if !ok {
		t.Fatalf("extractSetbackPatches: ok=false")
	}
	for i, lp := range loops {
		if _, ok := resolveBlend(lp, res); !ok {
			t.Fatalf("loop %d: resolveBlend did not certify a valid patch", i)
		}
	}
}

// TestExtractSetbackPatches_NonS1Rejected pins the honest-reject scope: a non-two-boss / non-dual-host
// input (here the behind-band fixture, which detects no bands at all) must yield ok=false, never a
// partial or wrong tiling.
func TestExtractSetbackPatches_NonS1Rejected(t *testing.T) {
	t.Parallel()
	ef, res := runoutFixtureBehindBand(t)
	b, _ := detectSetbackBands(ef, res)
	if _, ok := extractSetbackPatches(b, ef, res); ok {
		t.Fatalf("extractSetbackPatches: want ok=false when no S1 two-boss bands are present")
	}
}

// assertG1FootprintOnIntactWall is the load-bearing assertion: every footprint side (a side whose
// Adjacent is a boss cylinder — axis NOT parallel to the fillet spine, distinguishing it from the
// fillet-end arc's own fillet-cylinder Adjacent) must be G1 and carry a full, intact geom.Cylinder
// wall (r8 or r6), NOT a nil/G0 split sub-arc as the old split-boss tiler emitted.
func assertFootprintOnIntactWall(t *testing.T, lp RailLoop, ef edgeFillet) {
	t.Helper()
	found := 0
	for _, s := range lp.Sides {
		cyl, isCyl := s.Adjacent.(geom.Cylinder)
		if !isCyl || axisParallel(cyl.AxisDir, ef.cyl.AxisDir) {
			continue // nil/plane Adjacent, or the fillet-end arc's own fillet cylinder: not a footprint
		}
		found++
		if s.Cont != G0 {
			t.Errorf("footprint side (boss r=%v): Cont=%v, want G0 (the ball passes THROUGH the footprint edge)", cyl.Radius, s.Cont)
		}
		if cyl.Radius != 8 && cyl.Radius != 6 {
			t.Errorf("footprint Adjacent radius=%v, want an intact boss wall (r8 or r6)", cyl.Radius)
		}
	}
	if found == 0 {
		t.Errorf("loop has no footprint side on an intact boss cylinder wall")
	}
}

// assertBothBossRadiiPresent checks the tiling reaches BOTH intact boss walls across the three loops:
// the r8 outer wall (both flanks + central) and the r6 inner wall (central only) must each appear as
// a footprint Adjacent — proving the central patch runs out to two walls, not one (D2).
func assertBothBossRadiiPresent(t *testing.T, loops []RailLoop, ef edgeFillet) {
	t.Helper()
	var saw8, saw6 bool
	for _, lp := range loops {
		for _, s := range lp.Sides {
			cyl, isCyl := s.Adjacent.(geom.Cylinder)
			if !isCyl || axisParallel(cyl.AxisDir, ef.cyl.AxisDir) {
				continue
			}
			saw8 = saw8 || cyl.Radius == 8
			saw6 = saw6 || cyl.Radius == 6
		}
	}
	if !saw8 || !saw6 {
		t.Errorf("intact boss walls reached: r8=%v r6=%v, want both (central runs out to both)", saw8, saw6)
	}
}

// axisParallel reports whether two unit axes are (anti)parallel — used to tell a boss-wall Adjacent
// (axis perpendicular to the fillet spine) from the fillet cylinder's own axis.
func axisParallel(a, b math.UnitVector3) bool {
	return stdmath.Abs(float64(a.Dot(b))) > 0.99
}
