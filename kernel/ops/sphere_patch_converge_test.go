// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestSpherePatchDeficitConverges pins the patchgridcap fix's signature property: on a
// budget-honoured patch, the inscribed-area deficit against the closed form SHRINKS as the chord
// tolerance tightens, instead of plateauing. The old silent per-axis clamp (patchGridCap=80) held
// the deficit FLAT at every swept tolerance — a plateau twice misread as a converged value
// (sphere-notch-report.md; patchgridcap-report.md). RED with the clamp restored: all three sweeps
// then read the same clamped deficit and the shrink ratios fail.
//
// Fixture: R=13, 45°-polar cap rim (gnomonic chart, bbox 26: ~161² cells at ct 1e-3 and ~322²
// at 2.5e-4 — inside the 2^18 budget, so every swept grid is tolerance-honoured). The rim is
// sampled at 4096 points so the fixed boundary's own inscription (~2π³r²/3N² ≈ 1e-4) stays two
// decades under the finest interior deficit being asserted.
func TestSpherePatchDeficitConverges(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~5s): `make test-corpus`")
	}
	t.Parallel()
	const radius = 13.0
	sph, err := geom.NewSphere(math.P3(0, 0, 0), radius)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	closed := 2 * stdmath.Pi * radius * radius * (1 - stdmath.Sqrt2/2) // 2πR²(1−cos 45°)
	deficits := make([]float64, 0, 3)
	for _, ct := range []float64{1e-2, 1e-3, 2.5e-4} {
		m, ok := spherePatchMesh(nil, sph, cap45Rim(radius, 4096), nil, Quality{ChordTolerance: ct, AngleTolerance: stdmath.Pi / 180})
		if !ok {
			t.Fatalf("spherePatchMesh declined the 45° cap rim at ct=%g", ct)
		}
		if hasDiag(m.Diagnostics, CodeTessellateCapSaturated) {
			t.Fatalf("ct=%g saturated the cell budget — the fixture must stay budget-honoured for the sweep to mean convergence", ct)
		}
		deficits = append(deficits, stdmath.Abs(MeshArea(m)-closed))
	}
	for i := 1; i < len(deficits); i++ {
		if deficits[i] > 0.6*deficits[i-1] {
			t.Errorf("deficit plateaus: |Δ| %.6f → %.6f across a 4–10× tolerance tightening (want < 0.6×) — a density floor is back",
				deficits[i-1], deficits[i])
		}
	}
}

// cap45Rim samples the 45°-polar-angle cap rim of an origin-centred radius-r sphere at n points.
func cap45Rim(r float64, n int) []math.Point3 {
	ring := make([]math.Point3, n)
	for i := range ring {
		phi := 2 * stdmath.Pi * float64(i) / float64(n)
		s, c := stdmath.Sincos(phi)
		ring[i] = math.P3(math.Scalar(r*stdmath.Sin(stdmath.Pi/4)*c),
			math.Scalar(r*stdmath.Sin(stdmath.Pi/4)*s), math.Scalar(r*stdmath.Cos(stdmath.Pi/4)))
	}
	return ring
}
