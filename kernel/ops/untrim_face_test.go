// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

func TestUntrimFaceRecoversFullSurface(t *testing.T) {
	t.Parallel()
	src := multiSpanPatch(t) // helper in rebuild_faces_test.go
	body := brepfixture.SurfaceFaceBody(t, src)
	out, err := ops.UntrimFace(body, body.Faces()[0].ReferenceKey())
	if err != nil {
		t.Fatalf("UntrimFace: %v", err)
	}
	if len(out.Faces()) != 1 {
		t.Fatalf("untrimmed body has %d faces, want 1", len(out.Faces()))
	}
	face := out.Faces()[0]
	if len(out.Edges()) != 4 {
		t.Errorf("untrimmed face should have 4 boundary edges, got %d", len(out.Edges()))
	}
	// The base surface is recovered unchanged (so re-applying the original trim reproduces the face).
	bs, ok := face.Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("untrimmed face geometry is %T, want geom.BSplineSurface", face.Geometry())
	}
	for i := 0; i <= 6; i++ {
		for j := 0; j <= 6; j++ {
			u, v := float64(i)/6, float64(j)/6
			if !bs.PointAt(u, v).IsEqualTo(src.PointAt(u, v), 1e-12) {
				t.Fatalf("untrim changed the base surface at (%g,%g)", u, v)
			}
		}
	}
	// The boundary edges lie on the surface (iso-curves): the bottom edge's midpoint is on v=0.
	bottomMid := out.Edges()[0].Geometry().PointAt(0.5)
	if !onSurfaceBoundary(bs, bottomMid) {
		t.Errorf("untrim boundary edge does not lie on the surface boundary: %v", bottomMid)
	}
}

// onSurfaceBoundary reports whether p matches the surface at some boundary parameter (rough check
// against the four edges at the midpoint).
func onSurfaceBoundary(s geom.BSplineSurface, p math.Point3) bool {
	for _, e := range [][2]float64{{0.5, 0}, {0.5, 1}, {0, 0.5}, {1, 0.5}} {
		if s.PointAt(e[0], e[1]).IsEqualTo(p, 1e-9) {
			return true
		}
	}
	return false
}

func TestUntrimFaceErrorsOnNonNurbs(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 1, 1, 1) // planar faces
	if _, err := ops.UntrimFace(box, box.Faces()[0].ReferenceKey()); err == nil {
		t.Error("untrimming a planar face should error (not a NURBS surface)")
	}
	if _, err := ops.UntrimFace(box, []byte("nope")); err == nil {
		t.Error("untrimming with an unknown key should error")
	}
}
