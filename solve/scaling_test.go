// SPDX-License-Identifier: GPL-2.0-only

package solve

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func TestModelScaleTracksLargestCoordinate(t *testing.T) {
	a, b := math.Scalar(-7), math.Scalar(3)
	if s := modelScale([]*math.Scalar{&a, &b}); s != 7 {
		t.Errorf("modelScale = %v, want 7 (largest |coordinate|)", s)
	}
	// All-at-origin (degenerate) must not yield a zero scale (a zero tolerance threshold).
	z := math.Scalar(0)
	if s := modelScale([]*math.Scalar{&z}); s != 1 {
		t.Errorf("degenerate modelScale = %v, want the 1 floor", s)
	}
}

func TestRowWeightsHandlesZeroRow(t *testing.T) {
	// A non-zero row gets 1/‖row‖; an all-zero row (a constraint with no active variable)
	// gets weight 1 rather than dividing by zero.
	w := rowWeights([][]float64{{3, 4}, {0, 0}})
	if stdmath.Abs(w[0]-0.2) > 1e-12 {
		t.Errorf("w[0] = %v, want 0.2 (1/5)", w[0])
	}
	if w[1] != 1 {
		t.Errorf("zero-row weight = %v, want 1", w[1])
	}
}

func TestConflictingSourcesReportsUnsatisfiableSubset(t *testing.T) {
	// x pinned to 0 by one source and to 5 by another — irreconcilable. The least-squares
	// solution lands at 2.5, leaving both sources unsatisfied; both are reported, most
	// severe first.
	x := math.Scalar(0)
	res := []Residual{fixedValue{&x, 0}, fixedValue{&x, 5}}
	r := Solve(res, []*math.Scalar{&x}, Options{})
	if r.Converged {
		t.Fatal("expected the contradictory system not to converge")
	}
	if len(r.Conflicts) == 0 {
		t.Fatal("Result.Conflicts empty for a contradictory system, want the offending subset")
	}
	// The source asking for 5 is farther from x=2.5? Both are 2.5 away — but the report must
	// at least contain the offenders, not be a bare failure.
	for _, idx := range r.Conflicts {
		if idx < 0 || idx >= len(res) {
			t.Errorf("conflict index %d out of range", idx)
		}
	}
}

func TestSolveScaleInvariance(t *testing.T) {
	// gap{a,b,dist} at unit scale and ×1e6: both converge and report 1 DOF — the relative
	// tolerance makes the outcome scale-independent.
	run := func(k float64) Result {
		a := math.Scalar(0)
		b := math.Scalar(0.1 * k)
		return Solve([]Residual{gap{&a, &b, k}}, []*math.Scalar{&a, &b}, Options{})
	}
	unit, big := run(1), run(1e6)
	if !unit.Converged || !big.Converged {
		t.Fatalf("convergence parity broken: unit=%v big=%v", unit.Converged, big.Converged)
	}
	if unit.DOF != big.DOF || unit.Status != big.Status {
		t.Errorf("DOF/status parity broken: unit %+v vs big %+v", unit.DOFAnalysis, big.DOFAnalysis)
	}
}

func TestAnalyzeDOFMatchesSolveResult(t *testing.T) {
	a, b := math.Scalar(0), math.Scalar(0)
	vars := []*math.Scalar{&a, &b}
	res := []Residual{gap{&a, &b, 2}}
	Solve(res, vars, Options{})
	if got, want := AnalyzeDOF(res, vars), (DOFAnalysis{Variables: 2, Equations: 1, Rank: 1, DOF: 1, Redundant: 0, Status: UnderConstrained}); got != want {
		t.Errorf("AnalyzeDOF = %+v, want %+v", got, want)
	}
}
