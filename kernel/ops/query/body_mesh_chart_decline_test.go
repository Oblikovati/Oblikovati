// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The corpus row for Oblikovati/Oblikovati#3520: a chart-driven mesher decline reaches the harvest a
// feature reply, the API and the UI read.
//
// The body is a real boolean through the general pipeline, not a hand-built face: two crossing rods
// intersected, r = 3 along +x and r = 2.5 along +z offset 1 mm off its axis. At chord 0.001 the
// intersection's r = 3 wall carries a chart, the classification selects the chart-driven mesher for
// it, and the mesher gives it up — its covering is not bounded by its own rim — so the face ships from
// the router's fall-through instead.
//
// This body is the case that was FULLY silent. It is not the wrapping wall recordUnmeshedWallWrap
// already flags and not the whole-domain grid recordIgnoredTrim already flags; measured, it raises
// neither, so before this change nothing anywhere said the selected mesher had given the face up. The
// row asserts that absence too, or it would stop being the silent case the moment a wrapping body were
// substituted for it.

// TestAChartMesherDeclineReachesTheBodysMeshDiagnostics is the wiring, on a body that reaches it.
func TestAChartMesherDeclineReachesTheBodysMeshDiagnostics(t *testing.T) {
	t.Parallel()
	body := offsetCrossingRods(t)
	got := BodyMeshDiagnostics(body, PropertyQuality())
	d, found := findCode(got, tessellate.CodeChartMesherDeclined)
	if !found {
		t.Fatalf("the harvest carries %v; the chart mesher gave a face of this body up and nothing said so",
			codeList(got))
	}
	if d.Severity != diag.Defect {
		t.Errorf("the decline is reported at %v, want a Defect — the covering that shipped is one the "+
			"chart never certified", d.Severity)
	}
	if !strings.Contains(d.Detail, "not bounded by its own rim") {
		t.Errorf("the report does not name WHY the mesher gave up, so a reader cannot act on it: %q", d.Detail)
	}
}

// TestTheSilentDeclineWasSilent is what makes the row above a regression rather than a restatement:
// the two codes that already existed for a lost trim are BOTH absent from this body, so the decline it
// carries was reported by nothing before #3520.
func TestTheSilentDeclineWasSilent(t *testing.T) {
	t.Parallel()
	got := codeSet(BodyMeshDiagnostics(offsetCrossingRods(t), PropertyQuality()))
	for _, c := range []diag.Code{tessellate.CodeWallWrapUnmeshed, tessellate.CodeTrimIgnoredFullDomain} {
		if got[c] {
			t.Errorf("this body also raises %q, so it is not the silent case the row exists to pin — "+
				"pick a body that raises neither", c)
		}
	}
}

// TestTheDisplayFacetingIsQuietOnThisBody guards the false-positive side, on the faceting that reaches
// a user: feature health harvests at DefaultQuality (model/feature/result_diagnostics.go), and this
// body meshes cleanly there. A channel that fired at the display faceting on geometry this ordinary is
// one users learn to ignore — #2058's third acceptance.
func TestTheDisplayFacetingIsQuietOnThisBody(t *testing.T) {
	t.Parallel()
	if got := BodyMeshDiagnostics(offsetCrossingRods(t), tessellate.DefaultQuality()); len(got) != 0 {
		t.Errorf("the display faceting reports %v on two ordinary crossing rods, want nothing", codeList(got))
	}
}

// offsetCrossingRods is the corpus body: r = 3 along +x met by r = 2.5 along +z, whose axis is offset
// 1 mm. The offset is what makes it this row's body rather than a coaxial-symmetric crossing — the
// intersection's two walls are then unequal and the r = 3 one's chart mesh fails its own rim gate.
func offsetCrossingRods(t *testing.T) *topo.Body {
	t.Helper()
	along, err := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	if err != nil {
		t.Fatalf("along-x rod: %v", err)
	}
	across, err := brep.SolidCylinder(math.P3(0, 1, -6), math.V3(0, 0, 1), 2.5, 12)
	if err != nil {
		t.Fatalf("along-z rod: %v", err)
	}
	body, err := brep.Boolean(brep.Intersection, along, across)
	if err != nil {
		t.Fatalf("crossing rods ∩: %v", err)
	}
	return body
}

// findCode is the one diagnostic carrying code, if the list holds it.
func findCode(ds []diag.Diagnostic, code diag.Code) (diag.Diagnostic, bool) {
	for _, d := range ds {
		if d.Code == code {
			return d, true
		}
	}
	return diag.Diagnostic{}, false
}

// codeList renders a diagnostic list's codes for a failure message.
func codeList(ds []diag.Diagnostic) []diag.Code {
	out := make([]diag.Code, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Code)
	}
	return out
}
