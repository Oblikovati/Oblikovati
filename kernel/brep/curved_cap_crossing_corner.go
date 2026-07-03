// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cap-crossing corner solve (EPIC Oblikovati/Oblikovati#1724, ADR-0046, slice 2). The interior-exit slice
// (slice 1) handles a tool whose exit ellipse lies strictly INSIDE the cap rim. When the ellipse CROSSES
// the rim the exit is a rim-crossing corner: the tool exits partly through the cap and partly through the
// wall, meeting the rim at the triple points {rim ∩ tool_wall ∩ cap_plane}. This file computes those
// corner points exactly — the load-bearing shared vertices every rim-crossing face must reference so the
// cross-face weld stays watertight (the slice-1 by-value/T-junction lesson generalizes to these corners).

// capRimCorner is one point where the tool cylinder wall crosses the target cap's rim circle: the rim
// angle θ (radians) and the 3D point on the rim at that angle.
type capRimCorner struct {
	angle float64
	point math.Point3
}

// capRimCorners returns every point where the tool cylinder wall crosses the cap's rim circle — the exact
// triple points rim ∩ tool_wall ∩ cap_plane. It solves g(θ) = |w(θ)|² − (w(θ)·d)² − r² = 0 on the rim
// (w(θ) = rim(θ) − toolOrigin, d the tool axis), a first-and-second-harmonic trig equation, by dense-sample
// bracketing + bisection. The COUNT is the recognizer gate: 0 → ellipse clear of the rim (slice 1 or no
// contact), 2 → the single rim-crossing corner slice 2 handles, 4 → a two-lens footprint (deferred).
func capRimCorners(tool geom.Cylinder, rim geom.Circle) []capRimCorner {
	g := rimCornerEquation(tool, rim)
	roots := bracketedRoots(g, cornerSamples)
	out := make([]capRimCorner, 0, len(roots))
	for _, th := range roots {
		out = append(out, capRimCorner{angle: th, point: rim.PointAt(th / (2 * stdmath.Pi))})
	}
	return out
}

// cornerSamples is the number of equal rim-angle stations the corner equation is sampled at to bracket its
// sign changes. 1440 = one bracket per 0.25°, dense enough that two genuine crossings are never in one cell
// (they would have to sit within 0.25° — a near-tangency the slice-2 gate declines anyway).
const cornerSamples = 1440

// rimCornerEquation builds g(θ), the exact tool-cylinder-distance residual along the rim, as a closure over
// the harmonic coefficients (derived in the rim's own in-plane basis e1=RefDir, e2=Normal×RefDir). Keeping
// the tool-cylinder implicit — never reconstructing the exit ellipse's r/|n·d| semi-major — keeps the solve
// well-conditioned for shallow tools (ADR-0046 slice-2 numerics).
func rimCornerEquation(tool geom.Cylinder, rim geom.Circle) func(float64) float64 {
	e1 := rim.RefDir.AsVector()
	e2 := rim.Normal.Cross(rim.RefDir)
	d := tool.AxisDir.AsVector()
	w0 := tool.Origin.VectorTo(rim.Center)
	rr, tr := rim.Radius, tool.Radius
	d0 := float64(w0.Dot(d))
	p1, p2 := float64(w0.Dot(e1)), float64(w0.Dot(e2))
	al, be := float64(e1.Dot(d)), float64(e2.Dot(d))
	a0 := float64(w0.Dot(w0)) + rr*rr - d0*d0 - tr*tr - rr*rr*(al*al+be*be)/2
	a1, b1 := 2*rr*(p1-d0*al), 2*rr*(p2-d0*be)
	a2, b2 := -rr*rr*(al*al-be*be)/2, -rr*rr*al*be
	return func(th float64) float64 {
		return a0 + a1*stdmath.Cos(th) + b1*stdmath.Sin(th) + a2*stdmath.Cos(2*th) + b2*stdmath.Sin(2*th)
	}
}

// bracketedRoots returns the roots of a 2π-periodic function in [0, 2π) by sampling n equal cells, bracketing
// each sign change, and bisecting it to convergence. A cell whose endpoints share a sign is skipped, so a
// tangential (double) root that never changes sign is deliberately NOT reported — slice 2 wants only genuine
// transversal crossings, and the tangency boundary is a distinct declined case.
func bracketedRoots(g func(float64) float64, n int) []float64 {
	var roots []float64
	prevTh := 0.0
	prev := g(prevTh)
	for i := 1; i <= n; i++ {
		th := 2 * stdmath.Pi * float64(i) / float64(n)
		cur := g(th)
		if prev == 0 {
			roots = append(roots, prevTh)
		} else if prev*cur < 0 {
			roots = append(roots, bisectRoot(g, prevTh, th))
		}
		prevTh, prev = th, cur
	}
	return roots
}

// bisectRoot refines a sign-change bracket [lo, hi] of g to a model-independent angular tolerance by
// bisection — unconditionally convergent, no derivative needed (the corner equation is smooth but its
// derivative adds no robustness a 60-step bisection to ~1e-16 rad lacks).
func bisectRoot(g func(float64) float64, lo, hi float64) float64 {
	glo := g(lo)
	for k := 0; k < 60; k++ {
		mid := (lo + hi) / 2
		gm := g(mid)
		if gm == 0 || (hi-lo) < 1e-15 {
			return mid
		}
		if glo*gm < 0 {
			hi = mid
		} else {
			lo, glo = mid, gm
		}
	}
	return (lo + hi) / 2
}
