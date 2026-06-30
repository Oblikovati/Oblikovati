// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"strconv"
	"sync"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// buildKeyedStrip builds a finalized body of n independent triangles laid along x, every
// vertex/edge/face carrying a distinct lineage. It is the fixture for the reference-key
// index: a body with many distinct keys, so a lookup that scans all entities is O(n) but an
// indexed lookup is O(1) (#1580).
func buildKeyedStrip(n int) *Body {
	bld := NewBuilder(false, NewLineage(Tok("strip", "body", 0)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	for i := 0; i < n; i++ {
		x := float64(i) * 2
		a := bld.AddVertex(math.P3(x, 0, 0), NewLineage(Tok("strip", "vertex", i*3)))
		b := bld.AddVertex(math.P3(x+1, 0, 0), NewLineage(Tok("strip", "vertex", i*3+1)))
		c := bld.AddVertex(math.P3(x, 1, 0), NewLineage(Tok("strip", "vertex", i*3+2)))
		ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, NewLineage(Tok("strip", "edge", i*3)))
		bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, NewLineage(Tok("strip", "edge", i*3+1)))
		ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, NewLineage(Tok("strip", "edge", i*3+2)))
		bld.AddFace(pl, NewLineage(Tok("strip", "face", i)), OuterLoop(Fwd(ab), Fwd(bc), Fwd(ca)))
	}
	return bld.Build()
}

// TestEdgesByKeyReturnsDefensiveCopy pins that a caller mutating the returned slice cannot
// corrupt a memoized index: each lookup must hand back a fresh slice, not the backing one.
func TestEdgesByKeyReturnsDefensiveCopy(t *testing.T) {
	body := buildKeyedStrip(8)
	want := body.Edges()[3]
	got := body.EdgesByKey(want.ReferenceKey())
	if len(got) != 1 {
		t.Fatalf("EdgesByKey bound %d edges, want 1", len(got))
	}
	got[0] = nil // a hostile caller scribbling on the result

	again := body.EdgesByKey(want.ReferenceKey())
	if len(again) != 1 || again[0] == nil {
		t.Error("EdgesByKey returned an aliased slice; the caller's mutation corrupted the index")
	}
}

// TestFacesByKeyReturnsDefensiveCopy is the face counterpart of the copy-safety contract.
func TestFacesByKeyReturnsDefensiveCopy(t *testing.T) {
	body := buildKeyedStrip(8)
	want := body.Faces()[5]
	got := body.FacesByKey(want.ReferenceKey())
	if len(got) != 1 {
		t.Fatalf("FacesByKey bound %d faces, want 1", len(got))
	}
	got[0] = nil

	if again := body.FacesByKey(want.ReferenceKey()); len(again) != 1 || again[0] == nil {
		t.Error("FacesByKey returned an aliased slice; the caller's mutation corrupted the index")
	}
}

// TestKeyLookupMemoizedConsistent walks every entity twice — the second pass hits the
// memoized index — and checks both passes resolve each key to the exact same entity, so the
// index agrees with a linear scan and is stable across calls.
func TestKeyLookupMemoizedConsistent(t *testing.T) {
	body := buildKeyedStrip(16)
	for pass := 0; pass < 2; pass++ {
		for _, e := range body.Edges() {
			got := body.EdgesByKey(e.ReferenceKey())
			if len(got) != 1 || got[0].ID() != e.ID() {
				t.Fatalf("pass %d: edge %d resolved to %v, want exactly itself", pass, e.ID(), got)
			}
		}
		for _, f := range body.Faces() {
			got := body.FacesByKey(f.ReferenceKey())
			if len(got) != 1 || got[0].ID() != f.ID() {
				t.Fatalf("pass %d: face %d resolved to %v, want exactly itself", pass, f.ID(), got)
			}
		}
	}
}

// TestKeyLookupMissReturnsEmpty checks an unknown key resolves to nothing on every accessor.
func TestKeyLookupMissReturnsEmpty(t *testing.T) {
	body := buildKeyedStrip(4)
	miss := []byte("no-such-lineage")
	if got := body.EdgesByKey(miss); len(got) != 0 {
		t.Errorf("EdgesByKey(miss) = %d, want 0", len(got))
	}
	if got := body.FacesByKey(miss); len(got) != 0 {
		t.Errorf("FacesByKey(miss) = %d, want 0", len(got))
	}
	if _, ok := body.FindEdgeByKey(miss); ok {
		t.Error("FindEdgeByKey(miss) reported a hit")
	}
	if _, ok := body.FindFaceByKey(miss); ok {
		t.Error("FindFaceByKey(miss) reported a hit")
	}
	if _, ok := body.FindVertexByKey(miss); ok {
		t.Error("FindVertexByKey(miss) reported a hit")
	}
}

// TestKeyIndexConcurrentLookupsAreRaceFree drives many goroutines into the first lookup of
// each kind at once, so the lazy sync.Once index build races with concurrent readers. Run
// under -race it backs the "memoized once, read lock-free" claim on [Body]'s index fields.
func TestKeyIndexConcurrentLookupsAreRaceFree(t *testing.T) {
	body := buildKeyedStrip(64)
	edges, faces, verts := body.Edges(), body.Faces(), body.Vertices()
	var wg sync.WaitGroup
	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			body.EdgesByKey(edges[g%len(edges)].ReferenceKey())
			body.FacesByKey(faces[g%len(faces)].ReferenceKey())
			body.FindVertexByKey(verts[g%len(verts)].ReferenceKey())
		}(g)
	}
	wg.Wait()
}

// benchKeys returns one reference key per edge of a body — the per-recompute workload of
// resolving a feature's stored edge references against the running body.
func benchKeys(edges []*Edge) [][]byte {
	keys := make([][]byte, len(edges))
	for i, e := range edges {
		keys[i] = e.ReferenceKey()
	}
	return keys
}

// BenchmarkEdgesByKey measures resolving every edge reference of a high-edge-count body
// against that body — the O(refs × n) pattern #1580 turns into O(refs) with the index.
func BenchmarkEdgesByKey(b *testing.B) {
	for _, n := range []int{50, 500, 5000} {
		body := buildKeyedStrip(n)
		keys := benchKeys(body.Edges())
		b.Run("tris="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				body.EdgesByKey(keys[i%len(keys)])
			}
		})
	}
}
