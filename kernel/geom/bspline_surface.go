// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// BSplineSurface is a NURBS surface (contract: BSplineSurface): a rational
// B-spline over a rectangular control net Ctrl[uIndex][vIndex] with matching
// Weights and two knot vectors. It satisfies [Surface].
type BSplineSurface struct {
	UDegree, VDegree int
	Ctrl             [][]math.Point3
	Weights          [][]float64
	UKnots, VKnots   []float64
}

// NewBSplineSurface builds a rational B-spline surface, validating that the
// control net is rectangular and that both directions satisfy the
// degree/control/knot size relationships with positive weights. All slices are
// deep-copied so the value stays immutable.
func NewBSplineSurface(uDeg, vDeg int, ctrl [][]math.Point3, weights [][]float64, uKnots, vKnots []float64) (BSplineSurface, error) {
	uCount, vCount, err := rectangularDims(ctrl, weights)
	if err != nil {
		return BSplineSurface{}, err
	}
	if err := validateBSpline(uDeg, uCount, uCount, len(uKnots)); err != nil {
		return BSplineSurface{}, fmt.Errorf("u direction: %w", err)
	}
	if err := validateBSpline(vDeg, vCount, vCount, len(vKnots)); err != nil {
		return BSplineSurface{}, fmt.Errorf("v direction: %w", err)
	}
	if err := positiveNet(weights); err != nil {
		return BSplineSurface{}, err
	}
	return BSplineSurface{
		UDegree: uDeg, VDegree: vDeg, Ctrl: copyNet(ctrl), Weights: copyWeights(weights),
		UKnots: append([]float64(nil), uKnots...), VKnots: append([]float64(nil), vKnots...),
	}, nil
}

// PointAt returns the surface position at parameters (u, v).
func (s BSplineSurface) PointAt(u, v float64) math.Point3 {
	us, vs := s.spans(u, v)
	nu := basisFuns(us, s.UDegree, u, s.UKnots)
	nv := basisFuns(vs, s.VDegree, v, s.VKnots)
	var h homog
	for k := 0; k <= s.UDegree; k++ {
		for l := 0; l <= s.VDegree; l++ {
			ui, vj := us-s.UDegree+k, vs-s.VDegree+l
			h.add(s.Ctrl[ui][vj], s.Weights[ui][vj]*nu[k]*nv[l])
		}
	}
	return h.point()
}

// DerivativesAt returns the partials ∂P/∂u and ∂P/∂v at (u, v).
func (s BSplineSurface) DerivativesAt(u, v float64) (du, dv math.Vector3) {
	us, vs := s.spans(u, v)
	nu, dnu := basisAndFirstDerivs(us, s.UDegree, u, s.UKnots)
	nv, dnv := basisAndFirstDerivs(vs, s.VDegree, v, s.VKnots)
	var val, accU, accV homog
	for k := 0; k <= s.UDegree; k++ {
		for l := 0; l <= s.VDegree; l++ {
			ui, vj := us-s.UDegree+k, vs-s.VDegree+l
			p, w := s.Ctrl[ui][vj], s.Weights[ui][vj]
			val.add(p, w*nu[k]*nv[l])
			accU.add(p, w*dnu[k]*nv[l])
			accV.add(p, w*nu[k]*dnv[l])
		}
	}
	return val.deriv(accU), val.deriv(accV)
}

// NormalAt returns the unit normal (∂u×∂v normalized), or zero where degenerate.
func (s BSplineSurface) NormalAt(u, v float64) math.Vector3 {
	return normalFromPartials(s.DerivativesAt(u, v))
}

// spans returns the active knot spans in each direction for (u, v).
func (s BSplineSurface) spans(u, v float64) (us, vs int) {
	us = findSpan(len(s.Ctrl)-1, s.UDegree, u, s.UKnots)
	vs = findSpan(len(s.Ctrl[0])-1, s.VDegree, v, s.VKnots)
	return us, vs
}

// UDomain returns [UKnots[UDegree], UKnots[len−1−UDegree]].
func (s BSplineSurface) UDomain() (lo, hi float64) {
	return s.UKnots[s.UDegree], s.UKnots[len(s.UKnots)-1-s.UDegree]
}

// VDomain returns [VKnots[VDegree], VKnots[len−1−VDegree]].
func (s BSplineSurface) VDomain() (lo, hi float64) {
	return s.VKnots[s.VDegree], s.VKnots[len(s.VKnots)-1-s.VDegree]
}
