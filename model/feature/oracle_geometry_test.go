//go:build oracle

// SPDX-License-Identifier: GPL-2.0-only

// Package feature's analytic work-feature geometry (tangent/axis) cross-checked against an
// external CAD kernel oracle — OpenCASCADE via the _oracles/oracle_service.py JSON CLI (see
// architecture/decisions or memory oracle-kernels-occt-solvespace). These tests are excluded from
// the normal build (the `oracle` build tag; CI has no Python/OCCT) and skip unless the oracle env is
// set. Run locally:
//
//	OBK_ORACLE_PY=/c/Users/vmiguel/miniforge3/envs/occ/python.exe \
//	OBK_ORACLE_SVC=/c/Users/vmiguel/git/oblikovati-workspace/_oracles/oracle_service.py \
//	go test -tags oracle -run Oracle ./model/feature/...
//
// The oracle is the source of truth for the golden constants frozen into the normal (CI-run)
// geometry tests; these tests re-derive those values live and assert the Go kernel matches.
package feature

import (
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// oracleResult shells out to the OCCT oracle service for op(args) and returns its JSON object.
func oracleResult(t *testing.T, op string, args ...string) map[string]any {
	t.Helper()
	py, svc := os.Getenv("OBK_ORACLE_PY"), os.Getenv("OBK_ORACLE_SVC")
	if py == "" || svc == "" {
		t.Skip("oracle cross-check: set OBK_ORACLE_PY (conda occ python) and OBK_ORACLE_SVC (_oracles/oracle_service.py)")
	}
	out, err := exec.Command(py, append([]string{svc, op}, args...)...).Output()
	if err != nil {
		t.Fatalf("oracle %s %v: %v", op, args, err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("oracle %s: bad JSON %q: %v", op, out, err)
	}
	return m
}

func ftoa(v float64) string { return strconv.FormatFloat(v, 'g', 17, 64) }

// pairs2D reads a JSON [[x,y],[x,y]] value into a sorted [][2]float64 for order-independent compare.
func pairs2D(t *testing.T, v any) [][2]float64 {
	t.Helper()
	rows := v.([]any)
	out := make([][2]float64, len(rows))
	for i, r := range rows {
		xy := r.([]any)
		out[i] = [2]float64{xy[0].(float64), xy[1].(float64)}
	}
	if len(out) == 2 && out[0][1] > out[1][1] { // sort by y (both share x here)
		out[0], out[1] = out[1], out[0]
	}
	return out
}

// TestOracleCylinderTangentNormals cross-checks cylinderTangentNormals (#1844) against OCCT's
// GccAna_Lin2d2Tan (line tangent to a circle): the two contact points of the axis-parallel tangent
// lines from an external line must match to kernel tolerance.
func TestOracleCylinderTangentNormals(t *testing.T) {
	t.Parallel()
	const R, dist = 2.0, 5.0
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), R)
	if err != nil {
		t.Fatal(err)
	}
	m1, m2, err := cylinderTangentNormals(math.P3(dist, 0, 0), cyl)
	if err != nil {
		t.Fatal(err)
	}
	// Contact point = axis foot (origin here) + R·m; the normal m is the contact radial.
	got := [][2]float64{
		{R * float64(m1.AsVector().X), R * float64(m1.AsVector().Y)},
		{R * float64(m2.AsVector().X), R * float64(m2.AsVector().Y)},
	}
	if got[0][1] > got[1][1] {
		got[0], got[1] = got[1], got[0]
	}
	want := pairs2D(t, oracleResult(t, "tangent", ftoa(R), ftoa(dist))["contacts"])
	for i := range want {
		if abs(got[i][0]-want[i][0]) > 1e-9 || abs(got[i][1]-want[i][1]) > 1e-9 {
			t.Errorf("tangent contact %d: Go=%v OCCT=%v", i, got[i], want[i])
		}
	}
}

// TestOracleRevolvedFaceAxisCone cross-checks revolvedFaceAxis (#1840) against OCCT gp_Cone: the
// axis direction matches and the apex OCCT reports is exactly on that axis (so using Apex as the
// axis point is sound).
func TestOracleRevolvedFaceAxisCone(t *testing.T) {
	t.Parallel()
	const halfAngle, refRadius = 0.5, 2.0
	res := oracleResult(t, "revolved_axis", "cone", ftoa(halfAngle), ftoa(refRadius))
	ap := res["apex"].([]any)
	apex := math.P3(ap[0].(float64), ap[1].(float64), ap[2].(float64))
	cone, err := geom.NewCone(apex, math.V3(0, 0, 1), halfAngle)
	if err != nil {
		t.Fatal(err)
	}
	o, d, err := revolvedFaceAxis(cone)
	if err != nil {
		t.Fatal(err)
	}
	if !d.AsVector().IsParallelTo(math.V3(0, 0, 1), 1e-9) {
		t.Errorf("cone axis dir = %v, want +Z (OCCT axis_dir %v)", d, res["axis_dir"])
	}
	if !o.IsEqualTo(apex, 1e-9) {
		t.Errorf("cone axis origin = %v, want the OCCT apex %v", o, apex)
	}
	if res["apex_on_axis_dist"].(float64) > 1e-9 {
		t.Errorf("OCCT reports apex off axis by %v", res["apex_on_axis_dist"])
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
