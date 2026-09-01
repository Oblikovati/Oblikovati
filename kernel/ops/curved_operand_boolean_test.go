// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A boolean with a curved (fillet/cylinder) operand cannot go through the planar B-rep path.
// With the ADR-0056 Layer-5 cutover (reconstructionCutover), such a boolean is rebuilt
// ANALYTICALLY from the exact mesh boolean's provenance whenever its new boundary edges are all
// lines and circles (weldableSSICurve): the fillet cylinder SURVIVES and the volume is exact —
// where the pre-reconstruction triangle-soup CSG fallback would have faceted the fillet away.
// These drive that path through every operation. The triangle-soup CSG fallback still runs for
// the cases reconstruction declines (an oblique-conic SSI); TestReconstructDeclinesObliqueConicCut
// covers that it stays correct.

// analyticFilletedBarVolume is the exact volume of a 4×3×2 bar with one vertical edge (height 2)
// filleted at r=0.5: the fillet removes r²(1−π/4) of cross-section over the full height.
func analyticFilletedBarVolume() float64 {
	const r, h = 0.5, 2.0
	return 4*3*2 - r*r*(1-stdmath.Pi/4)*h
}

// curvedBarWithStraddlingTool builds a 4×3×2 bar with one vertical edge filleted (r=0.5) and a
// 2×1×1 tool that pokes out the +X face — overlap [3,4]×[1,2]×[0.5,1.5] = 1, parked away from the
// filleted corner so the overlap is an exact planar 1.0 while the operand stays curved.
func curvedBarWithStraddlingTool(t *testing.T) (bar, tool *topo.Body) {
	t.Helper()
	box := shellBox(4, 3, 2)
	curved, err := ops.FilletEdges(box, [][]byte{verticalEdgeKey(t, box)}, 0.5)
	if err != nil {
		t.Fatalf("fillet: %v", err)
	}
	if hasCylinderFaces(curved) == 0 {
		t.Fatal("operand has no curved face — the reconstruction path would not be exercised")
	}
	return curved, csgBox(math.P3(3, 1, 0.5), 2, 1, 1)
}

// assertReconstructedSolid checks the result is a valid solid whose fillet wall survives as an
// analytic cylinder (wantCyl of them — proof reconstruction, not the faceting CSG, ran) at the
// exact analytic volume.
func assertReconstructedSolid(t *testing.T, res *topo.Body, wantCyl int, wantVol float64) {
	t.Helper()
	if res == nil {
		t.Fatal("nil result body")
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("result is not a valid solid: %+v", r)
	}
	if n := hasCylinderFaces(res); n != wantCyl {
		t.Errorf("result kept %d cylinder faces, want %d (reconstruction preserves the fillet)", n, wantCyl)
	}
	if got := csgVolume(res); stdmath.Abs(got-wantVol) > 1e-2 {
		t.Errorf("volume = %g, want ≈ %g (±0.01)", got, wantVol)
	}
}

// TestCurvedBarJoinReconstructs: curved bar ∪ straddling tool adds the tool's outside half
// (tool − overlap = 2 − 1 = 1) and keeps the fillet analytic.
func TestCurvedBarJoinReconstructs(t *testing.T) {
	t.Parallel()
	bar, tool := curvedBarWithStraddlingTool(t)
	res, err := ops.Boolean(ops.Join, bar, tool)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	assertReconstructedSolid(t, res, 1, analyticFilletedBarVolume()+1)
}

// TestCurvedBarCutReconstructs: curved bar − straddling tool removes the overlap slab (1.0) and
// keeps the fillet analytic.
func TestCurvedBarCutReconstructs(t *testing.T) {
	t.Parallel()
	bar, tool := curvedBarWithStraddlingTool(t)
	res, err := ops.Boolean(ops.Cut, bar, tool)
	if err != nil {
		t.Fatalf("cut: %v", err)
	}
	assertReconstructedSolid(t, res, 1, analyticFilletedBarVolume()-1)
}

// TestCurvedBarIntersectReconstructs: curved bar ∩ straddling tool keeps only the overlap slab,
// which lies away from the filleted corner — so the result is a plain planar box (no cylinder).
func TestCurvedBarIntersectReconstructs(t *testing.T) {
	t.Parallel()
	bar, tool := curvedBarWithStraddlingTool(t)
	res, err := ops.Boolean(ops.Intersect, bar, tool)
	if err != nil {
		t.Fatalf("intersect: %v", err)
	}
	assertReconstructedSolid(t, res, 0, 1)
}
