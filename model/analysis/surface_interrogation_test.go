// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// sphereMesh builds a UV sphere of radius r (outward unit normals = position/r), with nLat×nLong
// quads split into triangles — a strongly curved surface whose isophotes are exact latitude circles.
func sphereMesh(r float64, nLat, nLong int) SurfaceSamples {
	var m SurfaceSamples
	idx := func(i, j int) int { return i*(nLong+1) + j }
	for i := 0; i <= nLat; i++ {
		theta := stdmath.Pi * float64(i) / float64(nLat) // 0..π
		for j := 0; j <= nLong; j++ {
			phi := 2 * stdmath.Pi * float64(j) / float64(nLong)
			x := r * stdmath.Sin(theta) * stdmath.Cos(phi)
			y := r * stdmath.Sin(theta) * stdmath.Sin(phi)
			z := r * stdmath.Cos(theta)
			m.Positions = append(m.Positions, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)))
			m.Normals = append(m.Normals, math.V3(math.Scalar(x/r), math.Scalar(y/r), math.Scalar(z/r)))
		}
	}
	for i := 0; i < nLat; i++ {
		for j := 0; j < nLong; j++ {
			a, b, c, d := idx(i, j), idx(i, j+1), idx(i+1, j+1), idx(i+1, j)
			m.Triangles = append(m.Triangles, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	return m
}

func TestIsophotesOnSphereLieOnSurface(t *testing.T) {
	const r = 2.0
	m := sphereMesh(r, 24, 48)
	segs := Isophotes(m, math.V3(0, 0, 1), 6)
	if len(segs) == 0 {
		t.Fatal("a sphere should produce isophote contours")
	}
	for _, s := range segs {
		for _, p := range [2]math.Point3{s.A, s.B} {
			// Points lie on the tessellated chords, slightly inside the analytic sphere (chord sag).
			if d := float64(p.AsVector().Length()); d > r+1e-9 || d < r-0.02 {
				t.Fatalf("isophote point off the sphere mesh: |p|=%g, want ~%g", d, r)
			}
		}
	}
}

func TestIsophotesFlatPlaneAreEmpty(t *testing.T) {
	// A flat patch has a constant normal, so N·L is constant and there are no iso-contours.
	var m SurfaceSamples
	for i := 0; i <= 4; i++ {
		for j := 0; j <= 4; j++ {
			m.Positions = append(m.Positions, math.P3(math.Scalar(i), math.Scalar(j), 0))
			m.Normals = append(m.Normals, math.V3(0, 0, 1))
		}
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			a := i*5 + j
			m.Triangles = append(m.Triangles, [3]int{a, a + 1, a + 6}, [3]int{a, a + 6, a + 5})
		}
	}
	if segs := Isophotes(m, math.V3(0.3, 0, 1), 5); len(segs) != 0 {
		t.Errorf("a flat plane should have no isophotes, got %d segments", len(segs))
	}
}

func TestIsophotesLinearFieldAreStraightAndParallel(t *testing.T) {
	// Normals tilt only with x → N·L (L=+Z) depends only on x → isophotes are vertical lines (x const).
	var m SurfaceSamples
	const n = 20
	idx := func(i, j int) int { return i*(n+1) + j }
	for i := 0; i <= n; i++ {
		x := -1 + 2*float64(i)/float64(n)
		nrm := math.V3(math.Scalar(x), 0, 1)
		for j := 0; j <= n; j++ {
			m.Positions = append(m.Positions, math.P3(math.Scalar(x), math.Scalar(float64(j)/float64(n)), 0))
			m.Normals = append(m.Normals, nrm)
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a, b, c, d := idx(i, j), idx(i, j+1), idx(i+1, j+1), idx(i+1, j)
			m.Triangles = append(m.Triangles, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	segs := Isophotes(m, math.V3(0, 0, 1), 4)
	if len(segs) == 0 {
		t.Fatal("a tilting-normal strip should produce isophotes")
	}
	for _, s := range segs {
		if dx := stdmath.Abs(float64(s.A.X - s.B.X)); dx > 0.06 { // within one cell width
			t.Errorf("isophote segment is not vertical: Δx=%g", dx)
		}
	}
}

func TestReflectionAndHighlightLinesOnSphere(t *testing.T) {
	m := sphereMesh(1, 20, 40)
	if segs := ReflectionLines(m, math.P3(0, 0, 5), math.V3(1, 0, 0), 8); len(segs) == 0 {
		t.Error("reflection lines should be non-empty on a sphere")
	}
	if segs := HighlightLines(m, math.P3(0, 0, 5), math.V3(0.4, 0.4, 1), 6); len(segs) == 0 {
		t.Error("highlight lines should be non-empty on a sphere")
	}
}

func TestInteriorLevels(t *testing.T) {
	lv := interiorLevels([]float64{-1, 1}, 3)
	want := []float64{-0.5, 0, 0.5}
	if len(lv) != 3 {
		t.Fatalf("got %d levels, want 3", len(lv))
	}
	for i := range want {
		if stdmath.Abs(lv[i]-want[i]) > 1e-12 {
			t.Fatalf("interiorLevels = %v, want %v", lv, want)
		}
	}
	// Auto-fits to the actual range, not a fixed (−1,1): a [0.5,1] field gives levels inside it.
	if lv := interiorLevels([]float64{0.5, 1}, 1); len(lv) != 1 || lv[0] <= 0.5 || lv[0] >= 1 {
		t.Errorf("interiorLevels([0.5,1],1) = %v, want one level inside (0.5,1)", lv)
	}
	if lv := interiorLevels([]float64{0.7, 0.7}, 4); lv != nil {
		t.Errorf("a constant field should yield no levels, got %v", lv)
	}
}
