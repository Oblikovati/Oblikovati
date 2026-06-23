// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Surface fairing (M36-F04): smooth curvature wrinkles out of a surface by relaxing its INTERIOR
// control points toward a minimal-strain-energy configuration (discrete Laplacian / membrane
// smoothing), while HOLDING a band of boundary control rows fixed so the surface keeps its boundary
// position/tangent/curvature — hence its continuity to neighbours (G0/G1/G2). The held band depth is
// holdOrder+1 rows from each edge (G2 holds 3 rows, so the boundary's 2nd derivative is preserved).

// FairSurface returns s faired: holdOrder selects the boundary continuity to preserve (0=G0/position,
// 1=G1/tangent, 2=G2/curvature), strength∈(0,1] is the per-iteration relaxation, iterations the count.
// Control points within holdOrder+1 of any edge are fixed; interior points relax (Jacobi) toward the
// average of their 4 grid neighbours. Knots, degree and weights are unchanged. A surface with no
// interior points (too small for the held band) is returned unchanged.
func FairSurface(s BSplineSurface, holdOrder int, strength float64, iterations int) BSplineSurface {
	nu, nv := len(s.Ctrl), len(s.Ctrl[0])
	band := holdOrder + 1
	if strength <= 0 || iterations <= 0 || nu <= 2*band || nv <= 2*band {
		return s
	}
	if strength > 1 {
		strength = 1
	}
	cur := cloneNet(s.Ctrl)
	for it := 0; it < iterations; it++ {
		next := cloneNet(cur)
		for i := band; i < nu-band; i++ {
			for j := band; j < nv-band; j++ {
				avg := neighbourAverage(cur, i, j)
				next[i][j] = lerpP(cur[i][j], avg, strength)
			}
		}
		cur = next
	}
	out, err := NewBSplineSurface(s.UDegree, s.VDegree, cur, cloneWeights(s.Weights), s.UKnots, s.VKnots)
	if err != nil {
		return s
	}
	return out
}

// neighbourAverage is the mean of control point (i,j)'s four grid neighbours — the discrete Laplacian
// target that minimizes membrane strain energy.
func neighbourAverage(net [][]math.Point3, i, j int) math.Point3 {
	n := net[i-1][j].AsVector().Add(net[i+1][j].AsVector()).Add(net[i][j-1].AsVector()).Add(net[i][j+1].AsVector())
	return n.Scale(0.25).AsPoint()
}

// cloneNet deep-copies a control net.
func cloneNet(net [][]math.Point3) [][]math.Point3 {
	out := make([][]math.Point3, len(net))
	for i := range net {
		out[i] = append([]math.Point3(nil), net[i]...)
	}
	return out
}

// cloneWeights deep-copies a weight net.
func cloneWeights(w [][]float64) [][]float64 {
	out := make([][]float64, len(w))
	for i := range w {
		out[i] = append([]float64(nil), w[i]...)
	}
	return out
}
