// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// F07 boundary-fill acceptance gate (M36): a G2 fill must read as one fair surface across the seam to
// its neighbour. We build a curved neighbour, fill the opening against it at G2, and use the F13
// cross-edge checker (CrossEdgeContinuity) as the numeric gate — the seam must be G0 (no gap), G1 (no
// tangent break) and G2 (curvature matches), the same gate a human applies with reflection lines.

const fillAcceptKnots5 = 5 // control points per fill boundary (degree-3, one interior knot)

func fillKnots() []float64 { return []float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1} }

// acceptancePatch is a 5×5 bicubic surface over [xoff,xoff+1]×[0,1] with per-control z = z(i,j).
func acceptancePatch(t *testing.T, xoff float64, z func(i, j int) float64) geom.BSplineSurface {
	t.Helper()
	const n = fillAcceptKnots5
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := 0; i < n; i++ {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			ctrl[i][j] = math.P3(math.Scalar(xoff+float64(i)*0.25), math.Scalar(float64(j)*0.25), math.Scalar(z(i, j)))
			w[i][j] = 1
		}
	}
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, fillKnots(), fillKnots())
	if err != nil {
		t.Fatalf("acceptance patch: %v", err)
	}
	return s
}

func fillBoundary(t *testing.T, pts ...math.Point3) geom.BSplineCurve {
	t.Helper()
	c, err := geom.NewBSplineCurveUniformWeights(3, pts, fillKnots())
	if err != nil {
		t.Fatalf("boundary curve: %v", err)
	}
	return c
}

// TestG2FillIsCurvatureContinuousToNeighbour fills the opening west of a curved neighbour at G2 and
// asserts the shared seam is curvature-continuous by the F13 gate.
func TestG2FillIsCurvatureContinuousToNeighbour(t *testing.T) {
	nb := acceptancePatch(t, -1, func(i, j int) float64 { return 0.5 * float64(i*i) })
	row := nb.Ctrl[len(nb.Ctrl)-1] // neighbour's u-max iso (constant along v: x=0, z=8)
	d0, err := geom.NewBSplineCurveUniformWeights(3, row, fillKnots())
	if err != nil {
		t.Fatalf("d0 iso: %v", err)
	}
	z0 := float64(row[0].Z)
	c0 := fillBoundary(t, math.P3(0, 0, math.Scalar(z0)), math.P3(0.25, 0, math.Scalar(z0*0.75)), math.P3(0.5, 0, math.Scalar(z0*0.5)), math.P3(0.75, 0, math.Scalar(z0*0.25)), math.P3(1, 0, 0))
	c1 := fillBoundary(t, math.P3(0, 1, math.Scalar(z0)), math.P3(0.25, 1, math.Scalar(z0*0.75)), math.P3(0.5, 1, math.Scalar(z0*0.5)), math.P3(0.75, 1, math.Scalar(z0*0.25)), math.P3(1, 1, 0))
	d1 := fillBoundary(t, math.P3(1, 0, 0), math.P3(1, 0.25, 0), math.P3(1, 0.5, 0), math.P3(1, 0.75, 0), math.P3(1, 1, 0))

	fill, err := geom.FillSurface(c0, c1, d0, d1, [4]geom.FillSide{2: {Adjacent: nb, AdjEdge: geom.UMaxEdge, Order: 2}})
	if err != nil {
		t.Fatalf("FillSurface G2: %v", err)
	}
	// F13 gate across the shared seam: neighbour u=1 meets fill u=0.
	nbEdge := func(p float64) (u, v float64) { return 1, p }
	fillEdge := func(p float64) (u, v float64) { return 0, p }
	rep := CrossEdgeContinuity(nb, fill, nbEdge, fillEdge, 12)
	if rep.MaxGap > 1e-6 {
		t.Errorf("G2 fill seam should be G0 (no gap): MaxGap = %g", rep.MaxGap)
	}
	if rep.MaxNormalDeg > 0.05 {
		t.Errorf("G2 fill seam should be G1 (tangent): MaxNormalDeg = %g°", rep.MaxNormalDeg)
	}
	if rep.MaxCurvPct > 1 {
		t.Errorf("G2 fill seam should be curvature-continuous: MaxCurvPct = %g%%", rep.MaxCurvPct)
	}
}

// TestG0FillBreaksCurvatureAtSeam is the negative control: a position-only (G0) fill against the same
// curved neighbour reads as a curvature break, so the gate above is meaningful.
func TestG0FillBreaksCurvatureAtSeam(t *testing.T) {
	nb := acceptancePatch(t, -1, func(i, j int) float64 { return 0.5 * float64(i*i) })
	row := nb.Ctrl[len(nb.Ctrl)-1]
	d0, _ := geom.NewBSplineCurveUniformWeights(3, row, fillKnots())
	z0 := float64(row[0].Z)
	c0 := fillBoundary(t, math.P3(0, 0, math.Scalar(z0)), math.P3(0.25, 0, math.Scalar(z0*0.75)), math.P3(0.5, 0, math.Scalar(z0*0.5)), math.P3(0.75, 0, math.Scalar(z0*0.25)), math.P3(1, 0, 0))
	c1 := fillBoundary(t, math.P3(0, 1, math.Scalar(z0)), math.P3(0.25, 1, math.Scalar(z0*0.75)), math.P3(0.5, 1, math.Scalar(z0*0.5)), math.P3(0.75, 1, math.Scalar(z0*0.25)), math.P3(1, 1, 0))
	d1 := fillBoundary(t, math.P3(1, 0, 0), math.P3(1, 0.25, 0), math.P3(1, 0.5, 0), math.P3(1, 0.75, 0), math.P3(1, 1, 0))

	fill, err := geom.FillSurface(c0, c1, d0, d1, [4]geom.FillSide{}) // all G0
	if err != nil {
		t.Fatalf("FillSurface G0: %v", err)
	}
	rep := CrossEdgeContinuity(nb, fill, func(p float64) (float64, float64) { return 1, p }, func(p float64) (float64, float64) { return 0, p }, 12)
	if rep.MaxGap > 1e-6 {
		t.Errorf("even a G0 fill interpolates the boundary (no gap): MaxGap = %g", rep.MaxGap)
	}
	if rep.MaxCurvPct < 5 {
		t.Errorf("a G0 fill should break curvature at the seam: MaxCurvPct = %g%%, want a clear break", rep.MaxCurvPct)
	}
}
