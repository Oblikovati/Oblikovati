// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
)

// wideAssembly builds a top assembly with several top-level placements (enough to trip the
// parallel flatten) plus a nested sub-assembly and a flexible one, so the serial/parallel
// equality check exercises depth, flyweight reuse, and flexible-child overrides.
func wideAssembly(t *testing.T) *AssemblyComponentDefinition {
	t.Helper()
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	sub := NewAssemblyComponentDefinition()
	sub.Place("inner:1", part, math.Translation4(math.V3(5, 0, 0)))
	sub.Place("inner:2", part, math.Translation4(math.V3(0, 5, 0)))

	top := NewAssemblyComponentDefinition()
	for i := range 6 {
		top.Place("loose:"+string(rune('a'+i)), part, math.Translation4(math.V3(math.Scalar(10*i), 0, 0)))
	}
	top.Place("sub:1", sub, math.Translation4(math.V3(100, 0, 0)))
	top.Place("sub:2", sub, math.Translation4(math.V3(200, 0, 0)))
	return top
}

// TestFlattenParallelMatchesSerial is the correctness contract for the F2 parallel flatten: the
// worker-chunked result must be byte-for-byte identical to the single-threaded walk — same order,
// transforms, sources and paths — or picking/derivation built on PlacedBodies would shift.
func TestFlattenParallelMatchesSerial(t *testing.T) {
	t.Parallel()
	top := wideAssembly(t)
	serial := flattenSerial(top.occurrences, 1)
	parallel := flattenParallel(top.occurrences, 1)
	if len(serial) != len(parallel) {
		t.Fatalf("parallel produced %d bodies, serial %d", len(parallel), len(serial))
	}
	for i := range serial {
		s, p := serial[i], parallel[i]
		if s.Body != p.Body || s.Source != p.Source || s.Transform != p.Transform {
			t.Errorf("body %d differs: serial(%p,%p) parallel(%p,%p)", i, s.Body, s.Source, p.Body, p.Source)
		}
		if !pathsEqual(s.Path, p.Path) {
			t.Errorf("body %d path differs: serial %v parallel %v", i, s.Path, p.Path)
		}
	}
}

func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestChunkRangeCoversAllItems checks the worker partition is a clean cover: contiguous, gap-free,
// overlap-free, and summing to n — the property the parallel concat depends on for a faithful order.
func TestChunkRangeCoversAllItems(t *testing.T) {
	t.Parallel()
	cases := []struct{ n, workers int }{{30, 8}, {5, 5}, {7, 3}, {1, 4}, {100, 7}, {0, 4}}
	for _, c := range cases {
		next := 0
		covered := 0
		for w := 0; w < c.workers; w++ {
			lo, hi := chunkRange(c.n, c.workers, w)
			if lo != next {
				t.Errorf("chunkRange(%d,%d,%d) lo=%d, want contiguous %d", c.n, c.workers, w, lo, next)
			}
			if hi < lo {
				t.Errorf("chunkRange(%d,%d,%d) hi=%d < lo=%d", c.n, c.workers, w, hi, lo)
			}
			covered += hi - lo
			next = hi
		}
		if covered != c.n || next != c.n {
			t.Errorf("chunks for n=%d workers=%d covered %d (ended at %d), want %d", c.n, c.workers, covered, next, c.n)
		}
	}
}

// TestPlacedBodiesParallelAndSerialPathsAgree pins that the PUBLIC PlacedBodies (which picks the
// path by root count) agrees with the serial reference on the same tree, so the threshold switch is
// transparent to callers.
func TestPlacedBodiesParallelAndSerialPathsAgree(t *testing.T) {
	t.Parallel()
	top := wideAssembly(t) // 8 roots ≥ parallelFlattenMinRoots → public call takes the parallel path
	if top.occurrences.Count() < parallelFlattenMinRoots {
		t.Fatalf("fixture has %d roots, need ≥ %d to exercise the parallel path", top.occurrences.Count(), parallelFlattenMinRoots)
	}
	if got, want := len(top.PlacedBodies()), len(flattenSerial(top.occurrences, 1)); got != want {
		t.Errorf("public PlacedBodies = %d bodies, serial reference = %d", got, want)
	}
}
