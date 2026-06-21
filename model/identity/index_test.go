// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"strconv"
	"testing"
)

// revSource is an EntitySource that reports a revision, so the key manager indexes
// it for O(1) exact binding (M31-F08). setEntities models a recompute: it swaps the
// entity set AND bumps the revision, which is the signal the index rebuilds on.
type revSource struct {
	entities []Entity
	rev      uint64
}

func (s *revSource) Entities() []Entity { return s.entities }
func (s *revSource) Revision() uint64   { return s.rev }
func (s *revSource) setEntities(e []Entity) {
	s.entities = e
	s.rev++
}

func TestIndexedExactBind(t *testing.T) {
	m := NewKeyManager()
	src := &revSource{entities: []Entity{face("A"), face("B")}}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, face("B"))

	got, match := m.BindKeyToObject(key)
	if match != MatchExact {
		t.Fatalf("indexed match = %v, want exact", match)
	}
	if string(got.Lineage().LineageKey()) != "B" {
		t.Errorf("bound to %q, want B", got.Lineage().LineageKey())
	}
}

// TestIndexRefreshesOnRevisionBump is the no-behaviour-change guarantee for a
// recompute that drops the referenced entity: once the revision moves, the stale
// index must not keep resolving the vanished face.
func TestIndexRefreshesOnRevisionBump(t *testing.T) {
	m := NewKeyManager()
	src := &revSource{entities: []Entity{face("keep"), face("doomed")}}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, face("doomed"))

	if _, match := m.BindKeyToObject(key); match != MatchExact {
		t.Fatal("doomed face did not bind while present")
	}

	src.setEntities([]Entity{face("keep")}) // recompute removes the face, bumps rev
	if got, match := m.BindKeyToObject(key); match != MatchNone || got != nil {
		t.Errorf("bind after removal = (%v,%v), want (nil, none) — stale index?", got, match)
	}
	if m.CanBindKeyToObject(key) {
		t.Error("CanBind true after the indexed face vanished")
	}
}

// TestIndexedFaceReappears covers a face removed then recreated with the same
// lineage across two revisions — it must bind again, proving the index tracks the
// live set rather than a one-shot snapshot.
func TestIndexedFaceReappears(t *testing.T) {
	m := NewKeyManager()
	src := &revSource{entities: []Entity{face("A"), face("B")}}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, face("B"))

	src.setEntities([]Entity{face("A")}) // B gone
	if _, match := m.BindKeyToObject(key); match != MatchNone {
		t.Fatal("B should be unbound while removed")
	}
	src.setEntities([]Entity{face("A"), face("B")}) // B recreated
	if _, match := m.BindKeyToObject(key); match != MatchExact {
		t.Error("recreated B did not rebind after a later revision")
	}
}

// TestRebindSourceInvalidatesIndex guards the case the revision number alone cannot:
// a brand-new source that happens to reuse the old source's revision value. The
// explicit invalidation on RebindSource must force a rebuild against the new source.
func TestRebindSourceInvalidatesIndex(t *testing.T) {
	m := NewKeyManager()
	old := &revSource{entities: []Entity{face("A"), face("gone")}, rev: 7}
	ctx := m.CreateKeyContext(old)
	key, _ := m.GetReferenceKey(ctx, face("gone"))
	if _, match := m.BindKeyToObject(key); match != MatchExact {
		t.Fatal("setup: gone should bind against the old source")
	}

	// New source, same revision value, without the referenced face.
	fresh := &revSource{entities: []Entity{face("A")}, rev: 7}
	if err := m.RebindSource(ctx, fresh); err != nil {
		t.Fatalf("RebindSource: %v", err)
	}
	if _, match := m.BindKeyToObject(key); match != MatchNone {
		t.Error("stale index survived RebindSource into a same-revision source")
	}
}

func TestIndexedKindMustMatch(t *testing.T) {
	m := NewKeyManager()
	src := &revSource{entities: []Entity{fakeEntity{kind: KindEdge, lin: "L"}}}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, face("L")) // same lineage, FACE key

	if _, match := m.BindKeyToObject(key); match != MatchNone {
		t.Error("a face key bound to an edge of the same lineage via the index")
	}
}

func benchEntities(n int) []Entity {
	es := make([]Entity, n)
	for i := range es {
		es[i] = fakeEntity{kind: KindEdge, lin: "ext#1/edge#" + strconv.Itoa(i)}
	}
	return es
}

// BenchmarkBindExactIndexed binds the worst-case (last) entity in a 10k-entity body
// through the revisioned index; the index is built once and every bind is O(1).
func BenchmarkBindExactIndexed(b *testing.B) {
	m := NewKeyManager()
	src := &revSource{entities: benchEntities(10000)}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, src.entities[len(src.entities)-1])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, match := m.BindKeyToObject(key); match != MatchExact {
			b.Fatal("indexed bind missed")
		}
	}
}

// BenchmarkBindExactLinear is the same workload against an un-revisioned source —
// every bind linearly scans all 10k entities. The gap to the indexed benchmark is
// the F08 speedup.
func BenchmarkBindExactLinear(b *testing.B) {
	m := NewKeyManager()
	src := &fakeSource{entities: benchEntities(10000)}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, src.entities[len(src.entities)-1])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, match := m.BindKeyToObject(key); match != MatchExact {
			b.Fatal("linear bind missed")
		}
	}
}
