// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"sort"
	"testing"
)

// sortedIDs returns ids sorted, for order-independent comparison.
func sortedIDs(ids []ID) []ID {
	out := append([]ID(nil), ids...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func TestTrackRecordsModelValueReads(t *testing.T) {
	ps := NewParameters()
	a := mustAdd(t, ps, "a", "10 mm")
	b := mustAdd(t, ps, "b", "20 mm")
	mustAdd(t, ps, "c", "30 mm") // never read

	got := ps.Track(func() {
		_ = a.ModelValue()
		_ = b.ModelValue()
		_ = a.ModelValue() // a re-read must not duplicate
	})

	want := sortedIDs([]ID{a.ID(), b.ID()})
	if g := sortedIDs(got); len(g) != 2 || g[0] != want[0] || g[1] != want[1] {
		t.Errorf("Track captured %v, want %v (only the parameters actually read)", g, want)
	}
}

func TestTrackOutsideCaptureRecordsNothing(t *testing.T) {
	ps := NewParameters()
	a := mustAdd(t, ps, "a", "10 mm")
	// A read with no active capture must be a no-op (no panic, nothing leaked into a later Track).
	_ = a.ModelValue()
	if got := ps.Track(func() {}); len(got) != 0 {
		t.Errorf("Track with no reads captured %v, want empty", got)
	}
}

func TestNestedTrackBubblesReadsToOuter(t *testing.T) {
	ps := NewParameters()
	a := mustAdd(t, ps, "a", "10 mm")
	b := mustAdd(t, ps, "b", "20 mm")

	var inner []ID
	outer := ps.Track(func() {
		_ = a.ModelValue()
		inner = ps.Track(func() { _ = b.ModelValue() })
	})

	if len(inner) != 1 || inner[0] != b.ID() {
		t.Errorf("inner Track captured %v, want [b]", inner)
	}
	// The outer capture must see BOTH its own read and the inner one (composition).
	want := sortedIDs([]ID{a.ID(), b.ID()})
	if g := sortedIDs(outer); len(g) != 2 || g[0] != want[0] || g[1] != want[1] {
		t.Errorf("outer Track captured %v, want %v", g, want)
	}
}

func TestDrainChangedReturnsEditedAndDependents(t *testing.T) {
	ps := NewParameters()
	w := mustAdd(t, ps, "width", "10 cm")
	h := mustAdd(t, ps, "height", "2 * width") // depends on width
	mustAdd(t, ps, "other", "3 cm")

	ps.DrainChanged() // clear the records from construction

	if err := ps.SetExpression(w.ID(), "15 cm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	got := sortedIDs(ps.DrainChanged())
	want := sortedIDs([]ID{w.ID(), h.ID()}) // the edited parameter and its transitive dependent
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("DrainChanged = %v, want %v (edited + dependents, not the independent parameter)", got, want)
	}
	// Draining is destructive: a second drain is empty.
	if again := ps.DrainChanged(); len(again) != 0 {
		t.Errorf("second DrainChanged = %v, want empty", again)
	}
}
