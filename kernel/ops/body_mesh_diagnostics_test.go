// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/test-utilities/degenerate"
)

// TestBodyMeshDiagnosticsHarvestsTheTessellatorsReport is the #2058 kernel-side regression: what the
// mesher recorded onto its tessellate.Mesh must be readable from outside this package, or it reaches nobody.
func TestBodyMeshDiagnosticsHarvestsTheTessellatorsReport(t *testing.T) {
	t.Parallel()
	got := BodyMeshDiagnostics(degenerate.CrossedTrimBody(), tessellate.DefaultQuality())
	if len(got) != 1 {
		t.Fatalf("harvested %d diagnostics from a two-face self-crossing trim, want 1 collapsed entry: %v",
			len(got), got)
	}
	if got[0].Code != tessellate.CodePatchCoverage || got[0].Severity != diag.Defect {
		t.Errorf("harvested %v, want a %s Defect", got[0], tessellate.CodePatchCoverage)
	}
	// Both faces carry the flaw, so the collapsed entry must say so rather than imply a single face.
	if !strings.Contains(got[0].Detail, "meshing 2 of the body's faces") {
		t.Errorf("collapsed detail %q does not report the 2 affected faces", got[0].Detail)
	}
}

// TestBodyMeshDiagnosticsStaysSilentOnCleanBodies guards the false-positive side: the ordinary
// analytic corpus meshes cleanly, and a channel that cries on healthy geometry is one users learn to
// ignore (#2058's third acceptance).
func TestBodyMeshDiagnosticsStaysSilentOnCleanBodies(t *testing.T) {
	t.Parallel()
	for name, b := range cleanPrimitiveBodies(t) {
		if got := BodyMeshDiagnostics(b, tessellate.DefaultQuality()); len(got) != 0 {
			t.Errorf("clean %s reported %v, want none", name, got)
		}
	}
}

// cleanPrimitiveBodies builds one of each analytic solid the mesher has an exact path for.
func cleanPrimitiveBodies(t *testing.T) map[string]*topo.Body {
	t.Helper()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 10)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	sph, err := brep.SolidSphere(math.P3(0, 0, 0), 5, "s")
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	tor, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "t")
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	return map[string]*topo.Body{"cylinder": cyl, "sphere": sph, "torus": tor}
}

// TestBodyMeshDiagnosticsOnNilBodyIsEmpty: the harvest is called on whatever a feature returned, and
// a nil body in that slice must not panic the recompute it is reporting on.
func TestBodyMeshDiagnosticsOnNilBodyIsEmpty(t *testing.T) {
	t.Parallel()
	if got := BodyMeshDiagnostics(nil, tessellate.DefaultQuality()); got != nil {
		t.Errorf("nil body reported %v, want nil", got)
	}
}
