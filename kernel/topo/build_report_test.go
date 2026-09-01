// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestBuildReportTravelsWithTheBody pins the build-report channel: what an assembler could not do
// ideally must reach the caller ON the body it produced, in emission order, and the accessor must
// hand back a COPY so a reader cannot mutate a finished body's report. A body whose assembler
// recorded nothing reports nothing — the overwhelmingly common case, which must stay allocation-free
// of surprises.
func TestBuildReportTravelsWithTheBody(t *testing.T) {
	bld := NewBuilder(true, NewLineage(Tok("t", "body", 0)))
	if got := bld.Build().BuildDiagnostics(); len(got) != 0 {
		t.Fatalf("a clean assembly reported %v, want nothing", got)
	}

	bld = NewBuilder(true, NewLineage(Tok("t", "body", 1)))
	bld.Diagnose(diag.Diagnostic{Code: "t.first", Severity: diag.Warning, Detail: "one"})
	bld.Diagnose(diag.Diagnostic{Code: "t.second", Severity: diag.Defect, Detail: "two"})
	body := bld.Build()

	got := body.BuildDiagnostics()
	if len(got) != 2 || got[0].Code != "t.first" || got[1].Code != "t.second" {
		t.Fatalf("build report = %v, want t.first then t.second in emission order", got)
	}
	got[0].Code = "mutated"
	if again := body.BuildDiagnostics(); again[0].Code != "t.first" {
		t.Errorf("mutating the returned slice changed the body's own report to %q — BuildDiagnostics must copy", again[0].Code)
	}
}

// TestBuildReportDoesNotSurviveARebuild pins the OTHER half of the channel, the half a reader is
// most likely to assume the wrong way round: the report is dropped, not merged, by every path that
// builds a new body from an old one's shells.
//
// ★ This is why an empty report cannot be read as "the assembler had nothing to say". A gate that
// asserts "no shipped body reports X" over bodies that passed through one ops.Boolean would be
// asserting nothing at all, silently and forever. The fillet's edge catalog therefore stamps a
// POSITIVE marker (blend.CodeAssembleEdgeCatalog) and its corpus gate asserts the marked population
// exactly; this test is the property that makes that necessary.
func TestBuildReportDoesNotSurviveARebuild(t *testing.T) {
	bld := NewBuilder(true, NewLineage(Tok("t", "body", 2)))
	bld.Diagnose(diag.Diagnostic{Code: "t.only", Severity: diag.Warning, Detail: "one"})
	body := bld.Build()
	if len(body.BuildDiagnostics()) != 1 {
		t.Fatalf("the assembler's own body reported %v, want one entry", body.BuildDiagnostics())
	}
	for name, rebuilt := range map[string]*Body{
		"BodyFromShells": BodyFromShells(body.Lineage(), body.IsSolid(), body.Shells()...),
		"MergeBodies":    MergeBodies(body.Lineage(), body.IsSolid(), body),
	} {
		if got := rebuilt.BuildDiagnostics(); len(got) != 0 {
			t.Errorf("%s carried the donor's build report forward as %v — if this ever becomes true, say so at "+
				"Body.BuildDiagnostics, because every gate reading that report was written for the drop", name, got)
		}
	}
}

// TestReplaceEdgeCurveSwapsGeometryBeforeBuild pins the setter the edge catalog adopts a later
// consumer's curve with: it must replace the geometry the edge was created with, and it must refuse
// a nil (dropping geometry is never an improvement, so a caller with nothing to offer must not call).
func TestReplaceEdgeCurveSwapsGeometryBeforeBuild(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("t", "body", 3)))
	a := bld.AddVertex(math.P3(0, 0, 0), NewLineage(Tok("t", "v", 0)))
	b := bld.AddVertex(math.P3(10, 0, 0), NewLineage(Tok("t", "v", 1)))
	chord := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0))
	e := bld.AddEdge(chord, a, b, NewLineage(Tok("t", "e", 0)))

	bow := geom.NewLineSegment(math.P3(10, 0, 0), math.P3(0, 0, 0)) // any other curve
	e.SetSnappedCurve([]math.Point3{math.P3(0, 0, 0), math.P3(5, 1, 0), math.P3(10, 0, 0)}, 0.5)
	bld.ReplaceEdgeCurve(e, bow)
	if got := e.Geometry().PointAt(0); got.DistanceTo(math.P3(10, 0, 0)) > 1e-12 {
		t.Errorf("edge still starts at %v after ReplaceEdgeCurve, want the replacement's own start", got)
	}
	// A snapped polyline is a discretization of the OLD curve and outranks Geometry() in
	// tessellation, so a stale one would silently ship the replaced geometry (m-8).
	if e.SnappedCurve() != nil || e.Tolerance() != 0 {
		t.Errorf("ReplaceEdgeCurve left a stale snapped polyline (%d points, tol %g) describing the OLD curve — "+
			"it must clear it, or the new curve is never tessellated", len(e.SnappedCurve()), e.Tolerance())
	}
	assertPanics(t, "ReplaceEdgeCurve(e, nil)", func() { bld.ReplaceEdgeCurve(e, nil) })
	assertPanics(t, "ReplaceEdgeCurve(nil, curve)", func() { bld.ReplaceEdgeCurve(nil, bow) })
}

// assertPanics requires fn to panic, naming what was expected to reject.
func assertPanics(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic — it must reject rather than silently drop an edge's geometry", what)
		}
	}()
	fn()
}
