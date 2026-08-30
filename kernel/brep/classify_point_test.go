// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// classifyCase names a query point and the containment the exact classifier must report.
type classifyCase struct {
	name string
	p    math.Point3
	want Containment
}

func runClassify(t *testing.T, b *topo.Body, cases []classifyCase) {
	t.Helper()
	for _, c := range cases {
		if got := ClassifyPoint(b, c.p); got != c.want {
			t.Errorf("ClassifyPoint(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

// A box classifies interior, exterior and on-face points exactly, with no tessellation.
func TestClassifyPointBox(t *testing.T) {
	box, err := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	runClassify(t, box, []classifyCase{
		{"centre", math.P3(1, 1, 1), Inside},
		{"corner-interior", math.P3(0.1, 0.1, 0.1), Inside},
		{"outside +x", math.P3(3, 1, 1), Outside},
		{"outside -z", math.P3(1, 1, -0.5), Outside},
		{"on x=0 face", math.P3(0, 1, 1), OnSurface},
		{"on top face", math.P3(1, 1, 2), OnSurface},
		{"on an edge", math.P3(0, 0, 1), OnSurface},
		// The first candidate direction {2,3,5} exits this interior point exactly through the
		// top-face y=2 edge; the classifier must reselect a direction and still report Inside.
		{"interior, first ray grazes an edge", math.P3(1, 1.4, 1), Inside},
	})
}

// A cylinder classifies against its wall and both caps analytically.
func TestClassifyPointCylinder(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	runClassify(t, cyl, []classifyCase{
		{"axis interior", math.P3(0, 0, 2), Inside},
		{"off-axis interior", math.P3(1.5, 0, 2), Inside},
		{"outside radius", math.P3(3, 0, 2), Outside},
		{"above top", math.P3(0, 0, 5), Outside},
		{"below base", math.P3(0, 0, -1), Outside},
		{"on wall", math.P3(2, 0, 2), OnSurface},
		{"on bottom cap", math.P3(0, 0, 0), OnSurface},
		{"on top cap", math.P3(0.5, 0, 4), OnSurface},
	})
}

// A sphere classifies inside/outside/on the surface.
func TestClassifyPointSphere(t *testing.T) {
	sph, err := SolidSphere(math.P3(0, 0, 0), 3, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	runClassify(t, sph, []classifyCase{
		{"centre", math.P3(0, 0, 0), Inside},
		{"near surface interior", math.P3(2.9, 0, 0), Inside},
		{"outside", math.P3(5, 0, 0), Outside},
		{"on surface", math.P3(3, 0, 0), OnSurface},
		{"on surface -y", math.P3(0, -3, 0), OnSurface},
	})
}

// A torus classifies the tube interior, the central hole (outside), and both tube walls (on).
func TestClassifyPointTorus(t *testing.T) {
	tor, err := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	runClassify(t, tor, []classifyCase{
		{"tube centre", math.P3(5, 0, 0), Inside},
		{"tube interior high", math.P3(5, 0, 1), Inside},
		{"central hole", math.P3(0, 0, 0), Outside},
		{"outside the ring", math.P3(9, 0, 0), Outside},
		{"outer wall", math.P3(7, 0, 0), OnSurface},
		{"inner wall", math.P3(3, 0, 0), OnSurface},
	})
}

// An empty body contains nothing.
func TestClassifyPointEmptyBody(t *testing.T) {
	if got := ClassifyPoint(topo.BodyFromShells(topo.Lineage{}, true), math.P3(0, 0, 0)); got != Outside {
		t.Errorf("empty body classify = %v, want Outside", got)
	}
}

// pointSegmentDistance handles a normal segment and a degenerate zero-length one.
func TestPointSegmentDistance(t *testing.T) {
	a, b := math.P3(0, 0, 0), math.P3(4, 0, 0)
	if d := pointSegmentDistance(math.P3(2, 3, 0), a, b); d != 3 {
		t.Errorf("distance to mid-segment = %g, want 3", d)
	}
	if d := pointSegmentDistance(math.P3(-3, 0, 0), a, b); d != 3 {
		t.Errorf("distance past the start = %g, want 3 (clamped)", d)
	}
	if d := pointSegmentDistance(math.P3(0, 5, 0), a, a); d != 5 {
		t.Errorf("distance to a zero-length segment = %g, want 5", d)
	}
}

// cylinderSideFace returns the cylindrical side face of a solid cylinder (for grazing unit tests).
func cylFaceForTest(t *testing.T, b *topo.Body) curvedFace {
	t.Helper()
	for _, f := range facesOfAny(b) {
		if _, ok := f.surface.(geom.Cylinder); ok {
			return f
		}
	}
	t.Fatal("no cylindrical face found")
	return curvedFace{}
}

// rayGrazes flags a tangent ray and a pierce landing on a trim edge, but not a clean transversal
// crossing — the reselection trigger that keeps the parity count unambiguous.
func TestRayGrazesTangentAndBoundary(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	side := cylFaceForTest(t, cyl)
	tol := geom.ResolutionForBox(cyl.RangeBox()).Sew()

	clean, _ := geom.NewLine(math.P3(-5, 0, 2), math.V3(1, 0, 0)) // straight through the wall at mid-height
	hits := geom.RaySurfaceHits(side.surface, clean, 100)
	if len(hits) == 0 || rayGrazes(side, clean, hits[0], tol) {
		t.Errorf("a transversal mid-wall crossing must not graze (hits=%d)", len(hits))
	}

	tangent, _ := geom.NewLine(math.P3(2, -5, 2), math.V3(0, 1, 0)) // skims the wall at x=radius
	if th := geom.RaySurfaceHits(side.surface, tangent, 100); len(th) > 0 && !rayGrazes(side, tangent, th[0], tol) {
		t.Error("a tangent ray must graze (ambiguous crossing)")
	}

	atEdge, _ := geom.NewLine(math.P3(-5, 0, 0), math.V3(1, 0, 0)) // pierces the wall on the bottom cap circle
	eh := geom.RaySurfaceHits(side.surface, atEdge, 100)
	if len(eh) == 0 || !rayGrazes(side, atEdge, eh[0], tol) {
		t.Errorf("a pierce on the cap-edge boundary must graze (hits=%d)", len(eh))
	}
}

// ClassifyShellPoint classifies against a single shell's faces, matching the whole-body verdict for a
// one-shell solid.
func TestClassifyShellPoint(t *testing.T) {
	box, err := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	sh := box.Shells()[0]
	cases := []classifyCase{
		{"centre", math.P3(1, 1, 1), Inside},
		{"outside", math.P3(3, 1, 1), Outside},
		{"on face", math.P3(0, 1, 1), OnSurface},
	}
	for _, c := range cases {
		if got := ClassifyShellPoint(sh, c.p, 0); got != c.want {
			t.Errorf("ClassifyShellPoint(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

// ClassifyPointTol honors the caller's on-surface band: a point a small distance inside a face reads
// as OnSurface under a loose tolerance but Inside under a tight one.
func TestClassifyPointTol(t *testing.T) {
	box, err := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	p := math.P3(0.01, 1, 1) // 0.01 inside the x=0 face
	if got := ClassifyPointTol(box, p, 0.1); got != OnSurface {
		t.Errorf("loose onTol: got %v, want OnSurface", got)
	}
	if got := ClassifyPointTol(box, p, 1e-9); got != Inside {
		t.Errorf("tight onTol: got %v, want Inside", got)
	}
}

// InsideQuery flattens a body once and answers repeated strict-inside tests, matching PointInside.
func TestInsideQueryBatch(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	q := NewInsideQuery(cyl)
	pts := []struct {
		p    math.Point3
		want bool
	}{
		{math.P3(0, 0, 2), true},
		{math.P3(1.5, 0, 2), true},
		{math.P3(3, 0, 2), false},
		{math.P3(0, 0, 5), false},
	}
	for _, c := range pts {
		if got := q.Inside(c.p); got != c.want {
			t.Errorf("InsideQuery.Inside(%v) = %v, want %v", c.p, got, c.want)
		}
	}
	if NewInsideQuery(topo.BodyFromShells(topo.Lineage{}, true)).Inside(math.P3(0, 0, 0)) {
		t.Error("empty-body query must report not inside")
	}
}
