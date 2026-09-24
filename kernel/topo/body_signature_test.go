// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// squareFaceBody builds one square planar face as a body — enough topology to have edges, uses and
// loops, and small enough to reason about.
func squareFaceBody(t *testing.T, solid bool) *Body {
	t.Helper()
	lin := NewLineage(Tok("test", "sig", 0))
	bld := NewBuilder(solid, lin)
	corners := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)}
	verts := make([]*Vertex, len(corners))
	for i, p := range corners {
		verts[i] = bld.AddVertex(p, lin)
	}
	uses := make([]Use, len(corners))
	for i := range corners {
		j := (i + 1) % len(corners)
		uses[i] = Fwd(bld.AddEdge(geom.NewLineSegment(corners[i], corners[j]), verts[i], verts[j], lin))
	}
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	bld.AddFace(plane, lin, OuterLoop(uses...))
	return bld.Build()
}

// The signature counts what a topological verdict reads, and repeats itself: two readings of an
// untouched body must be identical, or every caller would see a change that never happened.
func TestTopologySignatureIsStableAndCountsTheTopology(t *testing.T) {
	t.Parallel()
	b := squareFaceBody(t, true)
	sig := b.TopologySignature()
	if sig != b.TopologySignature() {
		t.Error("two readings of an untouched body must agree")
	}
	want := TopologySignature{Solid: true, Vertices: 4, Edges: 4, Faces: 1, Loops: 1, EdgeUses: sig.EdgeUses}
	if sig != want {
		t.Errorf("TopologySignature(one square face) = %+v; want %+v", sig, want)
	}
	if sig.EdgeUses == fnvOffset64 {
		t.Error("the edge-use fold must actually fold the four edges")
	}
}

// The signature has to SEE the mutation that pointer identity misses: a builder that reuses an
// existing edge appends a use to it, leaving the body it came from with an edge used by two faces at
// the same address it always had.
func TestTopologySignatureSeesAStolenEdge(t *testing.T) {
	t.Parallel()
	b := squareFaceBody(t, true)
	before := b.TopologySignature()

	lin := NewLineage(Tok("test", "steal", 0))
	bld := NewBuilder(false, lin)
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	bld.AddFace(plane, lin, OuterLoop(Fwd(b.Edges()[0])))
	bld.Build()

	if after := b.TopologySignature(); after == before {
		t.Errorf("the body's edge is now used twice over; its signature must change (%+v)", after)
	}
}

// A surface body and a solid one with the same faces are not the same body to a validity verdict, so
// they must not be the same signature either.
func TestTopologySignatureSeparatesSolidFromSheet(t *testing.T) {
	t.Parallel()
	if squareFaceBody(t, true).TopologySignature() == squareFaceBody(t, false).TopologySignature() {
		t.Error("a solid and a sheet with identical topology must not share a signature")
	}
}
