// SPDX-License-Identifier: GPL-2.0-only

package solve

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// diffGap is a Differentiable version of gap: residual = (b−a)−dist, with exact
// partials ∂r/∂a = −1, ∂r/∂b = +1. It also records whether its variables were ever
// read at a perturbed value, to prove the analytic path never moves them.
type diffGap struct {
	a, b *math.Scalar
	dist float64
}

func (g diffGap) Residuals() []float64      { return []float64{(*g.b - *g.a) - g.dist} }
func (g diffGap) Variables() []*math.Scalar { return []*math.Scalar{g.a, g.b} }
func (g diffGap) Partials() [][]float64     { return [][]float64{{-1, 1}} }

// nonDiff is a plain residual (no Partials) — it forces the finite-difference fallback.
type nonDiff struct{ v *math.Scalar }

func (n nonDiff) Residuals() []float64 { return []float64{*n.v * *n.v} }

func TestAnalyticJacobianMatchesFiniteDifference(t *testing.T) {
	a, b := math.Scalar(1.5), math.Scalar(4.0)
	vars := []*math.Scalar{&a, &b}
	src := []Residual{diffGap{&a, &b, 2}}

	analytic, ok := analyticJacobian(src, vars)
	if !ok {
		t.Fatal("analyticJacobian rejected an all-differentiable system")
	}
	fd := finiteDiffJacobian(src, vars)
	for i := range analytic {
		for j := range analytic[i] {
			if stdmath.Abs(analytic[i][j]-fd[i][j]) > 1e-6 {
				t.Errorf("J[%d][%d]: analytic %v vs FD %v", i, j, analytic[i][j], fd[i][j])
			}
		}
	}
}

func TestAnalyticJacobianDoesNotPerturbLiveVariables(t *testing.T) {
	a, b := math.Scalar(1.5), math.Scalar(4.0)
	// A NaN guard: if the analytic path mutated a live variable mid-evaluation, a
	// concurrent reader could observe the perturbed value. We assert the exact values
	// are untouched after the call (the FD path would restore them, but only after
	// transiently writing orig±h).
	_, ok := analyticJacobian([]Residual{diffGap{&a, &b, 2}}, []*math.Scalar{&a, &b})
	if !ok {
		t.Fatal("expected analytic path")
	}
	if a != 1.5 || b != 4.0 {
		t.Errorf("live variables changed to (%v, %v), want (1.5, 4.0)", a, b)
	}
}

func TestJacobianFallsBackToFiniteDifference(t *testing.T) {
	v := math.Scalar(3)
	// A non-differentiable source forces FD; the derivative of v² at 3 is 6.
	j := Jacobian([]Residual{nonDiff{&v}}, []*math.Scalar{&v})
	if len(j) != 1 || stdmath.Abs(j[0][0]-6) > 1e-4 {
		t.Errorf("FD fallback Jacobian = %v, want ≈[[6]]", j)
	}
}

func TestMixedDifferentiabilityFallsBack(t *testing.T) {
	a, b := math.Scalar(1), math.Scalar(2)
	// One differentiable, one not → the whole system must finite-difference.
	if _, ok := analyticJacobian([]Residual{diffGap{&a, &b, 1}, nonDiff{&a}}, []*math.Scalar{&a, &b}); ok {
		t.Error("analyticJacobian accepted a system with a non-differentiable source")
	}
}

// sharedVar is a Differentiable whose Variables list the SAME pointer twice — the shape
// EqualLength takes on two lines sharing a vertex. residual = 3·x, with the dependence
// split across the two occurrences (partials 1 and 2), so the true column derivative is
// their sum, 3.
type sharedVar struct{ x *math.Scalar }

func (s sharedVar) Residuals() []float64      { return []float64{3 * *s.x} }
func (s sharedVar) Variables() []*math.Scalar { return []*math.Scalar{s.x, s.x} }
func (s sharedVar) Partials() [][]float64     { return [][]float64{{1, 2}} }

func TestAnalyticJacobianAccumulatesRepeatedVariable(t *testing.T) {
	x := math.Scalar(1)
	vars := []*math.Scalar{&x}
	j, ok := analyticJacobian([]Residual{sharedVar{&x}}, vars)
	if !ok {
		t.Fatal("expected analytic path")
	}
	// The repeated-variable partials (1 and 2) must SUM into the single column (3), the
	// same total the finite-difference path sees by perturbing x once.
	fd := finiteDiffJacobian([]Residual{sharedVar{&x}}, vars)
	if stdmath.Abs(j[0][0]-3) > 1e-9 || stdmath.Abs(j[0][0]-fd[0][0]) > 1e-6 {
		t.Errorf("repeated-variable column = %v, want 3 (FD says %v)", j[0][0], fd[0][0])
	}
}

func TestSolveReusesConvergedJacobian(t *testing.T) {
	a, b := math.Scalar(0), math.Scalar(0)
	r := Solve([]Residual{diffGap{&a, &b, 2}}, []*math.Scalar{&a, &b}, Options{})
	if r.Jacobian == nil {
		t.Fatal("Result.Jacobian not populated for reuse")
	}
	// The carried Jacobian must reproduce the reported DOF analysis.
	if got := analyzeJacobian(r.Jacobian, 2); got != r.DOFAnalysis {
		t.Errorf("DOF from carried Jacobian = %+v, want %+v", got, r.DOFAnalysis)
	}
}
