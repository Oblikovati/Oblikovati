// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"sync"

	"oblikovati.org/math"
)

// bsplineSeedGrid memoizes a B-spline surface's inversion seed lattice: the knot-span seed
// parameters in each direction and the surface point at every (u, v) seed pair. Both are pure
// functions of the surface's knots, degrees and control net, which NewBSplineSurface deep-copies
// and nothing mutates afterwards — the immutability the memo rests on. Before this, every ParamAt
// call rebuilt the lattice and re-evaluated ≥256 seed points through basisFuns, and inversion sits
// inside the SSI corrector and every closest-point recursion, which call it thousands of times per
// trace (Oblikovati/Oblikovati#3490 — a blend-parity corpus body spent 8m30s of a CI leg there).
//
// The grid is shared by every copy of the surface value (the field is a pointer, set once at
// construction) and filled lazily under sync.Once, so concurrent inversions race only on who fills
// it first, never on its content.
type bsplineSeedGrid struct {
	once sync.Once
	us   []float64
	vs   []float64
	pts  [][]math.Point3
}

// seedLattice returns the surface's seed parameters and their evaluated points, from the memo when
// the surface was built by NewBSplineSurface. A zero-value literal (test fixtures) has no memo slot
// to fill and computes the same lattice per call — the verdict is identical either way, which
// TestSeedMemoChangesNoInversion asserts point for point rather than assumes.
func (s BSplineSurface) seedLattice() (us, vs []float64, pts [][]math.Point3) {
	if s.seed == nil {
		us = knotSpanSeedParams(s.UKnots, s.UDegree)
		vs = knotSpanSeedParams(s.VKnots, s.VDegree)
		return us, vs, evalSeedGrid(s, us, vs)
	}
	s.seed.once.Do(func() {
		s.seed.us = knotSpanSeedParams(s.UKnots, s.UDegree)
		s.seed.vs = knotSpanSeedParams(s.VKnots, s.VDegree)
		s.seed.pts = evalSeedGrid(s, s.seed.us, s.seed.vs)
	})
	return s.seed.us, s.seed.vs, s.seed.pts
}

// evalSeedGrid evaluates the surface at every seed pair — the once-per-surface half of the work.
func evalSeedGrid(s BSplineSurface, us, vs []float64) [][]math.Point3 {
	pts := make([][]math.Point3, len(us))
	for i, u := range us {
		row := make([]math.Point3, len(vs))
		for j, v := range vs {
			row[j] = s.PointAt(u, v)
		}
		pts[i] = row
	}
	return pts
}

// nearestSeedPoint returns the seed pair whose evaluated point lies closest to q — nearestSeed with
// the evaluations already paid for.
func nearestSeedPoint(us, vs []float64, pts [][]math.Point3, q math.Point3) (float64, float64) {
	bu, bv, bd := us[0], vs[0], stdmath.Inf(1)
	for i, u := range us {
		for j, v := range vs {
			if d := float64(pts[i][j].DistanceSquaredTo(q)); d < bd {
				bu, bv, bd = u, v, d
			}
		}
	}
	return bu, bv
}
