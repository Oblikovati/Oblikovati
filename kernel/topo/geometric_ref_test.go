// SPDX-License-Identifier: GPL-2.0-only

package topo_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// SPIKE (M8): prove a SERIALIZED geometric descriptor — a face's centroid+normal, an
// edge's midpoint+direction — re-finds the entity on a freshly built body, even though it
// is a different object graph with no shared lineage. This is what lets the NX exporter
// bind a fillet/chamfer/shell/hole selection it could only describe geometrically.

const spikeTol = math.Scalar(1e-6)

// box builds a solid box with its near corner translated to p.
func box(p math.Point3, sx, sy, sz float64) *topo.Body {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(p.AsVector())
	}
	return subd.ToBody(m, "box")
}

// TestGeometricFaceRefReboundsAcrossRebuild captures a face descriptor on one box and
// resolves it on an independently built, geometrically identical box.
func TestGeometricFaceRefReboundsAcrossRebuild(t *testing.T) {
	a := box(math.P3(0, 0, 0), 2, 2, 2)
	b := box(math.P3(0, 0, 0), 2, 2, 2) // same geometry, different objects

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
		gr := topo.DescribeFace(got)
		if gr.Centroid.DistanceTo(ref.Centroid) > spikeTol || gr.Normal.Dot(ref.Normal) < 0.9999 {
			t.Errorf("face %d: resolved a different face (centroid %v vs %v)", i, gr.Centroid, ref.Centroid)
		}
	}
}

// TestGeometricEdgeRefReboundsAcrossRebuild does the same for edges.
func TestGeometricEdgeRefReboundsAcrossRebuild(t *testing.T) {
	a := box(math.P3(0, 0, 0), 2, 2, 2)
	b := box(math.P3(0, 0, 0), 2, 2, 2)

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

// TestClosedCircularEdgeDescribedByCentreAndAxis captures a bore/boss rim — a closed
// circular edge whose start and end are the same vertex, so the chord midpoint/direction
// degenerate — and confirms it is described by the circle centre + axis and re-binds on a
// geometrically identical rebuilt edge. This is what lets a fillet/chamfer on a circular
// edge (the case DressUpExtractor emits) resolve at recompute.
func TestClosedCircularEdgeDescribedByCentreAndAxis(t *testing.T) {
	// A cylinder's cap rim is a closed circular edge: start == end vertex, so the chord
	// midpoint/direction degenerate — it must be described by the circle centre + axis.
	cyl := func() *topo.Body {
		b, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
		if err != nil {
			t.Fatalf("cylinder: %v", err)
		}
		return b
	}
	body := cyl()

	var rim *topo.Edge
	for _, e := range body.Edges() {
		if _, ok := e.Geometry().(geom.Circle); ok {
			rim = e
			break
		}
	}
	if rim == nil {
		t.Fatal("setup: cylinder had no circular edge")
	}

	ref := topo.DescribeEdge(rim)
	if ref.Midpoint.DistanceTo(math.P3(0, 0, 0)) > spikeTol &&
		ref.Midpoint.DistanceTo(math.P3(0, 0, 4)) > spikeTol {
		t.Errorf("circle rim midpoint = %v, want a cap centre (0,0,0) or (0,0,4)", ref.Midpoint)
	}
	if stdmath.Abs(ref.Direction.Dot(math.V3(0, 0, 1))) < 0.9999 {
		t.Errorf("circle rim direction = %v, want the cylinder axis (0,0,1)", ref.Direction)
	}

	if self, ok := body.FindEdgeByGeometry(ref, spikeTol); !ok || self != rim {
		t.Fatalf("circular edge did not self-resolve (ok=%v)", ok)
	}
	if _, ok := cyl().FindEdgeByGeometry(ref, spikeTol); !ok {
		t.Error("closed circular edge descriptor did not re-bind on the rebuilt cylinder")
	}
}

// TestGeometricRefMissIsHonest confirms a descriptor with no match returns false rather
// than binding the nearest unrelated face.
func TestGeometricRefMissIsHonest(t *testing.T) {
	a := box(math.P3(0, 0, 0), 2, 2, 2)
	far := topo.GeometricFaceRef{Centroid: math.P3(100, 100, 100), Normal: math.Vector3{X: 0, Y: 0, Z: 1}}
	if _, ok := a.FindFaceByGeometry(far, spikeTol); ok {
		t.Error("a far-away descriptor must not bind any face")
	}
}

// TestGeometricEdgeRefDirectionIsSignAgnostic confirms an edge resolves even when the
// descriptor's direction points the opposite way (an edge has no inherent sense).
func TestGeometricEdgeRefDirectionIsSignAgnostic(t *testing.T) {
	a := box(math.P3(0, 0, 0), 2, 2, 2)
	e := a.Edges()[0]
	ref := topo.DescribeEdge(e)
	ref.Direction = ref.Direction.Scale(-1) // flip it

	if got, ok := a.FindEdgeByGeometry(ref, spikeTol); !ok || got != e {
		t.Errorf("reversed-direction descriptor did not resolve its edge (ok=%v)", ok)
	}
}

// TestGeometricFaceRefMatchesWithoutNormal confirms a zero normal skips the normal filter
// and binds purely by centroid nearness (each box face centroid is unique).
func TestGeometricFaceRefMatchesWithoutNormal(t *testing.T) {
	a := box(math.P3(0, 0, 0), 2, 2, 2)
	f := a.Faces()[0]
	ref := topo.GeometricFaceRef{Centroid: topo.DescribeFace(f).Centroid} // no normal

	if got, ok := a.FindFaceByGeometry(ref, spikeTol); !ok || got != f {
		t.Errorf("centroid-only descriptor did not resolve its face (ok=%v)", ok)
	}
}

// TestGeometricFaceRefAmbiguousTieIsLost builds a body with two same-normal faces
// equidistant from a descriptor and confirms the resolver refuses to guess.
func TestGeometricFaceRefAmbiguousTieIsLost(t *testing.T) {
	a := box(math.P3(0, 0, 0), 2, 2, 2)
	b := box(math.P3(6, 0, 0), 2, 2, 2) // disjoint, same orientation
	u, err := ops.Boolean(ops.Join, a, b)
	if err != nil {
		t.Fatalf("union: %v", err)
	}

	up := math.Vector3{X: 0, Y: 0, Z: 1}
	var tops []math.Point3
	for _, f := range u.Faces() {
		if topo.DescribeFace(f).Normal.Dot(up) > 0.99 {
			tops = append(tops, topo.DescribeFace(f).Centroid)
		}
	}
	if len(tops) != 2 {
		t.Fatalf("expected 2 up-facing faces, got %d", len(tops))
	}
	mid := math.P3((tops[0].X+tops[1].X)/2, (tops[0].Y+tops[1].Y)/2, (tops[0].Z+tops[1].Z)/2)

	if _, ok := u.FindFaceByGeometry(topo.GeometricFaceRef{Centroid: mid, Normal: up}, 1000); ok {
		t.Error("an equidistant tie between two aligned faces must not bind either")
	}
}

// TestFindPlanarFaceThroughBindsByCentre binds a placement face by a point lying on it (a hole's
// drill centre), even off the centroid, and refuses a point off every face plane — the resolution
// the hole binder falls back to when a centroid descriptor is lost.
func TestFindPlanarFaceThroughBindsByCentre(t *testing.T) {
	a := box(math.P3(0, 0, 0), 2, 2, 2)
	var top *topo.Face
	for _, f := range a.Faces() {
		if topo.DescribeFace(f).Normal.Z > 0.9 {
			top = f
			break
		}
	}
	if top == nil {
		t.Fatal("setup: no +Z face on the box")
	}
	ref := topo.DescribeFace(top)

	got, ok := a.FindPlanarFaceThrough(ref.Centroid, ref.Normal, spikeTol)
	if !ok || got != top {
		t.Fatalf("centre-on-plane did not bind the top face (ok=%v, same=%v)", ok, got == top)
	}
	if _, ok := a.FindPlanarFaceThrough(ref.Centroid.TranslateBy(math.Vector3{Z: 100}), ref.Normal, spikeTol); ok {
		t.Error("a point far off the face plane must not bind")
	}
}

// TestGeometricFaceRefNormalIsSignAgnostic: a face descriptor whose normal is recorded with the
// opposite sign (an external author's outward-normal convention may differ) still binds the same
// face — a planar face's identity does not depend on which way its normal is named.
func TestGeometricFaceRefNormalIsSignAgnostic(t *testing.T) {
	a := box(math.P3(0, 0, 0), 2, 2, 2)
	f := a.Faces()[0]
	ref := topo.DescribeFace(f)
	ref.Normal = ref.Normal.Scale(-1) // flip the sign
	if got, ok := a.FindFaceByGeometry(ref, spikeTol); !ok || got != f {
		t.Errorf("flipped-normal descriptor did not resolve its face (ok=%v, same=%v)", ok, got == f)
	}
}

// TestFindPlanarFaceThroughDisambiguatesCoplanarByContainment: two coplanar, same-normal faces
// (two disjoint boxes joined share the Z=2 plane) no longer tie — the face that CONTAINS the
// point binds, so a hole whose drill point lands inside one of several coplanar faces resolves.
func TestFindPlanarFaceThroughDisambiguatesCoplanarByContainment(t *testing.T) {
	a := box(math.P3(0, 0, 0), 2, 2, 2)
	b := box(math.P3(6, 0, 0), 2, 2, 2) // disjoint, tops coplanar at Z=2
	u, err := ops.Boolean(ops.Join, a, b)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	up := math.Vector3{X: 0, Y: 0, Z: 1}
	p := math.P3(1, 1, 2) // inside the FIRST box's top face only
	got, ok := u.FindPlanarFaceThrough(p, up, spikeTol)
	if !ok {
		t.Fatal("a point inside one of two coplanar faces should bind that face")
	}
	if c := topo.DescribeFace(got).Centroid; c.X > 3 {
		t.Errorf("bound the wrong coplanar face (centroid X=%v, want the box at X~1)", c.X)
	}
}
