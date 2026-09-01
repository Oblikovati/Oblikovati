// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// meshArea sums the triangle areas of a mesh.
func meshArea(m *ops.Mesh) float64 {
	area := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a, b, c := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		area += 0.5 * float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length())
	}
	return area
}

// cylindricalFaceKey returns the reference key of the body's lateral cylindrical face.
func cylindricalFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f.ReferenceKey()
		}
	}
	t.Fatal("no cylindrical face found")
	return nil
}

// TestOffsetFaceSurfacesCylinderArea is the offset oracle: offsetting a radius-5, height-10 cylinder's
// lateral face outward by 2 yields a radius-7 cylindrical surface, whose tessellated area must match
// the analytic lateral area 2π·7·10 (per the tessellation-correctness rule, validated against the
// closed form). The reverse direction gives the radius-3 inner surface (2π·3·10).
func TestOffsetFaceSurfacesCylinderArea(t *testing.T) {
	t.Parallel()
	const r, h = 5.0, 10.0
	body, err := brep.SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, h), r, r, "test")
	if err != nil {
		t.Fatal(err)
	}
	key := cylindricalFaceKey(t, body)

	cases := []struct {
		name    string
		dist    float64
		reverse bool
		wantR   float64
	}{
		{"outward", 2, false, r + 2},
		{"inward", 2, true, r - 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			off, err := ops.OffsetFaceSurfaces(body, [][]byte{key}, c.dist, c.reverse)
			if err != nil {
				t.Fatal(err)
			}
			if len(off.Faces()) != 1 {
				t.Fatalf("offset body has %d faces, want 1 (just the offset wall)", len(off.Faces()))
			}
			got := meshArea(ops.TessellateFace(off.Faces()[0], ops.DefaultQuality()))
			want := 2 * stdmath.Pi * c.wantR * h
			if stdmath.Abs(got-want)/want > 0.02 {
				t.Errorf("offset (%s) area = %.3f, want %.3f (2π·%g·%g)", c.name, got, want, c.wantR, h)
			}
		})
	}

	if _, err := ops.OffsetFaceSurfaces(body, nil, 1, false); err == nil {
		t.Error("offsetting no faces must error")
	}
	if _, err := ops.OffsetFaceSurfaces(body, [][]byte{[]byte("nope")}, 1, false); err == nil {
		t.Error("offsetting an unknown face key must error")
	}
}
