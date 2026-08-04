// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// runoutFixture solves a real edgeFillet on the occtparity S1 corpus fixture (a box slab carrying
// two independent bosses — an r8 vertical boss on the top plane, an r6 horizontal boss on the front
// plane — filleted along their shared front-top edge). Driving the real computeEdgeFillet off real
// imported topology (mirroring solvedFilsForCase in fillet_runout_fan_test.go), rather than hand-
// assembling a synthetic edgeFillet, is deliberate: edgeFillet's corner fields (ta/tb tangent
// points, cen, axis) are interdependent invariants that filletFrame/solvedEdgeFillet solve for —
// hand-rolling them risks a fixture that looks plausible but violates those invariants silently.
// S1 is also the exact motivating case for this detector (plan
// docs/superpowers/plans/2026-07-14-curved-runout-imprint-fillet.md), so this is the highest-
// fidelity fixture available, not just the path of least resistance.
func runoutFixture(t *testing.T, radius float64) (edgeFillet, Resolution) {
	t.Helper()
	b := importCorpusSolid(t, "simple/S1")
	e := edgeAtMidpoint(b, math.P3(0, -10, 10))
	if e == nil {
		t.Fatal("runoutFixture: front-top edge (midpoint 0,-10,10) not found on S1")
	}
	fil, err := computeEdgeFillet(b, filletPick{edge: e, r0: radius, r1: radius},
		map[uint64]*cornerBlend{}, map[uint64]*cornerMiter{}, FillConcaveOutward)
	if err != nil {
		t.Fatalf("runoutFixture: computeEdgeFillet(r=%v): %v", radius, err)
	}
	return fil, ResolutionForSize(50)
}

// runoutFixtureCrossingBoss is S1 at its own corpus radius (6): both the r8 top boss and the r6
// front boss dip into the receded fillet band, one on each of the fillet's two host planes.
func runoutFixtureCrossingBoss(t *testing.T) (edgeFillet, Resolution) {
	t.Helper()
	return runoutFixture(t, 6)
}

// runoutFixtureBehindBand is the SAME S1 body and picked edge at a much smaller radius (1): the
// receded boundary then sits well inside the edge (top host y=-9, front host z=9), clear of both
// bosses (top boss spans y∈[-8,8], front boss spans z∈[-6,6]) — so neither footprint reaches the
// band at all. Verified by exploration (go test -run TestExploreS1Fillet, since removed): both
// hosts report zero crossings at r=1, vs two independent crossings each at r=6.
func runoutFixtureBehindBand(t *testing.T) (edgeFillet, Resolution) {
	t.Helper()
	return runoutFixture(t, 1)
}

// TestDetectRunouts_S1BossesCrossBothHostsIndependently is the crux case this detector exists for:
// S1's two bosses each cross their own host plane's receded band, and BOTH must be admitted as
// independent imprints — the opposite of detectObstacle, which rejects this exact shape as a
// dual-host corner pierce (qualifying==2). Named for what it actually asserts (two imprints, not
// one) rather than the brief's original single-boss working title, since the highest-fidelity
// fixture available (real S1) genuinely has two independent crossing bosses.
func TestDetectRunouts_S1BossesCrossBothHostsIndependently(t *testing.T) {
	ef, res := runoutFixtureCrossingBoss(t)
	got := detectRunouts(ef, res)
	if len(got) != 2 {
		t.Fatalf("want 2 imprints (one per host, S1's two independent bosses), got %d: %+v", len(got), got)
	}
	assertDistinctNodesAndHosts(t, got)
}

// assertDistinctNodesAndHosts checks each returned imprint has two distinct band-crossing nodes,
// and that the two imprints (when there are two) are on different hosts — proving detectRunouts
// truly ran both hosts independently rather than reporting the same host twice.
func assertDistinctNodesAndHosts(t *testing.T, got []runoutImprint) {
	t.Helper()
	for i, im := range got {
		if im.nodes[0].P == im.nodes[1].P {
			t.Errorf("imprint %d: want two distinct nodes, got coincident: %+v", i, im.nodes)
		}
	}
	if len(got) == 2 && got[0].hostIsA == got[1].hostIsA {
		t.Errorf("want the two imprints on different hosts, both had hostIsA=%v", got[0].hostIsA)
	}
	if len(got) == 2 && got[0].host == got[1].host {
		t.Errorf("want the two imprints on different *topo.Face hosts, got the same face twice")
	}
}

// TestDetectRunouts_NonCrossingBoss proves the honest-reject side: when neither boss reaches the
// receded band (the fixture's radius shrunk to 1, see runoutFixtureBehindBand), detectRunouts must
// return no imprints at all — leaving the fillet whole, the benign non-crossing case (S3/S6/S7/T3
// siblings, per fillet_obstacle_detect_face.go's dual-host doc comment).
func TestDetectRunouts_NonCrossingBoss(t *testing.T) {
	ef, res := runoutFixtureBehindBand(t)
	if got := detectRunouts(ef, res); len(got) != 0 {
		t.Fatalf("non-crossing bosses must produce no imprint, got %d: %+v", len(got), got)
	}
}

// TestDetectRunouts_VaryingEdgeRejected pins the varying-radius honest-reject: a curved-spine
// (variable-radius) fillet sweeps a torus/canal band, not the constant-radius cylinder this
// detector's cross-section math assumes, so it must bail before touching any host.
func TestDetectRunouts_VaryingEdgeRejected(t *testing.T) {
	ef, res := runoutFixtureCrossingBoss(t)
	ef.varying = true
	if got := detectRunouts(ef, res); len(got) != 0 {
		t.Fatalf("varying fillet edge must be rejected outright, got %d imprints", len(got))
	}
}
