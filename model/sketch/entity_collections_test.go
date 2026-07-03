// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "testing"

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
