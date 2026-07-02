// SPDX-License-Identifier: GPL-2.0-only

package solve

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// Audit A13 (#1609): solver robustness vs solvespace practice — redundant-constraint
// identification, scale-invariant CPQR rank, Marquardt-scaled Moré/Nielsen LM damping, and a
// relative finite-difference step.

// linRes is a linear residual source a·x − c with exact partials, for constructing systems
// with known rank/redundancy.
type linRes struct {
	vars []*math.Scalar
	a    [][]float64
	c    []float64
}

func (l linRes) Residuals() []float64 {
	out := make([]float64, len(l.a))
	for i, row := range l.a {
		s := -l.c[i]
		for k, v := range l.vars {
			s += row[k] * float64(*v)
		}
		out[i] = s
	}
	return out
}

func (l linRes) Partials(int) []float64 { return nil } // not used; FD covers it

// TestRedundantSourcesNamesTheRemovableConstraint: x=1, y=2, and a REDUNDANT duplicate of x=1
// — the leave-one-out search must name both copies of the dependent constraint (removing
// either restores full rank) and never the independent y constraint.
func TestRedundantSourcesNamesTheRemovableConstraint(t *testing.T) {
	x, y := math.Scalar(0), math.Scalar(0)
	vars := []*math.Scalar{&x, &y}
	res := []Residual{
		linRes{vars, [][]float64{{1, 0}}, []float64{1}}, // 0: x = 1
		linRes{vars, [][]float64{{0, 1}}, []float64{2}}, // 1: y = 2
		linRes{vars, [][]float64{{2, 0}}, []float64{2}}, // 2: 2x = 2 — dependent on 0
	}
	got := RedundantSources(res, vars)
	want := map[int]bool{0: true, 2: true}
	if len(got) != 2 || !want[got[0]] || !want[got[1]] {
		t.Errorf("RedundantSources = %v, want the two dependent x-constraints {0, 2}", got)
	}
}

// TestRankIsScaleInvariant: the same rank-2 system evaluated at 1e-6…1e6 coordinate scales
// must report identical rank — the retired absolute Gauss–Jordan threshold dropped rank at
// scale (audit A13).
func TestRankIsScaleInvariant(t *testing.T) {
	for _, scale := range []float64{1e-6, 1e-3, 1, 1e3, 1e6} {
		j := [][]float64{
			{scale, 0, 0},
			{0, scale, 0},
			{scale, scale, 0}, // dependent
		}
		if rank := pivotedQRRank(rowNormalized(j)); rank != 2 {
			t.Errorf("scale %g: rank = %d, want 2", scale, rank)
		}
	}
}

// TestPivotedQRRankKnownDeficiency: constructed Jacobians with known ranks across magnitudes.
func TestPivotedQRRankKnownDeficiency(t *testing.T) {
	cases := []struct {
		j    [][]float64
		want int
	}{
		{[][]float64{{1, 2}, {2, 4}}, 1}, // proportional rows
		{[][]float64{{1, 0}, {0, 1}}, 2}, // identity
		// NOTE deliberately absent: {{1e6,0},{0,1e-6}} — a 1e12 pivot spread IS numerical
		// rank 1 at relTol 1e-9 (Golub & Van Loan numerical rank); the production path
		// row-normalizes first, covered by TestRankIsScaleInvariant.
		{[][]float64{{1, 1}, {1, 1 + 1e-13}}, 1},             // numerically dependent
		{[][]float64{{0, 0}, {0, 0}}, 0},                     // zero
		{[][]float64{{3, 1, 2}, {6, 2, 4}, {1, 0, 0}}, 2},    // one dependent of three
		{[][]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1e-4}}, 3}, // small but genuine pivot
	}
	for i, c := range cases {
		if got := pivotedQRRank(c.j); got != c.want {
			t.Errorf("case %d: rank = %d, want %d", i, got, c.want)
		}
	}
}

// distRes is a nonlinear distance residual |p−q|−d with FD partials, for convergence fixtures.
type distRes struct {
	px, py, qx, qy *math.Scalar
	d              float64
}

func (r distRes) Residuals() []float64 {
	dx := float64(*r.qx - *r.px)
	dy := float64(*r.qy - *r.py)
	return []float64{stdmath.Hypot(dx, dy) - r.d}
}

// TestIllScaledSystemConverges: dimensions spanning 1e-2…1e3 in one system — the retired ×10
// 8-try damping cap declared such systems falsely stuck; Marquardt scaling with Moré/Nielsen
// damping must converge (audit A13). The false-stuck verdict froze whole dialog flows through
// the sick-config commit gate (#1595).
func TestIllScaledSystemConverges(t *testing.T) {
	ax, ay := math.Scalar(0), math.Scalar(0)
	bx, by := math.Scalar(900), math.Scalar(100) // far from the 1000-length target
	cx, cy := math.Scalar(900.004), math.Scalar(100.002)
	vars := []*math.Scalar{&bx, &by, &cx, &cy}
	res := []Residual{
		distRes{&ax, &ay, &bx, &by, 1000}, // large dimension
		distRes{&bx, &by, &cx, &cy, 0.01}, // tiny dimension hanging off it
	}
	r := Solve(res, vars, Options{})
	if !r.Converged {
		t.Fatalf("ill-scaled but solvable system reported stuck: residual %g after %d iterations", r.Residual, r.Iterations)
	}
	if got := stdmath.Hypot(float64(bx), float64(by)); stdmath.Abs(got-1000) > 1e-6*1000 {
		t.Errorf("|b| = %g, want 1000", got)
	}
	if got := stdmath.Hypot(float64(cx-bx), float64(cy-by)); stdmath.Abs(got-0.01) > 1e-4 {
		t.Errorf("|c-b| = %g, want 0.01", got)
	}
}

// TestLargeCoordinateSystemKeepsRank: a well-constrained system translated to (1e5, 1e5) must
// keep DOF=0 — the absolute FD step and absolute rank threshold both used to break this.
func TestLargeCoordinateSystemKeepsRank(t *testing.T) {
	const off = 1e5
	px, py := math.Scalar(off), math.Scalar(off)
	qx, qy := math.Scalar(off+3), math.Scalar(off)
	vars := []*math.Scalar{&qx, &qy}
	res := []Residual{
		distRes{&px, &py, &qx, &qy, 5},                            // distance from the fixed p
		linRes{vars, [][]float64{{0, 1}}, []float64{float64(py)}}, // qy pinned at the offset height
	}
	r := Solve(res, vars, Options{})
	if !r.Converged {
		t.Fatalf("large-coordinate solve stuck: residual %g", r.Residual)
	}
	if r.DOF != 0 || r.Redundant != 0 {
		t.Errorf("DOF analysis at 1e5 = DOF %d, redundant %d, want 0/0 (phantom rank drop)", r.DOF, r.Redundant)
	}
}

// TestVerdictsAreScaleInvariant: the same system at 1e-3× and 1e3× uniform scale must produce
// identical DOF and redundancy verdicts.
func TestVerdictsAreScaleInvariant(t *testing.T) {
	verdict := func(scale float64) DOFAnalysis {
		px, py := math.Scalar(0), math.Scalar(0)
		qx, qy := math.Scalar(3*scale), math.Scalar(0)
		vars := []*math.Scalar{&qx, &qy}
		res := []Residual{
			distRes{&px, &py, &qx, &qy, 5 * scale},
			linRes{vars, [][]float64{{0, 1}}, []float64{0}},
		}
		return Solve(res, vars, Options{}).DOFAnalysis
	}
	small, big := verdict(1e-3), verdict(1e3)
	if small.DOF != big.DOF || small.Redundant != big.Redundant || small.Status != big.Status {
		t.Errorf("verdicts differ across scale: 1e-3 → %+v, 1e3 → %+v", small, big)
	}
}
