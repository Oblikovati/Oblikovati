// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdrand "math/rand"
	"sort"
	"testing"

	"oblikovati.org/math"
)

// randomTestBoxes returns n reproducible random boxes inside [0,20)³ with edges up to ~2.
func randomTestBoxes(n int, seed int64) []math.Box {
	rng := stdrand.New(stdrand.NewSource(seed))
	boxes := make([]math.Box, n)
	for i := range boxes {
		p := math.P3(rng.Float64()*20, rng.Float64()*20, rng.Float64()*20)
		q := p.TranslateBy(math.V3(rng.Float64()*2, rng.Float64()*2, rng.Float64()*2))
		boxes[i] = math.NewBox(p, q)
	}
	return boxes
}

// TestBoxTreeQueryMatchesBruteOverlap is the culling-soundness gate (#1607): for random query
// boxes, Query must report EXACTLY the items a brute scan finds overlapping — no false
// negatives (which would change boolean results) and no spurious extras beyond the box test.
func TestBoxTreeQueryMatchesBruteOverlap(t *testing.T) {
	boxes := randomTestBoxes(200, 1)
	tree := NewBoxTree(boxes)
	for qi, q := range randomTestBoxes(50, 2) {
		var got []int
		tree.Query(q, func(item int) bool { got = append(got, item); return false })
		sort.Ints(got)
		var want []int
		for i, b := range boxes {
			if b.Intersects(q) {
				want = append(want, i)
			}
		}
		if len(got) != len(want) {
			t.Fatalf("query %d: tree found %d overlaps, brute found %d", qi, len(got), len(want))
		}
		for k := range want {
			if got[k] != want[k] {
				t.Fatalf("query %d: tree overlap set %v != brute %v", qi, got, want)
			}
		}
	}
}

// TestBoxTreeQueryEarlyStop pins the early-out contract: a hit callback returning true ends
// the walk after that item.
func TestBoxTreeQueryEarlyStop(t *testing.T) {
	boxes := randomTestBoxes(64, 3)
	tree := NewBoxTree(boxes)
	visited := 0
	all := math.NewBox(math.P3(-1, -1, -1), math.P3(25, 25, 25))
	tree.Query(all, func(int) bool { visited++; return true })
	if visited != 1 {
		t.Fatalf("early-stop query visited %d items, want exactly 1", visited)
	}
}

// TestBoxTreeEmptyAndSelfQuery covers the degenerate empty tree, and that every item is
// reachable: querying each item's own box must report the item itself (a leaf-partitioning
// bug would silently drop items — a false negative the boolean cannot tolerate).
func TestBoxTreeEmptyAndSelfQuery(t *testing.T) {
	empty := NewBoxTree(nil)
	empty.Query(math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1)), func(int) bool {
		t.Fatal("empty tree Query must visit nothing")
		return true
	})
	boxes := randomTestBoxes(100, 4)
	tree := NewBoxTree(boxes)
	for i, b := range boxes {
		found := false
		tree.Query(b, func(item int) bool { found = found || item == i; return found })
		if !found {
			t.Fatalf("item %d not reachable through its own box", i)
		}
	}
}
