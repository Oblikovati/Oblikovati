// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Surface extension (M36-F11) — lengthen a B-spline surface past one of its edges with a chosen
// continuation: linear/tangent (order 1, a straight ruling along the boundary tangent) or curvature
// (order 2, continuing the boundary's second derivative). The appended span is the Taylor polynomial
// of the surface at the edge, expanded to that order and laid down as a new Bézier-degree span, so
// the join is G^order by construction (the F13 checker confirms it across the original boundary).
//
// Any of the four edges is supported by reorienting the surface so the edge is the u-max end,
// extending there, and reorienting back — so the core only handles one case.

// ExtendSurface returns s lengthened past `edge` by `distance` (model-space, along the boundary
// tangent) with the given continuation order (1=linear/tangent G1, 2=curvature G2, 3=G3). It errors
// on a non-positive distance or an order outside 1..3.
//
// Example: longer, _ := ExtendSurface(s, UMaxEdge, 5, 2) // extend the far edge, curvature-continuous.
func ExtendSurface(s BSplineSurface, edge Boundary, distance float64, order int) (BSplineSurface, error) {
	if order < 1 || order > 3 {
		return BSplineSurface{}, fmt.Errorf("geom: extend continuation order %d must be 1..3", order)
	}
	if distance <= 0 {
		return BSplineSurface{}, fmt.Errorf("geom: extend distance %g must be positive", distance)
	}
	switch edge {
	case UMaxEdge:
		return extendUMax(s, distance, order)
	case UMinEdge:
		return reorient(s, distance, order, reverseU)
	case VMaxEdge:
		return reorient(s, distance, order, transposeSurface)
	default: // VMinEdge
		return reorient(s, distance, order, func(x BSplineSurface) BSplineSurface { return reverseU(transposeSurface(x)) })
	}
}

// reorient extends a non-u-max edge by mapping it to u-max with `to`, extending, and mapping back
// (the transforms are their own inverses).
func reorient(s BSplineSurface, distance float64, order int, to func(BSplineSurface) BSplineSurface) (BSplineSurface, error) {
	ext, err := extendUMax(to(s), distance, order)
	if err != nil {
		return BSplineSurface{}, err
	}
	return to(ext), nil
}

// extendUMax appends a continuation span past the u=max edge. Each v-column's boundary derivatives
// (up to order) define a Taylor polynomial, converted to degree-p Bézier control rows and appended
// with a new clamped knot span; the join knot keeps the position only (the control points carry the
// geometric continuity).
func extendUMax(s BSplineSurface, distance float64, order int) (BSplineSurface, error) {
	p, nv := s.UDegree, len(s.Ctrl[0])
	srows := crossRows(s, UMaxEdge)
	coeff := edgeDerivCoeffs(s.UKnots, p, order, true, 1) // +u (outward) derivatives by edge-inward row
	d := boundaryDerivatives(srows, coeff, order, nv)
	e := extensionParam(d, distance)
	ctrl := copyNet(s.Ctrl)
	weights := copyWeights(s.Weights)
	for j := 1; j <= p; j++ {
		row, wr := make([]math.Point3, nv), unitWeights(nv)
		for a := 0; a < nv; a++ {
			row[a] = bezierExtensionPoint(d, j, p, e, order, a)
		}
		ctrl, weights = append(ctrl, row), append(weights, wr)
	}
	umax := s.UKnots[len(s.UKnots)-1-p]
	newU := append(append([]float64(nil), s.UKnots[:len(s.UKnots)-1]...), repeatedKnot(umax+e, p+1)...)
	return NewBSplineSurface(p, s.VDegree, ctrl, weights, newU, s.VKnots)
}

// boundaryDerivatives returns d[k][a] = the k-th outward cross-derivative at the edge for control
// column a, as the basis-weighted sum of the edge-inward rows.
func boundaryDerivatives(srows [][]math.Point3, coeff [][]float64, order, nv int) [][]math.Vector3 {
	d := make([][]math.Vector3, order+1)
	for k := 0; k <= order; k++ {
		d[k] = make([]math.Vector3, nv)
		for a := 0; a < nv; a++ {
			var v math.Vector3
			for j := 0; j <= order; j++ {
				v = v.Add(srows[j][a].AsVector().Scale(math.Scalar(coeff[k][j])))
			}
			d[k][a] = v
		}
	}
	return d
}

// bezierExtensionPoint returns the j-th appended degree-p Bézier control point of the Taylor
// extension over the local span [0, e] at column a: the monomial→Bézier image of Σ_k (d_k/k!) t^k.
func bezierExtensionPoint(d [][]math.Vector3, j, p int, e float64, order, a int) math.Point3 {
	var v math.Vector3
	for k := 0; k <= order && k <= j; k++ {
		coef := binomial(j, k) / binomial(p, k) * powInt(e, k) / factorial(k)
		v = v.Add(d[k][a].Scale(math.Scalar(coef)))
	}
	return v.AsPoint()
}

// extensionParam converts a model-space extension distance to a parameter length, using the mean
// boundary tangent speed so the appended span is roughly `distance` long.
func extensionParam(d [][]math.Vector3, distance float64) float64 {
	speed, n := 0.0, len(d[1])
	for _, t := range d[1] {
		speed += float64(t.Length())
	}
	if speed < 1e-12 {
		return distance
	}
	return distance / (speed / float64(n))
}

// reverseU returns s with its u parameterization reversed (control rows and knots flipped about the
// u domain), so the u-min edge becomes u-max. Geometry is unchanged.
func reverseU(s BSplineSurface) BSplineSurface {
	nu := len(s.Ctrl)
	ctrl := make([][]math.Point3, nu)
	weights := make([][]float64, nu)
	for i := 0; i < nu; i++ {
		ctrl[i] = append([]math.Point3(nil), s.Ctrl[nu-1-i]...)
		weights[i] = append([]float64(nil), s.Weights[nu-1-i]...)
	}
	r, _ := NewBSplineSurface(s.UDegree, s.VDegree, ctrl, weights, reversedKnots(s.UKnots), s.VKnots)
	return r
}

// transposeSurface returns s with its u and v directions swapped. Geometry is unchanged.
func transposeSurface(s BSplineSurface) BSplineSurface {
	nu, nv := len(s.Ctrl), len(s.Ctrl[0])
	ctrl := make([][]math.Point3, nv)
	weights := make([][]float64, nv)
	for j := 0; j < nv; j++ {
		ctrl[j] = make([]math.Point3, nu)
		weights[j] = make([]float64, nu)
		for i := 0; i < nu; i++ {
			ctrl[j][i], weights[j][i] = s.Ctrl[i][j], s.Weights[i][j]
		}
	}
	r, _ := NewBSplineSurface(s.VDegree, s.UDegree, ctrl, weights, s.VKnots, s.UKnots)
	return r
}

// reversedKnots returns the knot vector reflected about its own domain (a+b−u, re-sorted ascending),
// preserving the clamped structure.
func reversedKnots(knots []float64) []float64 {
	n := len(knots)
	lo, hi := knots[0], knots[n-1]
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		out[i] = lo + hi - knots[n-1-i]
	}
	return out
}

// repeatedKnot returns count copies of v.
func repeatedKnot(v float64, count int) []float64 {
	out := make([]float64, count)
	for i := range out {
		out[i] = v
	}
	return out
}

// powInt returns x^n for a small non-negative n.
func powInt(x float64, n int) float64 {
	p := 1.0
	for i := 0; i < n; i++ {
		p *= x
	}
	return p
}

// factorial returns k! for small k.
func factorial(k int) float64 {
	f := 1.0
	for i := 2; i <= k; i++ {
		f *= float64(i)
	}
	return f
}
