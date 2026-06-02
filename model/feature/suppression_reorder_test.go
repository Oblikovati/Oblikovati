// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/model/health"
	"github.com/Oblikovati/oblikovati/model/param"
)

func TestExplicitSuppressionPassesBodiesThrough(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	fs.Add(body())
	mid := fs.Add(body())
	fs.Add(body())
	mid.SetSuppressed(true)
	fs.Recompute()
	if mid.Health().Status != health.Suppressed {
		t.Errorf("suppressed feature health = %v, want suppressed", mid.Health().Status)
	}
	if mid.RecomputeCount() != 0 {
		t.Error("suppressed feature should not run")
	}
	// Two active features → two bodies (the suppressed one contributes none).
	if len(fs.Result()) != 2 {
		t.Errorf("result has %d bodies, want 2 (suppressed contributes none)", len(fs.Result()))
	}
}

func TestConditionalSuppressionTogglesWithExpression(t *testing.T) {
	ps := param.NewParameters()
	count, _ := ps.AddUserParameter("count", "1")
	fs := NewPartFeatures(ps, nil)
	f := fs.Add(body())
	// Suppress the feature when count < 2.
	f.SetSuppressionCondition("count", LessThan, 2)

	fs.Recompute()
	if f.Health().Status != health.Suppressed {
		t.Fatalf("count=1 < 2 should suppress, got %v", f.Health().Status)
	}
	// Raise count above the threshold → no longer suppressed.
	if err := count.SetExpression("5"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	fs.MarkDirty(f)
	fs.Recompute()
	if !f.Health().OK() {
		t.Errorf("count=5 should un-suppress, got %v", f.Health())
	}
}

func TestReorderRejectedPastDependency(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	base := fs.Add(body())
	dependent := fs.Add(body(), base.ID()) // dependent must stay after base
	fs.Recompute()

	// Moving the dependent before its dependency must be rejected.
	if err := fs.Reorder(dependent, 0); err == nil {
		t.Error("reorder past a dependency should be rejected")
	}
	if fs.Item(0) != base {
		t.Error("rejected reorder must not change the order")
	}
}

func TestValidReorderReEvaluates(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	a := fs.Add(body())
	b := fs.Add(body())
	c := fs.Add(body())
	fs.Recompute()
	// Independent features: moving c to the front is valid.
	if err := fs.Reorder(c, 0); err != nil {
		t.Fatalf("valid reorder rejected: %v", err)
	}
	if fs.Item(0) != c || fs.Item(1) != a || fs.Item(2) != b {
		t.Errorf("order after reorder = [%s %s %s], want [c a b]", fs.Item(0).Name(), fs.Item(1).Name(), fs.Item(2).Name())
	}
	fs.Recompute()
	if len(fs.Result()) != 3 {
		t.Errorf("result has %d bodies after reorder, want 3", len(fs.Result()))
	}
}

func TestEndOfPartExcludesTrailingFeatures(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	fs.Add(body())
	fs.Add(body())
	last := fs.Add(body())
	fs.Recompute()
	if len(fs.Result()) != 3 {
		t.Fatalf("baseline result = %d bodies, want 3", len(fs.Result()))
	}
	// Roll back before the last feature → it is excluded.
	if err := fs.SetEndOfPart(last); err != nil {
		t.Fatalf("SetEndOfPart: %v", err)
	}
	fs.Recompute()
	if !fs.IsRolledBack() || len(fs.Result()) != 2 {
		t.Errorf("after EOP move: rolledBack=%v result=%d, want true/2", fs.IsRolledBack(), len(fs.Result()))
	}
	// Rolling to the end re-includes it.
	fs.RollToEnd()
	fs.Recompute()
	if fs.IsRolledBack() || len(fs.Result()) != 3 {
		t.Errorf("after RollToEnd: rolledBack=%v result=%d, want false/3", fs.IsRolledBack(), len(fs.Result()))
	}
}

func TestComparisonOperators(t *testing.T) {
	cases := []struct {
		v   float64
		cmp ComparisonType
		thr float64
		out bool
	}{
		{2, Equal, 2, true},
		{2, NotEqual, 3, true},
		{1, LessThan, 2, true},
		{3, GreaterThan, 2, true},
		{2, LessOrEqual, 2, true},
		{2, GreaterOrEqual, 2, true},
		{2, Equal, 3, false},
		{2, GreaterThan, 5, false},
	}
	for _, c := range cases {
		if got := compare(c.v, c.cmp, c.thr); got != c.out {
			t.Errorf("compare(%v, %d, %v) = %v, want %v", c.v, c.cmp, c.thr, got, c.out)
		}
	}
}

func TestFeatureAccessorsAndEdges(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	f := fs.Add(body())
	if f.Name() != "box" || f.Definition() == nil || f.Suppressed() {
		t.Error("basic accessors wrong")
	}
	f.SetDependencies([]ID{ID(99)})
	if len(f.Dependencies()) != 1 || f.Dependencies()[0] != ID(99) {
		t.Error("SetDependencies/Dependencies wrong")
	}
	// A condition with no parameter store never suppresses (holds → false).
	f.SetSuppressionCondition("x", LessThan, 1)
	fs.Recompute()
	if !f.Health().OK() {
		t.Error("condition with nil params should not suppress")
	}
	f.ClearSuppressionCondition()
	if fs.EndOfPartIndex() != -1 {
		t.Errorf("EndOfPartIndex = %d, want -1 (at end)", fs.EndOfPartIndex())
	}

	// Reordering / EOP on a feature not in the program errors.
	stray := &PartFeature{id: nextID(), name: "stray", feature: body()}
	if err := fs.Reorder(stray, 0); err == nil {
		t.Error("Reorder of a stray feature should error")
	}
	if err := fs.SetEndOfPart(stray); err == nil {
		t.Error("SetEndOfPart of a stray feature should error")
	}
	if err := fs.Reorder(f, 5); err == nil {
		t.Error("out-of-range reorder index should error")
	}
}
