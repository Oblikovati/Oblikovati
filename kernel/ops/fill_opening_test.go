// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// flatNeighbour builds a single-face bicubic surface body covering the planar rectangle
// [x0,x1]×[y0,y1] at z=0. When flipV is set the v (y) parameterization runs high→low, which forces
// FillFourSided to reverse that neighbour's inner edge when chaining the loop.
func flatNeighbour(t *testing.T, x0, x1, y0, y1 float64, flipV bool) *topo.Body {
	t.Helper()
	ctrl := make([][]math.Point3, 4)
	w := make([][]float64, 4)
	for i := range 4 {
		ctrl[i] = make([]math.Point3, 4)
		w[i] = []float64{1, 1, 1, 1}
		for j := range 4 {
			yj := y0 + (y1-y0)*float64(j)/3
			if flipV {
				yj = y1 - (y1-y0)*float64(j)/3
			}
			ctrl[i][j] = math.P3(x0+(x1-x0)*float64(i)/3, yj, 0)
		}
	}
	bez := []float64{0, 0, 0, 0, 1, 1, 1, 1}
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, bez, bez)
	if err != nil {
		t.Fatalf("flat neighbour patch: %v", err)
	}
	return surfaceFaceBody(t, s)
}

// TestFillFourSidedG0InterpolatesOpening fills the unit-square opening bounded by four planar
// neighbours (east flipped to exercise edge reversal) and checks the fill spans the opening at z=0.
func TestFillFourSidedG0InterpolatesOpening(t *testing.T) {
	t.Parallel()
	west := flatNeighbour(t, -1, 0, 0, 1, false)
	east := flatNeighbour(t, 1, 2, 0, 1, true) // flipped v ⇒ inner edge reversed during chaining
	south := flatNeighbour(t, 0, 1, -1, 0, false)
	north := flatNeighbour(t, 0, 1, 1, 2, false)

	out, err := ops.FillFourSided([4]*topo.Body{west, east, south, north}, 0)
	if err != nil {
		t.Fatalf("FillFourSided: %v", err)
	}
	bs, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("fill face is %T, want geom.BSplineSurface", out.Faces()[0].Geometry())
	}
	// The fill covers the opening corners and stays in the z=0 plane.
	corners := map[[2]float64]math.Point3{
		{0, 0}: math.P3(0, 0, 0), {1, 0}: math.P3(1, 0, 0),
		{0, 1}: math.P3(0, 1, 0), {1, 1}: math.P3(1, 1, 0),
	}
	for uv, want := range corners {
		got := bs.PointAt(uv[0], uv[1])
		if !cornerSpansOpening(got, want) {
			t.Errorf("fill corner (%g,%g) = %v, want one of the opening corners near %v", uv[0], uv[1], got, want)
		}
	}
	for i := 0; i <= 4; i++ {
		for j := 0; j <= 4; j++ {
			if p := bs.PointAt(float64(i)/4, float64(j)/4); p.Z < -1e-7 || p.Z > 1e-7 {
				t.Fatalf("planar opening fill left the z=0 plane at (%d,%d): z=%g", i, j, p.Z)
			}
		}
	}
}

// cornerSpansOpening checks p lies on the unit-square opening boundary near any of its corners (the
// fill's parametric corners map to the opening corners; orientation may permute which is which).
func cornerSpansOpening(p, _ math.Point3) bool {
	onX := (p.X > -1e-7 && p.X < 1e-7) || (p.X > 1-1e-7 && p.X < 1+1e-7)
	onY := (p.Y > -1e-7 && p.Y < 1e-7) || (p.Y > 1-1e-7 && p.Y < 1+1e-7)
	return onX && onY
}

func TestFillFourSidedErrorsOnNonNurbs(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 1, 1, 1)
	good := flatNeighbour(t, -1, 0, 0, 1, false)
	if _, err := ops.FillFourSided([4]*topo.Body{good, box, good, good}, 0); err == nil {
		t.Error("a non-NURBS neighbour should make FillFourSided error")
	}
}

// planeNeighbour builds a single-face PLANAR surface body over [x0,x1]×[y0,y1] at z=0 (geom.Plane,
// not NURBS) — the kind boundaryPatch produces.
func planeNeighbour(t *testing.T, tag string, x0, x1, y0, y1 float64) *topo.Body {
	t.Helper()
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	c := [4]math.Point3{math.P3(math.Scalar(x0), math.Scalar(y0), 0), math.P3(math.Scalar(x1), math.Scalar(y0), 0), math.P3(math.Scalar(x1), math.Scalar(y1), 0), math.P3(math.Scalar(x0), math.Scalar(y1), 0)}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(tag, "body", 0)))
	v := make([]*topo.Vertex, 4)
	for i, p := range c {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(tag, "v", i)))
	}
	uses := make([]topo.Use, 4)
	for i := range 4 {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(c[i], c[j]), v[i], v[j], topo.NewLineage(topo.Tok(tag, "e", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(pl, topo.NewLineage(topo.Tok(tag, "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// TestFillFourSidedPlanarNeighbours: a fill among four PLANAR (non-NURBS) neighbours — the common
// boundary-patch case — closes the opening with a valid surface body in the shared plane (G0 on every
// side, since planar neighbours have no curvature to match).
func TestFillFourSidedPlanarNeighbours(t *testing.T) {
	t.Parallel()
	ns := [4]*topo.Body{
		planeNeighbour(t, "w", -1, 0, 0, 1),
		planeNeighbour(t, "e", 1, 2, 0, 1),
		planeNeighbour(t, "s", 0, 1, -1, 0),
		planeNeighbour(t, "n", 0, 1, 1, 2),
	}
	out, err := ops.FillFourSided(ns, 2) // G2 requested; planar ⇒ falls back to G0 per side
	if err != nil {
		t.Fatalf("FillFourSided (planar): %v", err)
	}
	bs, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("fill face is %T, want geom.BSplineSurface", out.Faces()[0].Geometry())
	}
	for i := 0; i <= 4; i++ {
		for j := 0; j <= 4; j++ {
			if p := bs.PointAt(float64(i)/4, float64(j)/4); p.Z < -1e-7 || p.Z > 1e-7 {
				t.Fatalf("planar fill left z=0 at (%d,%d): z=%g", i, j, p.Z)
			}
		}
	}
}
