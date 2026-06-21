//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/head/viewport"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// triMesh is a fake flattened source with `verts` triangle vertices and one triangle's indices,
// enough to exercise the instanceBuilder's stream concatenation and offset math without a GPU.
func triMesh(verts int) viewport.Mesh {
	m := viewport.Mesh{TriVerts: make([]float32, verts*16), TriIndices: []uint32{0, 1, 2}, TriVCount: verts}
	m.TriBiasFirst = len(m.TriIndices) // all opaque (a body has no depth-biased fills)
	return m
}

// TestInstanceBuilderOffsets pins ADR-0038's record math under the F1b atlas/assemble split: two
// source regions concatenate into one tri stream (the retained atlas), and the per-frame assemble
// emits each region's record pointing (via vertexOffset + its 0-based indices) back into ITS OWN
// vertex block, with instance counts/first-instances matching that frame's visible transforms.
func TestInstanceBuilderOffsets(t *testing.T) {
	srcA, srcB := &topo.Body{}, &topo.Body{}
	var b instanceBuilder
	b.addSource(srcA, triMesh(3))
	b.addSource(srcB, triMesh(3))
	atlas := b.finishAtlas("k")
	if got := atlas.mesh.TriVCount; got != 6 {
		t.Fatalf("merged tri vertex count = %d, want 6 (3+3)", got)
	}
	mats, recs := atlas.assemble(map[*topo.Body][]math.Matrix4{
		srcA: {math.Identity4(), math.Translation4(math.V3(5, 0, 0))}, // 2 instances
		srcB: {math.Translation4(math.V3(0, 9, 0))},                   // 1 instance
	})
	if got := len(mats) / 16; got != 3 {
		t.Fatalf("instance matrices = %d, want 3 (2+1)", got)
	}
	if len(recs) != 2*7 {
		t.Fatalf("records = %d ints, want %d (two tri records)", len(recs), 2*7)
	}
	// Record A: tri stream, firstInstance 0, instanceCount 2, vertexOffset 0.
	a := recs[0:7]
	if a[0] != 1 || a[4] != 0 || a[5] != 2 || a[3] != 0 {
		t.Errorf("region A record = %v, want stream 1, firstInstance 0, instanceCount 2, vertexOffset 0", a)
	}
	// Record B: firstInstance 2 (after A's two), instanceCount 1, vertexOffset 3 (after A's 3 verts).
	bRec := recs[7:14]
	if bRec[4] != 2 || bRec[5] != 1 || bRec[3] != 3 {
		t.Errorf("region B record = %v, want firstInstance 2, instanceCount 1, vertexOffset 3", bRec)
	}
}

// TestInstanceBuilderStreamBase checks the absolute base when an earlier stream is non-empty: a tri
// record's vertexOffset must skip past the occluder stream's vertices in the concatenated atlas.
func TestInstanceBuilderStreamBase(t *testing.T) {
	srcOcc, srcTri := &topo.Body{}, &topo.Body{}
	var b instanceBuilder
	occ := viewport.Mesh{OccVerts: make([]float32, 4*16), OccIndices: []uint32{0, 1, 2}, OccVCount: 4}
	b.addSource(srcOcc, occ)
	b.addSource(srcTri, triMesh(3))
	atlas := b.finishAtlas("k")

	_, recs := atlas.assemble(map[*topo.Body][]math.Matrix4{srcOcc: {math.Identity4()}, srcTri: {math.Identity4()}})
	// Streams concatenate occ then tri, so the tri record's vertexOffset starts after occ's 4 verts.
	var tri []int32
	for i := 0; i < len(recs); i += 7 {
		if recs[i] == 1 {
			tri = recs[i : i+7]
		}
	}
	if tri == nil {
		t.Fatal("no tri record emitted")
	}
	if tri[3] != 4 { // vertexOffset
		t.Errorf("tri record vertexOffset = %d, want 4 (after the occluder stream's 4 verts)", tri[3])
	}
	if tri[1] != 3 { // firstIndex: after occ's 3 indices
		t.Errorf("tri record firstIndex = %d, want 3 (after the occluder stream's 3 indices)", tri[1])
	}
}

// TestFrameAtlasCullsHiddenInstances pins the F1b/F1 interaction: the atlas holds both sources'
// geometry, but assemble emits records only for the sources visible this frame — a culled source
// contributes no draw, while the atlas (vertex data) is unchanged.
func TestFrameAtlasCullsHiddenInstances(t *testing.T) {
	srcA, srcB := &topo.Body{}, &topo.Body{}
	var b instanceBuilder
	b.addSource(srcA, triMesh(3))
	b.addSource(srcB, triMesh(3))
	atlas := b.finishAtlas("k")
	// Only srcA is visible this frame.
	mats, recs := atlas.assemble(map[*topo.Body][]math.Matrix4{srcA: {math.Identity4()}})
	if got := len(mats) / 16; got != 1 {
		t.Fatalf("instances = %d, want 1 (srcB culled)", got)
	}
	if len(recs) != 7 {
		t.Fatalf("records = %d ints, want 7 (only srcA drawn)", len(recs))
	}
	if atlas.mesh.TriVCount != 6 {
		t.Errorf("atlas still holds both sources' verts = %d, want 6", atlas.mesh.TriVCount)
	}
}
