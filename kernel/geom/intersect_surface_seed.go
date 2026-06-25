// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Adaptive seeding for the surface–surface intersection tracer (Oblikovati#1400). The old seeder sampled
// a fixed 161×161 lattice and kept any node whose signed-distance field was within 2·max(du,dv) of zero —
// which SILENTLY DROPPED any intersection loop thinner than that grid spacing, and spent ~26k full point
// projections regardless of how much of the domain is nowhere near the other surface. This seeder instead
// refines a quadtree only where the field can still reach zero, pruning a cell as soon as its smallest
// corner |f| exceeds a conservative bound on how much the field can change across it. The signed-distance
// field is 1-Lipschitz in 3D (|∇f| ≤ 1), so |Δf| across a cell ≤ |ΔP| ≤ Tᵤ·Δu + Tᵥ·Δv where Tᵤ, Tᵥ bound
// the base tangent magnitudes — a cell whose corners are all farther from zero than that bound cannot
// contain a crossing, while an arbitrarily thin loop keeps its cell un-prunable down to march-step size.
// Corner evaluations are cached on a shared dyadic lattice so neighbouring cells reuse them.

const (
	// ssiSeedCoarse is the initial subdivision per axis (a component wider than 1/8 of the domain
	// separates at the coarse level); ssiSeedMaxDepth is the extra quadtree depth, so the finest lattice
	// is ssiSeedCoarse·2^depth = 1024 nodes per axis — far finer than the old 160 WHERE the field is near
	// zero, and never sampled where it is not.
	ssiSeedCoarse   = 8
	ssiSeedMaxDepth = 7
	// ssiSeedSafety inflates the tangent-magnitude variation bound so the prune stays conservative when
	// corner sampling underestimates a larger interior tangent. With it no zero is pruned in practice;
	// a rigorous bound would use the NURBS control-net derivative hodograph (deferred).
	ssiSeedSafety = 2.0
)

// ssiSeedField evaluates and caches the SSI signed-distance field and the base tangent magnitudes on a
// dyadic integer lattice over the base parameter window, so the quadtree's shared cell corners are
// computed once. evals counts the (expensive, projection-bearing) field evaluations actually performed —
// the metric the adaptive seeder drives down versus the fixed grid.
type ssiSeedField struct {
	base, other Surface
	u0, v0      float64 // lattice origin (UMin, VMin)
	du, dv      float64 // finest lattice spacing (one max-depth cell)
	cache       map[[2]int]ssiSample
	evals       int
}

// ssiSample is one lattice node: the signed-distance field f and the base surface tangent magnitudes
// (tu, tv), the latter bounding how fast f can change in each parameter direction across a cell.
type ssiSample struct {
	f, tu, tv float64
}

// at returns the cached sample at lattice node (i, j), evaluating and storing it on first touch.
func (c *ssiSeedField) at(i, j int) ssiSample {
	key := [2]int{i, j}
	if s, ok := c.cache[key]; ok {
		return s
	}
	u, v := c.u0+float64(i)*c.du, c.v0+float64(j)*c.dv
	du, dv := c.base.DerivativesAt(u, v)
	s := ssiSample{
		f:  SignedDistanceToSurface(c.other, c.base.PointAt(u, v)),
		tu: float64(du.Length()),
		tv: float64(dv.Length()),
	}
	c.cache[key] = s
	c.evals++
	return s
}

// point returns the base surface point at lattice node (i, j).
func (c *ssiSeedField) point(i, j int) math.Point3 {
	return c.base.PointAt(c.u0+float64(i)*c.du, c.v0+float64(j)*c.dv)
}

// newSSISeedField builds the cached dyadic-lattice field over the base parameter window. The finest
// lattice is ssiSeedCoarse·2^ssiSeedMaxDepth nodes per axis; the quadtree samples it adaptively.
func newSSISeedField(base, other Surface, g SurfaceGrid) *ssiSeedField {
	nodes := float64(ssiSeedCoarse << ssiSeedMaxDepth)
	return &ssiSeedField{
		base: base, other: other,
		u0: g.UMin, v0: g.VMin,
		du:    (g.UMax - g.UMin) / nodes,
		dv:    (g.VMax - g.VMin) / nodes,
		cache: map[[2]int]ssiSample{},
	}
}

// ssiSeedSink collects seeds in two tiers so transversal crossings are returned BEFORE interior minima.
// A loop is traced from the first seed that lands on it (later duplicates are deduped away by the
// tracer), so seeding edge crossings first makes a transversal loop start at a stable generic point on
// it — not at a curvature extreme — which keeps the downstream imprint seam well-behaved (#1400). The
// interior tier (a cell's smallest-|f| corner) only seeds a feature no edge crossing reveals: a sub-grid
// loop or a tangency.
type ssiSeedSink struct {
	crossings, interior []math.Point3
}

// all returns the crossing seeds followed by the interior seeds.
func (s *ssiSeedSink) all() []math.Point3 {
	return append(s.crossings, s.interior...)
}

// ssiSeeds returns candidate 3D points near the base∩other intersection by adaptively refining a quadtree
// over the base parameter window and emitting a seed wherever the field crosses zero or pins a sub-cell
// minimum (a thin loop or tangency the contour never straddles). It replaces the fixed-grid scan that
// dropped sub-grid loops (Oblikovati#1400).
func ssiSeeds(base, other Surface, g SurfaceGrid) []math.Point3 {
	return newSSISeedField(base, other, g).seeds(ssiStep(base, g))
}

// seeds runs the coarse-to-fine quadtree and returns the accumulated seeds (crossings first). leaf is the
// 3D cell size (a march step) below which a cell is seeded rather than split further.
func (c *ssiSeedField) seeds(leaf float64) []math.Point3 {
	res := 1 << ssiSeedMaxDepth
	sink := &ssiSeedSink{}
	for ci := 0; ci < ssiSeedCoarse; ci++ {
		for cj := 0; cj < ssiSeedCoarse; cj++ {
			c.refine(ci*res, cj*res, (ci+1)*res, (cj+1)*res, leaf, sink)
		}
	}
	return sink.all()
}

// refine processes the lattice cell [i0,i1]×[j0,j1]: prune it when the field provably cannot reach zero
// inside, emit seeds when it is a leaf, else split into four. A cell is a leaf — needs no finer split —
// when it is below a march step, at the finest lattice, OR shows a clean two-edge transversal crossing
// (the marcher resolves that curve from one seed, so refining it further is wasted bisection). Only an
// un-pruned cell that does NOT yet bracket a simple crossing keeps subdividing, because that is where a
// sub-cell loop or tangency can still hide. The prune is the speed-up; the keep-refining-the-ambiguous
// rule is the correctness (a thin loop's cell is never pruned and is split until seen).
func (c *ssiSeedField) refine(i0, j0, i1, j1 int, leaf float64, sink *ssiSeedSink) {
	s00, s10, s01, s11 := c.at(i0, j0), c.at(i1, j0), c.at(i0, j1), c.at(i1, j1)
	minAbs := minOf4(stdmath.Abs(s00.f), stdmath.Abs(s10.f), stdmath.Abs(s01.f), stdmath.Abs(s11.f))
	su, sv := float64(i1-i0)*c.du, float64(j1-j0)*c.dv
	tu := maxOf4(s00.tu, s10.tu, s01.tu, s11.tu)
	tv := maxOf4(s00.tv, s10.tv, s01.tv, s11.tv)
	variation := ssiSeedSafety * (tu*su + tv*sv) // upper bound on |Δf| across the cell (|∇f| ≤ 1)
	if minAbs > variation {
		return // every interior point keeps a corner's sign: no crossing here
	}
	if variation <= leaf || i1-i0 <= 1 || j1-j0 <= 1 || ssiCrossings(s00, s10, s01, s11) == 2 {
		c.emitLeaf(i0, j0, i1, j1, s00, s10, s01, s11, sink)
		return
	}
	im, jm := (i0+i1)/2, (j0+j1)/2
	c.refine(i0, j0, im, jm, leaf, sink)
	c.refine(im, j0, i1, jm, leaf, sink)
	c.refine(i0, jm, im, j1, leaf, sink)
	c.refine(im, jm, i1, j1, leaf, sink)
}

// ssiCrossings counts how many of the cell's four edges change sign — 2 for a simple transversal curve
// passing through, 0 for an interior-only feature, 4 for a saddle (ambiguous, keep refining).
func ssiCrossings(s00, s10, s01, s11 ssiSample) int {
	n := 0
	for _, e := range [4][2]float64{{s00.f, s10.f}, {s00.f, s01.f}, {s10.f, s11.f}, {s01.f, s11.f}} {
		if straddlesZero(e[0], e[1]) {
			n++
		}
	}
	return n
}

// emitLeaf appends seeds for a leaf cell: a bisected crossing on each edge that changes sign, and — when
// no edge crosses — the cell's smallest-|f| corner, the seed for a sub-cell loop or tangency that never
// straddles an edge. Duplicate seeds along a shared curve are harmless: the tracer dedups them.
func (c *ssiSeedField) emitLeaf(i0, j0, i1, j1 int, s00, s10, s01, s11 ssiSample, sink *ssiSeedSink) {
	field := func(u, v float64) float64 { return SignedDistanceToSurface(c.other, c.base.PointAt(u, v)) }
	u0, v0 := c.u0+float64(i0)*c.du, c.v0+float64(j0)*c.dv
	u1, v1 := c.u0+float64(i1)*c.du, c.v0+float64(j1)*c.dv
	crossed := false
	emit := func(b math.Point3) { sink.crossings = append(sink.crossings, b); crossed = true }
	if straddlesZero(s00.f, s10.f) {
		emit(bisectEdge(c.base, field, u0, v0, s00.f, u1, v0))
	}
	if straddlesZero(s00.f, s01.f) {
		emit(bisectEdge(c.base, field, u0, v0, s00.f, u0, v1))
	}
	if straddlesZero(s10.f, s11.f) {
		emit(bisectEdge(c.base, field, u1, v0, s10.f, u1, v1))
	}
	if straddlesZero(s01.f, s11.f) {
		emit(bisectEdge(c.base, field, u0, v1, s01.f, u1, v1))
	}
	if !crossed {
		sink.interior = append(sink.interior, c.minCorner(i0, j0, i1, j1, s00, s10, s01, s11))
	}
}

// minCorner returns the base point at the cell corner whose field is smallest in magnitude — the best
// single seed for a sub-cell loop or tangency that does not straddle any edge.
func (c *ssiSeedField) minCorner(i0, j0, i1, j1 int, s00, s10, s01, s11 ssiSample) math.Point3 {
	best, bi, bj := stdmath.Abs(s00.f), i0, j0
	for _, cand := range []struct {
		f    float64
		i, j int
	}{
		{stdmath.Abs(s10.f), i1, j0},
		{stdmath.Abs(s01.f), i0, j1},
		{stdmath.Abs(s11.f), i1, j1},
	} {
		if cand.f < best {
			best, bi, bj = cand.f, cand.i, cand.j
		}
	}
	return c.point(bi, bj)
}

// minOf4 returns the smallest of four values.
func minOf4(a, b, c, d float64) float64 {
	return stdmath.Min(stdmath.Min(a, b), stdmath.Min(c, d))
}

// maxOf4 returns the largest of four values.
func maxOf4(a, b, c, d float64) float64 {
	return stdmath.Max(stdmath.Max(a, b), stdmath.Max(c, d))
}
