// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math"
	"testing"
)

func TestInsideMeshCube(t *testing.T) {
	cube := boxMesh([3]float64{0, 0, 0}, [3]float64{1, 1, 1})
	inside := []Point{
		pt([3]float64{0.5, 0.5, 0.5}),
		pt([3]float64{0.1, 0.9, 0.5}),
		pt([3]float64{0.01, 0.01, 0.01}),
	}
	outside := []Point{
		pt([3]float64{2, 2, 2}),
		pt([3]float64{-0.5, 0.5, 0.5}),
		pt([3]float64{0.5, 0.5, 1.5}),
	}
	for _, p := range inside {
		if !insideMesh(p, cube) {
			t.Fatalf("point %v classified outside the cube (w=%.3f)", p.Round(), windingNumber(p, cube))
		}
	}
	for _, p := range outside {
		if insideMesh(p, cube) {
			t.Fatalf("point %v classified inside the cube (w=%.3f)", p.Round(), windingNumber(p, cube))
		}
	}
}

func TestWindingNumberMagnitude(t *testing.T) {
	cube := boxMesh([3]float64{0, 0, 0}, [3]float64{2, 2, 2})
	if w := windingNumber(pt([3]float64{1, 1, 1}), cube); math.Abs(math.Abs(w)-1) > 1e-9 {
		t.Fatalf("winding number inside = %.6f, want ±1", w)
	}
	if w := windingNumber(pt([3]float64{5, 5, 5}), cube); math.Abs(w) > 1e-9 {
		t.Fatalf("winding number far outside = %.6f, want 0", w)
	}
}

func TestCentroidInterior(t *testing.T) {
	tr := tri([3]float64{0, 0, 0}, [3]float64{6, 0, 0}, [3]float64{0, 6, 0})
	if got := centroid(tr); !got.Equal(pt([3]float64{2, 2, 0})) {
		t.Fatalf("centroid = %v, want (2,2,0)", got.Round())
	}
}

// TestBoxMeshVolumeSign guards the test fixture itself: boxMesh must be a closed,
// outward-oriented solid, so its divergence-theorem volume is the positive box
// volume. Downstream Boolean volume checks rely on this.
func TestBoxMeshVolumeSign(t *testing.T) {
	got := meshVolume(boxMesh([3]float64{0, 0, 0}, [3]float64{2, 1, 3}))
	if math.Abs(got-6) > 1e-9 {
		t.Fatalf("box volume = %.6f, want 6 (outward orientation)", got)
	}
}

// --- shared fixtures ---

// boxMesh returns the 12 outward-oriented triangles of the axis-aligned box
// [lo,hi]. Each face is two triangles fanned from a corner, wound CCW as seen from
// outside so the mesh normal points out.
func boxMesh(lo, hi [3]float64) [][3]Point {
	x0, y0, z0 := lo[0], lo[1], lo[2]
	x1, y1, z1 := hi[0], hi[1], hi[2]
	var m [][3]Point
	add := func(a, b, c, d [3]float64) {
		m = append(m, [3]Point{pt(a), pt(b), pt(c)}, [3]Point{pt(a), pt(c), pt(d)})
	}
	add([3]float64{x1, y0, z0}, [3]float64{x1, y1, z0}, [3]float64{x1, y1, z1}, [3]float64{x1, y0, z1}) // +x
	add([3]float64{x0, y0, z0}, [3]float64{x0, y0, z1}, [3]float64{x0, y1, z1}, [3]float64{x0, y1, z0}) // -x
	add([3]float64{x0, y1, z0}, [3]float64{x0, y1, z1}, [3]float64{x1, y1, z1}, [3]float64{x1, y1, z0}) // +y
	add([3]float64{x0, y0, z0}, [3]float64{x1, y0, z0}, [3]float64{x1, y0, z1}, [3]float64{x0, y0, z1}) // -y
	add([3]float64{x0, y0, z1}, [3]float64{x1, y0, z1}, [3]float64{x1, y1, z1}, [3]float64{x0, y1, z1}) // +z
	add([3]float64{x0, y0, z0}, [3]float64{x0, y1, z0}, [3]float64{x1, y1, z0}, [3]float64{x1, y0, z0}) // -z
	return m
}

// meshVolume returns the signed volume enclosed by mesh (divergence theorem); for a
// closed, outward-oriented mesh it is the positive enclosed volume.
func meshVolume(mesh [][3]Point) float64 {
	vol := 0.0
	for _, t := range mesh {
		vol += fdot(t[0].floats(), fcross(t[1].floats(), t[2].floats())) / 6
	}
	return vol
}
