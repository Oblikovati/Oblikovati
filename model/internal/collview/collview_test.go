// SPDX-License-Identifier: GPL-2.0-only

package collview

import "testing"

// noisemaker/quiet exercise the Elem→Iface conversion the contract collections rely on.
type quiet interface{ hush() }

type noisemaker struct{ id int }

func (*noisemaker) hush() {}

func asQuiet(n *noisemaker) quiet { return n }

// TestIndexedGuardNeverPanics pins the safety property of the whole contract collection
// surface (#1655): out-of-range Item returns a TRUE nil interface (add-ins probe with
// raw indices), valid indices convert the element, Count matches the slice.
func TestIndexedGuardNeverPanics(t *testing.T) {
	items := []*noisemaker{{id: 1}, {id: 2}}
	v := Over(items, asQuiet)
	if v.Count() != 2 {
		t.Fatalf("Count = %d, want 2", v.Count())
	}
	if got := v.Item(1); got != asQuiet(items[1]) {
		t.Errorf("Item(1) = %#v, want the converted second element", got)
	}
	for _, i := range []int{-1, 2, 1 << 20} {
		if got := v.Item(i); got != nil {
			t.Errorf("Item(%d) = %#v, want nil interface for out-of-range", i, got)
		}
	}
}

// TestIndexedEmptyView pins the empty-collection case: Count 0, every probe nil.
func TestIndexedEmptyView(t *testing.T) {
	v := Over([]*noisemaker(nil), asQuiet)
	if v.Count() != 0 || v.Item(0) != nil || v.Item(-1) != nil {
		t.Errorf("empty view: Count=%d Item(0)=%v Item(-1)=%v, want 0/nil/nil",
			v.Count(), v.Item(0), v.Item(-1))
	}
}

// TestAtReturnsZeroElemOutOfRange pins the identity-typed guard used by collections whose
// Item returns the element type itself: out of range yields a nil POINTER (the caller
// compares against the concrete type, not an interface).
func TestAtReturnsZeroElemOutOfRange(t *testing.T) {
	items := []*noisemaker{{id: 7}}
	if got := At(items, 0); got == nil || got.id != 7 {
		t.Errorf("At(0) = %#v, want the first element", got)
	}
	if At(items, -1) != nil || At(items, 1) != nil {
		t.Error("At out of range should return the zero (nil) element")
	}
}
