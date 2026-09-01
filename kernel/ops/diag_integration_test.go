// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/math"
)

// TestBooleanRecordsCSGFallbackDiagnostic proves the diag channel end to end on the boolean side: a
// curved configuration with no exact analytic/planar path (two overlapping spheres) falls back to
// triangle-soup CSG, and that fallback is now RECORDED as a searchable Defect instead of silently
// shipping a faceted mesh (the #1407 guardrail this infrastructure enables).
func TestBooleanRecordsCSGFallbackDiagnostic(t *testing.T) {
	t.Parallel()
	a, err := brep.SolidSphere(math.P3(0, 0, 0), 2, "a")
	if err != nil {
		t.Fatalf("sphere a: %v", err)
	}
	b, err := brep.SolidSphere(math.P3(2, 0, 0), 2, "b") // overlaps a; sphere∩sphere has no exact handler
	if err != nil {
		t.Fatalf("sphere b: %v", err)
	}

	var rec diag.Recorder
	res, err := BooleanWithDiagnostics(Intersect, a, b, &rec)
	if err != nil {
		t.Fatalf("BooleanWithDiagnostics: %v", err)
	}
	if res == nil {
		t.Fatal("nil result")
	}
	if !rec.Has(CodeBooleanCSGFallback) {
		t.Errorf("curved boolean fell back to CSG but recorded no %q diagnostic; got %v", CodeBooleanCSGFallback, rec.Records())
	}
	if rec.Count(diag.Defect) == 0 {
		t.Error("a CSG fallback must record a Defect-severity diagnostic")
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
	if _, err := BooleanWithDiagnostics(Intersect, fat, thin, &rec); err != nil {
		t.Fatalf("BooleanWithDiagnostics: %v", err)
	}
	if n := rec.Count(diag.Defect); n != 0 {
		t.Errorf("an exact crossing-cylinder boolean recorded %d defect(s), want 0: %v", n, rec.Records())
	}
}

// TestBooleanNilRecorderStillWorks confirms the legacy Boolean entry point (which passes a nil recorder)
// is unaffected: the same fallback runs, just unobserved.
func TestBooleanNilRecorderStillWorks(t *testing.T) {
	t.Parallel()
	a, _ := brep.SolidSphere(math.P3(0, 0, 0), 2, "a")
	b, _ := brep.SolidSphere(math.P3(2, 0, 0), 2, "b")
	if res, err := Boolean(Intersect, a, b); err != nil || res == nil {
		t.Fatalf("Boolean (nil recorder) = %v, %v; want a result and no error", res, err)
	}
}

// TestMeshCarriesDiagnostics covers the tessellation carrier: a diagnostic recorded on a component mesh
// surfaces on the composed mesh through mergeMesh — the path a deep tessellation degradation takes to
// the final face/body mesh (#1412).
func TestMeshCarriesDiagnostics(t *testing.T) {
	t.Parallel()
	child := &Mesh{}
	child.Diagnose(diag.Diagnostic{Code: "tessellate.cap-saturated", Severity: diag.Defect, Detail: "face X below tol"})
	parent := &Mesh{}
	mergeMesh(parent, child)
	if len(parent.Diagnostics) != 1 || parent.Diagnostics[0].Code != "tessellate.cap-saturated" {
		t.Errorf("mergeMesh did not carry the child mesh's diagnostics up: %v", parent.Diagnostics)
	}
}
