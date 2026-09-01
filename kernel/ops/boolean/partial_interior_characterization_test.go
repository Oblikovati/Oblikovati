// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"testing"

	"oblikovati.org/kernel/ops"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Keep-interior characterization (#1591, ADR-0049 D-b, Slice C). The partial curved-on-planar paths
// (CutEdgeScallop, JoinPartialBoss) must take ONLY the pierced-but-not-clean contact; a circle wholly inside
// one face stays the strictly-interior fast-path's job (DrillThroughHole / JoinCylindricalBoss). These tests
// pin that boundary: the partial recognizers decline the interior cases, and ops.Boolean still routes the
// interior contact to its analytic fast-path (a handful of faces, an analytic cylinder wall), not to the
// partial path nor to CSG.

// TestInteriorDrillStaysThroughHole: a centered drill (circle strictly inside both caps) declines the scallop
// path and stays DrillThroughHole — the interior/partial gate boundary is clean on the subtractive side.
func TestInteriorDrillStaysThroughHole(t *testing.T) {
	t.Parallel()
	plate, _ := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	drill, _ := brep.SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 2, 4) // centered → strictly interior
	if _, ok := brep.CutEdgeScallop(plate, drill); ok {
		t.Error("scallop recognizer accepted a strictly-interior hole; want decline")
	}
	res, err := ops.Boolean(ops.Cut, plate, drill)
	if err != nil {
		t.Fatalf("ops.Boolean(ops.Cut): %v", err)
	}
	assertAnalyticCylinderSolid(t, res, "interior drill")
}

// TestInteriorBossStaysCylindricalBoss: a boss whose base circle is strictly inside the seat face declines the
// straddling path and stays JoinCylindricalBoss — the gate boundary is clean on the additive side.
func TestInteriorBossStaysCylindricalBoss(t *testing.T) {
	t.Parallel()
	plate, _ := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	boss, _ := brep.SolidCylinder(math.P3(0, 0, 2), math.V3(0, 0, 1), 2, 3) // centered on the top face → interior
	if _, ok := brep.JoinPartialBoss(plate, boss); ok {
		t.Error("boss recognizer accepted a strictly-interior seat; want decline")
	}
	res, err := ops.Boolean(ops.Join, plate, boss)
	if err != nil {
		t.Fatalf("ops.Boolean(ops.Join): %v", err)
	}
	assertAnalyticCylinderSolid(t, res, "interior boss")
}

// assertAnalyticCylinderSolid checks a boolean result is the analytic fast-path output: a watertight solid
// with few faces (not CSG triangle-soup) that preserves an analytic cylinder wall.
func assertAnalyticCylinderSolid(t *testing.T, res *topo.Body, label string) {
	t.Helper()
	if !res.IsSolid() {
		t.Errorf("%s: result is not a solid", label)
	}
	if n := len(res.Faces()); n > 20 {
		t.Errorf("%s: result has %d faces; want the analytic fast-path (few), not CSG", label, n)
	}
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return
		}
	}
	t.Errorf("%s: result kept no analytic cylinder face (a CSG fallback)", label)
}
