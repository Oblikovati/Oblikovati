// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Rigorous per-box tangent-magnitude bounds for the SSI quadtree prune (Oblikovati#1608,
// audit A12). The signed-distance field f the seeder marches is 1-Lipschitz in 3D, so
// across a parameter cell |Δf| ≤ ∫|dS| ≤ (max|S_u|)·Δu + (max|S_v|)·Δv. The prune discards
// a cell only when its smallest corner |f| exceeds that bound; the bound must therefore be
// a TRUE upper bound on the tangent magnitude anywhere in the cell, not a sampled estimate.
// The retired seeder inflated the four corner tangents by a guessed factor of 2.0, which
// still UNDER-estimated the interior tangent on a high-curvature span and silently pruned
// cells that held a crossing. These closed-form / hodograph bounds replace that guess:
//   - analytic surfaces: their tangent magnitude has a known closed form (Piegl & Tiller);
//   - non-rational B-spline: the derivative surface is itself a B-spline whose control
//     points bound |S_u|,|S_v| by the convex-hull property (P&T §3.3, the hodograph net).
// A surface with no rigorous bound (a rational NURBS, an unknown type) returns ok=false so
// the seeder records the fallback (issue #1608 point 5) instead of pruning on a guess.

// boxTangentBounder reports rigorous upper bounds on |∂S/∂u| and |∂S/∂v| anywhere in the
// parameter box [u0,u1]×[v0,v1]. ok=false means no certified bound is available and the
// caller must fall back (and record the decline). It is consumed by the SSI seeder prune.
type boxTangentBounder interface {
	tangentBoundOverBox(u0, u1, v0, v1 float64) (maxSu, maxSv float64, ok bool)
}

// tangentBoundOverBox: the plane's partials are the constant unit in-plane axes.
func (p Plane) tangentBoundOverBox(_, _, _, _ float64) (float64, float64, bool) {
	return 1, 1, true
}

// tangentBoundOverBox: |S_u| = Radius (around), |S_v| = 1 (unit axis) — both constant.
func (c Cylinder) tangentBoundOverBox(_, _, _, _ float64) (float64, float64, bool) {
	return c.Radius, 1, true
}

// tangentBoundOverBox: on a cone |S_u| = v·tan(HalfAngle) grows with v, so it is bounded by
// the larger |v| end of the box; |S_v| = sqrt(1+tan²) is constant.
func (c Cone) tangentBoundOverBox(_, _, v0, v1 float64) (float64, float64, bool) {
	t := stdmath.Abs(stdmath.Tan(c.HalfAngle))
	vMax := stdmath.Max(stdmath.Abs(v0), stdmath.Abs(v1))
	return vMax * t, stdmath.Sqrt(1 + t*t), true
}

// tangentBoundOverBox: |S_u| = R·cos v ≤ R, |S_v| = R (P&T sphere parameterisation).
func (s Sphere) tangentBoundOverBox(_, _, _, _ float64) (float64, float64, bool) {
	return s.Radius, s.Radius, true
}

// tangentBoundOverBox: |S_u| = Major + Minor·cos v ≤ Major+Minor, |S_v| = Minor.
func (t Torus) tangentBoundOverBox(_, _, _, _ float64) (float64, float64, bool) {
	return t.MajorRadius + t.MinorRadius, t.MinorRadius, true
}

// tangentBoundOverBox bounds a non-rational B-spline surface's partials by its hodograph
// (derivative) control net: S_u is a B-spline in the control points Q^u_{i,j} =
// deg·(P_{i+1,j}−P_{i,j})/(U_{i+deg+1}−U_{i+1}), a convex combination of them, so |S_u| ≤
// max|Q^u| (P&T §3.3). The bound is over the whole net (an over-estimate for a sub-box, but
// always sound). A rational net has no such simple hodograph, so it declines (ok=false).
func (s BSplineSurface) tangentBoundOverBox(_, _, _, _ float64) (float64, float64, bool) {
	if !s.hasUnitWeights() {
		return 0, 0, false
	}
	return s.hodographBound(s.UDegree, s.UKnots, true), s.hodographBound(s.VDegree, s.VKnots, false), true
}

// hodographBound returns max|Q| over the derivative control net in one direction: alongU
// selects the u-differences P_{i+1,j}−P_{i,j} scaled by deg/(knot[i+deg+1]−knot[i+1]),
// else the v-differences. Zero-width knot spans (clamped ends never differenced) are skipped.
func (s BSplineSurface) hodographBound(degree int, knots []float64, alongU bool) float64 {
	best := 0.0
	nu, nv := len(s.Ctrl), len(s.Ctrl[0])
	iMax, jMax := nu, nv
	if alongU {
		iMax = nu - 1
	} else {
		jMax = nv - 1
	}
	for i := 0; i < iMax; i++ {
		for j := 0; j < jMax; j++ {
			best = stdmath.Max(best, hodographCoeff(s.Ctrl, i, j, degree, knots, alongU))
		}
	}
	return best
}

// hodographCoeff is |Q_{i,j}| for one derivative-net control point (0 if its knot span is
// degenerate: a repeated knot contributes no derivative there).
func hodographCoeff(ctrl [][]math.Point3, i, j, degree int, knots []float64, alongU bool) float64 {
	var diff math.Vector3
	var span float64
	if alongU {
		diff, span = ctrl[i][j].VectorTo(ctrl[i+1][j]), knots[i+degree+1]-knots[i+1]
	} else {
		diff, span = ctrl[i][j].VectorTo(ctrl[i][j+1]), knots[j+degree+1]-knots[j+1]
	}
	if span <= knotEps {
		return 0
	}
	return float64(degree) * float64(diff.Length()) / span
}

// hasUnitWeights reports whether every control weight is 1 (within knotEps), i.e. the
// surface is polynomial (non-rational) and its hodograph is the plain control differences.
func (s BSplineSurface) hasUnitWeights() bool {
	for _, row := range s.Weights {
		for _, w := range row {
			if stdmath.Abs(w-1) > knotEps {
				return false
			}
		}
	}
	return true
}
