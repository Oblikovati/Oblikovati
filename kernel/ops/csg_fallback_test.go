// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// The B-rep boolean (kernel/brep) only handles planar-faceted solids; when an operand has a
// curved face it returns ErrNonPlanar and ops.Boolean falls back to the triangle-soup BSP
// CSG (csg.go / csg_body.go / csg_split.go). These tests drive that fallback through every
// operation — the path a user hits whenever a boolean involves a fillet/cylinder/sphere.
//
// To reach the fallback the operands must (a) have a curved face and (b) be classified as
// genuinely intersecting (partial overlap), so classify() routes Join/Intersect/Cut through
// booleanGeneral rather than the disjoint/contained shortcuts. A straddling tool box parked
// clear of the filleted corner gives an exact planar overlap volume while the operand stays
// curved.

// csgFallbackTol bounds the volume check. The CSG result re-tessellates the curved bar at
// boolInputQuality (the faceting ops.Boolean meshes curved operands at — see
// booleanInputQuality in csg_body.go), the same quality the expected bar volume is measured
// at, so the curved contribution cancels and only float/weld noise remains; the planar
// overlap (1.0) is exact.
const csgFallbackTol = 1e-2

// boolInputQuality mirrors the (unexported) faceting ops.Boolean meshes curved operands at:
// the display chord tolerance with the angular refinement disabled. The fallback bakes the
// curved operand as planar facets at THIS quality, so the expected result volume must measure
// the curved bar here (not the finer DefaultQuality) for the curved part to cancel exactly.
func boolInputQuality() ops.Quality {
	return ops.Quality{ChordTolerance: ops.DefaultQuality().ChordTolerance, AngleTolerance: stdmath.Pi}
}

// curvedBarWithStraddlingTool builds a 4×3×2 bar with one vertical edge filleted (r=0.5) and a
// 2×1×1 tool that pokes out the +X face — overlap [3,4]×[1,2]×[0.5,1.5] = 1, parked away from
// any filleted corner so the overlap is exact. It returns the operands and the measured
// (tessellated) volume of the curved bar, against which results are checked.
func curvedBarWithStraddlingTool(t *testing.T) (bar, tool *topo.Body, barVol float64) {
	t.Helper()
	box := shellBox(4, 3, 2)
	curved, err := ops.FilletEdges(box, [][]byte{verticalEdgeKey(t, box)}, 0.5)
	if err != nil {
		t.Fatalf("fillet: %v", err)
	}
	if hasCylinderFaces(curved) == 0 {
		t.Fatal("operand has no curved face — fallback would not be exercised")
	}
	return curved, csgBox(math.P3(3, 1, 0.5), 2, 1, 1), ops.BodyGeometryProperties(curved, boolInputQuality()).Volume
}

// assertCSGSolid checks the fallback produced a valid solid whose faces are all planar (the
// triangle-soup CSG re-tessellates the curved operand, so no cylinder face survives — proof
// the fallback, not the B-rep path, ran) with the expected volume.
func assertCSGSolid(t *testing.T, res *topo.Body, wantVol float64) {
	t.Helper()
	if res == nil {
		t.Fatal("nil result body")
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("CSG result not a valid solid: %+v", r)
	}
	if n := hasCylinderFaces(res); n != 0 {
		t.Errorf("CSG result kept %d cylinder faces; the fallback should triangulate", n)
	}
	if got := csgVolume(res); stdmath.Abs(got-wantVol) > csgFallbackTol {
		t.Errorf("CSG volume = %g, want ≈ %g (±%g)", got, wantVol, csgFallbackTol)
	}
}

// TestCSGFallbackJoin: curved bar ∪ straddling tool adds the tool's outside half (tool − overlap
// = 2 − 1 = 1) to the bar.
func TestCSGFallbackJoin(t *testing.T) {
	bar, tool, barVol := curvedBarWithStraddlingTool(t)
	res, err := ops.Boolean(ops.Join, bar, tool)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	assertCSGSolid(t, res, barVol+1)
}

// TestCSGFallbackCut: curved bar − straddling tool removes the overlap slab (1.0).
func TestCSGFallbackCut(t *testing.T) {
	bar, tool, barVol := curvedBarWithStraddlingTool(t)
	res, err := ops.Boolean(ops.Cut, bar, tool)
	if err != nil {
		t.Fatalf("cut: %v", err)
	}
	assertCSGSolid(t, res, barVol-1)
}

// TestCSGFallbackIntersect: curved bar ∩ straddling tool keeps only the overlap slab (1.0).
func TestCSGFallbackIntersect(t *testing.T) {
	bar, tool, _ := curvedBarWithStraddlingTool(t)
	res, err := ops.Boolean(ops.Intersect, bar, tool)
	if err != nil {
		t.Fatalf("intersect: %v", err)
	}
	assertCSGSolid(t, res, 1)
}
