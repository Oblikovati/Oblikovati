// SPDX-License-Identifier: GPL-2.0-only

package topo

// Transient keys (M07-F07, Oblikovati/Oblikovati#630): every topology entity's
// session id doubles as its transient key — stable within the session, NOT
// persisted (reference keys are the persistent identity). BindTransientKey
// resolves a key back to the entity, the reference
// SurfaceBody.BindTransientKeyToObject.

// TransientRef is the tagged result of a transient-key lookup: Kind says which
// pointer is set.
type TransientRef struct {
	Kind   EntityKind
	Vertex *Vertex
	Edge   *Edge
	Face   *Face
	Shell  *Shell
	Wire   *Wire
}

// BindTransientKey finds the entity with the given session id anywhere in the
// body (vertices, edges, faces, shells, wires, the body itself is excluded —
// the caller already holds it).
//
// Example: ref, ok := body.BindTransientKey(faceID)
func (b *Body) BindTransientKey(key uint64) (TransientRef, bool) {
	for _, f := range b.Faces() {
		if f.ID() == key {
			return TransientRef{Kind: KindFace, Face: f}, true
		}
	}
	for _, e := range b.Edges() {
		if e.ID() == key {
			return TransientRef{Kind: KindEdge, Edge: e}, true
		}
	}
	for _, v := range b.Vertices() {
		if v.ID() == key {
			return TransientRef{Kind: KindVertex, Vertex: v}, true
		}
	}
	return b.bindStructural(key)
}

// bindStructural covers shells and wires.
func (b *Body) bindStructural(key uint64) (TransientRef, bool) {
	for _, s := range b.shells {
		if s.ID() == key {
			return TransientRef{Kind: KindShell, Shell: s}, true
		}
	}
	for _, w := range b.wires {
		if w.ID() == key {
			return TransientRef{Kind: KindWire, Wire: w}, true
		}
	}
	return TransientRef{}, false
}
