// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestSolveRunoutSpreadRejectsSelfIntersecting exercises solveRunoutSpread's own crossing
// certificate directly, on a synthetic over-radius fan (radius grossly oversized relative to the
// far edge) — the n-valent analogue of the #1800 over-radius reject. It proves the certificate
// itself still fires (Task 4/5 wired it). Renamed from TestValidateRunoutFansRejectsSelfIntersecting
// (a test-coverage-review finding): the old name claimed to test validateRunoutFans, the honest-
// reject GLUE, but called solveRunoutSpread directly — validateRunoutFans could have been replaced
// with `return nil` and this test would still pass. TestValidateRunoutFansRejectsRealFan and
// TestFilletEdgesRejectsNValentRunout below close that gap on real topology.
func TestSolveRunoutSpreadRejectsSelfIntersecting(t *testing.T) {
	t.Parallel()
	fan := endCornerFan{
		radius: 100, center: math.P3(0, 0, 0), axis: math.V3(1, 0, 0), apex: math.P3(0, 0, 0),
		ta: math.P3(0, 2, 0), tb: math.P3(0, -2, 0),
		fan:      []fanFace{{face: 1, normal: math.V3(0, 1, 0), exitEdge: 9}, {face: 2, normal: math.V3(0, -1, 0), entryEdge: 9}},
		farEdges: []fanEdge{{edge: 9, from: math.P3(0, 0, 0), to: math.P3(0, 1, 0), leftFace: 1, rightFace: 2}},
	}
	if _, err := solveRunoutSpread(fan); err == nil {
		t.Fatal("expected honest-reject on an over-radius runout")
	}
}

// TestValidateRunoutFansAcceptsV3 is the no-blanket-reject guard: V3's real valence-5 fan is
// genuinely valid (it closes to a solid, TestV3FilletClosesToSolid), so validateRunoutFans must let
// it through — proving the pre-pass rejects only genuinely invalid fans, not every fan.
func TestValidateRunoutFansAcceptsV3(t *testing.T) {
	t.Parallel()
	b := importCorpusSolid(t, "simple/V3")
	fils := solvedFilsForCase(t, b, "simple/V3")
	if err := validateRunoutFans(fils); err != nil {
		t.Fatalf("validateRunoutFans rejected V3's valid runout: %v", err)
	}
}

// tolblendC4Edge locates the occtparity tolblend_simple/C4 pick edge by its two exact endpoint
// vertices (25,0,0) and (25,0,25) — both valence-4, so classifyEndCorners builds a fan at each end.
// Found by sweeping every single-edge-pick case in the occtparity "simple"/"complex"/"bfuseblend"/
// "tolblend_simple" grids for a >3-valent runout vertex where computeEdgeFillet accepts the corpus
// radius (10) but the resulting fan's far edge is too short for the fillet tube at that radius to
// cross it once — solveRunoutSpread's "no single crossing on far edge" certificate then fires. See
// task-7-report.md "Fix pass" for the full sweep (V3's own pick and every other candidate found
// stayed acceptable across their whole convex-radius range, so this is the one real case that
// exercises the reject path).
func tolblendC4Edge(t *testing.T, b *topo.Body) *topo.Edge {
	t.Helper()
	v0 := vertexNear(t, b, math.P3(25, 0, 0))
	v1 := vertexNear(t, b, math.P3(25, 0, 25))
	return edgeBetween(t, b, v0, v1)
}

// TestValidateRunoutFansRejectsRealFan is the honest-reject GLUE proof the test-coverage finding
// asked for: tolblend_simple/C4's real valence-4 runout vertex (both ends) genuinely fails
// solveRunoutSpread's crossing certificate at its own OCCT-corpus radius (10) — unlike the earlier
// misnamed test, this calls validateRunoutFans on a fils slice built by the real computeEdgeFillet
// off real topology, so `return nil` in validateRunoutFans would now fail this test.
func TestValidateRunoutFansRejectsRealFan(t *testing.T) {
	t.Parallel()
	b := importCorpusSolid(t, "tolblend_simple/C4")
	e := tolblendC4Edge(t, b)
	fil, err := computeEdgeFillet(b, filletPick{edge: e, r0: 10, r1: 10},
		map[uint64]*cornerBlend{}, map[uint64]*cornerMiter{}, FillConcaveOutward)
	if err != nil {
		t.Fatalf("computeEdgeFillet: %v", err)
	}
	if err := validateRunoutFans([]edgeFillet{fil}); err == nil {
		t.Fatal("validateRunoutFans accepted a fan whose far edge has no single crossing (expected honest reject)")
	}
}

// TestFilletEdgesRejectsNValentRunout is the strongest form of the proof: driving the real
// production entry point (FilletEdges) on tolblend_simple/C4 at its own corpus radius must fail
// with the runout-certificate wording, not silently ship an open shell — showing the reject
// surfaces all the way through filletResolvedEdges, not just through the pre-pass in isolation.
func TestFilletEdgesRejectsNValentRunout(t *testing.T) {
	t.Parallel()
	b := importCorpusSolid(t, "tolblend_simple/C4")
	e := tolblendC4Edge(t, b)
	_, err := FilletEdges(b, [][]byte{e.ReferenceKey()}, 10)
	if err == nil {
		t.Fatal("FilletEdges accepted a 4-valent runout fan with no single crossing (expected honest reject)")
	}
	if !strings.Contains(err.Error(), "valent runout vertex") {
		t.Errorf("FilletEdges error = %q, want it to mention the runout-certificate wording", err)
	}
}
