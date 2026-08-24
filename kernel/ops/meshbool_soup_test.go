// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/subd"
)

// TestBodyToSoupClosedBox checks the IN adapter turns a body into a valid operand
// soup: one triangle per tessellation face-triangle, and a closed, outward-oriented
// mesh whose divergence-theorem volume equals the box volume (2·3·4 = 24). A
// non-closed or inward soup would not give the exact positive volume.
func TestBodyToSoupClosedBox(t *testing.T) {
	body := subd.ToBody(subd.Box(2, 3, 4), "box")
	soup := bodyToSoup(body, PropertyQuality())

	mesh, _ := TessellateBody(body, PropertyQuality())
	if len(soup) != mesh.TriangleCount() {
		t.Fatalf("soup has %d triangles, tessellation %d", len(soup), mesh.TriangleCount())
	}
	if len(soup) == 0 {
		t.Fatal("empty soup")
	}
	if v := soupVolume(soup); stdmath.Abs(v-24) > 1e-9 {
		t.Fatalf("soup volume = %.6f, want 24 (closed, outward-oriented)", v)
	}
	for i, tri := range soup {
		if soupTriDegenerate(tri) {
			t.Fatalf("degenerate triangle %d in soup — would violate the boolean's precondition", i)
		}
	}
}

// soupVolume is the signed volume enclosed by the soup (divergence theorem); for a
// closed, outward-oriented mesh it is the positive enclosed volume.
func soupVolume(soup [][3]meshbool.Point) float64 {
	vol := 0.0
	for _, tri := range soup {
		a, b, c := tri[0].Round(), tri[1].Round(), tri[2].Round()
		vol += (a.X*(b.Y*c.Z-b.Z*c.Y) + a.Y*(b.Z*c.X-b.X*c.Z) + a.Z*(b.X*c.Y-b.Y*c.X)) / 6
	}
	return vol
}

// soupTriDegenerate reports whether a soup triangle has zero area (its edge cross
// product vanishes).
func soupTriDegenerate(tri [3]meshbool.Point) bool {
	a, b, c := tri[0].Round(), tri[1].Round(), tri[2].Round()
	ux, uy, uz := b.X-a.X, b.Y-a.Y, b.Z-a.Z
	vx, vy, vz := c.X-a.X, c.Y-a.Y, c.Z-a.Z
	nx, ny, nz := uy*vz-uz*vy, uz*vx-ux*vz, ux*vy-uy*vx
	return nx*nx+ny*ny+nz*nz == 0
}
