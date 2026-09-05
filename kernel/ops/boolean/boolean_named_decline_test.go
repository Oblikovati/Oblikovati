// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/math"
)

// The guarded curved entry has four exits, and three of them recorded. "No exact path claims this" did
// not: the operation quietly produced triangle soup and nothing downstream could say why. A fallback is
// a diag.Defect that must reach feature health, the API and the UI (ADR-0061).
func TestABooleanWithNoExactCurvedPathDeclinesByName(t *testing.T) {
	t.Parallel()
	a, b := grazingSphereOnCylinder(t)
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Join, a, b, rec); err != nil {
		t.Fatalf("boolean returned error: %v", err)
	}
	if !rec.Has(CodeBooleanNoExactCurvedPath) {
		t.Error("a curved join no analytic path claims must record the named decline")
	}
}

// The counterpart: an ALL-PLANAR pair loses nothing when the curved paths decline — the planar B-rep
// path takes it exactly — so recording a degradation there would be noise on every boolean in the
// system.
func TestAnAllPlanarBooleanDeclinesSilently(t *testing.T) {
	t.Parallel()
	a, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "a")
	if err != nil {
		t.Fatalf("block a: %v", err)
	}
	b, err := brep.SolidBlock(math.P3(1, 1, 1), math.P3(3, 3, 3), "b")
	if err != nil {
		t.Fatalf("block b: %v", err)
	}
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Cut, a, b, rec); err != nil {
		t.Fatalf("boolean returned error: %v", err)
	}
	if rec.Has(CodeBooleanNoExactCurvedPath) {
		t.Error("an all-planar cut is exact on the planar path: it must not report a curved decline")
	}
}
