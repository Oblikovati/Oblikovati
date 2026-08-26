// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// splitFixture builds the canonical "one parent face halved by a single cutting face" naming
// input: a 4×2 parent face on z=0 cut along x=2 by face C, yielding a front (x<2) and back (x>2)
// fragment, each bordered by the SAME imprint segment (the cut line) recorded on the parent. It
// returns the two fragments and the provenance the cut produced. Shared by the naming-correctness
// and the #1578 key-rebuild benchmark so they exercise the identical input.
func splitFixture() (fromFace []subFace, parent topo.Lineage, prov []imprintSeg) {
	parent = topo.NewLineage(topo.Tok("base", "face", 1))
	cut := topo.NewLineage(topo.Tok("tool", "face", 7))
	front := subFace{outer: []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(2, 2, 0), math.P3(0, 2, 0)}}
	back := subFace{outer: []math.Point3{math.P3(2, 0, 0), math.P3(4, 0, 0), math.P3(4, 2, 0), math.P3(2, 2, 0)}}
	// The imprint segment is the cut line x=2 (y from 0..2), recorded ON the parent face, with the
	// cutting face C as `other` — the border both fragments share.
	prov = []imprintSeg{{a: math.P3(2, 0, 0), b: math.P3(2, 2, 0), owner: parent, other: cut}}
	return []subFace{front, back}, parent, prov
}

// TestNameFragmentsNamesBothHalvesByCuttingFace is the characterization test the #1578 hot-path
// refactor must not break: a single straight cut halves a face into two fragments that share the
// one cutting face C as their border, so both are named parent/brep:cut#0/brep:by#0/<C> and the
// dup index disambiguates the otherwise-identical pair (front=0, back=1).
func TestNameFragmentsNamesBothHalvesByCuttingFace(t *testing.T) {
	fromFace, parent, prov := splitFixture()
	nameFragments(fromFace, parent, false, prov)

	want0 := "base:face#1/brep:cut#0/brep:by#0/tool:face#7"
	want1 := want0 + "/brep:frag#1"
	if got := string(fromFace[0].lineage.Key()); got != want0 {
		t.Errorf("front fragment lineage = %q, want %q", got, want0)
	}
	if got := string(fromFace[1].lineage.Key()); got != want1 {
		t.Errorf("back fragment lineage = %q, want %q", got, want1)
	}
}

// TestNameFragmentsSingleSurvivorKeepsParentKey pins the K1a path: a face that survives whole keeps
// its parent lineage (the refactor must leave this branch untouched).
func TestNameFragmentsSingleSurvivorKeepsParentKey(t *testing.T) {
	parent := topo.NewLineage(topo.Tok("base", "face", 1))
	whole := []subFace{{outer: []math.Point3{math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 2, 0), math.P3(0, 2, 0)}}}
	nameFragments(whole, parent, false, nil)
	if got := string(whole[0].lineage.Key()); got != "base:face#1" {
		t.Errorf("single survivor lineage = %q, want base:face#1", got)
	}
}

// BenchmarkNameFragments measures the #1578 hot path: naming the fragments of a cut face. The
// original implementation rebuilt parent.Key()/owner.Key() on every (ring vertex × imprint
// segment), so its cost ballooned with the imprint-set size the outrunner motor produces.
func BenchmarkNameFragments(b *testing.B) {
	fromFace, parent, prov := splitFixture()
	// Pad the provenance with unrelated segments to mimic a dense imprint set (many crossing
	// faces), the regime where the redundant key rebuilds dominated.
	other := topo.NewLineage(topo.Tok("tool", "face", 99))
	for range 200 {
		prov = append(prov, imprintSeg{a: math.P3(0, 5, 0), b: math.P3(1, 5, 0), owner: parent, other: other})
	}
	scratch := make([]subFace, len(fromFace))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(scratch, fromFace)
		nameFragments(scratch, parent, false, prov)
	}
}
