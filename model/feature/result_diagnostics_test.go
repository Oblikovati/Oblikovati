// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/test-utilities/degenerate"
)

// crossedTrimFeature is a fake feature whose result is deliberately BAD INPUT for the tessellator (a
// self-crossing trim boundary, see test-utilities/degenerate). It stands in for the real thing —
// malformed geometry arriving from an import or a degenerate sketch — because a body the kernel
// itself meshes badly is a tessellation bug to fix, not a fixture to build a regression test on.
type crossedTrimFeature struct{}

func (crossedTrimFeature) Kind() string { return "crossed-trim" }

func (crossedTrimFeature) Recompute(Input) (Output, error) {
	return Output{Bodies: []*topo.Body{degenerate.CrossedTrimBody()}}, nil
}

// passThroughFeature returns the running bodies untouched — a suppressible marker, a work feature,
// any feature that adds no geometry.
type passThroughFeature struct{}

func (passThroughFeature) Kind() string { return "pass-through" }

func (passThroughFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: in.Bodies}, nil
}

// discardBodiesFeature drops the running bodies — a delete-body feature, or any operation that
// consumes what came before it.
type discardBodiesFeature struct{}

func (discardBodiesFeature) Kind() string { return "discard-bodies" }

func (discardBodiesFeature) Recompute(Input) (Output, error) { return Output{}, nil }

// markedBodyFeature produces a fresh, geometrically clean body carrying a planted build report, so
// the routing of topo.Body.BuildDiagnostics can be tested independently of any tessellation.
type markedBodyFeature struct {
	marks []diag.Diagnostic
}

func (markedBodyFeature) Kind() string { return "marked-body" }

func (m markedBodyFeature) Recompute(Input) (Output, error) {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("test", "marked", 0)))
	for _, d := range m.marks {
		bld.Diagnose(d)
	}
	return Output{Bodies: []*topo.Body{bld.Build()}}, nil
}

// TestFeatureReportsTheTessellatorsDefect is the #2058 engine-side regression: a feature whose result
// does not mesh must say so. Before this, kernel/ops knew and the feature reply was clean — which is
// how #2038's 77%-low body reached the user without a word.
func TestFeatureReportsTheTessellatorsDefect(t *testing.T) {
	fs := NewPartFeatures(nil)
	pf := fs.Add(crossedTrimFeature{})
	fs.Recompute()
	if !hasDiagCode(pf.Diagnostics(), ops.CodePatchCoverage) {
		t.Fatalf("a feature whose body fails to mesh reports %v, want a %s defect",
			pf.Diagnostics(), ops.CodePatchCoverage)
	}
}

// TestDegradationIsFiledOnTheProducingFeature: a body's defect belongs to whoever MADE the body. A
// feature that hands its input straight back returns the same *topo.Body, and blaming it would point
// the user at the wrong row of the browser.
func TestDegradationIsFiledOnTheProducingFeature(t *testing.T) {
	fs := NewPartFeatures(nil)
	bad := fs.Add(crossedTrimFeature{})
	quiet := fs.Add(passThroughFeature{})
	fs.Recompute()
	if !hasDiagCode(bad.Diagnostics(), ops.CodePatchCoverage) {
		t.Fatalf("the producing feature lost its %s defect: %v", ops.CodePatchCoverage, bad.Diagnostics())
	}
	if n := len(quiet.Diagnostics()); n != 0 {
		t.Errorf("the pass-through feature reports %d diagnostics, want 0: %v", n, quiet.Diagnostics())
	}
}

// TestBuildReportReachesTheFeatureWithoutItsInfoMarkers: the assembler's report on the body is the
// other result-carried channel nothing outside the kernel read. Its Warning/Defect entries must
// arrive; its Info markers (e.g. "assembled through the fillet edge catalog", which a kernel corpus
// gate reads) must not, or every fillet would ship noise on the wire.
func TestBuildReportReachesTheFeatureWithoutItsInfoMarkers(t *testing.T) {
	const marker, degradation diag.Code = "test.path-marker", "test.degradation"
	fs := NewPartFeatures(nil)
	pf := fs.Add(markedBodyFeature{marks: []diag.Diagnostic{
		{Code: marker, Severity: diag.Info, Detail: "a path was taken"},
		{Code: degradation, Severity: diag.Warning, Detail: "something came out worse than asked"},
	}})
	fs.Recompute()
	if !hasDiagCode(pf.Diagnostics(), degradation) {
		t.Errorf("the build report's Warning did not reach the feature: %v", pf.Diagnostics())
	}
	if hasDiagCode(pf.Diagnostics(), marker) {
		t.Errorf("an Info path marker reached the feature reply: %v", pf.Diagnostics())
	}
}

// TestRepeatedRecomputesDoNotAccumulateDiagnostics: the result drain re-files on every recompute, and
// a report that grew a copy each time would turn one defect into a hundred over a session. It also
// pins the memo — the second pass must reuse the verdict of a body it already judged, not re-mesh it.
func TestRepeatedRecomputesDoNotAccumulateDiagnostics(t *testing.T) {
	fs := NewPartFeatures(nil)
	pf := fs.Add(crossedTrimFeature{})
	fs.Recompute()
	first := len(pf.Diagnostics())
	for range 3 {
		fs.Recompute()
	}
	if got := len(pf.Diagnostics()); got != first {
		t.Errorf("after 4 recomputes the feature reports %d diagnostics, want the same %d: %v",
			got, first, pf.Diagnostics())
	}
}

// TestRecomputeDoesNotReadTheBodiesUntilAsked pins what the first cut of #2058 got wrong: reading a
// body's report means MESHING it, which can cost far more than building it — the modeled thread
// retypes one face in ~400 µs and meshes in ~11 ms, and doing that on every recompute made
// TestThreadCutModelsRealThreadFast's no-boolean budget fail by 2x on CI. The price belongs to the
// caller who wants the answer. The memo must also survive a recompute that changed nothing.
func TestRecomputeDoesNotReadTheBodiesUntilAsked(t *testing.T) {
	fs := NewPartFeatures(nil)
	pf := fs.Add(crossedTrimFeature{})
	fs.Recompute()
	if pf.resultRead {
		t.Fatalf("recompute meshed the result body; the report must wait until it is asked for")
	}
	pf.Diagnostics()
	if !pf.resultRead {
		t.Fatalf("asking for the diagnostics did not read the body")
	}
	fs.Recompute() // nothing dirty: the same bodies survive, so the verdict cannot have changed
	if !pf.resultRead {
		t.Errorf("a recompute that changed nothing threw the memoized report away")
	}
}

// TestDeletedGeometryStopsBeingReported: the report reads the RESULT, so a defect a later feature cut
// away must disappear from the report — the model no longer has it, and a stale alarm is as bad as a
// missing one.
func TestDeletedGeometryStopsBeingReported(t *testing.T) {
	fs := NewPartFeatures(nil)
	bad := fs.Add(crossedTrimFeature{})
	fs.Add(discardBodiesFeature{})
	fs.Recompute()
	if n := len(bad.Diagnostics()); n != 0 {
		t.Errorf("a feature whose body was removed downstream still reports %d diagnostics: %v",
			n, bad.Diagnostics())
	}
}
