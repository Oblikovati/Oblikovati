// SPDX-License-Identifier: GPL-2.0-only

package topo

import "testing"

// TestNameByParentsIsOrderIndependent pins the defining property (ADR-0043): the provenance name
// depends only on the SET of parents, not the order the generator discovered them in. Swapping the
// parents yields byte-identical keys; the parents' token runs are joined by the separator.
func TestNameByParentsIsOrderIndependent(t *testing.T) {
	a := NewLineage(Tok("Extrusion1", "side", 2))
	b := NewLineage(Tok("Extrusion1", "side", 0))
	sep := Tok("brep", "x", 0)

	ab := NameByParents([]Lineage{a, b}, sep, Tok("brep", "seg", 0), 0)
	ba := NameByParents([]Lineage{b, a}, sep, Tok("brep", "seg", 0), 0)
	if string(ab.Key()) != string(ba.Key()) {
		t.Fatalf("order changed the name: %q vs %q", ab.Key(), ba.Key())
	}
	// b sorts before a (…side#0 < …side#2), so the canonical form is b / brep:x#0 / a.
	if got, want := string(ab.Key()), "Extrusion1:side#0/brep:x#0/Extrusion1:side#2"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
}

// TestNameByParentsRank appends the disambiguator only when rank > 0, carrying the rank seed's
// feature/role with the rank as its index.
func TestNameByParentsRank(t *testing.T) {
	a := NewLineage(Tok("f", "a", 0))
	b := NewLineage(Tok("f", "b", 0))
	sep, seed := Tok("brep", "x", 0), Tok("brep", "seg", 0)

	if got, want := string(NameByParents([]Lineage{a, b}, sep, seed, 0).Key()), "f:a#0/brep:x#0/f:b#0"; got != want {
		t.Errorf("rank 0 name = %q, want %q (no disambiguator)", got, want)
	}
	if got, want := string(NameByParents([]Lineage{a, b}, sep, seed, 3).Key()), "f:a#0/brep:x#0/f:b#0/brep:seg#3"; got != want {
		t.Errorf("rank 3 name = %q, want %q", got, want)
	}
}

// TestNameByParentsThreeParents generalizes past the boolean's two-parent case: a vertex where
// three faces meet is named by all three, canonically ordered and sep-joined.
func TestNameByParentsThreeParents(t *testing.T) {
	p := []Lineage{NewLineage(Tok("f", "c", 0)), NewLineage(Tok("f", "a", 0)), NewLineage(Tok("f", "b", 0))}
	got := string(NameByParents(p, Tok("f", "at", 0), Tok("f", "dup", 0), 0).Key())
	want := "f:a#0/f:at#0/f:b#0/f:at#0/f:c#0"
	if got != want {
		t.Errorf("three-parent name = %q, want %q", got, want)
	}
}
