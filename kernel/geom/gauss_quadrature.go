// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"sync"
)

// Gauss–Legendre quadrature — the general numeric-integration primitive the analytic
// mass-properties integration (M48/C3, Oblikovati/Oblikovati#3453..3458) builds on. An
// n-point rule integrates polynomials up to degree 2n−1 exactly and converges spectrally
// on smooth integrands, so it computes surface integrals (volume, moment, area, inertia)
// over the ANALYTIC B-rep instead of over a triangle mesh. This is a derived numeric
// computation of the surface, not a tessellation read.

// gaussCache memoizes the nodes/weights per order — Newton iteration on the Legendre roots
// is not free, and the same handful of orders is reused across every face integration.
var gaussCache sync.Map // int → gaussRule

type gaussRule struct{ nodes, weights []float64 }

// GaussLegendre returns the n Gauss–Legendre nodes and weights on [−1, 1], nodes ascending.
// The rule is exact for polynomials of degree ≤ 2n−1.
//
// Example: nodes, weights := GaussLegendre(5) // the classic 5-point rule
func GaussLegendre(n int) (nodes, weights []float64) {
	if n < 1 {
		panic("geom.GaussLegendre: order must be ≥ 1, got " + itoa(n))
	}
	if r, ok := gaussCache.Load(n); ok {
		g := r.(gaussRule)
		return g.nodes, g.weights
	}
	g := computeGaussRule(n)
	gaussCache.Store(n, g)
	return g.nodes, g.weights
}

// Integrate1D returns ∫ₐᵇ f(x) dx via an n-point Gauss–Legendre rule mapped onto [a, b].
func Integrate1D(n int, a, b float64, f func(x float64) float64) float64 {
	nodes, weights := GaussLegendre(n)
	half, mid := (b-a)/2, (a+b)/2
	var sum float64
	for i, x := range nodes {
		sum += weights[i] * f(mid+half*x)
	}
	return sum * half
}

// computeGaussRule derives the nodes (roots of the degree-n Legendre polynomial Pₙ) by
// Newton's method from the standard cosine seed, and the weights wᵢ = 2/((1−xᵢ²)·Pₙ′(xᵢ)²).
func computeGaussRule(n int) gaussRule {
	nodes := make([]float64, n)
	weights := make([]float64, n)
	for i := range n {
		x := stdmath.Cos(stdmath.Pi * (float64(i) + 0.75) / (float64(n) + 0.5)) // i-th root seed
		x = refineLegendreRoot(n, x)
		_, dp := legendreValueDeriv(n, x)
		nodes[n-1-i] = x // seed runs high→low; store ascending
		weights[n-1-i] = 2 / ((1 - x*x) * dp * dp)
	}
	return gaussRule{nodes: nodes, weights: weights}
}

// refineLegendreRoot Newton-polishes a Legendre root to full double precision.
func refineLegendreRoot(n int, x float64) float64 {
	for range 100 {
		p, dp := legendreValueDeriv(n, x)
		dx := p / dp
		x -= dx
		if stdmath.Abs(dx) <= 1e-15 { // tol:numeric — Newton convergence on a Legendre root
			break
		}
	}
	return x
}

// legendreValueDeriv evaluates Pₙ(x) and Pₙ′(x) by the stable three-term recurrence.
func legendreValueDeriv(n int, x float64) (p, dp float64) {
	p0, p1 := 1.0, x
	for k := 1; k < n; k++ {
		p0, p1 = p1, ((2*float64(k)+1)*x*p1-float64(k)*p0)/(float64(k)+1)
	}
	if n == 0 {
		return 1, 0
	}
	dp = float64(n) * (x*p1 - p0) / (x*x - 1)
	return p1, dp
}

// itoa avoids an fmt import in the panic path.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
