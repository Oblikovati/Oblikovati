// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// grazingSphereOnCylinder is a small sphere set just inside a cylinder's wall so their surfaces graze
// — the near-tangent contact no exact path claims. It was the mesh-arrangement rescue's fixture; with
// that engine gone (ADR-0061 stage 6) it is the corpus row for the NAMED refusal.
func grazingSphereOnCylinder(t *testing.T) (*topo.Body, *topo.Body) {
	t.Helper()
	a, err := brep.SolidCylinder(math.P3(0, 0, -2), math.V3(0, 0, 1), 1, 4)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	b, err := brep.SolidSphere(math.P3(1.4, 0, 0), 0.5, "s")
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	return a, b
}

// The guarded curved entry has four exits, and three of them recorded. "No exact path claims this" did
// not: the operation quietly produced triangle soup and nothing downstream could say why. A refusal is
// a diag.Defect that must reach feature health, the API and the UI (ADR-0061), and with the faceted
// engines gone (stage 7) it is also the operation's error rather than a stand-in body.
func TestABooleanWithNoExactCurvedPathRefusesByName(t *testing.T) {
	t.Parallel()
	a, b := grazingSphereOnCylinder(t)
	rec := &diag.Recorder{}
	body, err := BooleanWithDiagnostics(Join, a, b, rec)
	if !errors.Is(err, ErrUnmodelledBoolean) {
		t.Fatalf("a curved join no analytic path claims must be refused by name; got err=%v", err)
	}
	if body != nil {
		t.Fatalf("a refused boolean must return no body; got %d faces", len(body.Faces()))
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
