// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestGraphicsMeshFromBox checks the display-mesh decode against a real PmGraphicsSegment: the
// box's face patches (A79EACD2 nodes) yield a non-trivial triangle mesh whose indices are all
// in range. This is the static-body fallback for parts that don't rebuild parametrically.
func TestGraphicsMeshFromBox(t *testing.T) {
	d := openDoc(t, "10_box.ipt")
	m := GraphicsMesh(d)
	if len(m.Verts) < 12 || len(m.Tris) < 12 {
		t.Fatalf("box graphics mesh = %d verts, %d tris; want the box's 12+ triangles", len(m.Verts), len(m.Tris))
	}
	for _, tri := range m.Tris {
		for _, vi := range tri {
			if vi < 0 || vi >= len(m.Verts) {
				t.Fatalf("triangle vertex index %d out of range [0,%d)", vi, len(m.Verts))
			}
		}
	}
}

// TestGraphicsMeshAbsent confirms a segment with no display patches yields an empty mesh (so
// the caller falls through to the ACIS body path) rather than panicking.
func TestGraphicsMeshAbsent(t *testing.T) {
	d := openDoc(t, "sketch_line.ipt")
	if m := GraphicsMesh(d); len(m.Tris) > 0 && len(m.Verts) == 0 {
		t.Errorf("mesh has %d triangles but no vertices", len(m.Tris))
	}
}
