// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/ops/boolean"

	"oblikovati.org/kernel/ops"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/math"
)

// TestBooleanRefusesAnUnmodelledConfigurationByName proves the diag channel end to end on the boolean
// side. It used to prove that a configuration with no exact path fell back to triangle-soup CSG and
// RECORDED the fallback as a searchable Defect (#1407). ADR-0061 stage 7 deleted the engine behind
// that record, so the same fixture now proves the stronger contract: the operation REFUSES by name,
// with the refusal both on the error and in the diagnostics, and no body is returned at all.
// It asserts the DECLINE, never a faceted body: when this configuration lands analytically the
// assertion converts to a positive corpus case rather than being deleted to move a number.
func TestBooleanRefusesAnUnmodelledConfigurationByName(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	// A ball joined to a TORUS: their crossing is a quartic space curve on both surfaces, which no
	// closed form in the intersector claims, so this is a configuration that genuinely has no exact
	// path. It replaces the sphere PAIR this test used to decline on, which now lands analytically —
	// the conversion the assertion was written to make (ADR-0061); TestSpherePairVolumesAreExact is
	// its positive form.
	tor, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "t")
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	ball, err := brep.SolidSphere(math.P3(5, 0, 0), 3, "ball")
	if err != nil {
		t.Fatalf("ball: %v", err)
	}

	var rec diag.Recorder
	res, err := ops.BooleanWithDiagnostics(ops.Join, tor, ball, &rec)
	if !errors.Is(err, ops.ErrUnmodelledBoolean) {
		t.Fatalf("a configuration no exact path models must be refused by name; got err=%v", err)
	}
	if res != nil {
		t.Fatalf("a refused boolean must return no body; got %d faces", len(res.Faces()))
	}
	if !rec.Has(ops.CodeBooleanNoExactCurvedPath) {
		t.Errorf("the refusal recorded no %q diagnostic; got %v", ops.CodeBooleanNoExactCurvedPath, rec.Records())
	}
	if rec.Count(diag.Defect) == 0 {
		t.Error("a refusal must record a Defect-severity diagnostic")
	}
}

// TestBooleanExactPathRecordsNoDiagnostic confirms the channel is quiet on success: an exact analytic
// crossing-cylinder boolean (unequal radii — handled by the curved exact path) records no defect, so a
// recorded defect always means a real degradation, never noise.
func TestBooleanExactPathRecordsNoDiagnostic(t *testing.T) {
	t.Parallel()
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	thin, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)

	var rec diag.Recorder
	if _, err := ops.BooleanWithDiagnostics(ops.Intersect, fat, thin, &rec); err != nil {
		t.Fatalf("ops.BooleanWithDiagnostics: %v", err)
	}
	if n := rec.Count(diag.Defect); n != 0 {
		t.Errorf("an exact crossing-cylinder boolean recorded %d defect(s), want 0: %v", n, rec.Records())
	}
}

// TestBooleanNilRecorderStillWorks confirms the legacy ops.Boolean entry point (which passes a nil
// recorder) is unaffected: the same path runs, just unobserved.
func TestBooleanNilRecorderStillWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	a, _ := brep.SolidSphere(math.P3(0, 0, 0), 2, "a")
	b, _ := brep.SolidSphere(math.P3(2, 0, 0), 2, "b")
	if res, err := ops.Boolean(ops.Intersect, a, b); err != nil || res == nil {
		t.Fatalf("ops.Boolean (nil recorder) = %v, %v; want a result and no error", res, err)
	}
}

// TestMeshCarriesDiagnostics covers the tessellation carrier: a diagnostic recorded on a component mesh
// surfaces on the composed mesh through tessellate.MergeMesh — the path a deep tessellation degradation takes to
// the final face/body mesh (#1412).
func TestMeshCarriesDiagnostics(t *testing.T) {
	t.Parallel()
	child := &boolean.Mesh{}
	child.Diagnose(diag.Diagnostic{Code: "tessellate.cap-saturated", Severity: diag.Defect, Detail: "face X below tol"})
	parent := &boolean.Mesh{}
	tessellate.MergeMesh(parent, child)
	if len(parent.Diagnostics) != 1 || parent.Diagnostics[0].Code != "tessellate.cap-saturated" {
		t.Errorf("tessellate.MergeMesh did not carry the child mesh's diagnostics up: %v", parent.Diagnostics)
	}
}
