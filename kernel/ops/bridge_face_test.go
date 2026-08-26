// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// bridgePatchBody builds a 5×5 bicubic NURBS surface body over [xoff,xoff+1]×[0,1] with per-control
// z = z(i,j) (a curved panel to bridge).
func bridgePatchBody(t *testing.T, xoff float64, z func(i, j int) float64) *topo.Body {
	t.Helper()
	const n = 5
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range n {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := range n {
			ctrl[i][j] = math.P3(math.Scalar(xoff+float64(i)*0.25), math.Scalar(float64(j)*0.25), math.Scalar(z(i, j)))
			w[i][j] = 1
		}
	}
	k := []float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1}
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, k, k)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	return surfaceFaceBody(t, s)
}

// TestBridgeBodiesG2IsValidSurface bridges the gap between two curved panels (x∈[0,1] and x∈[2,3]) at
// G2 on both sides and checks the result is a valid single-face surface body spanning the gap.
func TestBridgeBodiesG2IsValidSurface(t *testing.T) {
	a := bridgePatchBody(t, 0, func(i, j int) float64 { return 0.4 * float64(i*i) })
	b := bridgePatchBody(t, 2, func(i, j int) float64 { return 0.3 * float64((4-i)*(4-i)) })
	out, err := ops.BridgeBodies(a, b, 2, 2)
	if err != nil {
		t.Fatalf("BridgeBodies: %v", err)
	}
	bs, ok := out.Faces()[0].Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("bridge face is %T, want geom.BSplineSurface", out.Faces()[0].Geometry())
	}
	// The bridge spans the gap: its mid-section sits between the two panels (x≈1.5).
	if x := float64(bs.PointAt(0.5, 0.5).X); x < 1.2 || x > 1.8 {
		t.Errorf("bridge mid x = %g, want between the panels (~1.5)", x)
	}
}

func TestBridgeBodiesErrorsOnNonNurbs(t *testing.T) {
	box := csgBox(math.P3(0, 0, 0), 1, 1, 1)
	good := bridgePatchBody(t, 2, func(i, j int) float64 { return 0 })
	if _, err := ops.BridgeBodies(box, good, 1, 1); err == nil {
		t.Error("bridging a non-NURBS body should error")
	}
}
