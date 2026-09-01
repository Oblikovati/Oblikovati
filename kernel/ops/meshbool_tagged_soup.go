// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// The provenance-carrying IN half of the exact-boolean adapter (ADR-0056). Where
// bodyToSoup flattens a body to bare triangles, bodyToTaggedSoup tags every triangle
// with the id of the topo.Face it was meshed from and keeps a side table mapping each
// id to that face's exact analytic surface. Because tessellateBodyFaces meshes one
// submesh PER FACE (its returned faces and meshes are parallel), the tag is exact
// provenance — not a fit — so reconstruction rebuilds each output face on its
// original surface bit-for-bit.

// faceSurfaceRef is the reconstruction's record of one operand face: the exact
// analytic surface it carries and its original sense (a cut wall is Reversed). The
// tagged soup keys into a []faceSurfaceRef by the tag it carries.
type faceSurfaceRef struct {
	face     *topo.Face
	surface  geom.Surface
	reversed bool
}

// bodyToTaggedSoup tessellates body b at quality q into a tagged triangle soup: every
// triangle carries the id (tagBase + local face index) of the face it descends from,
// and the returned refs slice maps local face index i to that face's surface. The
// caller assigns operand A tagBase 0 and operand B tagBase len(aRefs), then indexes
// the concatenated refs by a result triangle's tag.
func bodyToTaggedSoup(b *topo.Body, q Quality, tagBase int) (meshbool.TaggedSoup, []faceSurfaceRef) {
	faces, fm := tessellate.TessellateBodyFaces(b, q)
	merged := &Mesh{}
	var triTags []int
	for i, m := range fm {
		before := merged.TriangleCount()
		tessellate.MergeMesh(merged, m)
		for t := before; t < merged.TriangleCount(); t++ {
			triTags = append(triTags, tagBase+i)
		}
	}
	refs := make([]faceSurfaceRef, len(faces))
	for i, f := range faces {
		refs[i] = faceSurfaceRef{face: f, surface: f.Geometry(), reversed: f.Reversed()}
	}
	return taggedSoupFromMesh(merged, triTags), refs
}

// taggedSoupFromMesh is soupFromMesh carrying a per-triangle tag: it welds positions
// (correcting per-face tessellation float-noise into a watertight input) and drops
// welded-degenerate slivers, keeping each surviving triangle's tag in lockstep. The
// tag of a dropped sliver is dropped with it, so len(Tags) == len(Tris) still holds.
func taggedSoupFromMesh(m *Mesh, triTags []int) meshbool.TaggedSoup {
	pos := weldPositions(m.Positions)
	s := meshbool.TaggedSoup{Tris: make([][3]meshbool.Point, 0, m.TriangleCount())}
	for t := 0; t < m.TriangleCount(); t++ {
		i, j, k := m.Indices[3*t], m.Indices[3*t+1], m.Indices[3*t+2]
		tri := [3]meshbool.Point{meshbool.FromPoint3(pos[i]), meshbool.FromPoint3(pos[j]), meshbool.FromPoint3(pos[k])}
		if tri[0].Equal(tri[1]) || tri[1].Equal(tri[2]) || tri[2].Equal(tri[0]) {
			continue // welded to a degenerate sliver — drop it and its tag
		}
		s.Tris = append(s.Tris, tri)
		s.Tags = append(s.Tags, triTags[t])
	}
	return s
}
