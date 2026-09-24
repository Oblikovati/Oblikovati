// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The recorder's own rows (#3520). The corpus rows live in kernel/ops/query, where a real body proves
// the record reaches query.BodyMeshDiagnostics; these pin the four decisions the recorder itself makes.

// TestAChartDeclineIsRecordedOnWhateverTheFaceShipsInstead: the mesher that gave up returns no mesh, so
// the report can only ride the mesh the face fell back to — which is why the recorder sits at the router
// and not at the point of decline.
func TestAChartDeclineIsRecordedOnWhateverTheFaceShipsInstead(t *testing.T) {
	t.Parallel()
	log := &chartDeclineLog{}
	log.declined("the covering kept no triangle inside the chart's own window")
	m := log.recordOn(&Mesh{}, mustDeclineSurface(t))
	if !hasCode(m, CodeChartMesherDeclined) {
		t.Fatalf("a pending decline recorded nothing; the mesh carries %v", m.Diagnostics)
	}
	if !strings.Contains(m.Diagnostics[0].Detail, "kept no triangle") {
		t.Errorf("the record does not carry the reason: %q", m.Diagnostics[0].Detail)
	}
	if m.Diagnostics[0].Severity != diag.Defect {
		t.Errorf("a chart decline recorded at severity %v, want a Defect — it changes the shipped mesh",
			m.Diagnostics[0].Severity)
	}
}

// TestNothingIsRecordedForAFaceTheChartMesherNeverOwned guards the false-positive side. chartFaceMesh
// answers false for every aperiodic or unrecorded face in the model, which is the ORDINARY route onto
// the generic (u,v) trim path; a channel that fired there would fire on almost every curved face there
// is, and #2058's third acceptance is that a channel which cries on healthy geometry is ignored.
func TestNothingIsRecordedForAFaceTheChartMesherNeverOwned(t *testing.T) {
	t.Parallel()
	log := &chartDeclineLog{}
	if m := log.recordOn(&Mesh{}, mustDeclineSurface(t)); hasCode(m, CodeChartMesherDeclined) {
		t.Error("a face the mesher never owned was reported as a degradation")
	}
}

// TestTheFirstDeclineReasonWins: a face can reach chartFaceMesh twice — the classification's kindChart
// arm, then chartedTrimMesh once ToUVLoops fails — and the second call re-derives the same answer. The
// report names the reason the mesher first gave, not the last route that asked.
func TestTheFirstDeclineReasonWins(t *testing.T) {
	t.Parallel()
	log := &chartDeclineLog{}
	log.declined("the first reason")
	log.declined("the second reason")
	m := log.recordOn(&Mesh{}, mustDeclineSurface(t))
	if !strings.Contains(m.Diagnostics[0].Detail, "the first reason") {
		t.Errorf("the record reads %q, want the reason the mesher first gave", m.Diagnostics[0].Detail)
	}
}

// TestADeclineIsReportedEvenWhenTheFallThroughMeshedNothing pins the worst case: no geometry AND no
// report is the one outcome the ground rules forbid outright, so the record does not depend on the
// fall-through having produced a mesh (#3520 review M3).
func TestADeclineIsReportedEvenWhenTheFallThroughMeshedNothing(t *testing.T) {
	t.Parallel()
	log := &chartDeclineLog{}
	log.declined("the covering kept no triangle inside the chart's own window")
	m := log.recordOn(nil, mustDeclineSurface(t))
	if m == nil {
		t.Fatal("a face that declined and meshed nothing shipped nothing AND said nothing")
	}
	if !hasCode(m, CodeChartMesherDeclined) {
		t.Errorf("the empty fall-through carries %v, want the decline", m.Diagnostics)
	}
}

// TestNothingIsAllocatedForAFaceThatNeverDeclined is the other half: the ordinary face — no chart, or a
// mesher that accepted it — keeps exactly what the router returned, nil included. Allocating there would
// turn every declined-nothing face into an empty mesh the body then has to carry.
func TestNothingIsAllocatedForAFaceThatNeverDeclined(t *testing.T) {
	t.Parallel()
	log := &chartDeclineLog{}
	if m := log.recordOn(nil, mustDeclineSurface(t)); m != nil {
		t.Errorf("recordOn(nil) with no decline returned %v, want nil", m)
	}
}

// mustDeclineSurface is the surface the recorder names in its report — any periodic surface will do,
// since the recorder reads only its type.
func mustDeclineSurface(t *testing.T) geom.Surface {
	t.Helper()
	s, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	return s
}

// The NAMED site, at last (#3520 review I4). singlyPeriodicWrapMesh is the call the issue is about:
// it falls to the FLAT best-fit-plane CDT when the chart mesher gives the face up, and it used to
// report that only when isDevelopableSide(s) held AND the outer loop wrapped the whole period. The
// silent set was never just "a non-wrapping developable" either — recordUnmeshedWallWrap returns early
// for anything that is not a developable side, and every SPHERE reaches this call
// (IsPeriodic(U) != IsPeriodic(V)), so a charted sphere giving up here was silent too.
//
// No corpus body reaches this site and declines — a 106-body boolean sweep put every decline at the
// classification's kindChart arm or at chartedTrimMesh — so the row is built at the lowest level that
// is still the production function: singlyPeriodicWrapMesh itself, with the outer boundary it takes as
// a parameter. The face is a real charted cylinder wall whose chart DISAGREES with its own boundary,
// which is what makes the mesher give it up.

// TestTheNamedSiteReportsADeclineItUsedToSwallow drives singlyPeriodicWrapMesh with a declining
// charted wall whose outer loop does NOT wrap the period, and asserts both halves: the old conditional
// report stays silent (that is the defect), and the new one speaks.
func TestTheNamedSiteReportsADeclineItUsedToSwallow(t *testing.T) {
	t.Parallel()
	f := chartDisagreeingWall(t)
	s := f.Geometry()
	log := &chartDeclineLog{}
	m := singlyPeriodicWrapMesh(f, s, partialArcBoundary(s), nil, DefaultQuality(), log)
	if log.reason() == "" {
		t.Fatal("the chart mesher accepted a wall whose chart omits its own window; the row needs a decline")
	}
	// The defect, stated as an assertion: the report this site already had cannot see this input.
	if hasCode(m, CodeWallWrapUnmeshed) {
		t.Fatal("recordUnmeshedWallWrap fired, so this input is the case that was ALREADY reported; " +
			"the row must use one it was blind to, or it asserts nothing about #3520")
	}
	if got := log.recordOn(m, s); !hasCode(got, CodeChartMesherDeclined) {
		t.Errorf("the named site swallowed a decline it fell back from: %v", got.Diagnostics)
	}
}

// chartDisagreeingWall is the windowed wall with a chart that OMITS its window: the covering then
// covers material the face does not carry, the window's rim segments go unbounded, and the mesher's own
// rim gate refuses the mesh. Nothing is faked — this is the gate deciding, on a real chart.
func chartDisagreeingWall(t *testing.T) *topo.Face {
	t.Helper()
	f := rimBoundedWindowedWall(t, 3.0, 4.0, 3.0, 6.0)
	side, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	outerContour := wallChart(side, 3.0, 4.0, 3.0, 6.0)[0] // the parameter rectangle, WITHOUT the window
	f.SetChart([][]math.Point2{outerContour})
	return f
}

// partialArcBoundary is an outer boundary spanning a QUARTER of the cylinder's azimuth — a developable
// side whose loop does not wrap the period, which is exactly what wrappedWallUV refuses and therefore
// what recordUnmeshedWallWrap has always been blind to.
func partialArcBoundary(s geom.Surface) []math.Point3 {
	cyl := s.(geom.Cylinder)
	var out []math.Point3
	for i := range 9 {
		out = append(out, cyl.PointAt(float64(i)*stdmath.Pi/16, 0))
	}
	for i := 8; i >= 0; i-- {
		out = append(out, cyl.PointAt(float64(i)*stdmath.Pi/16, wallH))
	}
	return out
}

// The full-domain report's CAUSE (#3520 review I5). recordIgnoredTrim used to be handed only the
// SPECIAL meshers' refusal, so on every face that reached it after the chart-driven mesher gave up —
// three bodies of the sweep carry both codes — it told the user "no mesher recognised its boundary on
// this surface" while one had recognised it and named exactly what it could not describe. The lie
// predates #3520; what #3520 added is the reason sitting in scope at that call site.

// TestTheFullDomainReportNamesTheMesherThatActuallyRefused: when the chart mesher gave the face up, its
// reason is the cause, not "nothing recognised it".
func TestTheFullDomainReportNamesTheMesherThatActuallyRefused(t *testing.T) {
	t.Parallel()
	f := chartDisagreeingWall(t)
	log := &chartDeclineLog{}
	m := chartedTrimMesh(f, f.Geometry(), DefaultQuality(), "", log)
	d, found := meshDiagnostic(m, CodeTrimIgnoredFullDomain)
	if !found {
		t.Fatalf("a trimmed face meshed over the whole domain reported nothing: %v", m.Diagnostics)
	}
	if strings.Contains(d.Detail, "no mesher recognised") {
		t.Errorf("the report says nothing recognised the face, but the chart-driven mesher did and gave "+
			"it up: %q", d.Detail)
	}
	if !strings.Contains(d.Detail, "not bounded by its own rim") {
		t.Errorf("the report does not carry the refusing mesher's own reason: %q", d.Detail)
	}
}

// TestAnUnrecognisedFaceStillReadsAsUnrecognised is the other direction, so the fix above cannot be
// read as "always blame a mesher": a face NO mesher claimed — it carries no chart, so the chart-driven
// mesher never owned it — keeps the honest "nothing recognised it" wording.
func TestAnUnrecognisedFaceStillReadsAsUnrecognised(t *testing.T) {
	t.Parallel()
	f := rimBoundedWindowedWall(t, 3.0, 4.0, 3.0, 6.0)
	f.SetChart(nil)
	log := &chartDeclineLog{}
	m := chartedTrimMesh(f, f.Geometry(), DefaultQuality(), "", log)
	d, found := meshDiagnostic(m, CodeTrimIgnoredFullDomain)
	if !found {
		t.Fatalf("a trimmed face meshed over the whole domain reported nothing: %v", m.Diagnostics)
	}
	if !strings.Contains(d.Detail, "no mesher recognised") {
		t.Errorf("a face no mesher claimed should read as unrecognised, not blame one: %q", d.Detail)
	}
}

// TestNamedRefusalPrefersTheSelectedMesher: when a SPECIAL arm recognised the face and refused it, that
// is the classification's own verdict and outranks the chart mesher's later attempt.
func TestNamedRefusalPrefersTheSelectedMesher(t *testing.T) {
	t.Parallel()
	if got := namedRefusal("the band arm gave up", "the chart gave up"); got != "the band arm gave up" {
		t.Errorf("namedRefusal picked %q; the arm the classification selected names the cause", got)
	}
	if got := namedRefusal("", "the chart gave up"); got != "the chart gave up" {
		t.Errorf("namedRefusal picked %q, want the chart mesher's reason when no arm refused", got)
	}
	if got := namedRefusal("", ""); got != "" {
		t.Errorf("namedRefusal invented %q; no mesher claimed the face", got)
	}
}

// meshDiagnostic is the mesh's record for one code, if it carries it.
func meshDiagnostic(m *Mesh, code diag.Code) (diag.Diagnostic, bool) {
	for _, d := range m.Diagnostics {
		if d.Code == code {
			return d, true
		}
	}
	return diag.Diagnostic{}, false
}
