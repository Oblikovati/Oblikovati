// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

func TestExtendFaceSurfaceGrowsDomain(t *testing.T) {
	t.Parallel()
	body := surfaceFaceBody(t, multiSpanPatch(t)) // helpers in rebuild_faces_test.go
	out, err := ops.ExtendFaceSurface(body, geom.UMaxEdge, 0.5, 2)
	if err != nil {
		t.Fatalf("ExtendFaceSurface: %v", err)
	}
	bs, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("extended face is %T, want geom.BSplineSurface", out.Faces()[0].Geometry())
	}
	if _, hi := bs.UDomain(); hi <= 1+1e-9 {
		t.Errorf("extended u-domain max = %g, want > 1", hi)
	}
	if len(out.Edges()) != 4 {
		t.Errorf("extended face should be a full-domain quad (4 edges), got %d", len(out.Edges()))
	}
}

func TestExtendFaceSurfaceErrorsOnNonNurbs(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 1, 1, 1)
	if _, err := ops.ExtendFaceSurface(box, geom.UMaxEdge, 0.5, 2); err == nil {
		t.Error("extending a planar body should error (no NURBS face)")
	}
}
