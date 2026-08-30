// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestFaceResolvesByAliasKey pins ADR-0057 multi-parent identity: a face that carries a merged
// parent's reference key as an alias resolves from BOTH its own key and the alias key, so a pick on
// either coplanar parent survives the merge. It also verifies AddAliasKey never lists the face twice
// under one key (a self-alias or a repeat is a no-op).
func TestFaceResolvesByAliasKey(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	a := bld.AddVertex(math.P3(0, 0, 0), NewLineage(Tok("f", "vertex", 0)))
	b := bld.AddVertex(math.P3(1, 0, 0), NewLineage(Tok("f", "vertex", 1)))
	c := bld.AddVertex(math.P3(0, 1, 0), NewLineage(Tok("f", "vertex", 2)))
	ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, NewLineage(Tok("f", "edge", 0)))
	bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, NewLineage(Tok("f", "edge", 1)))
	ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, NewLineage(Tok("f", "edge", 2)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	face := bld.AddFace(pl, NewLineage(Tok("f", "face", 0)), OuterLoop(Fwd(ab), Fwd(bc), Fwd(ca)))

	// The reference key a merged coplanar PARENT face would have carried.
	parentKey := referenceKey(KindFace, NewLineage(Tok("f", "parentface", 7)))
	face.AddAliasKey(parentKey)
	face.AddAliasKey(parentKey)           // repeat: must not double-list
	face.AddAliasKey(face.ReferenceKey()) // self-alias: must be a no-op

	body := bld.Build()

	if got := body.FacesByKey(face.ReferenceKey()); len(got) != 1 || got[0] != face {
		t.Fatalf("primary key resolved %d faces, want the one face", len(got))
	}
	if got := body.FacesByKey(parentKey); len(got) != 1 || got[0] != face {
		t.Fatalf("alias (merged-parent) key resolved %d faces, want the merged face", len(got))
	}
	if got := len(face.AliasKeys()); got != 1 {
		t.Fatalf("face has %d alias keys, want 1 (repeat and self-alias were no-ops)", got)
	}
}
