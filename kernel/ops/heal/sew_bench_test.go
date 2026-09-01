// SPDX-License-Identifier: GPL-2.0-only

package heal

import (
	"fmt"
	stdrand "math/rand"
	"testing"

	"oblikovati.org/math"
)

// sewBenchEndpoints spreads m endpoints over a shell-sized domain with every second point
// jittered within tol of another — the open-quilt boundary soup Sew clusters.
func sewBenchEndpoints(m int, tol float64) []math.Point3 {
	rng := stdrand.New(stdrand.NewSource(1607))
	pts := make([]math.Point3, 0, m)
	for len(pts) < m {
		p := math.P3(rng.Float64()*100, rng.Float64()*100, rng.Float64()*100)
		pts = append(pts, p, p.TranslateBy(math.V3(tol/2, 0, 0)))
	}
	return pts
}

// BenchmarkEndpointClusterSnap locks the sew clustering's asymptotics at 1×/4×/16× endpoint
// counts (#1607): with the tol-cell grid hash built first, per-tier cost must grow near
// linearly where the retired pre-pass grew with m².
func BenchmarkEndpointClusterSnap(b *testing.B) {
	const tol = 0.05
	for _, m := range []int{1024, 4096, 16384} {
		pts := sewBenchEndpoints(m, tol)
		b.Run(fmt.Sprintf("endpoints_%d", m), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				endpointClusterSnap(pts, tol)
			}
		})
	}
}
