// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

// addBody is a fake feature that appends one new body to the running state.
type addBody struct {
	kind string
	mk   func() *topo.Body
}

func (f addBody) Kind() string { return f.kind }
func (f addBody) Recompute(in Input) (Output, error) {
	return Output{Bodies: append(append([]*topo.Body(nil), in.Bodies...), f.mk())}, nil
}

// failer is a fake feature that always fails (→ sick).
type failer struct{}

func (failer) Kind() string                    { return "failer" }
func (failer) Recompute(Input) (Output, error) { return Output{}, errors.New("boom") }

// panicker is a fake feature whose Recompute panics — the alpha-kernel crash (a nil-deref /
// out-of-range mid-boolean) that #1415 must convert into a sick feature, not an app crash.
type panicker struct{}

func (panicker) Kind() string                    { return "panicker" }
func (panicker) Recompute(Input) (Output, error) { panic("kernel nil-deref") }

func makeBody() *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", 0)))
	v := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	e := bld.AddEdge(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), v, v, topo.NewLineage(topo.Tok("f", "edge", 0)))
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(e)))
	return bld.Build()
}

func body() addBody { return addBody{kind: "box", mk: makeBody} }

func TestRecomputeProducesBodies(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	fs.Add(body())
	fs.Add(body())
	fs.Recompute()
	if len(fs.Result()) != 2 {
		t.Fatalf("result has %d bodies, want 2", len(fs.Result()))
	}
	for i := 0; i < fs.Count(); i++ {
		if !fs.Item(i).Health().OK() {
			t.Errorf("feature %d not healthy", i)
		}
	}
}

func TestUniqueNameNumbersFromOneAndSkipsTaken(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	if got := fs.UniqueName("Extrusion"); got != "Extrusion1" {
		t.Errorf("first unique name = %q, want Extrusion1", got)
	}
	a := fs.Add(body())
	a.SetName("Extrusion1")
	b := fs.Add(body())
	b.SetName("Extrusion2")
	if got := fs.UniqueName("Extrusion"); got != "Extrusion3" {
		t.Errorf("unique name with 1,2 taken = %q, want Extrusion3", got)
	}
	// A gap is filled by the smallest free integer, not the next after the max.
	b.SetName("Extrusion9")
	if got := fs.UniqueName("Extrusion"); got != "Extrusion2" {
		t.Errorf("unique name with 1,9 taken = %q, want Extrusion2", got)
	}
}

func TestPreviewResultIsNonDestructive(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	fs.Add(body())
	fs.Add(body())
	fs.Recompute()
	before := fs.Result()
	beforeCounts := []int{fs.Item(0).RecomputeCount(), fs.Item(1).RecomputeCount()}

	// Previewing a candidate appended at end-of-part sees the 2 committed bodies and
	// returns 3, without touching the program.
	got, err := fs.PreviewResult(body())
	if err != nil {
		t.Fatalf("PreviewResult: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("preview result has %d bodies, want 3 (2 committed + 1 candidate)", len(got))
	}
	if len(fs.Result()) != 2 || &fs.Result()[0] != &before[0] {
		t.Errorf("PreviewResult mutated fs.Result(): now %d bodies", len(fs.Result()))
	}
	if fs.Count() != 2 {
		t.Errorf("PreviewResult changed the program length to %d, want 2", fs.Count())
	}
	if c0, c1 := fs.Item(0).RecomputeCount(), fs.Item(1).RecomputeCount(); c0 != beforeCounts[0] || c1 != beforeCounts[1] {
		t.Errorf("PreviewResult re-evaluated committed features: counts %d,%d → %d,%d", beforeCounts[0], beforeCounts[1], c0, c1)
	}
}

func TestPreviewResultPropagatesCandidateError(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	fs.Add(body())
	fs.Recompute()
	if _, err := fs.PreviewResult(failer{}); err == nil {
		t.Fatal("PreviewResult should surface a sick candidate's error, got nil")
	}
	if _, err := fs.PreviewResult(nil); err == nil {
		t.Fatal("PreviewResult(nil) should error")
	}
}

func TestEditingEarlyFeatureReusesCleanPrefix(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	a := fs.Add(body())
	b := fs.Add(body())
	c := fs.Add(body())
	fs.Recompute() // all evaluated once
	for _, f := range []*PartFeature{a, b, c} {
		if f.RecomputeCount() != 1 {
			t.Fatalf("after first recompute, %s count = %d, want 1", f.Name(), f.RecomputeCount())
		}
	}
	// Edit the middle feature: it and its tail re-evaluate; the prefix (a) does not.
	fs.MarkDirty(b)
	fs.Recompute()
	if a.RecomputeCount() != 1 {
		t.Errorf("clean prefix feature a recomputed again: count=%d", a.RecomputeCount())
	}
	if b.RecomputeCount() != 2 || c.RecomputeCount() != 2 {
		t.Errorf("dirty tail not re-evaluated: b=%d c=%d, want 2/2", b.RecomputeCount(), c.RecomputeCount())
	}
}

func TestFailingFeatureGoesSickWithoutAbortingRebuild(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	good := fs.Add(body())
	bad := fs.Add(failer{})
	after := fs.Add(body()) // independent of bad → must still evaluate
	fs.Recompute()

	if !good.Health().OK() {
		t.Error("healthy feature before the failure went sick")
	}
	if bad.Health().Status != health.Sick {
		t.Errorf("failing feature health = %v, want sick", bad.Health().Status)
	}
	if !after.Health().OK() {
		t.Error("independent feature after the failure should still be healthy (rebuild not aborted)")
	}
}

func TestSickFeaturePoisonsDependents(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	bad := fs.Add(failer{})
	dependent := fs.Add(body(), bad.ID()) // depends on the failing feature
	fs.Recompute()
	if dependent.Health().Status != health.Sick {
		t.Errorf("dependent of a sick feature = %v, want sick (poisoned)", dependent.Health().Status)
	}
	if dependent.RecomputeCount() != 0 {
		t.Error("poisoned feature should not run its recompute")
	}
}

// TestPanickingFeatureGoesSickWithoutAbortingRebuild is acceptance criterion 1/2 of #1415: a
// feature whose Recompute panics is marked Sick and the rebuild carries on — Recompute itself
// must return normally (no panic escaping to crash the app), and an independent later feature
// still evaluates healthy.
func TestPanickingFeatureGoesSickWithoutAbortingRebuild(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	good := fs.Add(body())
	bad := fs.Add(panicker{})
	after := fs.Add(body()) // independent of bad → must still evaluate
	fs.Recompute()          // must return, not unwind the panic

	if !good.Health().OK() {
		t.Error("healthy feature before the panic went sick")
	}
	if bad.Health().Status != health.Sick {
		t.Errorf("panicking feature health = %v, want sick", bad.Health().Status)
	}
	if !after.Health().OK() {
		t.Error("independent feature after the panic should still be healthy (rebuild not aborted)")
	}
}

// TestPanicSickMessageNamesFeatureAndValue is acceptance criterion 3 of #1415: the sick reason
// carries the offending feature's identity and the recovered panic value (per the CLAUDE.md
// exception-message rule), so the user can see WHAT failed and WHY.
func TestPanicSickMessageNamesFeatureAndValue(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	bad := fs.Add(panicker{})
	fs.Recompute()
	reason := bad.Health().Reason
	if !strings.Contains(reason, "panicker") || !strings.Contains(reason, "kernel nil-deref") {
		t.Errorf("sick reason %q must name the feature (panicker) and the panic value (kernel nil-deref)", reason)
	}
}

// TestPanickingFeaturePoisonsDependents confirms a panic is treated like any other failure
// downstream: a feature depending on the panicking one is poisoned sick, not evaluated.
func TestPanickingFeaturePoisonsDependents(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	bad := fs.Add(panicker{})
	dependent := fs.Add(body(), bad.ID())
	fs.Recompute()
	if bad.Health().Status != health.Sick {
		t.Errorf("panicking feature = %v, want sick", bad.Health().Status)
	}
	if dependent.Health().Status != health.Sick {
		t.Errorf("dependent of a panicking feature = %v, want sick (poisoned)", dependent.Health().Status)
	}
}

func TestLookupAndRename(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	f := fs.Add(body())
	f.SetName("Extrusion1")
	id := f.ID()
	f.SetName("Boss") // rename is id-stable
	if f.ID() != id {
		t.Error("rename changed the feature id")
	}
	if got, ok := fs.ByName("Boss"); !ok || got != f {
		t.Error("ByName did not find the renamed feature")
	}
	if got, ok := fs.ByID(id); !ok || got != f {
		t.Error("ByID failed after rename")
	}
}

func TestRemoveFeatureDropsItAndRebuilds(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	a := fs.Add(body())
	b := fs.Add(body())
	fs.Recompute()
	if len(fs.Result()) != 2 {
		t.Fatalf("precondition: result has %d bodies, want 2", len(fs.Result()))
	}
	if !fs.Remove(a.ID()) {
		t.Fatal("Remove reported the feature missing")
	}
	if fs.Count() != 1 || fs.Item(0) != b {
		t.Errorf("after Remove: count=%d, item0==b: %v", fs.Count(), fs.Item(0) == b)
	}
	if _, ok := fs.ByID(a.ID()); ok {
		t.Error("ByID still finds the removed feature")
	}
	fs.Recompute()
	if len(fs.Result()) != 1 {
		t.Errorf("after Remove+Recompute: result has %d bodies, want 1", len(fs.Result()))
	}
}

func TestRemoveFeatureMissingIsNoop(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	fs.Add(body())
	if fs.Remove(99999) {
		t.Error("Remove of an absent id reported success")
	}
	if fs.Count() != 1 {
		t.Errorf("Remove of an absent id changed the count to %d", fs.Count())
	}
}
