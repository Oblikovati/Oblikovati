// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "oblikovati.org/math"
)

// TestEntityListStorageCore pins the shared storage core the typed factory collections
// embed (#1656): append order, Count/Item indexing, and remove-first-occurrence
// semantics identical to removeItem (the regression contract for entity bookkeeping).
func TestEntityListStorageCore(t *testing.T) {
	var l entityList[int]
	l.append(10)
	l.append(20)
	l.append(10)
	if l.Count() != 3 || l.Item(0) != 10 || l.Item(1) != 20 || l.Item(2) != 10 {
		t.Fatalf("after appends: count=%d items=%v, want [10 20 10]", l.Count(), l.items)
	}
	l.remove(10) // first occurrence only
	if l.Count() != 2 || l.Item(0) != 20 || l.Item(1) != 10 {
		t.Fatalf("after remove: items=%v, want [20 10]", l.items)
	}
	l.remove(99) // absent value is a no-op
	if l.Count() != 2 {
		t.Fatalf("remove of an absent value changed the list: %v", l.items)
	}
}

// TestEntityListItemIsUnguarded pins that Item keeps the collections' historical
// unguarded semantics (panic out of range) — the nil-guarded shape is reserved for
// the contract-facing views (#1655).
func TestEntityListItemIsUnguarded(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Item out of range should panic, not return a zero value")
		}
	}()
	var l entityList[int]
	_ = l.Item(0)
}

// TestCircularCenters returns each circle's and arc's centre so a sketch overlay can mark them
// (#2159), and nothing for a sketch without circular geometry.
func TestCircularCenters(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	if got := s.CircularCenters(); len(got) != 0 {
		t.Fatalf("empty sketch CircularCenters = %v, want none", got)
	}
	s.Circles().AddByCenterRadius(gmath.P2(2, 3), 1)
	s.Arcs().AddByCenterStartEnd(gmath.P2(-4, 5), gmath.P2(-3, 5), gmath.P2(-4, 6), true)

	got := s.CircularCenters()
	if len(got) != 2 {
		t.Fatalf("CircularCenters = %d centres, want 2 (one circle, one arc)", len(got))
	}
	if !got[0].IsEqualTo(gmath.P2(2, 3), 1e-9) {
		t.Errorf("circle centre = %v, want (2,3)", got[0])
	}
	if !got[1].IsEqualTo(gmath.P2(-4, 5), 1e-9) {
		t.Errorf("arc centre = %v, want (-4,5)", got[1])
	}
}
