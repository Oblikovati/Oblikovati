// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// Incremental parameter recompute (Oblikovati#1414): a parameter edit must rebuild only the
// features it actually affects (the dependent tail), not the whole program. Before this, every
// parameter edit called MarkAllDirty, so editing one dimension on an N-feature part recomputed
// all N features from index 0. These tests pin the new behaviour: the earliest feature whose
// consumed-sketch dimension (or direct read) the edit touched, and its tail, recompute; the
// clean prefix is reused; and an edit that reaches no feature recomputes nothing — while the
// resulting geometry still equals a full rebuild.

// buildExtrudeStack builds n independent extruded blocks, each on its own sketch offset along x.
// The block at driveIndex is sized by a dimension referencing the user parameter driveParam, so
// editing that parameter moves only that block (and the tail the engine rebuilds after it).
func buildExtrudeStack(t *testing.T, def *compdef.PartComponentDefinition, n, driveIndex int, driveParam string) {
	t.Helper()
	for i := range n {
		sk := def.Sketches().Add(sketch.XYPlane())
		ox := float64(i) * 10
		c0 := sk.Points().Add(math.P2(ox, 0))
		c1 := sk.Points().Add(math.P2(ox+4, 0)) // 4 cm == 40 mm, the driveParam's initial value
		c2 := sk.Points().Add(math.P2(ox+4, 3))
		c3 := sk.Points().Add(math.P2(ox, 3))
		sk.Lines().Add(c0, c1)
		sk.Lines().Add(c1, c2)
		sk.Lines().Add(c2, c3)
		sk.Lines().Add(c3, c0)
		if i == driveIndex {
			if _, err := sk.DimensionConstraints().AddDistance(c0, c1, driveParam); err != nil {
				t.Fatalf("AddDistance(%q): %v", driveParam, err)
			}
		}
		feature.NewExtrudeFeatures(def.Features()).
			AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	}
}

// recomputeCounts snapshots the per-feature recompute counters.
func recomputeCounts(def *compdef.PartComponentDefinition) []int {
	fs := def.Features()
	counts := make([]int, fs.Count())
	for i := range counts {
		counts[i] = fs.Item(i).RecomputeCount()
	}
	return counts
}

// userParamID resolves a user parameter's id by name.
func userParamID(t *testing.T, def *compdef.PartComponentDefinition, name string) param.ID {
	t.Helper()
	p, ok := def.Parameters().ByName(name)
	if !ok {
		t.Fatalf("parameter %q not found", name)
	}
	return p.ID()
}

func TestParameterEditRebuildsOnlyDependentTail(t *testing.T) {
	t.Parallel()
	def := compdef.NewPartComponentDefinition()
	if _, err := def.Parameters().AddUserParameter("drive", "40 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	const n, driveIndex = 8, 5
	buildExtrudeStack(t, def, n, driveIndex, "drive")
	def.Recompute()
	before := recomputeCounts(def)

	if err := def.Parameters().SetExpression(userParamID(t, def, "drive"), "60 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	def.RecomputeAfterChange()
	after := recomputeCounts(def)

	for i := range n {
		switch {
		case i < driveIndex && after[i] != before[i]:
			t.Errorf("feature %d (before the edited one) recomputed: %d→%d, want reused", i, before[i], after[i])
		case i >= driveIndex && after[i] != before[i]+1:
			t.Errorf("feature %d (the edited one or its tail) count %d→%d, want exactly +1", i, before[i], after[i])
		}
	}
}

func TestIndependentParameterEditRebuildsNothing(t *testing.T) {
	t.Parallel()
	def := compdef.NewPartComponentDefinition()
	if _, err := def.Parameters().AddUserParameter("drive", "40 mm"); err != nil {
		t.Fatalf("AddUserParameter(drive): %v", err)
	}
	if _, err := def.Parameters().AddUserParameter("unused", "10 mm"); err != nil {
		t.Fatalf("AddUserParameter(unused): %v", err)
	}
	buildExtrudeStack(t, def, 6, 3, "drive")
	def.Recompute()
	before := recomputeCounts(def)

	// "unused" drives no dimension and is read by nothing, so editing it must recompute no feature.
	if err := def.Parameters().SetExpression(userParamID(t, def, "unused"), "20 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	def.RecomputeAfterChange()
	after := recomputeCounts(def)

	for i := range before {
		if after[i] != before[i] {
			t.Errorf("feature %d recomputed on an unrelated parameter edit: %d→%d", i, before[i], after[i])
		}
	}
}

func TestTargetedParameterEditMatchesFullRebuildGeometry(t *testing.T) {
	t.Parallel()
	// A part whose driven block is edited from 40 mm to 60 mm via the targeted seam...
	edited := compdef.NewPartComponentDefinition()
	if _, err := edited.Parameters().AddUserParameter("drive", "40 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	buildExtrudeStack(t, edited, 6, 3, "drive")
	edited.Recompute()
	if err := edited.Parameters().SetExpression(userParamID(t, edited, "drive"), "60 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	edited.RecomputeAfterChange()

	// ...must match a part built from scratch at 60 mm (a full evaluation).
	fresh := compdef.NewPartComponentDefinition()
	if _, err := fresh.Parameters().AddUserParameter("drive", "60 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	buildExtrudeStack(t, fresh, 6, 3, "drive")
	fresh.Recompute()

	a, b := edited.RangeBox(), fresh.RangeBox()
	if !a.Min.IsEqualTo(b.Min, 1e-9) || !a.Max.IsEqualTo(b.Max, 1e-9) {
		t.Errorf("targeted-edit geometry [%v..%v] != full-rebuild geometry [%v..%v]", a.Min, a.Max, b.Min, b.Max)
	}
}
