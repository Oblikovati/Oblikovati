// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Surface matching (M36-F05) — the defining Class-A construction move: rebuild one surface against
// a neighbour to a chosen continuity (G0 position, G1 tangent, G2 curvature, G3). It works on the
// control net directly: the first order+1 control rows of the matched surface at its edge are set
// from the target's edge-inward rows by the C^k patch-join (Hermite) conditions, which reflect and
// scale the target's polygon across the seam. Because C^k implies G^k, the result is curvature- (or
// higher-) continuous by construction — verifiable by the F13 cross-edge checker.
//
// The two edges must already correspond pointwise along the seam (same count of control columns and
// matching parameterization — F01's make-compatible is the precursor); only the cross-edge rows of
// the matched surface change, so its knots and weights are preserved.

// Boundary names one of a surface's four control-net edges.
type Boundary int

const (
	// UMinEdge is the u=0 edge, UMaxEdge the u=1 edge, VMinEdge v=0, VMaxEdge v=1.
	UMinEdge Boundary = iota
	UMaxEdge
	VMinEdge
	VMaxEdge
)

// MatchSurface returns a copy of s whose sEdge boundary is matched to t's tEdge boundary to the
// given continuity order (0=G0 … 3=G3). It errors when the order exceeds either surface's available
// cross-edge rows or the two edges have different control-column counts.
//
// Example: matched, _ := MatchSurface(flat, neighbour, UMinEdge, UMaxEdge, 2) // G2 across the seam.
func MatchSurface(s, t BSplineSurface, sEdge, tEdge Boundary, order int) (BSplineSurface, error) {
	if order < 0 || order > 3 {
		return BSplineSurface{}, fmt.Errorf("geom: match continuity order %d must be 0..3", order)
	}
	srows := crossRows(s, sEdge)
	trows := crossRows(t, tEdge)
	if order+1 > len(srows) || order+1 > len(trows) {
		return BSplineSurface{}, fmt.Errorf("geom: match to G%d needs %d cross-edge rows (s has %d, t has %d)", order, order+1, len(srows), len(trows))
	}
	if len(srows[0]) != len(trows[0]) {
		return BSplineSurface{}, fmt.Errorf("geom: match needs equal edge column counts (s %d, t %d)", len(srows[0]), len(trows[0]))
	}
	cs := edgeDerivCoeffs(crossKnots(s, sEdge), crossDegree(s, sEdge), order, atMax(sEdge), into(false, sEdge))
	ct := edgeDerivCoeffs(crossKnots(t, tEdge), crossDegree(t, tEdge), order, atMax(tEdge), into(true, tEdge))
	matched := matchedRows(srows, trows, cs, ct, order)
	return setCrossRows(s, sEdge, matched)
}

// matchedRows sets the matched surface's first order+1 edge-inward control rows so that, at every
// control column, its k-th into-the-seam cross-derivative control curve equals the target's. Each
// derivative is the basis-weighted sum of the edge-inward rows (cs/ct, the knot-correct coefficients),
// and the resulting per-k system is lower-triangular, so it solves directly. Rows beyond `order` keep
// their original positions.
func matchedRows(srows, trows [][]math.Point3, cs, ct [][]float64, order int) [][]math.Point3 {
	out := make([][]math.Point3, len(srows))
	for k := range out {
		out[k] = append([]math.Point3(nil), srows[k]...)
	}
	for a := 0; a < len(srows[0]); a++ {
		col := make([]math.Point3, order+1)
		for k := 0; k <= order; k++ {
			col[k] = solveRow(trows, col, cs, ct, k, a)
			out[k][a] = col[k]
		}
	}
	return out
}

// solveRow solves the k-th matching equation Σ_i cs[k][i]·Qᵢ = Σ_j ct[k][j]·Tⱼ for the matched
// surface's k-th edge-inward control point Qₖ at column a (cs is lower-triangular, so cs[k][k] is the
// pivot and only Q₀…Q_{k−1} are already known).
func solveRow(trows [][]math.Point3, q []math.Point3, cs, ct [][]float64, k, a int) math.Point3 {
	var rhs math.Vector3
	for j := 0; j <= k; j++ {
		rhs = rhs.Add(trows[j][a].AsVector().Scale(math.Scalar(ct[k][j])))
	}
	for i := 0; i < k; i++ {
		rhs = rhs.Sub(q[i].AsVector().Scale(math.Scalar(cs[k][i])))
	}
	return rhs.Scale(math.Scalar(1 / cs[k][k])).AsPoint()
}

// edgeDerivCoeffs returns coeff[k][j] such that the surface's k-th cross-derivative at the edge, in
// the `into` direction, has control-curve value Σ_j coeff[k][j]·(edge-inward row j). It evaluates the
// B-spline basis derivatives at the clamped edge (knot-correct for any interior knots) and applies
// the into-direction sign for odd orders.
func edgeDerivCoeffs(knots []float64, deg, order int, edgeAtMax bool, intoSign float64) [][]float64 {
	ders := edgeBasisDers(knots, deg, order, edgeAtMax)
	c := make([][]float64, order+1)
	for k := 0; k <= order; k++ {
		c[k] = make([]float64, order+1)
		s := 1.0
		if k%2 == 1 {
			s = intoSign
		}
		for j := 0; j <= order; j++ {
			c[k][j] = s * ders[k][j]
		}
	}
	return c
}

// edgeBasisDers returns ders[k][j] = the k-th derivative (w.r.t. the surface's natural +param) of the
// basis function on edge-inward row j, at the clamped edge. The max edge reverses the local basis
// order so j still indexes from the edge inward.
func edgeBasisDers(knots []float64, deg, order int, edgeAtMax bool) [][]float64 {
	nctrl := len(knots) - deg - 1
	u := knots[deg]
	span := deg
	if edgeAtMax {
		u = knots[nctrl]
		span = findSpan(nctrl-1, deg, u, knots)
	}
	raw := dersBasisFuns(span, deg, u, order, knots)
	out := make([][]float64, order+1)
	for k := 0; k <= order; k++ {
		out[k] = make([]float64, order+1)
		for j := 0; j <= order && j <= deg; j++ {
			if edgeAtMax {
				out[k][j] = raw[k][deg-j] // edge-inward row j = local basis (deg−j) at the max end
			} else {
				out[k][j] = raw[k][j]
			}
		}
	}
	return out
}

// into returns the sign of the "into the seam" parameter direction for one side: for the matched
// surface it is into its own interior (+ at a min edge, − at a max edge); for the target it is out of
// its interior across the edge (+ at a max edge, − at a min edge).
func into(target bool, edge Boundary) float64 {
	if (atMax(edge)) == target {
		return 1
	}
	return -1
}

// atMax reports whether the edge is the max end of its parametric direction.
func atMax(edge Boundary) bool { return edge == UMaxEdge || edge == VMaxEdge }

// crossDegree returns the surface degree in the edge's cross direction.
func crossDegree(s BSplineSurface, edge Boundary) int {
	if edge == UMinEdge || edge == UMaxEdge {
		return s.UDegree
	}
	return s.VDegree
}

// crossKnots returns the surface knot vector in the edge's cross direction.
func crossKnots(s BSplineSurface, edge Boundary) []float64 {
	if edge == UMinEdge || edge == UMaxEdge {
		return s.UKnots
	}
	return s.VKnots
}

// crossRows returns the control rows perpendicular to the edge, ordered from the edge inward; each
// inner slice is indexed by the along-edge position.
func crossRows(s BSplineSurface, edge Boundary) [][]math.Point3 {
	nu, nv := len(s.Ctrl), len(s.Ctrl[0])
	switch edge {
	case UMaxEdge:
		return uRows(s, nu, func(k int) int { return nu - 1 - k })
	case UMinEdge:
		return uRows(s, nu, func(k int) int { return k })
	case VMaxEdge:
		return vRows(s, nv, func(k int) int { return nv - 1 - k })
	default: // VMinEdge
		return vRows(s, nv, func(k int) int { return k })
	}
}

// uRows returns the u-indexed rows (along-edge = v) ordered by the edge-inward index map.
func uRows(s BSplineSurface, nu int, idx func(int) int) [][]math.Point3 {
	rows := make([][]math.Point3, nu)
	for k := 0; k < nu; k++ {
		rows[k] = append([]math.Point3(nil), s.Ctrl[idx(k)]...)
	}
	return rows
}

// vRows returns the v-indexed rows (along-edge = u) ordered by the edge-inward index map.
func vRows(s BSplineSurface, nv int, idx func(int) int) [][]math.Point3 {
	rows := make([][]math.Point3, nv)
	for k := 0; k < nv; k++ {
		row := make([]math.Point3, len(s.Ctrl))
		for i := range s.Ctrl {
			row[i] = s.Ctrl[i][idx(k)]
		}
		rows[k] = row
	}
	return rows
}

// setCrossRows writes the first len(rows) edge-inward control rows back into a copy of s, preserving
// its knots and weights, and returns the rebuilt surface.
func setCrossRows(s BSplineSurface, edge Boundary, rows [][]math.Point3) (BSplineSurface, error) {
	ctrl := copyNet(s.Ctrl)
	nu, nv := len(ctrl), len(ctrl[0])
	for k := range rows {
		switch edge {
		case UMaxEdge:
			ctrl[nu-1-k] = rows[k]
		case UMinEdge:
			ctrl[k] = rows[k]
		case VMaxEdge:
			for i := range ctrl {
				ctrl[i][nv-1-k] = rows[k][i]
			}
		default: // VMinEdge
			for i := range ctrl {
				ctrl[i][k] = rows[k][i]
			}
		}
	}
	return NewBSplineSurface(s.UDegree, s.VDegree, ctrl, s.Weights, s.UKnots, s.VKnots)
}
