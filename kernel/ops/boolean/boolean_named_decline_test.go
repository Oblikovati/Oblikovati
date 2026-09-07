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

// sphereOnCylinderWall is a small sphere set across a cylinder's wall, so their surfaces meet in one
// closed seam. It has been three things in a row, which is the retirement's whole story on one fixture:
// the mesh-arrangement rescue's case, then — with that engine gone (ADR-0061 stage 6) — the corpus row
// for the NAMED refusal, and now an exact result. What changed is the ruled∩quadric section: the ruling
// meets the sphere over part of the cylinder's sweep only, so the section FOLDS, and stage 5 gave the
// closed form the folded window it was missing (geom.RuledQuadricLoop).
func sphereOnCylinderWall(t *testing.T) (*topo.Body, *topo.Body) {
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

// torusMeetingSphere is a pair NO closed form covers: a torus is quartic, so it has no implicit quadric
// to substitute a ruling into and neither the wrap form nor the folded window applies. It is the corpus
// row for the named refusal — the exit that must stay loud.
func torusMeetingSphere(t *testing.T) (*topo.Body, *topo.Body) {
	t.Helper()
	a, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "t")
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	b, err := brep.SolidSphere(math.P3(5, 0, 0), 2, "s")
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
	a, b := torusMeetingSphere(t)
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

// TestSphereCrossingACylinderWallIsExact is the same fixture from the other side: the folded window is
// a closed form like any other, so all three operations come back as valid analytic solids with no
// degradation recorded. Their volumes are certified against an independent membership integral in
// boolean_folded_window_test.go; here the point is that the operation SUCCEEDS and says nothing.
func TestSphereCrossingACylinderWallIsExact(t *testing.T) {
	t.Parallel()
	a, b := sphereOnCylinderWall(t)
	for _, op := range []PartFeatureOperation{Join, Cut, Intersect} {
		rec := &diag.Recorder{}
		body, err := BooleanWithDiagnostics(op, a, b, rec)
		if err != nil {
			t.Fatalf("%v: %v", op, err)
		}
		if r := Validate(body); !r.ValidSolid() {
			t.Fatalf("%v: not a valid solid: %+v", op, r)
		}
		if rec.Has(CodeBooleanNoExactCurvedPath) || rec.Count(diag.Defect) != 0 {
			t.Errorf("%v: the folded window is exact and must record nothing; got %v", op, rec.Records())
		}
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
