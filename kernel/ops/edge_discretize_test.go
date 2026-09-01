// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// TestPlanarFaceFollowsCurvedEdge tessellates a half-disk whose curved boundary is an
// arc edge: the boundary must follow the arc (area → πr²/2), not chord straight across
// the diameter of the semicircle (which would halve the area). Regression for the
// tessellator that used loop vertices only (every edge treated as a chord).
func TestPlanarFaceFollowsCurvedEdge(t *testing.T) {
	t.Parallel()
	const r = 2.0
	f := brepfixture.HalfDiskFace(t, r)
	mesh := tessellate.TessellateFace(f, Quality{ChordTolerance: 1e-3})
	if mesh.VertexCount() <= 4 {
		t.Fatalf("arc boundary not subdivided: %d vertices", mesh.VertexCount())
	}
	want := stdmath.Pi * r * r / 2
	if got := mesh.Area(); stdmath.Abs(got-want) > 0.01 {
		t.Errorf("half-disk mesh area = %g, want ≈ %g", got, want)
	}
}

// TestDiscretizeEdgeIsShared checks both directions of the same arc edge produce the
// identical point set (reversed) — the property that keeps shared edges crack-free.
func TestDiscretizeEdgeIsShared(t *testing.T) {
	t.Parallel()
	f := brepfixture.HalfDiskFace(t, 2)
	var arc *topo.Edge
	for _, e := range f.Edges() {
		if _, ok := e.Geometry().(geom.Arc3d); ok {
			arc = e
		}
	}
	if arc == nil {
		t.Fatal("no arc edge on the half-disk")
	}
	fwd := tessellate.DiscretizeEdge(arc, Quality{ChordTolerance: 1e-2})
	rev := probe.ReversedPoints(tessellate.DiscretizeEdge(arc, Quality{ChordTolerance: 1e-2}))
	if len(fwd) != len(rev) {
		t.Fatalf("forward %d vs reversed %d points", len(fwd), len(rev))
	}
	for i := range fwd {
		if fwd[i].DistanceTo(rev[len(rev)-1-i]) > 1e-12 {
			t.Errorf("discretization not symmetric at %d", i)
		}
	}
}
