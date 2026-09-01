// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestBodyToSoupClosedBox checks the IN adapter turns a body into a valid operand
// soup: one triangle per tessellation face-triangle, and a closed, outward-oriented
// mesh whose divergence-theorem volume equals the box volume (2·3·4 = 24). A
// non-closed or inward soup would not give the exact positive volume.
func TestBodyToSoupClosedBox(t *testing.T) {
	t.Parallel()
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

// TestSoupToBodyRoundTripBox round-trips a box through the OUT adapter: body →
// soup → body. The reconstructed body must be a valid solid with the 6 merged
// square faces, and re-tessellating it must give back the box volume 24.
func TestSoupToBodyRoundTripBox(t *testing.T) {
	t.Parallel()
	orig := subd.ToBody(subd.Box(2, 3, 4), "box")
	body := soupToBody(bodyToSoup(orig, PropertyQuality()), "bool")

	if got := len(body.Faces()); got != 6 {
		t.Fatalf("round-trip body has %d faces, want 6 (merged)", got)
	}
	if !body.IsSolid() {
		t.Fatal("round-trip body is not a solid")
	}
	if r := Validate(body); !r.Valid {
		t.Fatalf("round-trip body invalid: %+v", r)
	}
	if v := soupVolume(bodyToSoup(body, PropertyQuality())); stdmath.Abs(v-24) > 1e-9 {
		t.Fatalf("round-trip volume = %.6f, want 24", v)
	}
}

// TestBooleanViaMeshboolBoxes is the end-to-end integration proof: two box BODIES
// (A=[0,2]^3, B=[1,3]^3) through the mesh-arrangement engine, each result a valid
// solid with the exact expected volume.
func TestBooleanViaMeshboolBoxes(t *testing.T) {
	t.Parallel()
	a := subd.ToBody(subd.Box(2, 2, 2), "a")
	b := boxBodyAt(2, 2, 2, math.V3(1, 1, 1), "b")
	cases := []struct {
		name string
		op   meshbool.Op
		vol  float64
	}{
		{"union", meshbool.Union, 15},
		{"difference", meshbool.Difference, 7},
		{"intersection", meshbool.Intersection, 1},
	}
	for _, tc := range cases {
		res := booleanViaMeshbool(a, b, tc.op, PropertyQuality(), "op")
		if !res.IsSolid() {
			t.Fatalf("%s: result is not a solid", tc.name)
		}
		if r := Validate(res); !r.Valid {
			t.Fatalf("%s: result invalid: %+v", tc.name, r)
		}
		if v := soupVolume(bodyToSoup(res, PropertyQuality())); stdmath.Abs(v-tc.vol) > 1e-9 {
			t.Fatalf("%s: volume = %.6f, want %.0f", tc.name, v, tc.vol)
		}
	}
}

// TestBooleanViaMeshboolCoplanar drives the coplanar-keep + face-merge path through
// the full brep adapter: two boxes overlapping only in x share coincident
// top/bottom/front/back faces. The union must be a valid solid of volume 12.
func TestBooleanViaMeshboolCoplanar(t *testing.T) {
	t.Parallel()
	a := subd.ToBody(subd.Box(2, 2, 2), "a")
	b := boxBodyAt(2, 2, 2, math.V3(1, 0, 0), "b")
	res := booleanViaMeshbool(a, b, meshbool.Union, PropertyQuality(), "op")
	if !res.IsSolid() || !Validate(res).Valid {
		t.Fatal("coplanar union is not a valid solid")
	}
	if v := soupVolume(bodyToSoup(res, PropertyQuality())); stdmath.Abs(v-12) > 1e-9 {
		t.Fatalf("coplanar union volume = %.6f, want 12", v)
	}
}

// boxBodyAt builds a box translated by off.
func boxBodyAt(sx, sy, sz float64, off math.Vector3, feat string) *topo.Body {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(off)
	}
	return subd.ToBody(m, feat)
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
