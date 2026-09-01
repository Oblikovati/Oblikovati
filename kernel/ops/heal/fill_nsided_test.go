// SPDX-License-Identifier: GPL-2.0-only

package heal_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/ops/heal"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// edgeNeighbour builds a single-face bicubic surface body that extends outward (away from center)
// from the segment a→b: row 0 is the inner edge a→b, row 3 the same edge pushed radially outward, so
// FillNSided sees a→b as this neighbour's inner boundary. All points stay at z=0.
func edgeNeighbour(t *testing.T, a, b, center math.Point3, reach float64) *topo.Body {
	t.Helper()
	mid := math.P3((a.X+b.X)/2, (a.Y+b.Y)/2, 0)
	out := center.VectorTo(mid)
	if l := out.Length(); l > 0 {
		out = out.Scale(math.Scalar(reach / float64(l)))
	}
	aOut := a.TranslateBy(out)
	bOut := b.TranslateBy(out)
	ctrl := make([][]math.Point3, 4)
	w := make([][]float64, 4)
	for i := range 4 {
		ctrl[i] = make([]math.Point3, 4)
		w[i] = []float64{1, 1, 1, 1}
		for j := range 4 {
			s := float64(j) / 3
			inner := a.Lerp(b, s)
			outer := aOut.Lerp(bOut, s)
			ctrl[i][j] = inner.Lerp(outer, float64(i)/3)
		}
	}
	bez := []float64{0, 0, 0, 0, 1, 1, 1, 1}
	srf, err := geom.NewBSplineSurface(3, 3, ctrl, w, bez, bez)
	if err != nil {
		t.Fatalf("edge neighbour: %v", err)
	}
	return brepfixture.SurfaceFaceBody(t, srf)
}

// polygonNeighbours builds one outward edge-neighbour per side of the regular n-gon of radius r,
// lifting vertex i to height z(i) so the opening can be made non-planar. Returns the bodies and the
// n-gon vertices (for interpolation checks).
func polygonNeighbours(t *testing.T, n int, r float64, z func(i int) float64) ([]*topo.Body, []math.Point3) {
	t.Helper()
	verts := make([]math.Point3, n)
	for i := range n {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		verts[i] = math.P3(math.Scalar(r*stdmath.Cos(a)), math.Scalar(r*stdmath.Sin(a)), math.Scalar(z(i)))
	}
	center := math.P3(0, 0, 0)
	bodies := make([]*topo.Body, n)
	for i := range n {
		bodies[i] = edgeNeighbour(t, verts[i], verts[(i+1)%n], center, r)
	}
	return bodies, verts
}

func flat(int) float64 { return 0 }

// fillBorderOnOpening checks the fill's own four parametric borders lie on the opening boundary (the
// n-gon polyline) — a forward-evaluation G0 closure check that avoids the fragile surface inverse on
// a warped patch.
func fillBorderOnOpening(t *testing.T, fill geom.BSplineSurface, verts []math.Point3) {
	t.Helper()
	for _, edge := range []struct {
		uFixed bool
		val    float64
	}{{true, 0}, {true, 1}, {false, 0}, {false, 1}} {
		for k := 0; k <= 16; k++ {
			s := float64(k) / 16
			var p math.Point3
			if edge.uFixed {
				p = fill.PointAt(edge.val, s)
			} else {
				p = fill.PointAt(s, edge.val)
			}
			if d := minDistToOpening(p, verts); d > 1e-6 {
				t.Errorf("fill border (uFixed=%v val=%g) strayed %.3g off the opening boundary at s=%g", edge.uFixed, edge.val, d, s)
			}
		}
	}
}

// minDistToOpening returns the distance from p to the closed n-gon boundary (point-to-segment over
// every edge), so a curved border on the exact boundary reads ~0 regardless of sampling resolution.
func minDistToOpening(p math.Point3, verts []math.Point3) float64 {
	best := stdmath.Inf(1)
	for i := range verts {
		if d := pointToSegment(p, verts[i], verts[(i+1)%len(verts)]); d < best {
			best = d
		}
	}
	return best
}

// pointToSegment returns the distance from p to segment a-b.
func pointToSegment(p, a, b math.Point3) float64 {
	ab := a.VectorTo(b)
	l2 := float64(ab.Dot(ab))
	if l2 == 0 {
		return float64(p.DistanceTo(a))
	}
	tt := float64(a.VectorTo(p).Dot(ab)) / l2
	if tt < 0 {
		tt = 0
	} else if tt > 1 {
		tt = 1
	}
	return float64(p.DistanceTo(a.Lerp(b, tt)))
}

// fillInterpolatesVertices checks the fill surface passes through every n-gon vertex (the corners
// where the boundary edges meet) — proof the opening boundary is interpolated.
func fillInterpolatesVertices(t *testing.T, fill geom.BSplineSurface, verts []math.Point3) {
	t.Helper()
	for k, v := range verts {
		u, w := fill.ParamAt(v)
		if d := float64(v.DistanceTo(fill.PointAt(u, w))); d > 1e-5 {
			t.Errorf("fill misses n-gon vertex %d (%v) by %.3g", k, v, d)
		}
	}
}

func TestFillNSidedThreeSided(t *testing.T) {
	t.Parallel()
	bodies, verts := polygonNeighbours(t, 3, 1, flat)
	out, err := heal.FillNSided(bodies, 1)
	if err != nil {
		t.Fatalf("FillNSided(3): %v", err)
	}
	fill, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("fill face is %T, want BSplineSurface", out.Faces()[0].Geometry())
	}
	fillInterpolatesVertices(t, fill, verts)
}

func TestFillNSidedFiveSided(t *testing.T) {
	t.Parallel()
	bodies, verts := polygonNeighbours(t, 5, 1, flat)
	out, err := heal.FillNSided(bodies, 2)
	if err != nil {
		t.Fatalf("FillNSided(5): %v", err)
	}
	fill, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("fill face is %T, want BSplineSurface", out.Faces()[0].Geometry())
	}
	fillInterpolatesVertices(t, fill, verts)
}

func TestFillNSidedFourDelegates(t *testing.T) {
	t.Parallel()
	bodies, verts := polygonNeighbours(t, 4, 1, flat)
	out, err := heal.FillNSided(bodies, 1)
	if err != nil {
		t.Fatalf("FillNSided(4): %v", err)
	}
	fill := out.Faces()[0].Geometry().(geom.BSplineSurface)
	fillInterpolatesVertices(t, fill, verts)
}

func TestFillNSidedNonPlanarFiveSided(t *testing.T) {
	t.Parallel()
	// A saddle-shaped pentagon opening (alternating raised/lowered vertices): the fill must still
	// close it through every curved boundary edge.
	z := func(i int) float64 {
		if i%2 == 0 {
			return 0.25
		}
		return -0.15
	}
	bodies, verts := polygonNeighbours(t, 5, 1, z)
	out, err := heal.FillNSided(bodies, 0) // G0 closure of a warped opening (G2 match quality is a neighbour property, gated separately by F13)
	if err != nil {
		t.Fatalf("FillNSided(non-planar 5): %v", err)
	}
	fill := out.Faces()[0].Geometry().(geom.BSplineSurface)
	fillBorderOnOpening(t, fill, verts)
}

func TestFillNSidedTooFew(t *testing.T) {
	t.Parallel()
	bodies, _ := polygonNeighbours(t, 3, 1, flat)
	if _, err := heal.FillNSided(bodies[:2], 1); err == nil {
		t.Error("fewer than 3 neighbours should error")
	}
}
