//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/head/viewport"
	"oblikovati.org/math"
)

// triMesh is a fake flattened source with `verts` triangle vertices and one triangle's indices,
// enough to exercise the instanceBuilder's stream concatenation and offset math without a GPU.
func triMesh(verts int) viewport.Mesh {
	m := viewport.Mesh{TriVerts: make([]float32, verts*16), TriIndices: []uint32{0, 1, 2}, TriVCount: verts}
	m.TriBiasFirst = len(m.TriIndices) // all opaque (a body has no depth-biased fills)
	return m
}

// TestInstanceBuilderOffsets pins ADR-0038's record math: two groups of one source each must
// concatenate into one tri stream, with each group's record pointing (via vertexOffset + its
// 0-based indices) back into ITS OWN appended vertex block, and the instance counts/first-instances
// matching the transforms.
func TestInstanceBuilderOffsets(t *testing.T) {
	var b instanceBuilder
	b.addGroup(triMesh(3), []math.Matrix4{math.Identity4(), math.Translation4(math.V3(5, 0, 0))}) // 2 instances
	b.addGroup(triMesh(3), []math.Matrix4{math.Translation4(math.V3(0, 9, 0))})                   // 1 instance

	m, mats, recs := b.finish()
	if got := m.TriVCount; got != 6 {
		t.Fatalf("merged tri vertex count = %d, want 6 (3+3)", got)
	}
	if got := len(mats) / 16; got != 3 {
		t.Fatalf("instance matrices = %d, want 3 (2+1)", got)
	}
	if len(recs) != 2*7 {
		t.Fatalf("records = %d ints, want %d (two tri records)", len(recs), 2*7)
	}
	// Record A: tri stream, firstInstance 0, instanceCount 2, vertexOffset 0.
	a := recs[0:7]
	if a[0] != 1 || a[4] != 0 || a[5] != 2 || a[3] != 0 {
		t.Errorf("group A record = %v, want stream 1, firstInstance 0, instanceCount 2, vertexOffset 0", a)
	}
	// Record B: firstInstance 2 (after A's two), instanceCount 1, vertexOffset 3 (after A's 3 verts).
	bRec := recs[7:14]
	if bRec[4] != 2 || bRec[5] != 1 || bRec[3] != 3 {
		t.Errorf("group B record = %v, want firstInstance 2, instanceCount 1, vertexOffset 3", bRec)
	}
}

// TestInstanceBuilderStreamBase checks the absolute base when an earlier stream is non-empty: a tri
// record's vertexOffset must skip past the occluder stream's vertices in the concatenated buffer.
func TestInstanceBuilderStreamBase(t *testing.T) {
	var b instanceBuilder
	occ := viewport.Mesh{OccVerts: make([]float32, 4*16), OccIndices: []uint32{0, 1, 2}, OccVCount: 4}
	b.addGroup(occ, []math.Matrix4{math.Identity4()})
	b.addGroup(triMesh(3), []math.Matrix4{math.Identity4()})

	_, _, recs := b.finish()
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
