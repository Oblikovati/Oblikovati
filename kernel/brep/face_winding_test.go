// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// squareFace builds one planar face on the plane z=0 with outward normal +z, its loop walked either
// counter-clockwise about +z (the contract) or clockwise (inverted).
func squareFace(t *testing.T, ccw bool) *topo.Face {
	t.Helper()
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)}
	if !ccw {
		pts[1], pts[3] = pts[3], pts[1]
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("t", "body", 0)))
	vs := make([]*topo.Vertex, 4)
	for i, p := range pts {
		vs[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("t", "v", i)))
	}
	uses := make([]topo.Use, 4)
	for i := range pts {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(pts[i], pts[j]), vs[i], vs[j], topo.NewLineage(topo.Tok("t", "e", i)))
		uses[i] = topo.Fwd(e)
	}
	f := bld.AddFace(pl, topo.NewLineage(topo.Tok("t", "f", 0)), topo.OuterLoop(uses...))
	bld.Build()
	return f
}

// TestFaceWindingCertificateReadsAPlanarLoop: a loop counter-clockwise about the outward normal
// passes, the same loop walked the other way is an inverted face — the case Validate's per-edge
// rule cannot see on its own.
func TestFaceWindingCertificateReadsAPlanarLoop(t *testing.T) {
	t.Parallel()
	if ok, certain := FaceWindingConsistent(squareFace(t, true)); !ok || !certain {
		t.Errorf("counter-clockwise square: ok=%v certain=%v, want a certified pass", ok, certain)
	}
	if ok, certain := FaceWindingConsistent(squareFace(t, false)); ok || !certain {
		t.Errorf("clockwise square: ok=%v certain=%v, want a certified failure", ok, certain)
	}
}

// TestTangentFigureEightLobesWindWithTheirNormals is the torus tangent cut through the general
// pipeline: a plane tangent to the R=5 r=2 torus's inner equator keeps a lens on the torus whose one
// boundary loop passes through the touch point twice, and two planar lobes that share it. Every face
// must wind with its outward normal. The second lobe used to come out inverted together with the
// torus edge it borders — a closed loop whose two runs carried differently parametrised curves, and
// whose stitch read each run's direction against its own curve (ADR-0061 stage 4).
func TestTangentFigureEightLobesWindWithTheirNormals(t *testing.T) {
	t.Parallel()
	tor, err := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
	if err != nil {
		t.Fatal(err)
	}
	box, err := SolidBlock(math.P3(-20, 3, -20), math.P3(20, 20, 20), "box")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Boolean(Intersection, tor, box)
	if err != nil {
		t.Fatalf("Boolean: %v", err)
	}
	if n := len(res.Faces()); n != 3 {
		t.Fatalf("%d faces, want the torus lens and two planar lobes", n)
	}
	assertEveryFaceWinds(t, res)
}

// TestDrilledSlabFacesWindWithTheirNormals covers the certificate's other two readings: a bore wall
// (a reversed face whose two rims turn the azimuth, read against its chart) and the slab caps, whose
// bore rims are holes wound against the outer loop.
func TestDrilledSlabFacesWindWithTheirNormals(t *testing.T) {
	t.Parallel()
	slab, err := SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "slab")
	if err != nil {
		t.Fatal(err)
	}
	rod, err := SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 1.5, 4)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Boolean(Difference, slab, rod)
	if err != nil {
		t.Fatalf("Boolean: %v", err)
	}
	assertEveryFaceWinds(t, res)
}

// TestCrossingCylinderBandIsCertified is the regression for the certificate's own false positive: the
// band a rod keeps where it crosses a fatter cylinder is bounded by two loops that each wrap the rod's
// azimuth, and the chart that carries it is the band cut open at a seam. Reading a rim's direction
// against the nearest CONTOUR SEGMENT put the verdict on that artificial seam — the rim's middle
// sample sits exactly on it — and condemned a body whose volume matches OCC to six figures. The trim
// answers instead: step off the boundary to the side the winding claims (ADR-0061 stage 4).
func TestCrossingCylinderBandIsCertified(t *testing.T) {
	t.Parallel()
	fat, err := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if err != nil {
		t.Fatal(err)
	}
	thin, err := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Boolean(Intersection, fat, thin)
	if err != nil {
		t.Fatalf("Boolean(Intersection) on crossing cylinders: %v", err)
	}
	if n := len(res.Faces()); n != 3 {
		t.Fatalf("crossing intersect has %d faces, want 3 (rod band + two lens caps)", n)
	}
	assertEveryFaceWinds(t, res)
}

// assertEveryFaceWinds fails on any face the certificate can read and finds inverted, and on any
// face it cannot read at all — a corpus body's faces must be certifiable.
func assertEveryFaceWinds(t *testing.T, b *topo.Body) {
	t.Helper()
	for i, f := range b.Faces() {
		ok, certain := FaceWindingConsistent(f)
		if !certain {
			t.Errorf("face %d (%T, %d loops): winding not certifiable", i, f.Geometry(), len(f.Loops()))
		} else if !ok {
			t.Errorf("face %d (%T, %d loops): wound against its outward normal", i, f.Geometry(), len(f.Loops()))
		}
	}
}
