// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"bytes"
	"fmt"

	"oblikovati.org/kernel/topo"
)

// DropFaces removes the selected faces (or with keepInstead, every face NOT
// selected) without healing — the openings stay open and the result is a
// surface body. This is the reference TransientBRep.DeleteFaces semantics;
// the healing variant for solids is [DeleteFaces].
//
// Example: open, err := ops.DropFaces(b, [][]byte{face.ReferenceKey()}, false)
func DropFaces(b *topo.Body, faceKeys [][]byte, keepInstead bool) (*topo.Body, error) {
	selected := func(f *topo.Face) bool {
		for _, k := range faceKeys {
			if bytes.Equal(f.ReferenceKey(), k) {
				return true
			}
		}
		return false
	}
	// Keep a face when its selection state matches keepInstead: plain delete
	// keeps the UNselected; keep-instead keeps the selected.
	bld := topo.NewBuilder(false, b.Lineage())
	kept := rebuildKeptFaces(bld, b, func(f *topo.Face) bool { return selected(f) == keepInstead })
	if kept == 0 {
		return nil, fmt.Errorf("ops.DropFaces: selection removes every face of body %d", b.ID())
	}
	return bld.Build(), nil
}

// rebuildKeptFaces clones the surviving faces (shared vertices/edges welded by
// identity) into the builder, returning how many were kept.
func rebuildKeptFaces(bld *topo.Builder, b *topo.Body, keep func(*topo.Face) bool) int {
	verts := map[*topo.Vertex]*topo.Vertex{}
	edges := map[*topo.Edge]*topo.Edge{}
	kept := 0
	for _, f := range b.Faces() {
		if !keep(f) {
			continue
		}
		kept++
		specs := cloneLoopSpecs(bld, f, verts, edges)
		if f.Reversed() {
			bld.AddReversedFace(f.Geometry(), f.Lineage(), specs...)
		} else {
			bld.AddFace(f.Geometry(), f.Lineage(), specs...)
		}
	}
	return kept
}

// cloneLoopSpecs rebuilds a face's loops against cloned (welded) topology.
func cloneLoopSpecs(bld *topo.Builder, f *topo.Face, verts map[*topo.Vertex]*topo.Vertex, edges map[*topo.Edge]*topo.Edge) []topo.LoopSpec {
	cloneV := func(v *topo.Vertex) *topo.Vertex {
		if c, ok := verts[v]; ok {
			return c
		}
		c := bld.AddVertex(v.Point(), v.Lineage())
		verts[v] = c
		return c
	}
	cloneE := func(e *topo.Edge) *topo.Edge {
		if c, ok := edges[e]; ok {
			return c
		}
		c := bld.AddEdge(e.Geometry(), cloneV(e.StartVertex()), cloneV(e.EndVertex()), e.Lineage())
		edges[e] = c
		return c
	}
	var specs []topo.LoopSpec
	for _, l := range f.Loops() {
		uses := make([]topo.Use, 0, len(l.EdgeUses()))
		for _, u := range l.EdgeUses() {
			uses = append(uses, topo.Use{Edge: cloneE(u.Edge()), Reversed: u.Reversed()})
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	return specs
}
