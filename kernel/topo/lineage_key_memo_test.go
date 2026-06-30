// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"
)

// TestKeyStringMatchesKey pins that the memoized KeyString and the byte Key agree and produce the
// exact serialization reference keys are persisted with — the encoding must not drift (#1578 only
// memoized it, it must not change it).
func TestKeyStringMatchesKey(t *testing.T) {
	l := NewLineage(Tok("base", "face", 1), Tok("brep", "by", 0), Tok("tool", "face", 7))
	const want = "base:face#1/brep:by#0/tool:face#7"
	if got := l.KeyString(); got != want {
		t.Errorf("KeyString = %q, want %q", got, want)
	}
	if got := string(l.Key()); got != want {
		t.Errorf("Key = %q, want %q", got, want)
	}
}

// TestZeroValueLineageKeyIsEmpty pins the sentinel: a zero-value Lineage (the not-found return of
// edgeParents) serializes to the empty key without panicking.
func TestZeroValueLineageKeyIsEmpty(t *testing.T) {
	var zero Lineage
	if got := zero.KeyString(); got != "" {
		t.Errorf("zero-value KeyString = %q, want empty", got)
	}
	if got := string(zero.Key()); got != "" {
		t.Errorf("zero-value Key = %q, want empty", got)
	}
}

// TestValueCopySharesMemoizedKey is the load-bearing #1578 property: copying a lineage by value (as
// imprintSeg does, holding owner/other lineages copied into every segment) carries the memoized key,
// so KeyString on the copy is correct without re-serializing.
func TestValueCopySharesMemoizedKey(t *testing.T) {
	orig := NewLineage(Tok("base", "face", 2), Tok("brep", "cut", 0))
	cp := orig // value copy, as in appendTagged's `owner: owner.lineage`
	if cp.KeyString() != orig.KeyString() {
		t.Errorf("value copy KeyString = %q, want %q", cp.KeyString(), orig.KeyString())
	}
	if cp.keyStr == "" {
		t.Error("value copy lost the memoized key (would force a re-serialize per call)")
	}
}

// TestNameByParentsMemoizesKey pins that the second constructor path also memoizes, so boolean edge/
// vertex names built via NameByParents are not re-serialized on every comparison either.
func TestNameByParentsMemoizesKey(t *testing.T) {
	a := NewLineage(Tok("a", "face", 0))
	b := NewLineage(Tok("b", "face", 0))
	got := NameByParents([]Lineage{b, a}, Tok("brep", "x", 0), Tok("brep", "seg", 0), 0)
	if got.keyStr == "" {
		t.Error("NameByParents result is not memoized")
	}
	const want = "a:face#0/brep:x#0/b:face#0" // sorted by key, so a precedes b
	if got.KeyString() != want {
		t.Errorf("NameByParents key = %q, want %q", got.KeyString(), want)
	}
}
