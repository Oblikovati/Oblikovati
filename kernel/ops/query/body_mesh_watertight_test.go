// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The body's tessellation post-condition has to reach a USER, and this package is the seam it reaches
// them through: BodyMeshDiagnostics is the #2058 harvest a feature reply, the API and the UI read
// (model/feature/result_diagnostics.go). A defect recorded only on the whole-body mesh reaches none of
// them — every TessellateBody caller discards that mesh — so the row that matters is this one.

// TestATornClosedBodyReportsThroughTheHarvest: a closed body whose mesh has a crack must carry it on
// BodyMeshDiagnostics — the list a feature reply, the API and the UI read.
//
// The #2167 cocylindrical join used to tear at DefaultQuality and that is what this row drove. It does
// not any more (ADR-0061 stage 5, Task 7 round 1: the router offers a seam-wrapping face on a singly
// periodic surface its own chart, the chart's membership test reads a slanted seam, and the rim gate no
// longer counts a seam SLIT twice). At PropertyQuality the same body still tears, on a chart the MERGE
// records 0.0198 rad off the edges the face carries, so the row drives it there and stays a real proof
// of the harvest. It fails loudly if that tear closes too — which is the correct signal to re-point it
// at whatever still tears, or to delete it if nothing does.
func TestATornClosedBodyReportsThroughTheHarvest(t *testing.T) {
	t.Parallel()
	body := cocylindricalBossOnWall(t)
	mesh, _ := tessellate.TessellateBody(body, PropertyQuality())
	if n := tessellate.FreeEdgeCount(mesh); n == 0 {
		t.Fatalf("the fixture meshes watertight; it is not the torn case this row needs")
	}
	assertHarvested(t, BodyMeshDiagnostics(body, PropertyQuality()), tessellate.CodeMeshNotWatertight)
}

// TestAWatertightBodyHarvestsNoTear is the control: a plain cylinder meshes closed, and the harvest
// must stay silent. A diagnostic that fires on the ordinary body is noise on every feature reply.
func TestAWatertightBodyHarvestsNoTear(t *testing.T) {
	t.Parallel()
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 6)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	for _, d := range BodyMeshDiagnostics(body, DefaultQuality()) {
		if d.Code == tessellate.CodeMeshNotWatertight {
			t.Errorf("a watertight body harvested %q: %s", d.Code, d.Detail)
		}
	}
}

// cocylindricalBossOnWall is the #2167 shape from primitives: a boss whose wall is cocylindrical with
// its host's, flattened on one side so the merged wall's second rim is notched.
func cocylindricalBossOnWall(t *testing.T) *topo.Body {
	t.Helper()
	host, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 6)
	if err != nil {
		t.Fatalf("host cylinder: %v", err)
	}
	upper, err := brep.SolidCylinder(math.P3(0, 0, 6), math.V3(0, 0, 1), 3, 4)
	if err != nil {
		t.Fatalf("boss cylinder: %v", err)
	}
	chop, err := brep.SolidBlock(math.P3(2.4, -4, 5), math.P3(5, 4, 11), "chop")
	if err != nil {
		t.Fatalf("chop block: %v", err)
	}
	boss, err := brep.Boolean(brep.Difference, upper, chop)
	if err != nil {
		t.Fatalf("flattening the boss: %v", err)
	}
	body, err := brep.Boolean(brep.Union, host, boss)
	if err != nil {
		t.Fatalf("seating the boss: %v", err)
	}
	return body
}

// assertHarvested requires the code on the harvest at Defect severity.
func assertHarvested(t *testing.T, ds []diag.Diagnostic, code diag.Code) {
	t.Helper()
	for _, d := range ds {
		if d.Code == code && d.Severity == diag.Defect {
			return
		}
	}
	t.Errorf("BodyMeshDiagnostics did not carry %q; got %v", code, ds)
}
