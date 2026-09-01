// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestFilletSlabColumnWatertight is the crux body-level gate for the mid-span obstacle slice (ADR-4,
// Task 6): a slab (z[-20,0]) carrying an elliptical column/tube (z[0,30]) whose footprint is a hole in
// the slab-top plane, filleted along the slab's front-top edge at r=6. The fillet's receded boundary
// (y=-7) dips into the column footprint ellipse (y∈[-10,10]), so without the obstacle rebuild the hole
// protrudes past the shrunken outer loop (HolesContained=false). With the rebuild — notched host,
// split obstacle wall, two wings and a corner-blend patch — the result must be a single, watertight,
// hole-contained solid. This drives the REAL fillet feature end to end, not the provider directly.
func TestFilletSlabColumnWatertight(t *testing.T) {
	t.Parallel()
	body := slabWithColumn(t)
	res, ok, reason := filletFrontTopEdge(body, 6)
	if !ok {
		t.Fatalf("fillet failed: %s", reason)
	}
	rep := validate.Validate(res)
	if !rep.HolesContained {
		t.Errorf("filleted slab+column must be hole-contained (no protrusion): %v", rep.Issues)
	}
	if !rep.Valid {
		t.Errorf("filleted slab+column must be a valid solid: %v", rep.Issues)
	}
	if !res.IsSolid() {
		t.Errorf("filleted slab+column must be a single closed solid")
	}
}

// TestFilletSlabObliqueColumnFallsBackToCoons is the ADR-3 do-no-harm fallback's END-TO-END gate. Since
// the surf-rst canal tier landed, EVERY obstacle in the parity corpus certifies as a canal — measured, 25
// canal builds and 0 Coons builds across all 475 corpus cases — so bsplineObstacleProvider and its four
// rails have unit tests but nothing exercising them through to a BODY. This case restores that coverage
// without touching the corpus and without a production toggle: it is the same slab-with-column shape, with
// the column's footprint rotated 45° (semi-axes 10.5/4.2 about y=-4), which puts the ellipse's
// spine-extremal point INSIDE the dip. The dip's rim then RE-ENTERS the band — its samples are not
// strictly monotone along the spine — which is exactly the honest decline obstacleCanalRimFeet documents,
// so the canal drops its payload and the straight-seam Coons model must carry the rebuild alone and still
// produce a watertight, hole-contained solid.
//
// All three legs are asserted, premise first, so the test cannot quietly stop testing the fallback: the
// dip must be non-monotone, the payload must be nil, and the patch that gets built must be the Coons tier.
func TestFilletSlabObliqueColumnFallsBackToCoons(t *testing.T) {
	t.Parallel()
	body := slabWithObliqueColumn(t)
	ef, of, _, res := obstacleFeatureFor(t, body, "oblique column", math.P3(0, -13, 0), 6)
	assertDipReEntersTheBand(t, ef, of)
	if of.Canal != nil {
		t.Fatalf("the canal tier must DECLINE on a re-entrant dip; it built %d stations", len(of.Canal.Centres))
	}
	patch, ok := resolveObstaclePatch(of, res)
	if !ok {
		t.Fatal("the Coons obstacle tier must carry the rebuild when the canal declines (ADR-3 do-no-harm)")
	}
	if patch.Kind != BlendKindBSpline {
		t.Errorf("patch Kind = %q, want the Coons tier %q", patch.Kind, BlendKindBSpline)
	}
	assertObstacleFilletIsASolid(t, body, "oblique column")
}

// assertDipReEntersTheBand pins the fixture's PREMISE: the dip's rim samples must fold back along the
// spine, which is what makes the canal decline. Without this the test would still pass if the fixture
// drifted into a monotone dip — and would then be gating the canal tier a second time, not the fallback.
func assertDipReEntersTheBand(t *testing.T, ef edgeFillet, of *ObstacleFeature) {
	t.Helper()
	ss := dipSpineStations(ef, of)
	if strictlyMonotone(ss) {
		t.Fatalf("the oblique-column dip is strictly monotone along the spine over its %d rim samples, so the canal would not decline and this test would not reach the Coons tier", len(ss))
	}
	t.Logf("dip re-enters the band: %d rim samples, spine span %.6f, first fold at sample %d",
		len(ss), ss[len(ss)-1]-ss[0], firstSpineFold(ss))
}

// firstSpineFold returns the index of the first rim sample whose spine coordinate reverses direction.
func firstSpineFold(ss []float64) int {
	for i := 2; i < len(ss); i++ {
		if (ss[i] > ss[i-1]) != (ss[1] > ss[0]) {
			return i
		}
	}
	return -1
}

// assertObstacleFilletIsASolid drives the real fillet feature on an obstacle body and requires a single
// watertight, hole-contained solid — the crux body-level condition of the whole obstacle rebuild.
func assertObstacleFilletIsASolid(t *testing.T, body *topo.Body, name string) {
	t.Helper()
	res, ok, reason := filletFrontTopEdge(body, 6)
	if !ok {
		t.Fatalf("%s: fillet failed: %s", name, reason)
	}
	rep := validate.Validate(res)
	if !rep.Valid || !rep.HolesContained || !res.IsSolid() {
		t.Errorf("%s: filleted body must be a valid, hole-contained, single closed solid (valid=%v holes=%v solid=%v): %v",
			name, rep.Valid, rep.HolesContained, res.IsSolid(), rep.Issues)
	}
}

// slabWithColumn imports the slab-with-elliptical-column fixture (an elliptical tube rising from a
// hole in a box top — the canonical mid-span obstacle shape). Built via STEP import, mirroring the
// other ops fillet fixtures (importPrismCylBorder / importNotchedPrism), so the topology is a real
// imported B-rep, not a hand-welded stub.
func slabWithColumn(t *testing.T) *topo.Body {
	t.Helper()
	return importStepSolid(t, filepath.Join("testdata", "obstacle_slab_column.step"))
}

// slabWithObliqueColumn imports the same slab+column shape with the column's elliptical footprint turned
// 45° to the filleted edge (semi-axes 10.5 / 4.2, centred at y=-4), the one difference that makes the dip
// re-enter the band. Derived from obstacle_slab_column.step by rotating and resizing its three ELLIPSE
// entities and their seam vertices; everything else — the slab, the extrusion lean, the pick — is
// identical, so the two fixtures differ in exactly the property under test.
func slabWithObliqueColumn(t *testing.T) *topo.Body {
	t.Helper()
	return importStepSolid(t, filepath.Join("testdata", "obstacle_slab_oblique_column.step"))
}

// filletFrontTopEdge drives the real fillet feature over the slab's front-top edge (midpoint
// (0,-13,0), along +X) at radius r, returning the result body plus a health flag/reason.
func filletFrontTopEdge(body *topo.Body, r float64) (*topo.Body, bool, string) {
	edge := edgeAtMidpoint(body, math.P3(0, -13, 0))
	if edge == nil {
		return nil, false, "front-top edge (midpoint 0,-13,0) not found on the imported body"
	}
	res, err := FilletEdges(body, [][]byte{edge.ReferenceKey()}, r)
	if err != nil {
		return nil, false, err.Error()
	}
	return res, true, ""
}

// edgeAtMidpoint returns the body edge whose endpoint midpoint matches mid (within a mm-scale
// tolerance), or nil when none does.
func edgeAtMidpoint(b *topo.Body, mid math.Point3) *topo.Edge {
	for _, e := range b.Edges() {
		if e.StartVertex().Point().Midpoint(e.EndVertex().Point()).DistanceTo(mid) < 1e-3 {
			return e
		}
	}
	return nil
}
