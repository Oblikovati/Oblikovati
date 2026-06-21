// SPDX-License-Identifier: GPL-2.0-only

package topo

import "testing"

func sameEdges(a, b []*Edge) bool {
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

func sameVertices(a, b []*Vertex) bool {
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

// TestDerivedCacheMatchesLiveDerivation is the F2b correctness contract: the cached adjacency lists
// a finalized body returns must be exactly the distinct faces/edges/vertices a fresh derivation
// produces — same elements, same first-seen order — or every consumer (range box, drawlist, picking,
// keys) shifts. A tetra has 4 faces, 6 edges, 4 vertices.
func TestDerivedCacheMatchesLiveDerivation(t *testing.T) {
	b := linedTetra(componentLineage)
	if !b.derived {
		t.Fatal("Build should finalize the derived caches")
	}
	if len(b.Faces()) != 4 || len(b.Edges()) != 6 || len(b.Vertices()) != 4 {
		t.Fatalf("tetra faces=%d edges=%d verts=%d, want 4/6/4", len(b.Faces()), len(b.Edges()), len(b.Vertices()))
	}
	if !sameEdges(b.Edges(), deriveEdgesFrom(b.deriveFaces())) {
		t.Error("cached body edges differ from a fresh derivation")
	}
	if !sameVertices(b.Vertices(), deriveVerticesFrom(deriveEdgesFrom(b.deriveFaces()))) {
		t.Error("cached body vertices differ from a fresh derivation")
	}
}

// TestDerivedCacheLivePathEqualsCached pins that the pre-finalize live path (derived==false, as
// during RegroupShells) yields the same result as the finalized cache, so finalization is
// transparent.
func TestDerivedCacheLivePathEqualsCached(t *testing.T) {
	b := linedTetra(componentLineage)
	cached := b.Edges()
	b.derived = false // force the uncached path the body uses mid-build
	if !sameEdges(cached, b.Edges()) {
		t.Error("live edge derivation differs from the finalized cache")
	}
}

// TestDerivedCacheReturnsCopy guards the fresh-slice contract: mutating a returned slice must not
// corrupt the cache, so a careless consumer cannot poison every later query.
func TestDerivedCacheReturnsCopy(t *testing.T) {
	b := linedTetra(componentLineage)
	edges := b.Edges()
	edges[0] = nil
	if b.Edges()[0] == nil {
		t.Error("Edges() handed out the cached backing slice; a mutation leaked into the cache")
	}
	faces := b.Faces()
	faces[0] = nil
	if b.Faces()[0] == nil {
		t.Error("Faces() handed out the cached backing slice")
	}
}

// TestFaceCacheFinalizedWithBody checks each face is finalized when its body is and returns its
// three bounding edges from the cache.
func TestFaceCacheFinalizedWithBody(t *testing.T) {
	b := linedTetra(componentLineage)
	for i, f := range b.Faces() {
		if !f.derived {
			t.Fatalf("face %d not finalized with its body", i)
		}
		if len(f.Edges()) != 3 {
			t.Errorf("tetra face %d edges = %d, want 3", i, len(f.Edges()))
		}
		if !sameEdges(f.Edges(), f.deriveEdges()) {
			t.Errorf("face %d cached edges differ from live derivation", i)
		}
	}
}

// TestMergeBodiesFinalizesUnion checks MergeBodies finalizes the merged body and its caches are the
// union of the inputs (two disjoint tetra = 8 faces / 12 edges / 8 vertices).
func TestMergeBodiesFinalizesUnion(t *testing.T) {
	a := linedTetra(componentLineage)
	c := linedTetra(placedLineage(1))
	m := MergeBodies(NewLineage(Tok("m", "body", 0)), true, a, c)
	if !m.derived {
		t.Fatal("MergeBodies should finalize the derived caches")
	}
	if len(m.Faces()) != 8 || len(m.Edges()) != 12 || len(m.Vertices()) != 8 {
		t.Errorf("merged faces=%d edges=%d verts=%d, want 8/12/8", len(m.Faces()), len(m.Edges()), len(m.Vertices()))
	}
}

// BenchmarkBodyEdgesCachedVsLive contrasts the finalized cached query against the pre-F2b live
// derivation (rebuilding the dedup map every call) — the per-frame cost the cache removes.
func BenchmarkBodyEdgesCachedVsLive(b *testing.B) {
	body := linedTetra(componentLineage)
	b.Run("cached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = body.Edges()
		}
	})
	b.Run("live", func(b *testing.B) {
		body.derived = false
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = body.Edges()
		}
		body.derived = true
	})
}
