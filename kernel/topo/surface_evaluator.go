// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/math"
)

// SurfaceEvaluator evaluates a surface: point, partial tangents, normal, and a
// numeric closest-parameter search.
type SurfaceEvaluator struct {
	surface geom.Surface
}

// NewSurfaceEvaluator wraps a surface.
func NewSurfaceEvaluator(s geom.Surface) SurfaceEvaluator { return SurfaceEvaluator{surface: s} }

// PointAt returns the position at (u, v).
func (e SurfaceEvaluator) PointAt(u, v float64) math.Point3 { return e.surface.PointAt(u, v) }

// NormalAt returns the unit normal at (u, v).
func (e SurfaceEvaluator) NormalAt(u, v float64) math.Vector3 { return e.surface.NormalAt(u, v) }

// TangentsAt returns the partial derivatives ∂P/∂u and ∂P/∂v.
func (e SurfaceEvaluator) TangentsAt(u, v float64) (du, dv math.Vector3) {
	return e.surface.DerivativesAt(u, v)
}

// ClosestParam returns the (u, v) whose point is nearest p, by a coarse grid sample
// then a few projected Gauss–Newton steps. Infinite domains are clamped to a finite
// search window.
func (e SurfaceEvaluator) ClosestParam(p math.Point3) (u, v float64) {
	uLo, uHi := clampDomain(e.surface.UDomain())
	vLo, vHi := clampDomain(e.surface.VDomain())
	u, v = gridSeed(e.surface, p, uLo, uHi, vLo, vHi)
	for i := 0; i < 40; i++ {
		u, v = projectStep(e.surface, p, u, v, uLo, uHi, vLo, vHi)
	}
	return u, v
}

// ClosestPoint returns the point on the surface nearest p.
func (e SurfaceEvaluator) ClosestPoint(p math.Point3) math.Point3 {
	u, v := e.ClosestParam(p)
	return e.surface.PointAt(u, v)
}

// gridSeed returns the sampled (u, v) nearest p over a coarse grid.
func gridSeed(s geom.Surface, p math.Point3, uLo, uHi, vLo, vHi float64) (float64, float64) {
	const n = 16
	bu, bv, bd := uLo, vLo, stdmath.Inf(1)
	for i := 0; i <= n; i++ {
		u := uLo + (uHi-uLo)*float64(i)/n
		for j := 0; j <= n; j++ {
			v := vLo + (vHi-vLo)*float64(j)/n
			if d := s.PointAt(u, v).DistanceSquaredTo(p); d < bd {
				bu, bv, bd = u, v, d
			}
		}
	}
	return bu, bv
}

// projectStep moves (u, v) toward p by projecting the residual onto each partial,
// staying within the domain.
func projectStep(s geom.Surface, p math.Point3, u, v, uLo, uHi, vLo, vHi float64) (float64, float64) {
	du, dv := s.DerivativesAt(u, v)
	res := s.PointAt(u, v).VectorTo(p)
	u = clamp(u+projectOnto(res, du), uLo, uHi)
	v = clamp(v+projectOnto(res, dv), vLo, vHi)
	return u, v
}

// projectOnto returns res·d / |d|², the parameter step that reduces the residual
// along d (zero if d is degenerate).
func projectOnto(res, d math.Vector3) float64 {
	denom := d.LengthSquared()
	if denom < math.DefaultTolerance {
		return 0
	}
	return res.Dot(d) / denom
}

// clampDomain replaces an infinite bound with a finite search window.
func clampDomain(lo, hi float64) (float64, float64) {
	const window = 1e3
	if stdmath.IsInf(lo, 0) {
		lo = -window
	}
	if stdmath.IsInf(hi, 0) {
		hi = window
	}
	return lo, hi
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
