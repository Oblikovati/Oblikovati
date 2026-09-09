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

// TestTheHarvestCarriesEveryCodeTheFaceMeshesDo is the wiring, stated as an identity.
//
// This row drove a body that TORE, and asserted the tear came out of BodyMeshDiagnostics. Nothing in
// the corpus tears any more — Task 7 gave the merged band its own mesher and ADR-0061 stage 5 round 3
// gave the merge a chart that matches its own edges — so that precondition is gone, and the row says
// so rather than hunting for a body that still cracks.
//
// What it guards instead is the defect that made the tear invisible in the first place: the check ran
// somewhere the harvest could not see it. The harvest's codes must be EXACTLY the codes the face
// meshes carry. That identity holds whether or not anything is torn today, and it fails the moment a
// tessellation defect is recorded anywhere the faces route does not reach — which is the whole of what
// went wrong (ADR-0061 stage 5, review round 2). The step before it, "the body's tear is recorded on a
// FACE mesh", is pinned in kernel/ops/tessellate.
func TestTheHarvestCarriesEveryCodeTheFaceMeshesDo(t *testing.T) {
	t.Parallel()
	body := nearPinchCrossingRods(t)
	_, meshes := tessellate.TessellateBodyFaces(body, PropertyQuality())
	want := codeSet(faceMeshDiagnostics(meshes))
	if len(want) == 0 {
		t.Fatal("no face of this body records anything; the identity would hold vacuously")
	}
	assertSameCodes(t, codeSet(BodyMeshDiagnostics(body, PropertyQuality())), want)
}

// nearPinchCrossingRods is the one corpus body whose faces still record a tessellation degradation at
// PropertyQuality — two rods crossing with a 4e-5 radius difference, whose wall's two lens windows
// leave a corridor narrower than the boundary's own chords. It is the fixture that makes the identity
// above a proof rather than a tautology.
func nearPinchCrossingRods(t *testing.T) *topo.Body {
	t.Helper()
	along, err := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	if err != nil {
		t.Fatalf("along-x rod: %v", err)
	}
	across, err := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3.00004, 12)
	if err != nil {
		t.Fatalf("along-z rod: %v", err)
	}
	body, err := brep.Boolean(brep.Union, along, across)
	if err != nil {
		t.Fatalf("crossing rods: %v", err)
	}
	return body
}

// faceMeshDiagnostics is everything the face meshes recorded, in meshing order.
func faceMeshDiagnostics(meshes []*tessellate.Mesh) []diag.Diagnostic {
	var out []diag.Diagnostic
	for _, m := range meshes {
		if m != nil {
			out = append(out, m.Diagnostics...)
		}
	}
	return out
}

// codeSet is the distinct codes of a diagnostic list.
func codeSet(ds []diag.Diagnostic) map[diag.Code]bool {
	out := map[diag.Code]bool{}
	for _, d := range ds {
		out[d.Code] = true
	}
	return out
}

// assertSameCodes requires the two sets to hold the same codes.
func assertSameCodes(t *testing.T, got, want map[diag.Code]bool) {
	t.Helper()
	for c := range want {
		if !got[c] {
			t.Errorf("the face meshes recorded %q and the harvest dropped it", c)
		}
	}
	for c := range got {
		if !want[c] {
			t.Errorf("the harvest reported %q that no face mesh recorded", c)
		}
	}
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
