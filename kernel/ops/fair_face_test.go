// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestFairFaceSurfaceFairsBody fairs a multi-span NURBS surface body (5×5 net) at G1 and checks it
// returns a valid single-face NURBS body with the same control dimensions (fairing moves interior
// points, it does not change degree/knots).
func TestFairFaceSurfaceFairsBody(t *testing.T) {
	body := surfaceFaceBody(t, multiSpanPatch(t))
	out, err := ops.FairFaceSurface(body, 1, 0.5, 10)
	if err != nil {
		t.Fatalf("FairFaceSurface: %v", err)
	}
	s, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("faired face is %T, want geom.BSplineSurface", out.Faces()[0].Geometry())
	}
	if len(s.Ctrl) != 5 || len(s.Ctrl[0]) != 5 {
		t.Errorf("faired net = %dx%d, want 5x5 (fairing keeps the control structure)", len(s.Ctrl), len(s.Ctrl[0]))
	}
}

func TestFairFaceSurfaceRejectsNonNurbs(t *testing.T) {
	box := csgBox(math.P3(0, 0, 0), 1, 1, 1)
	if _, err := ops.FairFaceSurface(box, 1, 0.5, 10); err == nil {
		t.Error("fairing a body with no NURBS face should error")
	}
}
