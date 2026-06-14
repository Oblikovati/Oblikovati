// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"bytes"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// linedTetra builds the unit tetrahedron with every entity's lineage produced by lin —
// used to build both a bare "component" body and an assembly-"placed" body whose lineages
// carry an extra occurrence prefix.
func linedTetra(lin func(role string, i int) Lineage) *Body {
	bld := NewBuilder(true, lin("body", 0))
	a := bld.AddVertex(math.P3(0, 0, 0), lin("vertex", 0))
	b := bld.AddVertex(math.P3(1, 0, 0), lin("vertex", 1))
	c := bld.AddVertex(math.P3(0, 1, 0), lin("vertex", 2))
	d := bld.AddVertex(math.P3(0, 0, 1), lin("vertex", 3))
	seg := func(p, q *Vertex) geom.LineSegment { return geom.NewLineSegment(p.Point(), q.Point()) }
	edge := func(p, q *Vertex, i int) *Edge { return bld.AddEdge(seg(p, q), p, q, lin("edge", i)) }
	ab, ac, ad := edge(a, b, 0), edge(a, c, 1), edge(a, d, 2)
	bc, bd, cd := edge(b, c, 3), edge(b, d, 4), edge(c, d, 5)
	plane := func(o, n math.Vector3) geom.Surface {
		p, _ := geom.NewPlane(o.AsPoint(), n)
		return p
	}
	bld.AddFace(plane(math.V3(0, 0, 0), math.V3(0, 0, 1)), lin("face", 0), OuterLoop(Fwd(ab), Fwd(bc), Rev(ac)))
	bld.AddFace(plane(math.V3(0, 0, 0), math.V3(0, 1, 0)), lin("face", 1), OuterLoop(Fwd(ab), Fwd(bd), Rev(ad)))
	bld.AddFace(plane(math.V3(0, 0, 0), math.V3(1, 0, 0)), lin("face", 2), OuterLoop(Fwd(ac), Fwd(cd), Rev(ad)))
	bld.AddFace(plane(math.V3(1, 1, 1), math.V3(1, 1, 1)), lin("face", 3), OuterLoop(Fwd(bc), Fwd(cd), Rev(bd)))
	return bld.Build()
}

func componentLineage(role string, i int) Lineage { return NewLineage(Tok("src", role, i)) }

func placedLineage(occ int) func(string, int) Lineage {
	return func(role string, i int) Lineage {
		return NewLineage(Tok("assemblyFeature", "occ", occ), Tok("src", role, i))
	}
}

// TestEdgeReferenceKeysWithLineageSuffix resolves a component-local edge key against a
// placed body (its lineage prefixed by an occurrence token), recovering the placed edge's
// full key regardless of the occurrence index — the occurrence-relative resolution the
// assembly dress-up features rely on (#735).
func TestEdgeReferenceKeysWithLineageSuffix(t *testing.T) {
	component := linedTetra(componentLineage)
	// The client picks an edge on the component; LineageSuffixOf yields the match suffix.
	componentEdgeKey := edgeKeyForRole(t, component, 3)
	suffix := LineageSuffixOf(componentEdgeKey)

	for _, occ := range []int{0, 7} {
		placed := linedTetra(placedLineage(occ))
		keys := placed.EdgeReferenceKeysWithLineageSuffix(suffix)
		if len(keys) != 1 {
			t.Fatalf("occ %d: resolved %d edges, want 1", occ, len(keys))
		}
		want := edgeKeyForRole(t, placed, 3) // the placed edge#3's full (prefixed) key
		if !bytes.Equal(keys[0], want) {
			t.Errorf("occ %d: resolved key %q, want the placed edge's full key %q", occ, keys[0], want)
		}
	}

	// A suffix for an edge index that does not exist resolves nothing.
	missing := LineageSuffixOf(referenceKey(KindEdge, componentLineage("edge", 99)))
	if got := linedTetra(placedLineage(0)).EdgeReferenceKeysWithLineageSuffix(missing); len(got) != 0 {
		t.Errorf("non-existent edge suffix resolved %d keys, want 0", len(got))
	}
}

// TestFaceReferenceKeysWithLineageSuffix is the face twin: a component face key resolves to
// the placed body's prefixed face.
func TestFaceReferenceKeysWithLineageSuffix(t *testing.T) {
	suffix := LineageSuffixOf(referenceKey(KindFace, componentLineage("face", 2)))
	placed := linedTetra(placedLineage(3))
	keys := placed.FaceReferenceKeysWithLineageSuffix(suffix)
	if len(keys) != 1 {
		t.Fatalf("resolved %d faces, want 1", len(keys))
	}
	if _, ok := placed.FindFaceByKey(keys[0]); !ok {
		t.Error("resolved face key does not bind back to a placed face")
	}
}

// TestLineageSuffixBoundary checks a partial-token suffix never matches across a token
// boundary: "edge#0" (no role/feature) must not match the full lineage "src:edge#0".
func TestLineageSuffixBoundary(t *testing.T) {
	placed := linedTetra(placedLineage(0))
	if got := placed.EdgeReferenceKeysWithLineageSuffix([]byte("edge#0")); len(got) != 0 {
		t.Errorf("partial-token suffix matched %d edges, want 0 (boundary guard)", len(got))
	}
	if got := placed.EdgeReferenceKeysWithLineageSuffix(nil); got != nil {
		t.Error("empty suffix should match nothing")
	}
}

// edgeKeyForRole returns the reference key of the body's edge whose lineage ends with
// edge#i (the i-th source edge), failing if absent.
func edgeKeyForRole(t *testing.T, b *Body, i int) []byte {
	t.Helper()
	suffix := LineageSuffixOf(referenceKey(KindEdge, componentLineage("edge", i)))
	for _, e := range b.Edges() {
		if lineageKeyHasSuffix(e.Lineage().Key(), suffix) {
			return e.ReferenceKey()
		}
	}
	t.Fatalf("no edge with source index %d", i)
	return nil
}
