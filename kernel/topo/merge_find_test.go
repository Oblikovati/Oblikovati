// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"
)

// TestFaceEntityAccessors covers the Face ID/Kind/Lineage accessors that the broader
// entity-accessor test reaches only on vertices/edges/loops.
func TestFaceEntityAccessors(t *testing.T) {
	f := buildTetra().Faces()[0]
	if f.ID() == 0 {
		t.Error("face ID should be non-zero")
	}
	if f.Kind() != KindFace {
		t.Errorf("face kind = %v, want KindFace", f.Kind())
	}
	if len(f.Lineage().Tokens()) == 0 {
		t.Error("face lineage should carry tokens")
	}
}

// TestEdgeUses checks that a solid edge reports exactly its two oriented uses (one per
// adjacent face), and that the returned slice is a copy (mutating it can't corrupt the edge).
func TestEdgeUses(t *testing.T) {
	e := buildTetra().Edges()[0]
	uses := e.Uses()
	if len(uses) != 2 {
		t.Fatalf("solid edge has %d uses, want 2", len(uses))
	}
	uses[0] = nil // must not affect the edge's own slice
	if e.Uses()[0] == nil {
		t.Error("Uses() returned the backing slice, not a copy")
	}
}

// TestFindVertexByKey resolves a vertex from its reference key (a round-trip) and reports a
// miss for an unknown key.
func TestFindVertexByKey(t *testing.T) {
	body := buildTetra()
	want := body.Vertices()[0]
	got, ok := body.FindVertexByKey(want.ReferenceKey())
	if !ok || got.ID() != want.ID() {
		t.Errorf("FindVertexByKey round-trip = (%v,%v), want the same vertex", got, ok)
	}
	if _, ok := body.FindVertexByKey([]byte("no-such-key")); ok {
		t.Error("FindVertexByKey should miss on an unknown key")
	}
}

// TestMergeBodies combines two solids into one multi-shell body and re-parents every shell
// (the boolean-Join-of-disjoint-bodies path in kernel/ops).
func TestMergeBodies(t *testing.T) {
	a, b := buildTetra(), buildTetra()
	lin := NewLineage(Tok("merge", "body", 0))
	merged := MergeBodies(lin, true, a, b)
	if got := len(merged.Shells()); got != len(a.Shells())+len(b.Shells()) {
		t.Fatalf("merged body has %d shells, want %d", got, len(a.Shells())+len(b.Shells()))
	}
	if !merged.IsSolid() {
		t.Error("merged body should be solid when built solid")
	}
	for _, sh := range merged.Shells() {
		if sh.Body() != merged {
			t.Error("shell not re-parented to the merged body")
		}
	}
}
