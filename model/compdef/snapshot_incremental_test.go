// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Incremental undo/redo recompute (Oblikovati#1424 PR2): restoring a snapshot that changed only a
// feature tail must reuse the live engine's cached prefix and recompute just the tail — not reset
// the whole program and re-evaluate every feature. These tests pin that the kept prefix features
// keep their identity and recompute counters across the restore (so they were genuinely reused),
// that the rebuilt tail re-evaluates, that a non-feature change still falls back to the full
// reset, and that every path lands the same geometry.

// addExtrudeOn appends one more extruded block reusing an existing sketch, so the recipe change is
// purely a feature-tail change (no new sketch or parameter) — the case the incremental path covers.
func addExtrudeOn(def *compdef.PartComponentDefinition, sketchIndex int) {
	sk := def.Sketches().Item(sketchIndex)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
}

func TestIncrementalUndoReusesFeaturePrefix(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	const n = 6
	buildExtrudeStack(t, def, n, -1, "") // n independent blocks, no driven dimension
	def.Recompute()
	base, err := def.MarshalSnapshot() // the n-feature state to undo back to
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}

	addExtrudeOn(def, 0) // pure feature-tail change: +1 feature, sketches/params untouched
	def.Recompute()
	if def.Features().Count() != n+1 {
		t.Fatalf("after adding a feature: %d features, want %d", def.Features().Count(), n+1)
	}
	before := recomputeCounts(def)
	prefixObjs := make([]*feature.PartFeature, n)
	for i := 0; i < n; i++ {
		prefixObjs[i] = def.Features().Item(i)
	}

	if err := def.RestoreSnapshot(base); err != nil { // undo the feature add
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	if def.Features().Count() != n {
		t.Fatalf("after incremental undo: %d features, want %d", def.Features().Count(), n)
	}
	after := recomputeCounts(def)
	for i := 0; i < n; i++ {
		if def.Features().Item(i) != prefixObjs[i] {
			t.Errorf("feature %d was rebuilt (different object); the incremental path should reuse it", i)
		}
		if after[i] != before[i] {
			t.Errorf("feature %d recomputed on incremental undo: %d→%d, want reused (cached prefix)", i, before[i], after[i])
		}
	}
}

func TestIncrementalRedoRebuildsOnlyTailFeature(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	const n = 5
	buildExtrudeStack(t, def, n, -1, "")
	def.Recompute()
	base, err := def.MarshalSnapshot() // n features
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	addExtrudeOn(def, 0)
	def.Recompute()
	withExtra, err := def.MarshalSnapshot() // n+1 features
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}

	if err := def.RestoreSnapshot(base); err != nil { // undo
		t.Fatalf("undo: %v", err)
	}
	before := recomputeCounts(def)
	if err := def.RestoreSnapshot(withExtra); err != nil { // redo: rebuild only the tail feature
		t.Fatalf("redo: %v", err)
	}
	if def.Features().Count() != n+1 {
		t.Fatalf("after redo: %d features, want %d", def.Features().Count(), n+1)
	}
	after := recomputeCounts(def)
	for i := 0; i < n; i++ {
		if after[i] != before[i] {
			t.Errorf("prefix feature %d recomputed on redo: %d→%d, want reused", i, before[i], after[i])
		}
	}
	if got := def.Features().Item(n).RecomputeCount(); got != 1 {
		t.Errorf("the redone tail feature evaluated %d times, want exactly 1", got)
	}
}

func TestIncrementalRestoreGeometryMatchesFreshBuild(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	const n = 6
	buildExtrudeStack(t, def, n, -1, "")
	def.Recompute()
	base, err := def.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	addExtrudeOn(def, 0)
	def.Recompute()
	if err := def.RestoreSnapshot(base); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}

	fresh := compdef.NewPartComponentDefinition()
	buildExtrudeStack(t, fresh, n, -1, "")
	fresh.Recompute()

	a, b := def.RangeBox(), fresh.RangeBox()
	if !a.Min.IsEqualTo(b.Min, 1e-9) || !a.Max.IsEqualTo(b.Max, 1e-9) {
		t.Errorf("incremental-restore geometry [%v..%v] != fresh-build geometry [%v..%v]", a.Min, a.Max, b.Min, b.Max)
	}
	if def.SurfaceBodies().Count() != fresh.SurfaceBodies().Count() {
		t.Errorf("incremental-restore body count %d != fresh %d", def.SurfaceBodies().Count(), fresh.SurfaceBodies().Count())
	}
}

// TestParameterSnapshotRestoreFallsBackToFullRebuild: a snapshot differing in a non-feature
// section (here a parameter expression) is NOT eligible for the incremental path, so the restore
// resets the engine — observable as the prefix features being rebuilt (new objects) — and still
// lands the snapshot's geometry.
func TestParameterSnapshotRestoreFallsBackToFullRebuild(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	if _, err := def.Parameters().AddUserParameter("drive", "40 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	buildExtrudeStack(t, def, 4, 2, "drive")
	def.Recompute()
	at40, err := def.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}

	if err := def.Parameters().SetExpression(userParamID(t, def, "drive"), "60 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	def.RecomputeAfterChange()
	firstFeatureAt60 := def.Features().Item(0)

	if err := def.RestoreSnapshot(at40); err != nil { // a parameter differs → full reset path
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	if def.Features().Item(0) == firstFeatureAt60 {
		t.Error("a parameter-differing restore reused the engine; it must fall back to a full reset")
	}
	if _, ok := def.Parameters().ByName("drive"); !ok {
		t.Fatal("restore lost the drive parameter")
	}
}

// TestIncrementalRestoreThenWholesaleParamEditStillFullRebuilds guards the wholesale-set
// invariant (#1424 PR2): an incremental restore must not shrink the set of parameters that force a
// full rebuild, or a later edit to such a parameter could leave a feature on stale geometry. Here
// a work-plane offset parameter ("lift") reaches geometry through an unmodelled path; after an
// incremental feature-tail restore, editing it must still rebuild the whole program and move the
// geometry — never silently skip it.
func TestIncrementalRestoreThenWholesaleParamEditStillFullRebuilds(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	lift, err := def.Parameters().AddUserParameter("lift", "20 mm")
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	base := def.Sketches().Add(sketch.XYPlane())
	rectangle(base, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(base, 0, ops.NewBody, func() float64 { return 5 })
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, lift.ModelValue)
	lifted := def.Sketches().Add(wp.Plane())
	lifted.SetPlaneHost(func() sketch.Plane { return wp.Plane() })
	rectangle(lifted, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(lifted, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()

	twoFeatures, err := def.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	addExtrudeOn(def, 0) // pure feature-tail change → eligible for the incremental path
	def.Recompute()
	if err := def.RestoreSnapshot(twoFeatures); err != nil { // incremental undo
		t.Fatalf("RestoreSnapshot: %v", err)
	}

	before := recomputeCounts(def)
	if err := def.Parameters().SetExpression(userParamID(t, def, "lift"), "50 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	def.RecomputeAfterChange()

	after := recomputeCounts(def)
	for i := range before {
		if after[i] != before[i]+1 {
			t.Errorf("feature %d count %d→%d after the lift edit, want a full rebuild (+1) — the wholesale set was shrunk by the incremental restore", i, before[i], after[i])
		}
	}
	if z := def.RangeBox().Max.Z; z < 9.999 || z > 10.001 {
		t.Errorf("after lift→50 mm, model max.z = %v, want ~10 (lifted block followed its plane)", z)
	}
}
