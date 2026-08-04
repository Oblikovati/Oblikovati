// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
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

// ReplaceEdgeCurve swaps the geometry of an edge still under construction. It exists for the one
// case where an edge's best geometry is not known when the edge is created: a shared edge welded
// from the first of its two faces, where that face had no curve to offer and the second one does
// (ops.edgeCatalog). A nil offer is an absence of information, not an assertion of straightness, so
// the later curve replaces the straight chord — oriented by the caller to the edge's own start→end
// sense, which this method cannot know.
//
// Example:
//
//	bld.ReplaceEdgeCurve(e, geom.ReverseCurve3(offered)) // the second face traverses e backwards
//
// Legal ONLY before [Builder.Build]: afterwards a body's geometry is immutable and its derived
// caches (RangeBox, key indices, tessellation memos) are already memoized against the old curve.
// A nil curve is rejected — dropping geometry is never an improvement, and the caller that has
// nothing to offer must simply not call.
func (bld *Builder) ReplaceEdgeCurve(e *Edge, curve geom.Curve3) {
	if e == nil || curve == nil {
		panic(fmt.Sprintf("topo: ReplaceEdgeCurve(edge=%v, curve=%v): both must be non-nil", e, curve))
	}
	e.curve = curve
	// A healed polyline (SetSnappedCurve, import healing M25) is a discretization of the OLD curve
	// and OUTRANKS e.curve in tessellation, so a stale one would silently ship the replaced
	// geometry. Unreachable today (nothing snaps an edge before Build), cleared defensively with
	// its residual so the invariant "snapped describes curve" survives any future caller
	// (adversarial-review finding m-8).
	e.snapped, e.tolerance = nil, 0
}

// Diagnose records d on the body under construction, so what the assembler could not do ideally
// travels WITH the body it produced instead of being swallowed (see [Body.BuildDiagnostics]).
// Legal only before [Builder.Build].
func (bld *Builder) Diagnose(d diag.Diagnostic) { bld.body.buildDiags = append(bld.body.buildDiags, d) }

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

// Build returns the assembled body, with its shells regrouped to true face
// connectivity (M07-F06, #629) — a builder receives faces in arbitrary order,
// so the single working shell may actually hold several disjoint groups (a
// stitched pair of quilts, a cavity cut's inner skin).
func (bld *Builder) Build() *Body {
	RegroupShells(bld.body)
	return bld.body
}
