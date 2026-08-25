// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"errors"
	"testing"
)

// fakeBLASBuilder counts Build/Destroy calls and hands back a distinct handle each
// build, so a test can assert exactly which hashes were (re)built.
type fakeBLASBuilder struct {
	nextHandle int
	built      []int // handles, in build order
	destroyed  []any
}

func (f *fakeBLASBuilder) BuildBLAS(_ []Triangle) (any, error) {
	f.nextHandle++
	f.built = append(f.built, f.nextHandle)
	return f.nextHandle, nil
}

func (f *fakeBLASBuilder) DestroyBLAS(handle any) { f.destroyed = append(f.destroyed, handle) }

func quadTriangles(z float32) []Triangle {
	return []Triangle{{V0: [3]float32{0, 0, z}, V1: [3]float32{1, 0, z}, V2: [3]float32{0, 1, z}}}
}

// TestBLASCacheSkipsUnchangedBodies is PBI-333's "rebuild triggers exactly on
// tessellation-dirty, not every frame" guard: syncing the same hash twice must not
// rebuild the second time.
func TestBLASCacheSkipsUnchangedBodies(t *testing.T) {
	fb := &fakeBLASBuilder{}
	c := NewBLASCache(fb)

	bodies := map[uint64][]Triangle{100: quadTriangles(0)}
	c.Sync(bodies)
	if c.BuildCount != 1 {
		t.Fatalf("first Sync BuildCount = %d, want 1", c.BuildCount)
	}
	c.Sync(bodies) // same hash, unchanged
	if c.BuildCount != 1 {
		t.Errorf("second Sync (unchanged) BuildCount = %d, want still 1 (no rebuild)", c.BuildCount)
	}
}

// TestBLASCacheEditingOneBodyRebuildsOnlyItsOwnBLAS is PBI-333's explicit acceptance
// criterion: editing body A (its hash changes) must rebuild only A's BLAS, leaving body
// B's untouched (same handle, no extra Build/Destroy call for B).
func TestBLASCacheEditingOneBodyRebuildsOnlyItsOwnBLAS(t *testing.T) {
	fb := &fakeBLASBuilder{}
	c := NewBLASCache(fb)

	const hashA, hashB = 100, 200
	handles := c.Sync(map[uint64][]Triangle{hashA: quadTriangles(0), hashB: quadTriangles(5)})
	handleBBefore := handles[hashB]
	if c.BuildCount != 2 {
		t.Fatalf("initial Sync BuildCount = %d, want 2", c.BuildCount)
	}

	// "Edit" body A: its content hash changes (a real caller derives this from the
	// tessellated triangle stream, e.g. an FNV hash — here we just use a new key,
	// which is exactly what a changed hash looks like from Sync's point of view).
	const hashAEdited = 101
	handles = c.Sync(map[uint64][]Triangle{hashAEdited: quadTriangles(0.1), hashB: quadTriangles(5)})

	if c.BuildCount != 3 {
		t.Errorf("BuildCount after editing A = %d, want 3 (one new build for the edited A)", c.BuildCount)
	}
	if c.DestroyCount != 1 {
		t.Errorf("DestroyCount after editing A = %d, want 1 (the old A hash evicted)", c.DestroyCount)
	}
	if handles[hashB] != handleBBefore {
		t.Errorf("body B's handle changed (%v → %v); editing A must not rebuild B", handleBBefore, handles[hashB])
	}
	if len(fb.destroyed) != 1 {
		t.Fatalf("destroyed = %v, want exactly 1 entry (A's old handle)", fb.destroyed)
	}
}

// TestBLASCacheRemovedBodyIsDestroyed checks a body dropped from the synced set (e.g.
// deleted) frees its BLAS.
func TestBLASCacheRemovedBodyIsDestroyed(t *testing.T) {
	fb := &fakeBLASBuilder{}
	c := NewBLASCache(fb)
	c.Sync(map[uint64][]Triangle{100: quadTriangles(0)})
	c.Sync(map[uint64][]Triangle{}) // body removed
	if c.Len() != 0 {
		t.Errorf("Len() after removal = %d, want 0", c.Len())
	}
	if c.DestroyCount != 1 {
		t.Errorf("DestroyCount after removal = %d, want 1", c.DestroyCount)
	}
}

// TestBLASCacheBuildErrorLeavesNoEntry checks a failed build doesn't poison the cache
// with a phantom entry.
func TestBLASCacheBuildErrorLeavesNoEntry(t *testing.T) {
	fb := &erroringBLASBuilder{}
	c := NewBLASCache(fb)
	c.Sync(map[uint64][]Triangle{100: quadTriangles(0)})
	if c.Len() != 0 {
		t.Errorf("Len() after a failed build = %d, want 0", c.Len())
	}
}

type erroringBLASBuilder struct{}

func (erroringBLASBuilder) BuildBLAS(_ []Triangle) (any, error) {
	return nil, errors.New("fake build failure")
}
func (erroringBLASBuilder) DestroyBLAS(_ any) {}
