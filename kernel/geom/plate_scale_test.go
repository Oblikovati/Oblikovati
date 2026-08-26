// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"
)

// P2.1 regression suite: the domain non-dimensionalization that cures the model-scale G1
// conditioning of PlateSolveMulti (advisory .superpowers/sdd/plate-conditioning-fix.md). Two
// invariants are pinned: (1) the transform is an EXACT identity — the reproduced surface is the
// same at any domain scale/offset (Duchon dilation-invariance); (2) it is load-bearing — the
// un-normalized (world-coordinate) assembly loses the interpolant at model scale where the
// normalized solve stays machine-accurate (the RED→GREEN witness).

// scaleField is a genuinely non-polynomial height field (engages the RBF weights λ, unlike the
// quadratic reproduction oracle whose λ vanish and so never stress the K conditioning) plus its
// exact partials, expressed at a domain length scale L so its gradient stays O(1) at any L.
func scaleField(u, v, u0, v0, l float64) float64 {
	return stdmath.Sin((u-u0)/l*3)*stdmath.Cos((v-v0)/l*3)*l + 0.25*(u-u0)
}
func scaleFieldU(u, v, u0, v0, l float64) float64 {
	return stdmath.Cos((u-u0)/l*3)*stdmath.Cos((v-v0)/l*3)*3 + 0.25
}
func scaleFieldV(u, v, u0, v0, l float64) float64 {
	return -stdmath.Sin((u-u0)/l*3) * stdmath.Sin((v-v0)/l*3) * 3
}

// scaledFieldConstraints samples scaleField on a 5×5 grid of diameter l centred at (u0,v0), with
// G0 + ∂u + ∂v rows at every station (the mixed G0/G1 pattern that diverges at model scale).
func scaledFieldConstraints(u0, v0, l float64) ([]PlateConstraint, [][]float64) {
	var cs []PlateConstraint
	var val []float64
	for i := range 5 {
		for j := range 5 {
			u := u0 + l*(float64(i)/4-0.5)
			v := v0 + l*(float64(j)/4-0.5)
			cs = append(cs, PlateConstraint{U: u, V: v, Order: [2]int{0, 0}})
			val = append(val, scaleField(u, v, u0, v0, l))
			cs = append(cs, PlateConstraint{U: u, V: v, Order: [2]int{1, 0}})
			val = append(val, scaleFieldU(u, v, u0, v0, l))
			cs = append(cs, PlateConstraint{U: u, V: v, Order: [2]int{0, 1}})
			val = append(val, scaleFieldV(u, v, u0, v0, l))
		}
	}
	return cs, [][]float64{val}
}

// TestPlateScaleInvariance is the identity witness: the SAME non-polynomial field, sampled at
// three very different domain scales/offsets — unit, N7 model scale (L≈10 centred), and a large
// off-origin patch — must be reproduced to the same relative accuracy. If normalization were not
// an exact similarity transform, the far/large instances would drift; they do not.
func TestPlateScaleInvariance(t *testing.T) {
	cases := []struct {
		name      string
		u0, v0, l float64
		relTol    float64
	}{
		{"unit", 0, 0, 1, 1e-12},
		{"n7-model-scale", 0, 0, 10, 1e-12},
		{"large-off-origin", 500, -300, 200, 1e-11},
	}
	for _, tc := range cases {
		cs, vals := scaledFieldConstraints(tc.u0, tc.v0, tc.l)
		coeffs, err := PlateSolveMulti(cs, vals)
		if err != nil {
			t.Fatalf("%s: PlateSolveMulti error at model scale: %v", tc.name, err)
		}
		worst := worstFieldReproduction(coeffs[0], tc.u0, tc.v0, tc.l)
		if worst > tc.relTol {
			t.Errorf("%s: worst relative reproduction error %.3e exceeds %.3e", tc.name, worst, tc.relTol)
		}
		t.Logf("%s (L=%.0f off=%.0f): worst rel reproduction = %.3e", tc.name, tc.l, tc.u0, worst)
	}
}

// worstFieldReproduction returns the worst relative |Eval−field| over an interior grid.
func worstFieldReproduction(c PlateCoeffs, u0, v0, l float64) float64 {
	worst := 0.0
	for i := 1; i < 4; i++ {
		for j := 1; j < 4; j++ {
			u := u0 + l*(float64(i)/4-0.5)
			v := v0 + l*(float64(j)/4-0.5)
			want := scaleField(u, v, u0, v0, l)
			rel := stdmath.Abs(c.Eval(u, v)-want) / (1 + stdmath.Abs(want))
			worst = stdmath.Max(worst, rel)
		}
	}
	return worst
}

// TestPlateNormalizationCuresConditioning is the RED→GREEN witness. At an extreme domain scale
// (L=1e5) the un-normalized bordered assembly — the pre-P2.1 code path, reproduced here by
// assembling and solving directly on the world constraints — collapses (the L⁴logL … L²logL block
// spread drives a pivot below the dense-solve floor: gaussSolve reports SINGULAR). The P2.1
// normalized PlateSolveMulti solves the identical data to machine accuracy. This is the exact
// mechanism (matrix scaling, not rank) the advisory §2/§3 identifies and cures.
func TestPlateNormalizationCuresConditioning(t *testing.T) {
	const l = 1e5
	cs, vals := scaledFieldConstraints(0, 0, l)

	// RED: the un-normalized world assembly (assemble on the raw constraints, no frame) is singular.
	res := ResolutionForSize(plateDomainDiameter(cs))
	worldA := assemblePlateMatrix(cs, plateRFloor(res))
	worldB := assemblePlateRHS(cs, vals)
	if _, err := gaussClone(worldA, worldB); err == nil {
		t.Fatalf("expected the un-normalized world assembly to be singular at L=%.0g (P2.1 RED); it solved", l)
	}

	// GREEN: the normalized solve reproduces the field to machine accuracy.
	coeffs, err := PlateSolveMulti(cs, vals)
	if err != nil {
		t.Fatalf("normalized PlateSolveMulti did not converge at L=%.0g: %v", l, err)
	}
	if worst := worstFieldReproduction(coeffs[0], 0, 0, l); worst > 1e-10 {
		t.Errorf("normalized solve worst relative reproduction %.3e exceeds 1e-10 at L=%.0g", worst, l)
	}
}

// TestPlateG1ConvergesAtModelScale is the G1-at-model-scale convergence check (advisory §6 a–c):
// a mixed G0+G1 non-polynomial field at N7 scale (L≈10 centred) — the regime whose G1 rows the
// pre-P2.1 solver could not resolve — is ACCEPTED (residual ≤ weld·‖b‖, else PlateSolveMulti
// errors), reproduces every G0 foot value, and yields a G1-consistent world gradient (EvalGrad,
// after the chain-rule 1/L rescale, matches the prescribed derivative targets).
func TestPlateG1ConvergesAtModelScale(t *testing.T) {
	const l = 10.0
	cs, vals := scaledFieldConstraints(0, 0, l)
	coeffs, err := PlateSolveMulti(cs, vals)
	if err != nil {
		t.Fatalf("G0+G1 solve at model scale (L=%.0f) not accepted: %v", l, err)
	}
	field := coeffs[0]
	tol := ResolutionForSize(plateDomainDiameter(cs)).Weld() * plateDomainDiameter(cs)
	for i, c := range cs {
		switch c.Order {
		case [2]int{0, 0}:
			if got := field.Eval(c.U, c.V); stdmath.Abs(got-vals[0][i]) > tol {
				t.Errorf("G0 foot %d Eval = %.9f, want %.9f (tol %.3e)", i, got, vals[0][i], tol)
			}
		case [2]int{1, 0}:
			fu, _ := field.EvalGrad(c.U, c.V)
			if stdmath.Abs(fu-vals[0][i]) > 1e-6 {
				t.Errorf("G1 ∂u foot %d = %.9f, want %.9f", i, fu, vals[0][i])
			}
		case [2]int{0, 1}:
			_, fv := field.EvalGrad(c.U, c.V)
			if stdmath.Abs(fv-vals[0][i]) > 1e-6 {
				t.Errorf("G1 ∂v foot %d = %.9f, want %.9f", i, fv, vals[0][i])
			}
		}
	}
}
