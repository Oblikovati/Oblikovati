// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// halfDiskFace builds a planar (XY) half-disk face of radius r: a CCW outer loop of
// the semicircle arc (angle 0→π through the top) plus the diameter segment back.
func halfDiskFace(t *testing.T, r float64) *topo.Face {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "halfdisk", 0))
	bld := topo.NewBuilder(false, lin)
	a := math.P3(r, 0, 0)  // angle 0
	b := math.P3(-r, 0, 0) // angle π
	va := bld.AddVertex(a, lin)
	vb := bld.AddVertex(b, lin)
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r, 0, stdmath.Pi)
	if err != nil {
		t.Fatal(err)
	}
	eArc := bld.AddEdge(arc, va, vb, lin)                       // A→B along the top arc
	eSeg := bld.AddEdge(geom.NewLineSegment(b, a), vb, va, lin) // B→A along the diameter
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	bld.AddFace(plane, lin,
		topo.OuterLoop(topo.Use{Edge: eArc}, topo.Use{Edge: eSeg}))
	return bld.Build().Faces()[0]
}

// meshArea sums the triangle areas of a mesh.
func meshArea(m *Mesh) float64 {
	var sum float64
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a, b, c := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		sum += a.VectorTo(b).Cross(a.VectorTo(c)).Length() / 2
	}
	return sum
}

// TestPlanarFaceFollowsCurvedEdge tessellates a half-disk whose curved boundary is an
// arc edge: the boundary must follow the arc (area → πr²/2), not chord straight across
// the diameter of the semicircle (which would halve the area). Regression for the
// tessellator that used loop vertices only (every edge treated as a chord).
func TestPlanarFaceFollowsCurvedEdge(t *testing.T) {
	t.Parallel()
	const r = 2.0
	f := halfDiskFace(t, r)
	mesh := TessellateFace(f, Quality{ChordTolerance: 1e-3})
	if mesh.VertexCount() <= 4 {
		t.Fatalf("arc boundary not subdivided: %d vertices", mesh.VertexCount())
	}
	want := stdmath.Pi * r * r / 2
	if got := meshArea(mesh); stdmath.Abs(got-want) > 0.01 {
		t.Errorf("half-disk mesh area = %g, want ≈ %g", got, want)
	}
}

// TestDiscretizeEdgeIsShared checks both directions of the same arc edge produce the
// identical point set (reversed) — the property that keeps shared edges crack-free.
func TestDiscretizeEdgeIsShared(t *testing.T) {
	t.Parallel()
	f := halfDiskFace(t, 2)
	var arc *topo.Edge
	for _, e := range f.Edges() {
		if _, ok := e.Geometry().(geom.Arc3d); ok {
			arc = e
		}
	}
	if arc == nil {
		t.Fatal("no arc edge on the half-disk")
	}
	fwd := discretizeEdge(arc, Quality{ChordTolerance: 1e-2})
	rev := probe.ReversedPoints(discretizeEdge(arc, Quality{ChordTolerance: 1e-2}))
	if len(fwd) != len(rev) {
		t.Fatalf("forward %d vs reversed %d points", len(fwd), len(rev))
	}
	for i := range fwd {
		if fwd[i].DistanceTo(rev[len(rev)-1-i]) > 1e-12 {
			t.Errorf("discretization not symmetric at %d", i)
		}
	}
}
