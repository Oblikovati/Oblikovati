// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/subd"
	"oblikovati.org/math"
)

// provFaces builds a named box and returns its planar faces — the operand form provenanceOf
// consumes. Box face indices follow subd.Box: 1 = top (+Z), 4 = left (−X).
func provFaces(name string, px, py, pz, sx, sy, sz float64) []planarFace {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(math.V3(px, py, pz))
	}
	fs, ok := facesOf(subd.ToBody(m, name))
	if !ok {
		panic("provFaces: box has a non-planar face")
	}
	return fs
}

// TestProvenanceTagsIntersectionEdgeWithBothParents: the slot rim where the tool's −X wall (x=3)
// crosses the base's top (z=4) is an intersection edge; edgeParents must return BOTH generating
// faces — the base top (base:face#1) and the tool left wall (tool:face#4) — canonically ordered.
func TestProvenanceTagsIntersectionEdgeWithBothParents(t *testing.T) {
	base := provFaces("base", 0, 0, 0, 16, 10, 4)
	tool := provFaces("tool", 3, -1, 2, 4, 12, 4) // x∈[3,7], pokes through the top
	prov := provenanceOf(base, tool)

	// The rim edge runs along x=3, z=4 from y=0 to y=10.
	lo, hi, ok := edgeParents(math.P3(3, 0, 4), math.P3(3, 10, 4), prov)
	if !ok {
		t.Fatal("the slot-rim intersection edge resolved to no parent faces")
	}
	got := []string{lo.String(), hi.String()}
	want := []string{"base:face#1", "tool:face#4"} // canonical (sorted) order
	if got[0] != want[0] || got[1] != want[1] {
		t.Errorf("edge parents = %v, want %v (base top × tool −X wall)", got, want)
	}
}

// TestProvenanceParentPairIsOperandOrderIndependent: tagging the same crossing from either face
// yields the same canonical pair, so an edge surviving on the A side or the B side gets one name.
func TestProvenanceParentPairIsOperandOrderIndependent(t *testing.T) {
	base := provFaces("base", 0, 0, 0, 16, 10, 4)
	tool := provFaces("tool", 3, -1, 2, 4, 12, 4)

	loAB, hiAB, okAB := edgeParents(math.P3(3, 0, 4), math.P3(3, 10, 4), provenanceOf(base, tool))
	loBA, hiBA, okBA := edgeParents(math.P3(3, 0, 4), math.P3(3, 10, 4), provenanceOf(tool, base))
	if !okAB || !okBA {
		t.Fatal("rim edge unresolved in one operand order")
	}
	if loAB.String() != loBA.String() || hiAB.String() != hiBA.String() {
		t.Errorf("parent pair depends on operand order: (base,tool)=%v,%v vs (tool,base)=%v,%v",
			loAB.String(), hiAB.String(), loBA.String(), hiBA.String())
	}
}

// TestProvenanceLeavesOriginalBoundaryEdgesUnparented: an edge that is not an intersection edge —
// an original face boundary, lying on no imprint segment — resolves to no parent pair, so F03 can
// fall back to carrying its source lineage rather than mislabelling it.
func TestProvenanceLeavesOriginalBoundaryEdgesUnparented(t *testing.T) {
	base := provFaces("base", 0, 0, 0, 16, 10, 4)
	tool := provFaces("tool", 3, -1, 2, 4, 12, 4)
	prov := provenanceOf(base, tool)

	// A bottom-front boundary edge of the base (z=0, y=0) crosses no imprint.
	if _, _, ok := edgeParents(math.P3(0, 0, 0), math.P3(16, 0, 0), prov); ok {
		t.Error("an original boundary edge was wrongly attributed to an intersection")
	}
}

// TestImprintAllMatchesProvenanceSegments pins that the imprintAll refactor still emits exactly
// the segments provenanceOf tags (same shared pairImprints), so geometry and provenance cannot
// drift: every segment imprintAll records on a face appears, tagged, in the provenance.
func TestImprintAllMatchesProvenanceSegments(t *testing.T) {
	base := provFaces("base", 0, 0, 0, 16, 10, 4)
	tool := provFaces("tool", 3, -1, 2, 4, 12, 4)

	impA, _ := imprintAll(base, tool)
	prov := provenanceOf(base, tool)
	tagged := map[string]bool{}
	for _, s := range prov {
		tagged[segKey(s.a, s.b)] = true
	}
	for i := range impA {
		for _, s := range impA[i] {
			if !tagged[segKey(s[0], s[1])] {
				t.Errorf("imprintAll segment %v–%v on face %d is absent from the provenance", s[0], s[1], i)
			}
		}
	}
}

// segKey is an endpoint-order-independent key for a 3D segment, rounded to the weld grid.
func segKey(a, b math.Point3) string {
	ka, kb := roundKey(a), roundKey(b)
	if ka <= kb {
		return ka + "|" + kb
	}
	return kb + "|" + ka
}
