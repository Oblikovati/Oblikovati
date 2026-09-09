// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The corpus rows for Oblikovati/Oblikovati#3520: a chart-driven mesher decline reaches the harvest a
// feature reply, the API and the UI read. Both bodies are real booleans through the general pipeline,
// not hand-built faces.
//
// They pin the two ends of what the new code means. offsetCrossingRods is the SILENT case — it carries
// neither of the two codes that already existed for a lost trim, so before this change nothing anywhere
// said the selected mesher had given the face up. wideCrossingRods is the LOUD one — it raises the code
// at the DISPLAY faceting, where feature health reads it, on a face that comes back 28.7 % short of its
// analytic area. Together they say the report is neither redundant nor cosmetic.
//
// A note for whoever extends this: the row's declining face is `selected` by the classification AND
// declines, so adding either body to `classificationCorpus()` in kernel/ops/tessellate would turn
// TestNoChartedFaceEmitsARimOnlyTriangle red. That is the acceptance assertion working, not a
// regression — the bodies live here on purpose.
//
// They are built with brep.Boolean rather than ops.Boolean, and must be: kernel/ops depends on
// kernel/ops/query, so this internal test cannot import ops without an import cycle. The sweep behind
// chart_decline.go's THE MEASUREMENT block runs through ops.Boolean, which admits fewer bodies — both
// of these are in that set too, so the rows assert on bodies the general pipeline really produces.

// TestAChartMesherDeclineReachesTheBodysMeshDiagnostics is the wiring plus the "was silent" half, on
// one harvest. The two were separate rows and re-meshed the same body twice for no gain.
func TestAChartMesherDeclineReachesTheBodysMeshDiagnostics(t *testing.T) {
	t.Parallel()
	got := BodyMeshDiagnostics(offsetCrossingRods(t), PropertyQuality())
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
	// What makes this a regression and not a restatement: the two codes that already existed for a lost
	// trim are BOTH absent, so this decline was reported by nothing before #3520.
	codes := codeSet(got)
	for _, c := range []diag.Code{tessellate.CodeWallWrapUnmeshed, tessellate.CodeTrimIgnoredFullDomain} {
		if codes[c] {
			t.Errorf("this body also raises %q, so it is not the silent case the row exists to pin — "+
				"pick a body that raises neither", c)
		}
	}
}

// TestTheDisplayFacetingIsQuietOnThisBody is the false-positive guard, on the faceting that reaches a
// user: feature health harvests at DefaultQuality (model/feature/result_diagnostics.go), and this body
// meshes cleanly there. A channel that fired at the display faceting on every ordinary crossing would
// be one users learn to ignore — #2058's third acceptance.
func TestTheDisplayFacetingIsQuietOnThisBody(t *testing.T) {
	t.Parallel()
	if got := BodyMeshDiagnostics(offsetCrossingRods(t), tessellate.DefaultQuality()); len(got) != 0 {
		t.Errorf("the display faceting reports %v on two ordinary crossing rods, want nothing", codeList(got))
	}
}

// TestADeclineAlsoReachesTheDisplayFacetingWhenItIsEarned is the other end, and the row the first cut of
// #3520 was missing: the new Defect DOES reach feature health at DefaultQuality on at least one body of
// the sweep, and that body is not a false alarm. Measured here against the analytic oracle rather than
// asserted: the declining face ships 28.7 % less area than it has.
//
// The row asserts the SHORTFALL, not just the code, because "the Defect reaches the display faceting"
// is only reassuring if the face it names is genuinely wrong. A future change that makes this face
// correct should delete the row, not loosen it.
func TestADeclineAlsoReachesTheDisplayFacetingWhenItIsEarned(t *testing.T) {
	t.Parallel()
	body := wideCrossingRods(t)
	if got := codeSet(BodyMeshDiagnostics(body, tessellate.DefaultQuality())); !got[tessellate.CodeChartMesherDeclined] {
		t.Fatalf("the display faceting no longer reports the decline on this body (%v); the row pins that "+
			"it does, because the severity argument depends on knowing when a user sees it", got)
	}
	short := declinedFaceAreaShortfall(t, body)
	if short < 0.2 {
		t.Errorf("the declining face is only %.4f short of its analytic area; the row exists to say this "+
			"Defect is EARNED at the display faceting, and a face this close no longer says that", short)
	}
}

// declinedFaceAreaShortfall is the fractional area a declining face is missing against
// AnalyticFaceArea — a per-face oracle, not a whole-body smoke test. It fails the row if no face of the
// body declines at the display faceting, so the number can never be read off the wrong face.
func declinedFaceAreaShortfall(t *testing.T, b *topo.Body) float64 {
	t.Helper()
	for _, f := range b.Faces() {
		m := tessellate.TessellateFace(f, tessellate.DefaultQuality())
		if m == nil || !meshCarries(m, tessellate.CodeChartMesherDeclined) {
			continue
		}
		want, ok := AnalyticFaceArea(f)
		if !ok || want == 0 {
			t.Fatalf("the declining %T face has no analytic area to gate against", f.Geometry())
		}
		return stdmath.Abs(want-m.Area()) / want
	}
	t.Fatal("no face of this body declines at the display faceting; the shortfall has nothing to measure")
	return 0
}

// meshCarries reports whether a face mesh recorded the given code.
func meshCarries(m *tessellate.Mesh, code diag.Code) bool {
	for _, d := range m.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

// offsetCrossingRods is the SILENT case: r = 3 along +x met by r = 2.5 along +z, whose axis is offset
// 1 mm. The offset is what makes it this row's body rather than a symmetric crossing — the
// intersection's two walls are then unequal and the r = 3 one's chart mesh fails its own rim gate.
func offsetCrossingRods(t *testing.T) *topo.Body {
	t.Helper()
	return crossingRods(t, 2.5, 1)
}

// wideCrossingRods is the LOUD case: the same r = 3 rod met by a WIDER r = 3.5 rod offset 0.5 mm, the
// one body of the sweep whose decline reaches the display faceting.
func wideCrossingRods(t *testing.T) *topo.Body {
	t.Helper()
	return crossingRods(t, 3.5, 0.5)
}

// crossingRods intersects the r = 3 rod along +x with a rod of radius r along +z, offset off in y.
func crossingRods(t *testing.T, r, off float64) *topo.Body {
	t.Helper()
	along, err := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	if err != nil {
		t.Fatalf("along-x rod: %v", err)
	}
	across, err := brep.SolidCylinder(math.P3(0, off, -6), math.V3(0, 0, 1), r, 12)
	if err != nil {
		t.Fatalf("along-z rod r=%g: %v", r, err)
	}
	body, err := brep.Boolean(brep.Intersection, along, across)
	if err != nil {
		t.Fatalf("crossing rods r=%g off=%g ∩: %v", r, off, err)
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
