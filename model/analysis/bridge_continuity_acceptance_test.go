// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"testing"

	"oblikovati.org/kernel/geom"
)

// edgeRowCurve builds a boundary iso-curve from a surface's u-min/u-max control row.
func edgeRowCurve(t *testing.T, s geom.BSplineSurface, atUMax bool) geom.BSplineCurve {
	t.Helper()
	i := 0
	if atUMax {
		i = len(s.Ctrl) - 1
	}
	c, err := geom.NewBSplineCurve(s.VDegree, s.Ctrl[i], s.Weights[i], s.VKnots)
	if err != nil {
		t.Fatalf("edge row: %v", err)
	}
	return c
}

// F09 bridge acceptance (M36): a G2 bridge between two offset surfaces must read as one fair surface
// across each seam — the F13 cross-edge checker is the gate. A G0 bridge of the same gap is the
// negative control (it breaks curvature), so the gate is meaningful.
func TestBridgeG2IsCurvatureContinuous(t *testing.T) {
	sA := acceptancePatch(t, 0, func(i, j int) float64 { return 0.4 * float64(i*i) })
	sB := acceptancePatch(t, 2, func(i, j int) float64 { return 0.3 * float64((4-i)*(4-i)) })
	cA := edgeRowCurve(t, sA, true)
	cB := edgeRowCurve(t, sB, false)

	sideAGap := func(order int) ContinuityReport {
		br, err := geom.BridgeSurface(cA, cB, sA, sB, geom.UMaxEdge, geom.UMinEdge, order, order)
		if err != nil {
			t.Fatalf("BridgeSurface order %d: %v", order, err)
		}
		// sA's u=1 edge meets the bridge's v=0 edge; the along-param is shared (sA.v ↔ bridge.u).
		return CrossEdgeContinuity(sA, br,
			func(p float64) (u, v float64) { return 1, p },
			func(p float64) (u, v float64) { return p, 0 }, 12)
	}

	g2 := sideAGap(2)
	if g2.MaxGap > 1e-6 {
		t.Errorf("G2 bridge seam should be G0 (no gap): MaxGap = %g", g2.MaxGap)
	}
	if g2.MaxNormalDeg > 0.05 {
		t.Errorf("G2 bridge seam should be G1 (tangent): MaxNormalDeg = %g°", g2.MaxNormalDeg)
	}
	if g2.MaxCurvPct > 1 {
		t.Errorf("G2 bridge seam should be curvature-continuous: MaxCurvPct = %g%%", g2.MaxCurvPct)
	}

	g0 := sideAGap(0)
	if g0.MaxGap > 1e-6 {
		t.Errorf("even a G0 bridge interpolates the edge (no gap): MaxGap = %g", g0.MaxGap)
	}
	if g0.MaxCurvPct < 5 {
		t.Errorf("a G0 bridge should break curvature at the seam: MaxCurvPct = %g%%, want a clear break", g0.MaxCurvPct)
	}
}
