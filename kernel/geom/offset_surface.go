// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// OffsetSurface is the parallel surface a fixed signed Distance from a base surface along its
// normal: S_d(u,v) = S(u,v) + Distance·N(u,v). It shares the base's (u,v) parameterization, so its
// normal at (u,v) equals the base normal there and its ParamAt is the base's — which is exactly what
// lets a face be offset by swapping its surface and KEEPING its loops: the trimmed-face tessellator
// projects the loop to (u,v) through ParamAt and evaluates through PointAt, so the same UV region is
// re-evaluated on the offset surface (no edge/loop surgery). CAM uses it to offset a part face by the
// tool radius and then sample the offset for drop-cutter projection.
//
// The offset is exact for the analytic surfaces (plane/cylinder/sphere/cone/torus, whose normal is
// closed-form); a large inward Distance on a tightly curved surface can self-intersect (the offset
// of a cylinder by more than its radius inverts), which the caller must avoid — there is no
// self-intersection trimming here.
type OffsetSurface struct {
	Base     Surface
	Distance float64
}

var _ Surface = OffsetSurface{}

// PointAt evaluates the base point displaced along the base normal by Distance.
func (o OffsetSurface) PointAt(u, v float64) math.Point3 {
	return o.Base.PointAt(u, v).TranslateBy(o.Base.NormalAt(u, v).Scale(math.Scalar(o.Distance)))
}

// NormalAt returns the base normal: a parallel surface has the same normal field as its base.
func (o OffsetSurface) NormalAt(u, v float64) math.Vector3 { return o.Base.NormalAt(u, v) }

// offsetParamStep is the half-width of the central difference used to take the normal field's rate
// of change; small enough to be accurate on the analytic surfaces' [0,2π] angular domains.
const offsetParamStep = 1e-6

// DerivativesAt returns the offset surface's partials, ∂S_d/∂u = ∂S/∂u + Distance·∂N/∂u (and v): the
// base's exact partials plus the Distance-scaled rate of change of the normal (by central
// difference, since the Surface interface exposes no second derivative).
func (o OffsetSurface) DerivativesAt(u, v float64) (du, dv math.Vector3) {
	bdu, bdv := o.Base.DerivativesAt(u, v)
	dNdu := centralDiff(o.Base.NormalAt(u+offsetParamStep, v), o.Base.NormalAt(u-offsetParamStep, v))
	dNdv := centralDiff(o.Base.NormalAt(u, v+offsetParamStep), o.Base.NormalAt(u, v-offsetParamStep))
	d := math.Scalar(o.Distance)
	return bdu.Add(dNdu.Scale(d)), bdv.Add(dNdv.Scale(d))
}

// centralDiff returns (hi − lo) / (2·offsetParamStep).
func centralDiff(hi, lo math.Vector3) math.Vector3 {
	return hi.Add(lo.Scale(-1)).Scale(math.Scalar(1.0 / (2 * offsetParamStep)))
}

// UDomain / VDomain are the base's: offsetting along the normal does not change the parameter range.
func (o OffsetSurface) UDomain() (lo, hi float64) { return o.Base.UDomain() }
func (o OffsetSurface) VDomain() (lo, hi float64) { return o.Base.VDomain() }

// ParamAt returns the base parameters of p. For a point on the offset surface this reproduces its
// (u,v) (the analytic surfaces' radial/perpendicular projection is offset-invariant), which is what
// the tessellator needs when the offset face reuses the base loops.
func (o OffsetSurface) ParamAt(p math.Point3) (u, v float64) { return o.Base.ParamAt(p) }
