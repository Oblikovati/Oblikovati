// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math"
	"testing"
)

func TestCentroidInterior(t *testing.T) {
	t.Parallel()
	tr := tri([3]float64{0, 0, 0}, [3]float64{6, 0, 0}, [3]float64{0, 6, 0})
	if got := centroid(tr); !got.Equal(pt([3]float64{2, 2, 0})) {
		t.Fatalf("centroid = %v, want (2,2,0)", got.Round())
	}
}

// TestBoxMeshVolumeSign guards the test fixture itself: boxMesh must be a closed,
// outward-oriented solid, so its divergence-theorem volume is the positive box
// volume. Downstream Boolean volume checks rely on this.
func TestBoxMeshVolumeSign(t *testing.T) {
	t.Parallel()
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
		a, b, c := t[0].Round(), t[1].Round(), t[2].Round()
		cx := b.Y*c.Z - b.Z*c.Y
		cy := b.Z*c.X - b.X*c.Z
		cz := b.X*c.Y - b.Y*c.X
		vol += (a.X*cx + a.Y*cy + a.Z*cz) / 6
	}
	return vol
}
