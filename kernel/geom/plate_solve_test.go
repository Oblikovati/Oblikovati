// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"
)

// quadratic q = 1 + 2u − v + 0.5u² + 3uv − 2v² and its gradient. q lies in the plate's
// reproduction span {1,u,v,u²,uv,v²}, so the UNIQUE Duchon interpolant of its samples is q
// itself (all RBF weights vanish). It is the closed-form oracle for the assembly + solve
// (kit §5b) — asserted against these exact values, NOT the solver's own fixed point.
func reproQuad(u, v float64) float64  { return 1 + 2*u - v + 0.5*u*u + 3*u*v - 2*v*v }
func reproQuadU(u, v float64) float64 { return 2 + u + 3*v }
func reproQuadV(u, v float64) float64 { return -1 + 3*u - 4*v }

// reproPoints are the 7 sample sites from kit §5b (unisolvent for the quadratic border).
func reproPoints() [][2]float64 {
	return [][2]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}, {0.5, 0.5}, {0.2, 0.8}, {0.8, 0.3}}
}

// TestPlateKernelPartialsMatchKit checks the §1 closed forms against the kit's confirmed values
// at P=(0.3,−0.4) (R=0.25, log R=−1.3862944) — this is the transcription-grade guard on the
// error-prone derivative table.
func TestPlateKernelPartialsMatchKit(t *testing.T) {
	const du, dv, floor = 0.3, -0.4, 1e-30
	cases := []struct {
		name string
		got  float64
		want float64
	}{
		{"E", plateE(du, dv, floor), -0.086643398},
		{"E_u", plateEu(du, dv, floor), -0.265888308},
		{"E_v", plateEv(du, dv, floor), 0.354517744},
		{"E_uu", plateEuu(du, dv, floor), -0.804426301},
		{"E_vv", plateEvv(du, dv, floor), -0.740751143},
		{"E_uv", plateEuv(du, dv, floor), -0.109157413},
	}
	for _, c := range cases {
		if stdmath.Abs(c.got-c.want) > 1e-8 {
			t.Errorf("%s = %.9f, want %.9f (kit §5a)", c.name, c.got, c.want)
		}
	}
}

// TestPlateKernelPartialsVsFiniteDiff cross-checks every §1 partial against a central finite
// difference of plateE at (0.3,−0.4) — the independent oracle for the derivative formulas.
func TestPlateKernelPartialsVsFiniteDiff(t *testing.T) {
	const du, dv, floor, h = 0.3, -0.4, 1e-30, 1e-6
	fdU := (plateE(du+h, dv, floor) - plateE(du-h, dv, floor)) / (2 * h)
	fdV := (plateE(du, dv+h, floor) - plateE(du, dv-h, floor)) / (2 * h)
	fdUU := (plateEu(du+h, dv, floor) - plateEu(du-h, dv, floor)) / (2 * h)
	fdVV := (plateEv(du, dv+h, floor) - plateEv(du, dv-h, floor)) / (2 * h)
	fdUV := (plateEu(du, dv+h, floor) - plateEu(du, dv-h, floor)) / (2 * h)
	maxErr := 0.0
	for _, p := range [][2]float64{
		{plateEu(du, dv, floor), fdU}, {plateEv(du, dv, floor), fdV},
		{plateEuu(du, dv, floor), fdUU}, {plateEvv(du, dv, floor), fdVV},
		{plateEuv(du, dv, floor), fdUV},
	} {
		if e := stdmath.Abs(p[0] - p[1]); e > maxErr {
			maxErr = e
		}
	}
	if maxErr > 1e-4 {
		t.Fatalf("analytic vs finite-difference partials disagree by %.3g (> 1e-4)", maxErr)
	}
	t.Logf("FD-partial max error = %.3g", maxErr)
}

// TestPlateSingularityIsRemovable confirms E and every partial return the exact 0 limit inside
// the rFloor guard (self-term P_i−P_i = 0) instead of log-ing a clamped R.
func TestPlateSingularityIsRemovable(t *testing.T) {
	const floor = 1e-18
	for _, f := range []func(a, b, c float64) float64{plateE, plateEu, plateEv, plateEuu, plateEvv, plateEuv} {
		if v := f(0, 0, floor); v != 0 {
			t.Errorf("kernel at r=0 = %v, want 0 (removable singularity)", v)
		}
	}
}

// TestPlateReproducesQuadratic is the RED self-check (kit §5b): 7 G0 samples of q → λ≈0, poly
// coefficients = q's, and Eval(0.4,0.6)=1.28, all against closed-form oracles to 1e-9.
func TestPlateReproducesQuadratic(t *testing.T) {
	var cs []PlateConstraint
	for _, p := range reproPoints() {
		cs = append(cs, PlateConstraint{U: p[0], V: p[1], Order: [2]int{0, 0}, Value: reproQuad(p[0], p[1])})
	}
	coeffs, err := PlateSolve(cs)
	if err != nil {
		t.Fatalf("PlateSolve returned error: %v", err)
	}
	assertLambdaVanishes(t, coeffs)
	assertPoly(t, coeffs.poly, [6]float64{1, 2, -1, 0.5, 3, -2})
	if got := coeffs.Eval(0.4, 0.6); stdmath.Abs(got-1.28) > 1e-9 {
		t.Errorf("Eval(0.4,0.6) = %.12f, want 1.28 (= q(0.4,0.6))", got)
	}
	t.Logf("Eval(0.4,0.6) = %.12f", coeffs.Eval(0.4, 0.6))
}

// TestPlateReproducesQuadraticWithG1 extends §5b with ∂u/∂v rows (values q_u,q_v) — exercising
// the derivative rows in K and P. The interpolant is still exactly q, λ still 0.
func TestPlateReproducesQuadraticWithG1(t *testing.T) {
	var cs []PlateConstraint
	for _, p := range reproPoints() {
		u, v := p[0], p[1]
		cs = append(cs,
			PlateConstraint{U: u, V: v, Order: [2]int{0, 0}, Value: reproQuad(u, v)},
			PlateConstraint{U: u, V: v, Order: [2]int{1, 0}, Value: reproQuadU(u, v)},
			PlateConstraint{U: u, V: v, Order: [2]int{0, 1}, Value: reproQuadV(u, v)})
	}
	coeffs, err := PlateSolve(cs)
	if err != nil {
		t.Fatalf("PlateSolve (G1) returned error: %v", err)
	}
	assertLambdaVanishes(t, coeffs)
	assertPoly(t, coeffs.poly, [6]float64{1, 2, -1, 0.5, 3, -2})
	if got := coeffs.Eval(0.4, 0.6); stdmath.Abs(got-1.28) > 1e-9 {
		t.Errorf("Eval(0.4,0.6) with G1 rows = %.12f, want 1.28", got)
	}
}

// TestPlateEvalGradMatchesQuadratic checks EvalGrad reproduces q's analytic gradient across Ω.
func TestPlateEvalGradMatchesQuadratic(t *testing.T) {
	var cs []PlateConstraint
	for _, p := range reproPoints() {
		cs = append(cs, PlateConstraint{U: p[0], V: p[1], Order: [2]int{0, 0}, Value: reproQuad(p[0], p[1])})
	}
	coeffs, err := PlateSolve(cs)
	if err != nil {
		t.Fatalf("PlateSolve returned error: %v", err)
	}
	for _, uv := range [][2]float64{{0.3, 0.7}, {0.9, 0.1}, {0.5, 0.5}} {
		fu, fv := coeffs.EvalGrad(uv[0], uv[1])
		wantU, wantV := reproQuadU(uv[0], uv[1]), reproQuadV(uv[0], uv[1])
		if stdmath.Abs(fu-wantU) > 1e-8 || stdmath.Abs(fv-wantV) > 1e-8 {
			t.Errorf("EvalGrad(%v) = (%.9f,%.9f), want (%.9f,%.9f)", uv, fu, fv, wantU, wantV)
		}
	}
}

// TestPlateSolveMultiSharesMatrix confirms the multi-RHS entry point (P4a's X/Y/Z path) yields
// exactly the same fields as independent single-field solves — one factorisation, many RHS.
func TestPlateSolveMultiSharesMatrix(t *testing.T) {
	pts := reproPoints()
	fieldA := make([]float64, len(pts))
	fieldB := make([]float64, len(pts))
	var cs []PlateConstraint
	for i, p := range pts {
		cs = append(cs, PlateConstraint{U: p[0], V: p[1], Order: [2]int{0, 0}})
		fieldA[i] = reproQuad(p[0], p[1])
		fieldB[i] = 3*p[0] - 2*p[1] // an affine field, also in the span
	}
	multi, err := PlateSolveMulti(cs, [][]float64{fieldA, fieldB})
	if err != nil {
		t.Fatalf("PlateSolveMulti error: %v", err)
	}
	if len(multi) != 2 {
		t.Fatalf("PlateSolveMulti returned %d fields, want 2", len(multi))
	}
	if got := multi[1].Eval(0.4, 0.6); stdmath.Abs(got-(3*0.4-2*0.6)) > 1e-9 {
		t.Errorf("field B Eval(0.4,0.6) = %.12f, want %.12f", got, 3*0.4-2*0.6)
	}
}

// TestPlateRejectsRankDeficient feeds collinear (u=v) G0 constraints: the quadratic border P
// collapses to rank 3 < 6, so the saddle system is singular and PlateSolve must return an error
// (→ the provider falls through to coons4). The failure is in the P block, unfixable by any
// K-diagonal ridge, so the honest-reject is robust.
func TestPlateRejectsRankDeficient(t *testing.T) {
	var cs []PlateConstraint
	for i := 0; i < 7; i++ {
		s := float64(i)
		cs = append(cs, PlateConstraint{U: s, V: s, Order: [2]int{0, 0}, Value: s})
	}
	if _, err := PlateSolve(cs); err == nil {
		t.Fatal("PlateSolve accepted a rank-deficient (collinear) constraint set; want error")
	}
}

// TestPlateRejectsMismatchedValues guards the input validation: a value field shorter than the
// constraint set is a caller error carrying both counts.
func TestPlateRejectsMismatchedValues(t *testing.T) {
	cs := []PlateConstraint{{U: 0, V: 0, Order: [2]int{0, 0}}, {U: 1, V: 0, Order: [2]int{0, 0}}}
	if _, err := PlateSolveMulti(cs, [][]float64{{1.0}}); err == nil {
		t.Fatal("PlateSolveMulti accepted a value field of the wrong length; want error")
	}
}

// assertLambdaVanishes asserts every RBF weight is within 1e-9 of 0 (kit §5b: q ∈ span ⇒ λ=0).
func assertLambdaVanishes(t *testing.T, c PlateCoeffs) {
	t.Helper()
	maxLambda := 0.0
	for _, l := range c.lambda {
		if a := stdmath.Abs(l); a > maxLambda {
			maxLambda = a
		}
	}
	if maxLambda > 1e-9 {
		t.Errorf("‖λ‖∞ = %.3g, want ≤ 1e-9 (all RBF weights vanish)", maxLambda)
	}
	t.Logf("‖λ‖∞ = %.3g", maxLambda)
}

// assertPoly asserts the recovered quadratic coefficients match want to 1e-9.
func assertPoly(t *testing.T, got, want [6]float64) {
	t.Helper()
	maxErr := 0.0
	for k := range got {
		if e := stdmath.Abs(got[k] - want[k]); e > maxErr {
			maxErr = e
		}
	}
	if maxErr > 1e-9 {
		t.Errorf("poly = %v, want %v (‖·‖∞ err %.3g)", got, want, maxErr)
	}
	t.Logf("poly = %v (‖·‖∞ err %.3g)", got, maxErr)
}
