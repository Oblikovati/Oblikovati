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

// The corpus rows for Oblikovati/Oblikovati#3520: a chart-driven mesher decline reaches the harvest a
// feature reply, the API and the UI read. Both bodies are real booleans through the general pipeline,
// not hand-built faces.
//
// They pin the two ends of what the new code means. ringMeetingADrill is the SILENT case — it carries
// neither of the two codes that already existed for a lost trim, so before this change nothing anywhere
// said the selected mesher had given the face up. wideCrossingRods is the LOUD one — it raises the code
// at the DISPLAY faceting, where feature health reads it, on a face that comes back 28.7 % short of its
// analytic area. Together they say the report is neither redundant nor cosmetic.
//
// The silent row's body CHANGED with #3518, twice, and the second move is the one that matters. It used
// to be offsetCrossingRods, whose r = 3 wall failed the chart mesher's rim gate because the region kept
// triangles between its chart's contour and the finer chord polygon of its shared edges. That is fixed
// (chart_face_rim_side.go), and swept over the crossing-rod family — radii 1.5 … 3.5 against the r = 3
// rod, offsets 0 … 2, all three operators, both facetings — not one rod-rod body declines silently any
// more. The first replacement was a rod-ball body, and it broke the PAIRING these two rows carry: it
// declined at BOTH facetings, on a face only 0.169 % short of its analytic area, so the
// display-faceting guard below had nothing left to guard and the Defect it raised there was unearned
// (#3518 review I3). ringMeetingADrill restores the pairing: measured, it is silent at DefaultQuality
// and declines with nothing else at PropertyQuality, so ONE body carries both rows again.
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

// TestAChartMesherDeclineReachesTheBodysMeshDiagnostics is the wiring: when the chart mesher owns a
// face and gives it up, the harvest a feature reply, the API and the UI read says so, at Defect
// severity, naming WHY.
//
// ITS FIXTURE HAS MOVED TWICE, and both moves were cures rather than losses. It read ringMeetingADrill
// until #3517's classification began selecting exactly one path, which sent that body's developing face
// to the structured grid so it had nothing to give up. It then read wideCrossingRods until #3551 made
// the covering lay one vertex per location, which cured that body too: swept on this tree over crossing
// rods (radius 1.5…4.0 × offset 0…2) at both facetings, NO body of that family declines any more.
//
// So it reads the tangent-plane piece, which is #3551's own family and the living declining case: the
// torus R=100 r=1 cut by the plane tangent to its inner equator, intersected, at PropertyQuality. Its
// torus face is refused by the mesher's own rim gate (6 unpaired edges that are no rim segment) and the
// router falls through to the surface's whole domain. tangent_plane_family_test.go carries what that
// residue is and why it is a covering DENSITY limit; this row only asserts that it is REPORTED.
func TestAChartMesherDeclineReachesTheBodysMeshDiagnostics(t *testing.T) {
	t.Parallel()
	got := BodyMeshDiagnostics(tangentPlanePieceForDecline(t), PropertyQuality())
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

// tangentPlanePieceForDecline is the torus R=100 r=1 cut by the plane tangent to its inner equator,
// intersected — the thin-tube end of #3551's tangent-plane family, whose torus face the chart mesher
// still refuses at PropertyQuality. Built with brep.Boolean like everything else in this file: kernel/ops
// depends on kernel/ops/query, so this internal test cannot import ops.
func tangentPlanePieceForDecline(t *testing.T) *topo.Body {
	t.Helper()
	const ringR, tubeR = 100.0, 1.0
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), ringR, tubeR, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	far := 20 * (ringR + tubeR)
	box, err := brep.SolidBlock(math.P3(-far, ringR-tubeR, -far), math.P3(far, far, far), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	body, err := brep.Boolean(brep.Intersection, ring, box)
	if err != nil {
		t.Fatalf("torus ∩ tangent half-space: %v", err)
	}
	return body
}

// TestTheBodyThatUsedToDeclineRaisesNoDeclineNow is the false-positive guard, and it is narrower than
// the sentence I first wrote for it. ringMeetingADrill declined at PropertyQuality and was quiet at
// DefaultQuality; it no longer DECLINES at either, because its charted face develops into one (u,v)
// branch and the classification sends it to the structured grid rather than the covering. A channel
// that fired on an ordinary crossing would be one users learn to ignore (#2058's third acceptance), and
// a body that stops needing the Defect is the better way to satisfy that.
//
// It is not SILENT, though, and round 5's version of this row asserted that it was — on the strength of
// a harvest that could not yet see a chord miss. Measured: the body's two torus faces come to 1.332×
// and 1.129× the display tolerance and 1.408× and 1.342× the property one, so both name
// CodeFaceChordNotMet (face_chord_achieved.go). They are inside 3.2e-4 of their exact analytic AREAS —
// the body is correct, not a suppressed complaint — and still coarser at the rim than they were asked
// to be, which is exactly the distinction the new report exists to draw. The row therefore asserts what
// it is really about: no DECLINE, in either direction, so a routing regression brings that code back.
func TestTheBodyThatUsedToDeclineRaisesNoDeclineNow(t *testing.T) {
	t.Parallel()
	for _, q := range []tessellate.Quality{tessellate.DefaultQuality(), PropertyQuality()} {
		got := codeSet(BodyMeshDiagnostics(ringMeetingADrill(t), q))
		for _, c := range []diag.Code{tessellate.CodeChartMesherDeclined, tessellate.CodeWallWrapUnmeshed,
			tessellate.CodeTrimIgnoredFullDomain, tessellate.CodeMeshNotWatertight} {
			if got[c] {
				t.Errorf("at chord tolerance %g this body raises %q; its charted face develops, so it is "+
					"meshed on the structured grid and has nothing to give up", q.Tol(), c)
			}
		}
	}
}

// TestTheCuredDisplayDeclineStaysCured is what is left of the pairing's DISPLAY-faceting end, and it is
// here rather than deleted because the ground rule says a bug fix adds its input to the corpus — and a
// CURED input is the one the corpus most wants, since it is the one that proves the cure is in force
// (Oblikovati/Oblikovati#3551).
//
// TestADeclineAlsoReachesTheDisplayFacetingWhenItIsEarned stood here and pinned that the Defect reaches
// feature health at DefaultQuality on this body, whose declining face shipped 28.7 % less area than it
// has. Its instruction was "a future change that makes this face correct should delete the row, not
// loosen it", and one did: the covering no longer lays two vertices at one location. Following that
// instruction literally cost the FIXTURE as well as the assertion, which was wrong — the assertion was
// what had to go, not the body.
//
// So the row is inverted instead. The same wideCrossingRods asserts the cured behaviour: no diagnostic
// at either faceting, and both walls meshed to their analytic areas within a chord deficit. A
// regression puts the decline back and this row says so; the numbers are the measurement, not a bound
// chosen in advance (0.392 % and 0.863 % at DefaultQuality, 0.0046 % and 0.0082 % at PropertyQuality).
//
// No body in this package's sweep declines at the display faceting any more, so the severity argument
// rests on the PropertyQuality row above plus TestTheDisplayFacetingIsQuietOnThisBody. If a shape that
// declines at DefaultQuality turns up again, its row goes beside this one.
func TestTheCuredDisplayDeclineStaysCured(t *testing.T) {
	t.Parallel()
	body := wideCrossingRods(t)
	for _, gq := range []struct {
		name string
		q    tessellate.Quality
		band float64
	}{{"default", tessellate.DefaultQuality(), 0.01}, {"property", tessellate.PropertyQuality(), 0.0001}} {
		// The cure this row names is the DECLINE and the tear behind it, so those are what it refuses.
		// It asserted no diagnostic AT ALL until #3517 gave a curved face that misses the chord tolerance
		// it was handed a voice (CodeFaceChordNotMet): at DefaultQuality this body is still completely
		// quiet, and at PropertyQuality it now says two true things about its own approximation — its two
		// cylinder walls reach 1.283× the 1e-3 mm asked for, and the generic path's interior refinement
		// saturated its 64-cell floor still above that chord. Neither is a lost trim, neither is a tear,
		// and the per-face areas below still hold to 1e-4, which is what says the body is correct rather
		// than quietened. A row that refused every report would have had to be loosened by the next
		// honest one; refusing the codes it is ABOUT cannot be.
		for _, c := range []diag.Code{tessellate.CodeChartMesherDeclined, tessellate.CodeWallWrapUnmeshed,
			tessellate.CodeTrimIgnoredFullDomain, tessellate.CodeMeshNotWatertight, tessellate.CodePatchCoverage} {
			if codeSet(BodyMeshDiagnostics(body, gq.q))[c] {
				t.Errorf("%s quality: wideCrossingRods raises %q; it was cured by #3551 and the cure must hold",
					gq.name, c)
			}
		}
		assertEveryWallMeshesItsAnalyticArea(t, body, gq.q, gq.name, gq.band)
	}
}

// assertEveryWallMeshesItsAnalyticArea holds every analytically integrable face of a body to its own
// area within band — a per-face oracle, so a body-level total cannot hide a face that is wrong.
func assertEveryWallMeshesItsAnalyticArea(t *testing.T, b *topo.Body, q tessellate.Quality, name string, band float64) {
	t.Helper()
	checked := 0
	for i, f := range b.Faces() {
		an, ok := AnalyticFaceArea(f)
		if !ok || an <= 0 {
			continue
		}
		checked++
		got := tessellate.MeshGeometryProperties(tessellate.TessellateFace(f, q)).Area
		if rel := (an - got) / an; rel < 0 || rel > band {
			t.Errorf("%s quality: face %d meshes %.5f against an analytic %.5f (deficit %.5f, want (0, %g])",
				name, i, got, an, rel, band)
		}
	}
	if checked == 0 {
		t.Errorf("%s quality: no face of the body integrates analytically; the per-face gate covers nothing", name)
	}
}

// wideCrossingRods is the body that #3551 cured: the r = 3 rod along +x met by a WIDER r = 3.5 rod
// offset 0.5 mm in y, whose merged wall used to be declined at the display faceting.
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
		t.Fatalf("across-z rod r=%g: %v", r, err)
	}
	body, err := brep.Boolean(brep.Intersection, along, across)
	if err != nil {
		t.Fatalf("crossing rods r=%g off=%g: %v", r, off, err)
	}
	return body
}

// ringMeetingADrill is the QUIET body: the R = 5, r = 1.5 ring met by an axial drill of radius 1.4
// standing at x = 5, offset 1 mm in y, kept. It used to be the SILENT case too — nothing at
// DefaultQuality, the chart-mesher decline alone at PropertyQuality — and it is now quiet at both,
// because its charted face develops into one (u,v) branch and the classification sends it to the
// structured grid (#3517 review 3 C1).
func ringMeetingADrill(t *testing.T) *topo.Body {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 1, -4), math.V3(0, 0, 1), 1.4, 8)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	body, err := brep.Boolean(brep.Intersection, ring, drill)
	if err != nil {
		t.Fatalf("ring ∩ drill: %v", err)
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
