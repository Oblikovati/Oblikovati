// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// OffsetSurface is the parallel surface a fixed signed Distance from a base surface along its
// normal: S_d(u,v) = S(u,v) + Distance·N(u,v). It shares the base's (u,v) parameterization and normal
// field, which is what lets a face be offset by swapping its surface and KEEPING its loops: the
// trimmed-face tessellator projects the loop to (u,v) through ParamAt and evaluates through PointAt,
// so the same UV region is re-evaluated on the offset surface (no edge/loop surgery). CAM uses it to
// offset a part face by the tool radius and then sample the offset for drop-cutter projection.
//
// A large inward Distance on a tightly curved surface self-intersects (the offset of a cylinder by
// more than its radius inverts). Construct through NewOffsetSurface to reject that regime; the bare
// struct literal is still valid for callers that have already guaranteed the offset is non-folding.
type OffsetSurface struct {
	Base     Surface
	Distance float64
}

var _ Surface = OffsetSurface{}

// NewOffsetSurface builds the offset and rejects a self-intersecting (folded) result: an inward offset
// larger than the base's local concave radius of curvature inverts the surface into invalid geometry.
//
//	off, err := geom.NewOffsetSurface(cyl, 2) // err != nil if 2 exceeds the cylinder radius
func NewOffsetSurface(base Surface, distance float64) (OffsetSurface, error) {
	o := OffsetSurface{Base: base, Distance: distance}
	if u, v, folds := o.foldPoint(); folds {
		return OffsetSurface{}, fmt.Errorf(
			"geom.NewOffsetSurface: distance %g self-intersects the base near (u,v)=(%g,%g): it exceeds the local concave radius of curvature",
			distance, u, v)
	}
	return o, nil
}

// PointAt evaluates the base point displaced along the base normal by Distance.
func (o OffsetSurface) PointAt(u, v float64) math.Point3 {
	return o.Base.PointAt(u, v).TranslateBy(o.Base.NormalAt(u, v).Scale(math.Scalar(o.Distance)))
}

// NormalAt returns the base normal: a parallel surface has the same normal field as its base.
func (o OffsetSurface) NormalAt(u, v float64) math.Vector3 { return o.Base.NormalAt(u, v) }

// DerivativesAt returns the offset surface's partials, ∂S_d/∂u = ∂S/∂u + Distance·∂N/∂u (and v): the
// base's exact partials plus the Distance-scaled rate of change of the normal.
func (o OffsetSurface) DerivativesAt(u, v float64) (du, dv math.Vector3) {
	bdu, bdv := o.Base.DerivativesAt(u, v)
	dNdu, dNdv := o.normalDerivs(u, v)
	d := math.Scalar(o.Distance)
	return bdu.Add(dNdu.Scale(d)), bdv.Add(dNdv.Scale(d))
}

// normalDerivs returns ∂N/∂u, ∂N/∂v of the base normal field by central differences — the Surface
// interface exposes no analytic ∂N, so this is the one place the offset's curvature-dependent partials
// come from. Each direction uses the first-difference-optimal step stepD1 = ε^{1/3} scaled by its own
// domain span, so the step adapts to the base's parameter scale (a NURBS [0,1] vs an analytic [0,2π]
// domain) and the truncation/roundoff balance holds at any model scale (#1322, #1402).
func (o OffsetSurface) normalDerivs(u, v float64) (dNdu, dNdv math.Vector3) {
	hu := stepD1 * spanOr1(o.Base.UDomain())
	hv := stepD1 * spanOr1(o.Base.VDomain())
	dNdu = centralDiff(o.Base.NormalAt(u+hu, v), o.Base.NormalAt(u-hu, v), hu)
	dNdv = centralDiff(o.Base.NormalAt(u, v+hv), o.Base.NormalAt(u, v-hv), hv)
	return dNdu, dNdv
}

// centralDiff returns (hi − lo) / (2·h).
func centralDiff(hi, lo math.Vector3, h float64) math.Vector3 {
	return hi.Add(lo.Scale(-1)).Scale(math.Scalar(1.0 / (2 * h)))
}

// UDomain / VDomain are the base's: offsetting along the normal does not change the parameter range.
func (o OffsetSurface) UDomain() (lo, hi float64) { return o.Base.UDomain() }
func (o OffsetSurface) VDomain() (lo, hi float64) { return o.Base.VDomain() }

// offsetInvertIters bounds the Gauss–Newton iterations inverting a point onto the offset
// surface (the inversion exits early on convergence, #1401).
const offsetInvertIters = 40

// ParamAt inverts PointAt by projecting p onto the OFFSET surface (not the base): seed from the base
// inversion — exact for the radial/perpendicular analytic projections (plane/cylinder/sphere), a good
// guess otherwise — then refine with projected Gauss–Newton on the offset's own partials. This fixes
// the silent error of returning o.Base.ParamAt(p) directly (#1322): for a NURBS base (and even the
// cone/torus, whose offset is not radially offset-invariant) that returned the wrong (u,v).
func (o OffsetSurface) ParamAt(p math.Point3) (u, v float64) {
	bu, bv := o.Base.ParamAt(p) // seed from the base inversion
	u, v = clampToSurface(o, bu, bv)
	u, v, _ = refineSurfaceParam(o, p, u, v, offsetInvertIters)
	return u, v
}

// SelfIntersects reports whether the offset folds onto itself anywhere in the domain. It samples the
// signed area element of the offset's partials against the base normal: an inward offset past the
// local concave radius of curvature inverts the surface, flipping that sign (or collapsing it).
func (o OffsetSurface) SelfIntersects() bool {
	_, _, folds := o.foldPoint()
	return folds
}

// foldTol is the tangent-scaling eigenvalue below which the offset is treated as folded — slightly
// above 0 so the razor-edge collapse (eigenvalue exactly 0, the offset pinched to a line/point) is
// rejected along with the inverted regime, with margin against finite-difference noise.
const foldTol = 1e-7

// foldPoint scans a grid for the first (u,v) where the offset has folded — where the smaller tangent-
// scaling eigenvalue 1 + Distance·κ has gone non-positive (the offset passed the local centre of
// curvature). Degenerate base points (a sphere pole, a degenerate tangent frame) are skipped.
func (o OffsetSurface) foldPoint() (u, v float64, folds bool) {
	uLo, uHi := finiteRange(o.UDomain())
	vLo, vHi := finiteRange(o.VDomain())
	const n = 24
	for i := 0; i <= n; i++ {
		uu := uLo + (uHi-uLo)*float64(i)/n
		for j := 0; j <= n; j++ {
			vv := vLo + (vHi-vLo)*float64(j)/n
			if s, ok := o.minTangentScale(uu, vv); ok && s < foldTol {
				return uu, vv, true
			}
		}
	}
	return 0, 0, false
}

// minTangentScale returns the smaller eigenvalue of the offset's tangent-scaling map I + Distance·W,
// where W is the base's shape operator (Weingarten map) recovered from ∂N/∂param in the tangent
// basis. The eigenvalues are 1 + Distance·κ₁,₂ (the principal curvatures); the offset self-intersects
// where the smaller one reaches 0 — i.e. |Distance| exceeds the local concave radius of curvature.
// This catches the ISOTROPIC fold (a sphere offset past its centre flips BOTH partials, so a mere
// area-element sign — the product of the two scalings — would stay positive and miss it). ok is false
// at a degenerate base point (zero normal or degenerate tangent frame).
func (o OffsetSurface) minTangentScale(u, v float64) (float64, bool) {
	if o.Base.NormalAt(u, v).LengthSquared() < math.DefaultTolerance {
		return 0, false
	}
	su, sv := o.Base.DerivativesAt(u, v)
	e, f, g := dot(su, su), dot(su, sv), dot(sv, sv)
	if degenerateFirstForm(e, f, g) { // scale-invariant (parallel/collapsed tangents), #1402
		return 0, false
	}
	det := e*g - f*f
	nu, nv := o.normalDerivs(u, v)
	// Express ∂N/∂u, ∂N/∂v in the {Su,Sv} basis (solve the metric system) → the 2×2 map W.
	w11, w21 := solveMetric(e, f, g, det, dot(nu, su), dot(nu, sv))
	w12, w22 := solveMetric(e, f, g, det, dot(nv, su), dot(nv, sv))
	d := o.Distance
	a, b, c, dd := 1+d*w11, d*w12, d*w21, 1+d*w22
	disc := (a-dd)*(a-dd) + 4*b*c
	if disc < 0 {
		disc = 0 // numerically tiny negative from FD noise on a (real-spectrum) shape operator
	}
	return ((a + dd) - stdmath.Sqrt(disc)) / 2, true
}

// solveMetric solves [[e,f],[f,g]]·x = [r1;r2] (det = e·g−f², precomputed nonzero).
func solveMetric(e, f, g, det, r1, r2 float64) (float64, float64) {
	return (g*r1 - f*r2) / det, (-f*r1 + e*r2) / det
}

// dot is float64(a·b), for the metric arithmetic above.
func dot(a, b math.Vector3) float64 { return float64(a.Dot(b)) }

// spanOr1 returns the domain span, or 1 for an unbounded/degenerate domain (so a relative step or
// tolerance built from it stays finite and nonzero).
func spanOr1(lo, hi float64) float64 {
	s := hi - lo
	if stdmath.IsInf(s, 0) || s <= 0 {
		return 1
	}
	return s
}

// finiteRange replaces ±Inf domain bounds with a finite sampling window for the fold scan.
func finiteRange(lo, hi float64) (float64, float64) {
	const window = 10
	if stdmath.IsInf(lo, 0) {
		lo = -window
	}
	if stdmath.IsInf(hi, 0) {
		hi = window
	}
	return lo, hi
}
