// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	stdmath "math"
	"strconv"
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/health"
	"oblikovati.org/model/occurrence"
)

// topHalfCutter builds an assembly-space tool box spanning a wide X range that, cut
// from a unit box, removes its top half (z ∈ [0.5,1]) — half the volume.
func topHalfCutter(t *testing.T) *topo.Body {
	t.Helper()
	tool, err := brep.SolidBlock(math.P3(-1, -1, 0.5), math.P3(100, 2, 2), "asmTool")
	if err != nil {
		t.Fatalf("SolidBlock cutter: %v", err)
	}
	return tool
}

// resultVolume sums o's machined assembly-space result bodies' volumes, for analytic
// gating of the assembly feature program.
func resultVolume(fs *AssemblyFeatures, o *occurrence.Occurrence) float64 {
	v := 0.0
	for _, b := range fs.Result(o) {
		v += ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	}
	return v
}

// assemblyOfUnitBoxes places one unit-box part at each given X translation and returns
// the assembly and the occurrences in order.
func assemblyOfUnitBoxes(t *testing.T, xs ...float64) (*AssemblyComponentDefinition, []*occurrence.Occurrence) {
	t.Helper()
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	asm := NewAssemblyComponentDefinition()
	occs := make([]*occurrence.Occurrence, len(xs))
	for i, x := range xs {
		occs[i] = asm.Place("box:"+strconv.Itoa(i+1), part, math.Translation4(math.V3(x, 0, 0)))
	}
	return asm, occs
}

// TestAddFeatureDefaultsParticipationToPresentComponents: an added feature participates
// on every component present at add time, and not on components added later (the
// reference API's default-participation rule).
func TestAddFeatureDefaultsParticipationToPresentComponents(t *testing.T) {
	t.Parallel()
	asm, occs := assemblyOfUnitBoxes(t, 0, 5)
	af := asm.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))

	if !af.Participates(occs[0]) || !af.Participates(occs[1]) {
		t.Error("present components should participate by default")
	}
	if got := len(af.Participants()); got != 2 {
		t.Fatalf("default participants = %d, want 2", got)
	}

	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	late := asm.Place("late:1", part, math.Translation4(math.V3(20, 0, 0)))
	if af.Participates(late) {
		t.Error("a component added after the feature should not participate by default")
	}
}

// TestAssemblyFeatureMachinesParticipantsGated gates the whole host against analytic
// volume: a top-half cut machines each participant to 0.5 and leaves a removed
// participant (and the shared part definition) untouched at 1.0.
func TestAssemblyFeatureMachinesParticipantsGated(t *testing.T) {
	t.Parallel()
	asm, occs := assemblyOfUnitBoxes(t, 0, 5, 20)
	af := asm.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))
	af.RemoveParticipant(occs[2]) // box 3 opts out

	asm.RecomputeFeatures()

	for i := range 2 {
		if got := resultVolume(asm.Features(), occs[i]); stdmath.Abs(got-0.5) > 1e-6 {
			t.Errorf("participant %d machined volume = %g, want 0.5", i, got)
		}
	}
	if got := resultVolume(asm.Features(), occs[2]); stdmath.Abs(got-1.0) > 1e-6 {
		t.Errorf("non-participant machined volume = %g, want 1.0 (untouched)", got)
	}
	// The shared part definition is unchanged — its own body is still a full unit box.
	shared := occs[0].Definition().(*PartComponentDefinition)
	sharedVol := ops.BodyGeometryProperties(shared.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume
	if stdmath.Abs(sharedVol-1.0) > 1e-6 {
		t.Errorf("shared part definition volume = %g, want 1.0 (assembly cut must not edit it)", sharedVol)
	}
}

// TestSuppressedFeaturePassesGeometryThrough: a suppressed assembly feature machines
// nothing (participants keep full volume) and reports Suppressed health; unsuppressing
// re-applies it.
func TestSuppressedFeaturePassesGeometryThrough(t *testing.T) {
	t.Parallel()
	asm, occs := assemblyOfUnitBoxes(t, 0)
	af := asm.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))

	asm.Features().SuppressFeatures(af.ID())
	asm.RecomputeFeatures()
	if got := resultVolume(asm.Features(), occs[0]); stdmath.Abs(got-1.0) > 1e-6 {
		t.Errorf("suppressed feature machined volume = %g, want 1.0 (passthrough)", got)
	}
	if af.Health().Status != health.Suppressed {
		t.Errorf("suppressed feature health = %v, want Suppressed", af.Health().Status)
	}

	asm.Features().UnsuppressFeatures(af.ID())
	asm.RecomputeFeatures()
	if got := resultVolume(asm.Features(), occs[0]); stdmath.Abs(got-0.5) > 1e-6 {
		t.Errorf("unsuppressed feature machined volume = %g, want 0.5", got)
	}
	if af.Health().Status != health.OK {
		t.Errorf("recomputed feature health = %v, want OK", af.Health().Status)
	}
}

// TestEndOfFeaturesRollsBackTrailingFeatures: the EOF marker bounds how far the program
// evaluates, so rolling back drops trailing features' machining.
func TestEndOfFeaturesRollsBackTrailingFeatures(t *testing.T) {
	t.Parallel()
	asm, occs := assemblyOfUnitBoxes(t, 0)
	// F1 removes the top half (→0.5); F2 removes everything above z=0.25 of F1's result (→0.25).
	asm.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))
	lower, _ := brep.SolidBlock(math.P3(-1, -1, 0.25), math.P3(100, 2, 2), "asmTool2")
	asm.AddFeature(feature.NewAssemblyCutFeature(lower, ops.Cut))

	asm.RecomputeFeatures()
	if got := resultVolume(asm.Features(), occs[0]); stdmath.Abs(got-0.25) > 1e-6 {
		t.Errorf("full program volume = %g, want 0.25 (both cuts)", got)
	}

	asm.Features().SetEndOfFeatures(1) // evaluate only F1
	if !asm.Features().IsRolledBack() {
		t.Error("IsRolledBack = false after SetEndOfFeatures(1), want true")
	}
	asm.RecomputeFeatures()
	if got := resultVolume(asm.Features(), occs[0]); stdmath.Abs(got-0.5) > 1e-6 {
		t.Errorf("rolled-back volume = %g, want 0.5 (only F1)", got)
	}

	asm.Features().RollToEnd()
	asm.RecomputeFeatures()
	if got := resultVolume(asm.Features(), occs[0]); stdmath.Abs(got-0.25) > 1e-6 {
		t.Errorf("rolled-to-end volume = %g, want 0.25 (both cuts again)", got)
	}
}

// pathVolume sums one placement's machined assembly-feature result volume.
func pathVolume(fs *AssemblyFeatures, path occurrence.OccurrencePath) float64 {
	v := 0.0
	for _, b := range fs.ResultPath(path) {
		v += ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	}
	return v
}

// TestNestedParticipationPathRestriction: a sub-assembly placed twice shares one leaf
// occurrence, so the two placements are distinguished only by path. By default a
// feature machines both; restricting it to one path machines that placement alone.
func TestNestedParticipationPathRestriction(t *testing.T) {
	t.Parallel()
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	sub := NewAssemblyComponentDefinition()
	leaf := sub.Place("part:1", part, math.Identity4())

	top := NewAssemblyComponentDefinition()
	top.Place("subA:1", sub, math.Identity4())
	top.Place("subB:1", sub, math.Translation4(math.V3(10, 0, 0)))

	af := top.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))
	top.RecomputeFeatures()

	pathA := occurrence.OccurrencePath{"subA:1", "part:1"}
	pathB := occurrence.OccurrencePath{"subB:1", "part:1"}
	if got := resultVolume(top.Features(), leaf); stdmath.Abs(got-1.0) > 1e-6 {
		t.Errorf("default leaf result = %g, want 1.0 (both placements machined to 0.5)", got)
	}
	if got := pathVolume(top.Features(), pathA); stdmath.Abs(got-0.5) > 1e-6 {
		t.Errorf("default subA placement = %g, want 0.5", got)
	}

	af.SetParticipantPaths([]occurrence.OccurrencePath{pathA})
	top.RecomputeFeatures()
	if got := pathVolume(top.Features(), pathA); stdmath.Abs(got-0.5) > 1e-6 {
		t.Errorf("restricted subA placement = %g, want 0.5 (still machined)", got)
	}
	if got := pathVolume(top.Features(), pathB); stdmath.Abs(got-1.0) > 1e-6 {
		t.Errorf("restricted subB placement = %g, want 1.0 (excluded by path)", got)
	}
}

// TestRecomputeRaisesFeaturesEvent checks that recomputing the feature program raises
// AssemblyFeaturesRecomputed on the assembly bus, carrying each feature's health.
func TestRecomputeRaisesFeaturesEvent(t *testing.T) {
	t.Parallel()
	asm, _ := assemblyOfUnitBoxes(t, 0)
	var got *AssemblyFeaturesRecomputed
	event.Subscribe(asm.Events().Bus(), event.After, func(_ event.Context, e AssemblyFeaturesRecomputed) event.Outcome {
		got = &e
		return event.Continue()
	})

	af := asm.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))
	asm.RecomputeFeatures()

	if got == nil || len(got.Features) != 1 || got.Features[0].ID != af.ID() {
		t.Fatalf("recompute event = %+v, want one feature snapshot for %d", got, af.ID())
	}
	if got.Features[0].Suppressed || got.Features[0].Health != "" {
		t.Errorf("feature snapshot = %+v, want unsuppressed + healthy", got.Features[0])
	}
}

// TestAssemblyFeaturesLookupsAndRemove covers the collection bookkeeping: unique
// naming, id/name lookup, removal, and count.
func TestAssemblyFeaturesLookupsAndRemove(t *testing.T) {
	t.Parallel()
	asm, _ := assemblyOfUnitBoxes(t, 0)
	fs := asm.Features()
	a := asm.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))
	a.SetName(fs.UniqueName("assemblyCut"))
	b := asm.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))
	b.SetName(fs.UniqueName("assemblyCut"))

	if a.Name() == b.Name() {
		t.Errorf("UniqueName returned duplicate %q", a.Name())
	}
	if got, ok := fs.ByID(a.ID()); !ok || got != a {
		t.Error("ByID did not return the feature")
	}
	if got, ok := fs.ByName(b.Name()); !ok || got != b {
		t.Error("ByName did not return the feature")
	}
	if !fs.Remove(a.ID()) || fs.Count() != 1 {
		t.Errorf("after Remove: count = %d, want 1", fs.Count())
	}
	if _, ok := fs.ByID(a.ID()); ok {
		t.Error("removed feature still resolves by id")
	}
}

// TestAddFeatureOwnsAttachPolicy: the aggregate names a new feature uniquely by
// kind and excludes a proxy cut's source from participation — policy that used
// to be duplicated across the router, the UI tools, and the load path (#1612).
func TestAddFeatureOwnsAttachPolicy(t *testing.T) {
	t.Parallel()
	asm, occs := assemblyOfUnitBoxes(t, 0, 5)
	first := asm.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))
	second := asm.AddFeature(feature.NewAssemblyCutFeature(topHalfCutter(t), ops.Cut))
	if first.Name() == "" || first.Name() == second.Name() {
		t.Errorf("default names = %q / %q, want unique non-empty names by kind", first.Name(), second.Name())
	}

	pc := asm.AddFeature(feature.NewAssemblyProxyCutFeature(occs[1], ops.Cut))
	for _, o := range pc.Participants() {
		if o == occs[1] {
			t.Error("a proxy cut's source must be excluded from its default participation")
		}
	}
}
