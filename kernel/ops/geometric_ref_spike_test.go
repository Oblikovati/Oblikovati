// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// SPIKE (M8): prove a SERIALIZED geometric descriptor — a face's centroid+normal, an
// edge's midpoint+direction — re-finds the entity on a freshly built body, even though it
// is a different object graph with no shared lineage. This is what lets the NX exporter
// bind a fillet/chamfer/shell/hole selection it could only describe geometrically.

const spikeTol = math.Scalar(1e-6)

// TestGeometricFaceRefReboundsAcrossRebuild captures a face descriptor on one box and
// resolves it on an independently built, geometrically identical box.
func TestGeometricFaceRefReboundsAcrossRebuild(t *testing.T) {
	a := opsBoxNamed("A", math.P3(0, 0, 0), 2, 2, 2)
	b := opsBoxNamed("B", math.P3(0, 0, 0), 2, 2, 2) // same geometry, different objects

	for i, f := range a.Faces() {
		ref := topo.DescribeFace(f)

		self, ok := a.FindFaceByGeometry(ref, spikeTol)
		if !ok || self != f {
			t.Fatalf("face %d: self-resolve ok=%v match=%v, want its own face", i, ok, self == f)
		}

		got, ok := b.FindFaceByGeometry(ref, spikeTol)
		if !ok {
			t.Fatalf("face %d: descriptor did not resolve on the rebuilt body", i)
		}
		// The rebuilt face must sit where the descriptor says (same centroid + normal).
		gr := topo.DescribeFace(got)
		if gr.Centroid.DistanceTo(ref.Centroid) > spikeTol || gr.Normal.Dot(ref.Normal) < 0.9999 {
			t.Errorf("face %d: resolved a different face (centroid %v vs %v)", i, gr.Centroid, ref.Centroid)
		}
	}
}

// TestGeometricEdgeRefReboundsAcrossRebuild does the same for edges.
func TestGeometricEdgeRefReboundsAcrossRebuild(t *testing.T) {
	a := opsBoxNamed("A", math.P3(0, 0, 0), 2, 2, 2)
	b := opsBoxNamed("B", math.P3(0, 0, 0), 2, 2, 2)

	for i, e := range a.Edges() {
		ref := topo.DescribeEdge(e)

		if self, ok := a.FindEdgeByGeometry(ref, spikeTol); !ok || self != e {
			t.Fatalf("edge %d: self-resolve failed", i)
		}
		got, ok := b.FindEdgeByGeometry(ref, spikeTol)
		if !ok {
			t.Fatalf("edge %d: descriptor did not resolve on the rebuilt body", i)
		}
		if topo.DescribeEdge(got).Midpoint.DistanceTo(ref.Midpoint) > spikeTol {
			t.Errorf("edge %d: resolved an edge at the wrong midpoint", i)
		}
	}
}

// TestGeometricRefMissIsHonest confirms a descriptor with no match returns false rather
// than binding the nearest unrelated face.
func TestGeometricRefMissIsHonest(t *testing.T) {
	a := opsBoxNamed("A", math.P3(0, 0, 0), 2, 2, 2)
	far := topo.GeometricFaceRef{Centroid: math.P3(100, 100, 100), Normal: math.Vector3{X: 0, Y: 0, Z: 1}}
	if _, ok := a.FindFaceByGeometry(far, spikeTol); ok {
		t.Error("a far-away descriptor must not bind any face")
	}
}
