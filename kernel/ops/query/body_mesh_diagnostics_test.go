// SPDX-License-Identifier: GPL-2.0-only

package query_test

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/test-utilities/degenerate"
)

// TestBodyMeshDiagnosticsHarvestsTheTessellatorsReport is the #2058 kernel-side regression: what the
// mesher recorded onto its tessellate.Mesh must be readable from outside this package, or it reaches nobody.
func TestBodyMeshDiagnosticsHarvestsTheTessellatorsReport(t *testing.T) {
	t.Parallel()
	got := query.BodyMeshDiagnostics(degenerate.CrossedTrimBody(), tessellate.DefaultQuality())
	// The harvest collapses PER CODE, so the row reads the code it is about rather than the total. It
	// used to assert a total of 1, which was the same thing while this body raised one code; it now
	// also misses its chord tolerance and says so (CodeFaceChordNotMet), and a count would have made
	// that honest new report look like a harvesting bug.
	d, found := harvestedCode(got, tessellate.CodePatchCoverage)
	if !found {
		t.Fatalf("harvested %v from a two-face self-crossing trim, want a %s entry",
			got, tessellate.CodePatchCoverage)
	}
	if d.Severity != diag.Defect {
		t.Errorf("harvested %v, want a %s Defect", d, tessellate.CodePatchCoverage)
	}
	// Both faces carry the flaw, so the collapsed entry must say so rather than imply a single face.
	if !strings.Contains(d.Detail, "meshing 2 of the body's faces") {
		t.Errorf("collapsed detail %q does not report the 2 affected faces", d.Detail)
	}
	// One entry per code is what "collapsed" means, and that is what the row actually guards.
	seen := map[diag.Code]int{}
	for _, e := range got {
		seen[e.Code]++
	}
	for code, n := range seen {
		if n != 1 {
			t.Errorf("the harvest carries %s %d times; per-face reports must collapse to one entry", code, n)
		}
	}
}

// TestBodyMeshDiagnosticsStaysSilentOnCleanBodies guards the false-positive side: the ordinary
// analytic corpus meshes cleanly, and a channel that cries on healthy geometry is one users learn to
// ignore (#2058's third acceptance).
func TestBodyMeshDiagnosticsStaysSilentOnCleanBodies(t *testing.T) {
	t.Parallel()
	for name, b := range cleanPrimitiveBodies(t) {
		if got := query.BodyMeshDiagnostics(b, tessellate.DefaultQuality()); len(got) != 0 {
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
	if got := query.BodyMeshDiagnostics(nil, tessellate.DefaultQuality()); got != nil {
		t.Errorf("nil body reported %v, want nil", got)
	}
}

// harvestedCode is the harvested entry for one code, if the body raised it.
func harvestedCode(ds []diag.Diagnostic, code diag.Code) (diag.Diagnostic, bool) {
	for _, d := range ds {
		if d.Code == code {
			return d, true
		}
	}
	return diag.Diagnostic{}, false
}
