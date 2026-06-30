// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// collidingTriBody builds a one-triangle body whose edges ab and bc deliberately share one
// lineage (and therefore one reference key), so a key resolves ambiguously — the topological-
// naming collision ADR-0043's guard must reject.
func collidingTriBody() (body *topo.Body, dupKey, uniqueKey []byte) {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	a := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	b := bld.AddVertex(math.P3(1, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 1)))
	c := bld.AddVertex(math.P3(0, 1, 0), topo.NewLineage(topo.Tok("f", "vertex", 2)))
	dup := topo.NewLineage(topo.Tok("f", "edge", 0))
	ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, dup)
	bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, dup)
	ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, topo.NewLineage(topo.Tok("f", "edge", 1)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(pl, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
	return bld.Build(), ab.ReferenceKey(), ca.ReferenceKey()
}

// TestResolveEdgesRejectsAmbiguousKey is the ADR-0043 resolution guard: a dress-up selection key
// that binds to MORE THAN ONE edge (a naming collision) must fail with a clear error instead of
// silently dressing up the first match — the wrong-rebind that breaks an unintended edge (#1536
// hazard class). A clean key still resolves, and a lost key reports honestly.
func TestResolveEdgesRejectsAmbiguousKey(t *testing.T) {
	body, dupKey, uniqueKey := collidingTriBody()

	if _, _, err := resolveEdges(body, [][]byte{uniqueKey}, nil); err != nil {
		t.Fatalf("unique key should resolve, got %v", err)
	}

	_, _, err := resolveEdges(body, [][]byte{dupKey}, nil)
	if err == nil {
		t.Fatal("ambiguous key resolved without error — the guard let a naming collision through")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error %q should name the ambiguity", err)
	}

	lost := topo.NewLineage(topo.Tok("ghost", "edge", 99))
	_, _, err = resolveEdges(body, [][]byte{appendKindByte(lost)}, nil)
	if err == nil || !strings.Contains(err.Error(), "lost") {
		t.Errorf("a missing key should report 'lost', got %v", err)
	}
}

// appendKindByte renders an edge lineage as a reference key (kind byte 0x02 + lineage string),
// matching topo.referenceKey(KindEdge, …) for a key the body cannot contain.
func appendKindByte(lin topo.Lineage) []byte {
	return append([]byte{0x02}, lin.Key()...)
}

// TestKeyText pins the diagnostic key renderer used in the resolution-guard errors: an empty key
// reads "<empty>", a real key has its leading kind byte stripped, and a key without a kind byte
// passes through unchanged.
func TestKeyText(t *testing.T) {
	if got := keyText(nil); got != "<empty>" {
		t.Errorf("keyText(nil) = %q, want <empty>", got)
	}
	if got := keyText([]byte{0x02, 'a', 'b'}); got != "ab" {
		t.Errorf("keyText(kind+ab) = %q, want ab", got)
	}
	if got := keyText([]byte("plain")); got != "plain" {
		t.Errorf("keyText(plain) = %q, want plain", got)
	}
}
