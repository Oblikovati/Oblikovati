// SPDX-License-Identifier: GPL-2.0-only

package geom

// Degree elevation on homogeneous control points (Piegl & Tiller A5.9). Elevation is
// *exact*: it raises the polynomial degree by t and adds control points while leaving
// the curve/surface geometry unchanged. It is the precursor to matching the degrees of
// two curves before a tensor-product surface op, and to reaching a higher-continuity
// (G2/G3) representation (M36-F01). The algorithm's many phases share mutable working
// state, so they are gathered on [degElevator] to keep each routine small.

// elevateDegreeHomog raises the degree of the homogeneous B-spline by t, returning the
// new knot vector and control points. t <= 0 is a no-op copy.
func elevateDegreeHomog(p, t int, U []float64, pw []hpoint4) (newU []float64, newPw []hpoint4) {
	if t <= 0 {
		return append([]float64(nil), U...), append([]hpoint4(nil), pw...)
	}
	return newDegElevator(p, t, U, pw).run()
}

// degElevator holds the working state of A5.9: the precomputed Bézier elevation
// coefficients, the per-segment scratch polygons, and the output cursors.
type degElevator struct {
	p, t, ph, ph2, m int
	U                []float64
	pw               []hpoint4
	bezalfs          [][]float64
	bpts, ebpts      []hpoint4
	nextbpts         []hpoint4
	alfs             []float64
	Uh               []float64
	Qw               []hpoint4
	kind, cind, mh   int
	r, a, b          int
	ua, ub           float64
}

// newDegElevator sizes the output and scratch arrays exactly (every distinct knot ends
// up with its multiplicity raised by t) and precomputes the Bézier elevation table.
func newDegElevator(p, t int, U []float64, pw []hpoint4) *degElevator {
	n := len(pw) - 1
	e := &degElevator{p: p, t: t, ph: p + t, ph2: (p + t) / 2, m: n + p + 1, U: U, pw: pw}
	knotLen := len(U) + t*distinctCount(U)
	e.Uh = make([]float64, knotLen)
	e.Qw = make([]hpoint4, knotLen-e.ph-1)
	e.bpts = make([]hpoint4, p+1)
	e.ebpts = make([]hpoint4, e.ph+1)
	e.nextbpts = make([]hpoint4, max(p-1, 0))
	e.alfs = make([]float64, max(p-1, 0))
	e.initBezalfs()
	return e
}

// initBezalfs fills the bezalfs[i][j] coefficients that elevate a degree-p Bézier
// segment to degree ph (A5.9, eqs. via binomials, with the symmetric upper half mirrored).
func (e *degElevator) initBezalfs() {
	e.bezalfs = make([][]float64, e.ph+1)
	for i := range e.bezalfs {
		e.bezalfs[i] = make([]float64, e.p+1)
	}
	e.bezalfs[0][0] = 1
	e.bezalfs[e.ph][e.p] = 1
	e.fillBezalfsLow()
	e.fillBezalfsHigh()
}

// fillBezalfsLow computes the lower-index half of the coefficient table directly.
func (e *degElevator) fillBezalfsLow() {
	for i := 1; i <= e.ph2; i++ {
		inv := 1 / binomial(e.ph, i)
		mpi := min(e.p, i)
		for j := max(0, i-e.t); j <= mpi; j++ {
			e.bezalfs[i][j] = inv * binomial(e.p, j) * binomial(e.t, i-j)
		}
	}
}

// fillBezalfsHigh mirrors the upper-index half from the (already computed) lower half.
func (e *degElevator) fillBezalfsHigh() {
	for i := e.ph2 + 1; i <= e.ph-1; i++ {
		mpi := min(e.p, i)
		for j := max(0, i-e.t); j <= mpi; j++ {
			e.bezalfs[i][j] = e.bezalfs[e.ph-i][e.p-j]
		}
	}
}

// run executes the main A5.9 loop, returning the elevated knot vector and control points.
func (e *degElevator) run() ([]float64, []hpoint4) {
	e.mh, e.kind, e.r, e.a, e.b, e.cind = e.ph, e.ph+1, -1, e.p, e.p+1, 1
	e.ua = e.U[0]
	e.Qw[0] = e.pw[0]
	for i := 0; i <= e.ph; i++ {
		e.Uh[i] = e.ua
	}
	copy(e.bpts, e.pw[:e.p+1])
	for e.b < e.m {
		e.elevateSegment()
	}
	nh := e.mh - e.ph - 1
	return e.Uh[:e.mh+1], e.Qw[:nh+1]
}

// elevateSegment processes one distinct-knot span: isolate its Bézier segment, elevate
// it, remove the surplus knots elevation introduced, and emit the new knots and points.
func (e *degElevator) elevateSegment() {
	mul := e.advanceToNextDistinct()
	oldr := e.r
	e.r = e.p - mul
	lbz, rbz := e.bezierBounds(oldr)
	if e.r > 0 {
		e.insertToBezier(mul)
	}
	e.elevateBezier(lbz)
	if oldr > 1 {
		e.removeAfterElevation(oldr, lbz)
	}
	e.loadKnot(oldr)
	e.loadCtrl(lbz, rbz)
	e.setupNext()
}

// advanceToNextDistinct scans b to the end of the current repeated-knot run, folding the
// knot's multiplicity into mh and recording the run's end value in ub.
func (e *degElevator) advanceToNextDistinct() (mul int) {
	i := e.b
	for e.b < e.m && e.U[e.b] == e.U[e.b+1] {
		e.b++
	}
	mul = e.b - i + 1
	e.mh += mul + e.t
	e.ub = e.U[e.b]
	return mul
}

// bezierBounds returns the Bézier index range [lbz, rbz] that the elevated segment's
// control points occupy, accounting for the knots shared with the neighbouring segments.
func (e *degElevator) bezierBounds(oldr int) (lbz, rbz int) {
	lbz, rbz = 1, e.ph
	if oldr > 0 {
		lbz = (oldr + 2) / 2
	}
	if e.r > 0 {
		rbz = e.ph - (e.r+1)/2
	}
	return lbz, rbz
}

// insertToBezier inserts the span-end knot r times so bpts becomes a pure Bézier segment,
// stashing the spill-over points for the next span in nextbpts.
func (e *degElevator) insertToBezier(mul int) {
	numer := e.ub - e.ua
	for k := e.p; k > mul; k-- {
		e.alfs[k-mul-1] = numer / (e.U[e.a+k] - e.ua)
	}
	for j := 1; j <= e.r; j++ {
		save := e.r - j
		s := mul + j
		for k := e.p; k >= s; k-- {
			e.bpts[k] = e.bpts[k].lerp(e.bpts[k-1], 1-e.alfs[k-s])
		}
		e.nextbpts[save] = e.bpts[e.p]
	}
}

// elevateBezier raises the current Bézier segment bpts to degree ph in ebpts via bezalfs.
func (e *degElevator) elevateBezier(lbz int) {
	for i := lbz; i <= e.ph; i++ {
		e.ebpts[i] = hpoint4{}
		mpi := min(e.p, i)
		for j := max(0, i-e.t); j <= mpi; j++ {
			e.ebpts[i] = e.ebpts[i].add(e.bpts[j].scale(e.bezalfs[i][j]))
		}
	}
}

// removeAfterElevation removes the knot at the span start oldr times — elevation
// over-inserts it, and these passes restore the correct multiplicity (A5.9 inner block).
func (e *degElevator) removeAfterElevation(oldr, lbz int) {
	first, last := e.kind-2, e.kind
	den := e.ub - e.ua
	bet := (e.ub - e.Uh[e.kind-1]) / den
	for tr := 1; tr < oldr; tr++ {
		e.removalPass(tr, oldr, lbz, first, last, den, bet)
		first--
		last++
	}
}

// removalPass runs one knot-removal sweep over the already-emitted Qw and the elevated
// ebpts for the converging index window.
func (e *degElevator) removalPass(tr, oldr, lbz, first, last int, den, bet float64) {
	i, j := first, last
	kj := j - e.kind + 1
	for j-i > tr {
		if i < e.cind {
			alf := (e.ub - e.Uh[i]) / (e.ua - e.Uh[i])
			e.Qw[i] = e.Qw[i].lerp(e.Qw[i-1], 1-alf)
		}
		if j >= lbz {
			e.ebpts[kj] = e.removalBlend(j, tr, kj, oldr, den, bet)
		}
		i, j, kj = i+1, j-1, kj-1
	}
}

// removalBlend chooses A5.9's interior (gamma) or boundary (beta) blend for one ebpts entry.
func (e *degElevator) removalBlend(j, tr, kj, oldr int, den, bet float64) hpoint4 {
	if j-tr <= e.kind-e.ph+oldr {
		gam := (e.ub - e.Uh[j-tr]) / den
		return e.ebpts[kj].lerp(e.ebpts[kj+1], 1-gam)
	}
	return e.ebpts[kj].lerp(e.ebpts[kj+1], 1-bet)
}

// loadKnot writes the span-start knot ua into Uh with its elevated multiplicity (ph−oldr),
// except for the very first span whose start knots are loaded before the loop.
func (e *degElevator) loadKnot(oldr int) {
	if e.a == e.p {
		return
	}
	for i := 0; i < e.ph-oldr; i++ {
		e.Uh[e.kind] = e.ua
		e.kind++
	}
}

// loadCtrl emits the elevated segment's control points [lbz, rbz] into the output Qw.
func (e *degElevator) loadCtrl(lbz, rbz int) {
	for j := lbz; j <= rbz; j++ {
		e.Qw[e.cind] = e.ebpts[j]
		e.cind++
	}
}

// setupNext primes bpts for the following span (or loads the clamped end knots when done).
func (e *degElevator) setupNext() {
	if e.b >= e.m {
		for i := 0; i <= e.ph; i++ {
			e.Uh[e.kind+i] = e.ub
		}
		return
	}
	for j := 0; j < e.r; j++ {
		e.bpts[j] = e.nextbpts[j]
	}
	for j := e.r; j <= e.p; j++ {
		e.bpts[j] = e.pw[e.b-e.p+j]
	}
	e.a, e.b, e.ua = e.b, e.b+1, e.ub
}

// distinctCount returns the number of distinct knot values in U (consecutive equal
// entries count once), the multiplier behind the elevated array sizing.
func distinctCount(U []float64) int {
	count := 0
	for i := range U {
		if i == 0 || U[i] != U[i-1] {
			count++
		}
	}
	return count
}

// binomial returns the binomial coefficient C(n, k) as a float64 (0 outside 0..n).
func binomial(n, k int) float64 {
	if k < 0 || k > n {
		return 0
	}
	if k > n-k {
		k = n - k
	}
	c := 1.0
	for i := 0; i < k; i++ {
		c = c * float64(n-i) / float64(i+1)
	}
	return c
}
