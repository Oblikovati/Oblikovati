// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// Builder assembles a [Body] with fully-wired adjacency. Modeling operations
// (kernel/ops, M07-F03) and feature recompute (M08) construct topology through it;
// tests use it directly. It guarantees the back-pointers (edge↔use↔loop↔face,
// vertex↔edge, face↔shell↔body) stay consistent.
type Builder struct {
	body  *Body
	shell *Shell
}

// NewBuilder starts a body with a single shell. A solid body's shell is closed.
func NewBuilder(solid bool, lineage Lineage) *Builder {
	body := &Body{id: nextID(), solid: solid, lineage: lineage}
	shell := &Shell{id: nextID(), body: body, closed: solid}
	body.shells = []*Shell{shell}
	return &Builder{body: body, shell: shell}
}

// AddVertex adds a vertex at p with the given lineage.
func (bld *Builder) AddVertex(p math.Point3, lineage Lineage) *Vertex {
	return &Vertex{id: nextID(), point: p, lineage: lineage}
}

// AddEdge adds an edge with the given curve and bounding vertices, recording the
// edge on each vertex's incidence list.
func (bld *Builder) AddEdge(curve geom.Curve3, start, end *Vertex, lineage Lineage) *Edge {
	e := &Edge{id: nextID(), curve: curve, start: start, end: end, lineage: lineage}
	start.edges = append(start.edges, e)
	if end != start {
		end.edges = append(end.edges, e)
	}
	return e
}

// Use is an oriented edge use within a loop (reversed = traversed end→start).
type Use struct {
	Edge     *Edge
	Reversed bool
}

// Fwd and Rev are terse Use constructors.
func Fwd(e *Edge) Use { return Use{Edge: e} }
func Rev(e *Edge) Use { return Use{Edge: e, Reversed: true} }

// LoopSpec describes one boundary loop of a face.
type LoopSpec struct {
	Outer bool
	Uses  []Use
}

// OuterLoop and InnerLoop build loop specs of the respective role.
func OuterLoop(uses ...Use) LoopSpec { return LoopSpec{Outer: true, Uses: uses} }
func InnerLoop(uses ...Use) LoopSpec { return LoopSpec{Outer: false, Uses: uses} }

// AddFace adds a face on the given surface bounded by the loop specs, wiring loops,
// edge-uses, and the face↔shell relationship.
func (bld *Builder) AddFace(surface geom.Surface, lineage Lineage, loops ...LoopSpec) *Face {
	f := &Face{id: nextID(), surface: surface, shell: bld.shell, lineage: lineage}
	for _, spec := range loops {
		f.loops = append(f.loops, bld.buildLoop(f, spec))
	}
	bld.shell.faces = append(bld.shell.faces, f)
	return f
}

// AddReversedFace adds a face whose outward (material) side is opposite its surface normal —
// the cut wall a Difference carves (see Face.Reversed). Identical to AddFace but for the sense.
func (bld *Builder) AddReversedFace(surface geom.Surface, lineage Lineage, loops ...LoopSpec) *Face {
	f := bld.AddFace(surface, lineage, loops...)
	f.reversed = true
	return f
}

// buildLoop creates a loop and its edge-uses, linking each use back onto its edge.
func (bld *Builder) buildLoop(f *Face, spec LoopSpec) *Loop {
	loop := &Loop{id: nextID(), face: f, outer: spec.Outer}
	for _, u := range spec.Uses {
		eu := &EdgeUse{edge: u.Edge, loop: loop, reversed: u.Reversed}
		loop.uses = append(loop.uses, eu)
		u.Edge.uses = append(u.Edge.uses, eu)
	}
	return loop
}

// Build returns the assembled body.
func (bld *Builder) Build() *Body { return bld.body }
