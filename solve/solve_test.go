// SPDX-License-Identifier: GPL-2.0-only

package solve

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// fixedValue is a residual source pinning a single scalar to a target — the minimal
// constraint for exercising the shared solver: residual = v − target.
type fixedValue struct {
	v      *math.Scalar
	target float64
}

func (f fixedValue) Residuals() []float64 { return []float64{*f.v - f.target} }

// gap holds two scalars a fixed distance apart: residual = (b − a) − dist.
type gap struct {
	a, b *math.Scalar
	dist float64
}

func (g gap) Residuals() []float64 { return []float64{(*g.b - *g.a) - g.dist} }

func TestSolveDrivesSingleVariableToTarget(t *testing.T) {
	x := math.Scalar(0)
	r := Solve([]Residual{fixedValue{&x, 5}}, []*math.Scalar{&x}, Options{})
	if !r.Converged {
		t.Fatalf("did not converge: %+v", r)
	}
	if stdmath.Abs(x-5) > 1e-6 {
		t.Errorf("x = %v, want 5", x)
	}
	if r.Status != WellConstrained {
		t.Errorf("status = %v, want well-constrained", r.Status)
	}
}

func TestSolveReportsUnderConstrained(t *testing.T) {
	a := math.Scalar(0)
	b := math.Scalar(0)
	// One equation, two free variables → 1 DOF remains (the pair can slide together).
	r := Solve([]Residual{gap{&a, &b, 2}}, []*math.Scalar{&a, &b}, Options{})
	if !r.Converged {
		t.Fatalf("did not converge: %+v", r)
	}
	if stdmath.Abs((b-a)-2) > 1e-6 {
		t.Errorf("b-a = %v, want 2", b-a)
	}
	if r.Status != UnderConstrained || r.DOF != 1 {
		t.Errorf("DOF analysis = %+v, want under-constrained with 1 DOF", r.DOFAnalysis)
	}
}

func TestSolveReportsOverConstrained(t *testing.T) {
	x := math.Scalar(0)
	// Two residuals pinning one variable — the second row is redundant.
	r := Solve([]Residual{fixedValue{&x, 3}, fixedValue{&x, 3}}, []*math.Scalar{&x}, Options{})
	if r.Status != OverConstrained || r.Redundant != 1 {
		t.Errorf("DOF analysis = %+v, want over-constrained with 1 redundant", r.DOFAnalysis)
	}
}

func TestStatusString(t *testing.T) {
	cases := map[Status]string{
		WellConstrained:  "well-constrained",
		UnderConstrained: "under-constrained",
		OverConstrained:  "over-constrained",
	}
	for st, want := range cases {
		if got := st.String(); got != want {
			t.Errorf("Status(%d).String() = %q, want %q", st, got, want)
		}
	}
}
