// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

func TestExtendSurfaceGrowsAndKeepsOriginal(t *testing.T) {
	t.Parallel()
	s := uPatch(t, 0, func(i, j int) float64 { return 0.5 * float64(i*i) }) // curved in u
	ext, err := ExtendSurface(s, UMaxEdge, 0.5, 2)
	if err != nil {
		t.Fatalf("ExtendSurface: %v", err)
	}
	if _, hi := ext.UDomain(); hi <= 1+1e-9 {
		t.Errorf("extended u-domain max = %g, want > 1", hi)
	}
	if len(ext.Ctrl) != len(s.Ctrl)+s.UDegree {
		t.Errorf("control rows = %d, want %d (original + degree appended)", len(ext.Ctrl), len(s.Ctrl)+s.UDegree)
	}
	// The original parameter range [0,1] is geometrically unchanged.
	for _, u := range []float64{0, 0.3, 0.7, 1} {
		for _, v := range []float64{0, 0.5, 1} {
			if !ext.PointAt(u, v).IsEqualTo(s.PointAt(u, v), 1e-9) {
				t.Fatalf("original surface changed at (%g,%g): %v vs %v", u, v, ext.PointAt(u, v), s.PointAt(u, v))
			}
		}
	}
}

// jointDerivs returns the parametric u-derivatives just below and just above the extension join at
// u=1 for column v, up to `order`.
func jointDerivs(ext BSplineSurface, v float64, order int) (below, above [][]math.Vector3) {
	const eps = 1e-6 // tiny, so a continuous derivative barely moves but a real jump still shows
	return ext.SurfaceDersAt(1-eps, v, order, 0), ext.SurfaceDersAt(1+eps, v, order, 0)
}

func TestExtendSurfaceG2IsCurvatureContinuous(t *testing.T) {
	t.Parallel()
	s := uPatch(t, 0, func(i, j int) float64 { return 0.5 * float64(i*i) })
	ext, err := ExtendSurface(s, UMaxEdge, 0.5, 2)
	if err != nil {
		t.Fatalf("ExtendSurface: %v", err)
	}
	for _, v := range []float64{0, 0.5, 1} {
		below, above := jointDerivs(ext, v, 2)
		if !below[1][0].IsEqualTo(above[1][0], 1e-3) {
			t.Errorf("G2 extend: tangent jumps at the join (v=%g): %v vs %v", v, below[1][0], above[1][0])
		}
		if !below[2][0].IsEqualTo(above[2][0], 1e-1) {
			t.Errorf("G2 extend: curvature jumps at the join (v=%g): %v vs %v", v, below[2][0], above[2][0])
		}
	}
}

func TestExtendSurfaceG1LeavesCurvatureBreak(t *testing.T) {
	t.Parallel()
	s := uPatch(t, 0, func(i, j int) float64 { return 0.5 * float64(i*i) })
	ext, err := ExtendSurface(s, UMaxEdge, 0.5, 1) // linear/tangent only
	if err != nil {
		t.Fatalf("ExtendSurface: %v", err)
	}
	below, above := jointDerivs(ext, 0.5, 2)
	if !below[1][0].IsEqualTo(above[1][0], 1e-3) {
		t.Errorf("G1 extend should still be tangent-continuous, got %v vs %v", below[1][0], above[1][0])
	}
	// The curved source has nonzero boundary curvature, so a linear extension breaks it.
	if jump := float64(below[2][0].Sub(above[2][0]).Length()); jump < 0.5 {
		t.Errorf("G1 extend should leave a curvature break, got jump %g", jump)
	}
}

func TestExtendSurfaceAllEdges(t *testing.T) {
	t.Parallel()
	s := uPatch(t, 0, func(i, j int) float64 { return 0.3 * float64(i+j) })
	for _, edge := range []Boundary{UMinEdge, UMaxEdge, VMinEdge, VMaxEdge} {
		ext, err := ExtendSurface(s, edge, 0.4, 1)
		if err != nil {
			t.Fatalf("ExtendSurface edge %d: %v", edge, err)
		}
		// Each extension adds control rows/cols in its direction.
		grew := len(ext.Ctrl) > len(s.Ctrl) || len(ext.Ctrl[0]) > len(s.Ctrl[0])
		if !grew {
			t.Errorf("edge %d: extension did not grow the control net", edge)
		}
	}
}

func TestExtendSurfaceValidates(t *testing.T) {
	t.Parallel()
	s := uPatch(t, 0, func(i, j int) float64 { return 0 })
	if _, err := ExtendSurface(s, UMaxEdge, 0.5, 0); err == nil {
		t.Error("order 0 should error")
	}
	if _, err := ExtendSurface(s, UMaxEdge, 0, 2); err == nil {
		t.Error("non-positive distance should error")
	}
}

func TestReverseAndTransposePreserveGeometry(t *testing.T) {
	t.Parallel()
	s := uPatch(t, 0, func(i, j int) float64 { return 0.2 * float64(i*i-j) })
	r := reverseU(s)
	tr := transposeSurface(s)
	for i := 0; i <= 6; i++ {
		for j := 0; j <= 6; j++ {
			u, v := float64(i)/6, float64(j)/6
			if !r.PointAt(1-u, v).IsEqualTo(s.PointAt(u, v), 1e-9) {
				t.Fatalf("reverseU changed geometry at (%g,%g)", u, v)
			}
			if !tr.PointAt(v, u).IsEqualTo(s.PointAt(u, v), 1e-9) {
				t.Fatalf("transpose changed geometry at (%g,%g)", u, v)
			}
		}
	}
}
