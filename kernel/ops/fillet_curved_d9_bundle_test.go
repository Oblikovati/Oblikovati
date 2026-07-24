// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// D9 is OCCT blend/simple's 270° REFLEX sphere-host trihedral corner. Its cylinder arm (edge 14) rolls
// on the top cap (plane 16) and a longitude plane (26); the cap ruling's true outer end is the far-vertex
// STATION (−10, 0, 129.9038). Because the wedge is reflex that station lies INTERIOR to the cap's 270°
// sector, on no loop edge — so armRulingEnd's loop-crossing contract is unsatisfiable and, before D9-T1,
// the whole arm rail bundle declined with "host contact rail or setback rail could not be built"
// (d9-rail-bundle-forensic.md §1). D9-T1 adds the interior-station fallback (rulingStationOuter) plus the
// rayArc2d forward-only filter; together they make the cylinder-arm bundle BUILD with the cap ruling
// ending at the station. The whole-body weld is T2, so the full D9 corpus case stays RED after T1 (it
// now floors later, at the top-cap corner-host retrim — asserted by TestOCCTBlendSimple/D9).

// d9CapStation is the cylinder arm's cap-ruling outer end — the far vertex (0,0,capZ) projected onto the
// ruling, landing at (−10, 0, capZ). capZ is the exact cap-plane height (the corpus/DRAWEXE value).
var d9CapStation = math.P3(-10, 0, 129.9038105676658)

// TestD9CylinderArmBundleBuilds drives the real imported D9 body through the corner solve to
// armRailBundle for the cylinder arm and asserts the bundle now builds (D9-T1 deliverable) with a host
// contact rail whose outer end is the interior far-vertex station. Reverting the rulingStationOuter
// fallback re-reddens this (the D9-T1 mutation check).
func TestD9CylinderArmBundleBuilds(t *testing.T) {
	body := corpusFixture(t, "simple/D9.step")
	arms, w, res := d9CornerArms(t, body)
	i := d9CylinderArmIndex(t, w)
	bundle, reason := armRailBundle(w.arms[i], arms[i], w, filletedEdgeSet(arms), res)
	if reason != "" {
		t.Fatalf("D9 cylinder-arm rail bundle declined: %q (want a built bundle)", reason)
	}
	if len(bundle.segs) == 0 {
		t.Fatalf("D9 cylinder-arm bundle built no rails (segs empty)")
	}
	tol := res.Weld() * w.radius
	outer, ok := d9CapRulingOuter(bundle, d9CapStation, tol)
	if !ok {
		t.Fatalf("no cylinder-arm host rail ends at the cap station %v within %.1e; hostA.from=%v hostB.from=%v",
			d9CapStation, tol, bundle.hostA.from, bundle.hostB.from)
	}
	t.Logf("D9 cap ruling outer end = %v, station %v, residual %.3e (tol %.1e)",
		outer, d9CapStation, outer.DistanceTo(d9CapStation), tol)
}

// d9CornerArms drives the real fillet pipeline on the imported D9 body up to the solved trihedral corner:
// it locates the three constant-radius picks, resolves + solves them (CornerMiter — the corpus strategy),
// gathers the corner arms at the shared vertex, and solves the sphere-host corner. It returns the arms
// (index-aligned with w.arms), the solved corner, and the body resolution — all armRailBundle needs.
func d9CornerArms(t *testing.T, body *topo.Body) ([]edgeFillet, cornerWeld, Resolution) {
	t.Helper()
	edges, err := resolveFilletPicks(body, d9Picks(t, body))
	if err != nil {
		t.Fatalf("resolve D9 picks: %v", err)
	}
	blends, miters, err := computeCorners(body, edges)
	if err != nil {
		t.Fatalf("compute D9 corners: %v", err)
	}
	fils, err := computeFillets(body, edges, blends, miters, FillConcaveOutward, nil)
	if err != nil {
		t.Fatalf("compute D9 fillets: %v", err)
	}
	vid, ok := sharedCornerVertex(curvedArmsOf(fils))
	if !ok {
		t.Fatalf("D9 curved arms share no trihedral vertex")
	}
	arms := cornerArms(fils, vid)
	res := ResolutionForBody(body)
	w, _, reason := solveCurvedArmCorner(arms, blends, vid, res)
	if reason != "" {
		t.Fatalf("D9 corner solve declined: %s", reason)
	}
	return arms, w, res
}

// d9Picks locates D9's three constant-radius (r=10) picks on the imported body by the OCCT corpus locator
// midpoints (test-utilities/occt-blend oracle) and returns them keyed by reference key.
func d9Picks(t *testing.T, body *topo.Body) []EdgeFilletRadii {
	t.Helper()
	mids := []math.Point3{
		math.P3(-53.03300859, 53.03300859, 129.9038106),
		math.P3(0, -37.5, 129.9038106),
		math.P3(0, -150, 0),
	}
	picks := make([]EdgeFilletRadii, len(mids))
	for i, m := range mids {
		picks[i] = EdgeFilletRadii{Key: d9EdgeNearestMidpoint(t, body, m).ReferenceKey(), R0: 10, R1: 10}
	}
	return picks
}

// d9EdgeNearestMidpoint returns the body edge whose parametric midpoint is closest to m (the corpus
// locator midpoint). It fails when nothing is within 1 model unit — a topology divergence, not a pick.
func d9EdgeNearestMidpoint(t *testing.T, body *topo.Body, m math.Point3) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := stdmath.Inf(1)
	for _, e := range body.Edges() {
		if d := float64(e.Geometry().PointAt(0.5).DistanceTo(m)); d < bestD {
			bestD, best = d, e
		}
	}
	if best == nil || bestD > 1 {
		t.Fatalf("no D9 edge near locator midpoint %v (nearest at %.3e)", m, bestD)
	}
	return best
}

// d9CylinderArmIndex returns the index of the corner's single cylinder arm (edge 14) among w.arms.
func d9CylinderArmIndex(t *testing.T, w cornerWeld) int {
	t.Helper()
	for i, a := range w.arms {
		if _, ok := a.arm.(geom.Cylinder); ok {
			return i
		}
	}
	t.Fatalf("no cylinder arm among the %d D9 corner arms", len(w.arms))
	return -1
}

// d9CapRulingOuter returns the bundle's host contact rail outer end that lands at the cap station (either
// hostA or hostB), or ok=false when neither does within tol.
func d9CapRulingOuter(b armRails, station math.Point3, tol float64) (math.Point3, bool) {
	for _, h := range []endSeg{b.hostA, b.hostB} {
		if float64(h.from.DistanceTo(station)) <= tol {
			return h.from, true
		}
	}
	return math.Point3{}, false
}
