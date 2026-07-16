// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"
)

// The Duchon order-3 polyharmonic (thin-plate) surface-spline solver — the variational core of
// M6's degenerate-corner plate fill (OCCT GeomPlate port). Over the flat average-plane domain Ω
// (built by AveragePlane, P1), the plate is the minimiser of the order-m=3 bending energy that
// interpolates a mix of G0 (point-value) and G1 (first-derivative) constraints. The interpolant
// is the RBF-plus-polynomial representer
//
//	f(u,v) = Σ_j λ_j·ψ_j(u,v)  +  a·[1,u,v,u²,uv,v²]
//
// where ψ_j is constraint j's representer (E or a signed partial of E; kit §1) and the quadratic
// polynomial is the reproduction (null-energy) space. The coefficients solve the bordered saddle
// system [K Pᵀ; P 0][λ;a] = [v;0] (Meinguet 1979; kit §2). See .superpowers/sdd/plate-math-kit.md
// for the transcription-grade math and .superpowers/sdd/n7-plate-solve-derivation.md for method.
//
// This package works purely in Ω's (u,v) floats; P3 discretises rail sides into PlateConstraints
// and P4a fits the evaluated grid to a BSpline. Both consume the interface below.

// PlateConstraint is one interpolation condition for the plate solver at domain point (U,V).
// Order selects the functional: {0,0} is a G0 point value, {1,0} is a ∂/∂u value, {0,1} is a
// ∂/∂v value. Value is the scalar target for the single-field PlateSolve; PlateSolveMulti reads
// targets from its own values argument and uses only U, V, Order here.
type PlateConstraint struct {
	U, V  float64
	Order [2]int
	Value float64
}

// PlateCoeffs is a solved scalar plate field: the RBF weights lambda (one per constraint, in
// input order), the quadratic polynomial coefficients poly (basis 1,u,v,u²,uv,v²), and the
// constraint centres plus the r→0 floor rFloor needed to re-evaluate the representers. Evaluate
// it anywhere in Ω with Eval / EvalGrad.
type PlateCoeffs struct {
	centers []PlateConstraint
	lambda  []float64
	poly    [6]float64
	rFloor  float64
}

// PlateSolve solves the Duchon order-3 plate for ONE scalar field: it packs each constraint's
// Value into the RHS, assembles and solves the bordered saddle system with one iterative-
// refinement step, and returns an error (carrying the residual norm and tolerance) when the
// system is singular or the refined residual exceeds the model-relative weld tolerance.
//
// Example:
//
//	cs := []PlateConstraint{{U: 0, V: 0, Order: [2]int{0, 0}, Value: 1}, ...}
//	f, err := PlateSolve(cs)
//	z := f.Eval(0.4, 0.6)
func PlateSolve(cs []PlateConstraint) (PlateCoeffs, error) {
	vals := make([]float64, len(cs))
	for i, c := range cs {
		vals[i] = c.Value
	}
	out, err := PlateSolveMulti(cs, [][]float64{vals})
	if err != nil {
		return PlateCoeffs{}, err
	}
	return out[0], nil
}

// PlateSolveMulti solves several scalar fields that share the SAME constraint geometry (U,V,
// Order) and therefore the SAME system matrix A — only the RHS differs. It assembles A once and
// solves all fields together through gaussSolve's built-in n×rhs support, so P4a can spline the
// X/Y/Z coordinate fields of a corner in a single factorisation (kit §2). values[f][i] is field
// f's target for constraint i; the returned slice is parallel to values.
func PlateSolveMulti(cs []PlateConstraint, values [][]float64) ([]PlateCoeffs, error) {
	if err := validatePlateInput(cs, values); err != nil {
		return nil, err
	}
	res := ResolutionForSize(plateDomainDiameter(cs))
	rFloor := plateRFloor(res)
	a := assemblePlateMatrix(cs, rFloor)
	b := assemblePlateRHS(cs, values)
	x, err := solvePlateSystem(a, b, res.Weld())
	if err != nil {
		return nil, err
	}
	return buildPlateCoeffs(cs, x, rFloor), nil
}

// validatePlateInput rejects an empty constraint set, an empty value list, or any value field
// whose length does not match the constraint count (one target per constraint).
func validatePlateInput(cs []PlateConstraint, values [][]float64) error {
	if len(cs) == 0 {
		return fmt.Errorf("geom: PlateSolve needs >=1 constraint, got 0")
	}
	if len(values) == 0 {
		return fmt.Errorf("geom: PlateSolveMulti needs >=1 value field, got 0")
	}
	for f, vf := range values {
		if len(vf) != len(cs) {
			return fmt.Errorf(
				"geom: plate value field %d has %d values, expected one per constraint (%d)", f, len(vf), len(cs))
		}
	}
	return nil
}

// plateDomainDiameter returns L, the max pairwise (u,v) distance of the constraints — the
// length scale every model-relative plate tolerance (weld, rFloor) is measured against (kit §0).
func plateDomainDiameter(cs []PlateConstraint) float64 {
	longest := 0.0
	for i := range cs {
		for j := i + 1; j < len(cs); j++ {
			d := stdmath.Hypot(cs[i].U-cs[j].U, cs[i].V-cs[j].V)
			if d > longest {
				longest = d
			}
		}
	}
	return longest
}

// plateRFloor is the removable-singularity guard R_floor = (res.Weld·L)² (kit §1). Below it,
// the E kernel and every partial return their exact r→0 limit of 0; above it, only genuinely
// distinct centres remain (P1 pre-conditions samples to be distinct to weld·L), so this fires
// solely on self-terms P_i−P_i = 0.
func plateRFloor(res Resolution) float64 {
	w := res.Weld() * res.Size()
	return w * w
}

// plateColSign is the column-order parity (−1)^{a_j+b_j} of the K sign rule
// K_ij = (−1)^{a_j+b_j}·D^{(a_i+a_j,b_i+b_j)}E(P_i−P_j) (kit §2): +1 for a G0 column, −1 for a
// first-derivative column. It is what makes ψ_j = (−1)^{a_j+b_j}D^{(a_j,b_j)}E the representer
// of a derivative constraint (differentiation w.r.t. the centre flips the sign).
func plateColSign(order [2]int) float64 {
	if (order[0]+order[1])%2 == 1 {
		return -1
	}
	return 1
}

// plateMonoRow is the constraint functional of order applied to the six quadratic monomials
// {1,u,v,u²,uv,v²} — the P-border row (kit §2). G0 is the monomials; {1,0} is ∂/∂u of them;
// {0,1} is ∂/∂v of them.
func plateMonoRow(u, v float64, order [2]int) [6]float64 {
	switch {
	case order[0] == 1 && order[1] == 0:
		return [6]float64{0, 1, 0, 2 * u, v, 0}
	case order[0] == 0 && order[1] == 1:
		return [6]float64{0, 0, 1, 0, u, 2 * v}
	default:
		return [6]float64{1, u, v, u * u, u * v, v * v}
	}
}

// assemblePlateMatrix builds the (N+6)×(N+6) bordered saddle matrix A = [K Pᵀ; P 0] (kit §2).
func assemblePlateMatrix(cs []PlateConstraint, rFloor float64) [][]float64 {
	m := len(cs) + 6
	a := make([][]float64, m)
	for i := range a {
		a[i] = make([]float64, m)
	}
	fillKBlock(a, cs, rFloor)
	fillPBorder(a, cs)
	return a
}

// fillKBlock writes the N×N RBF block K_ij = colSign(j)·D^{(o_i+o_j)}E(P_i−P_j) (kit §2 table).
// The result is symmetric because E's odd/even partials cancel the column-parity sign.
func fillKBlock(a [][]float64, cs []PlateConstraint, rFloor float64) {
	for i := range cs {
		for j := range cs {
			du := cs[i].U - cs[j].U
			dv := cs[i].V - cs[j].V
			oi, oj := cs[i].Order, cs[j].Order
			a[i][j] = plateColSign(oj) * plateDeriv(oi[0]+oj[0], oi[1]+oj[1], du, dv, rFloor)
		}
	}
}

// fillPBorder writes the polynomial border: P into A's top-right N×6 block and Pᵀ into the
// bottom-left 6×N block (kit §2). Dropping it leaves the quadratic null space unconstrained.
func fillPBorder(a [][]float64, cs []PlateConstraint) {
	n := len(cs)
	for i, c := range cs {
		row := plateMonoRow(c.U, c.V, c.Order)
		for k := 0; k < 6; k++ {
			a[i][n+k] = row[k]
			a[n+k][i] = row[k]
		}
	}
}

// assemblePlateRHS packs the value fields into the top N rows of the (N+6)×rhs RHS; the six
// border rows stay zero (the moment conditions Pᵀλ = 0; kit §2).
func assemblePlateRHS(cs []PlateConstraint, values [][]float64) [][]float64 {
	m := len(cs) + 6
	b := make([][]float64, m)
	for i := range b {
		b[i] = make([]float64, len(values))
	}
	for f, vf := range values {
		for i := range vf {
			b[i][f] = vf[i]
		}
	}
	return b
}

// solvePlateSystem solves A·x = b via gaussSolve, applies one iterative-refinement step, and
// accepts only when the refined residual is within weld·‖b‖ and all coefficients are finite
// (kit §2). A and b are treated as read-only; gaussSolve runs on clones.
func solvePlateSystem(a, b [][]float64, weld float64) ([][]float64, error) {
	xhat, err := gaussClone(a, b)
	if err != nil {
		return nil, err
	}
	x := refinePlateSolution(a, b, xhat)
	if !allFinite(x) {
		return nil, fmt.Errorf("geom: plate solve produced non-finite coefficients")
	}
	if err := acceptPlateResidual(a, b, x, weld); err != nil {
		return nil, err
	}
	return x, nil
}

// gaussClone solves A·x = b on deep copies (gaussSolve is in-place), returning x and leaving
// the caller's A and b untouched so they can be reused for the refinement residual.
func gaussClone(a, b [][]float64) ([][]float64, error) {
	aw := cloneRows(a)
	bw := cloneRows(b)
	if err := gaussSolve(aw, bw); err != nil {
		return nil, err
	}
	return bw, nil
}

// refinePlateSolution runs OCCT's one-step iterative refinement (nbiter=1, kit §2): r = b−A·x̂,
// solve A·δ = r on the same matrix, return x̂+δ. If the refinement solve fails it returns x̂
// unchanged — acceptPlateresidual is the sole accept gate either way.
func refinePlateSolution(a, b, xhat [][]float64) [][]float64 {
	resid := residualMatrix(a, b, xhat)
	delta, err := gaussClone(a, resid)
	if err != nil {
		return xhat
	}
	return addMatrix(xhat, delta)
}

// residualMatrix returns r = b − A·x (per RHS column).
func residualMatrix(a, b, x [][]float64) [][]float64 {
	r := make([][]float64, len(a))
	for i := range a {
		r[i] = make([]float64, len(b[i]))
		for f := range b[i] {
			r[i][f] = b[i][f] - dotRowColumn(a[i], x, f)
		}
	}
	return r
}

// dotRowColumn returns the dot product of matrix row with column f of x.
func dotRowColumn(row []float64, x [][]float64, f int) float64 {
	sum := 0.0
	for k, ak := range row {
		sum += ak * x[k][f]
	}
	return sum
}

// addMatrix returns the element-wise sum x+d.
func addMatrix(x, d [][]float64) [][]float64 {
	out := make([][]float64, len(x))
	for i := range x {
		out[i] = make([]float64, len(x[i]))
		for f := range x[i] {
			out[i][f] = x[i][f] + d[i][f]
		}
	}
	return out
}

// acceptPlateResidual rejects the solve (returning an error carrying the residual and tolerance)
// if any field's refined residual ‖b−A·x‖₂ exceeds the model-relative bound weld·‖b‖₂ (kit §2).
func acceptPlateResidual(a, b, x [][]float64, weld float64) error {
	resid := residualMatrix(a, b, x)
	for f := range b[0] {
		rn := columnNorm(resid, f)
		bn := columnNorm(b, f)
		if rn > weld*bn {
			return fmt.Errorf(
				"geom: plate solve did not converge: field %d residual %.6g exceeds weld·‖b‖ = %.6g "+
					"(weld %.6g, ‖b‖ %.6g)", f, rn, weld*bn, weld, bn)
		}
	}
	return nil
}

// columnNorm returns the Euclidean norm of column f of matrix m.
func columnNorm(m [][]float64, f int) float64 {
	sum := 0.0
	for i := range m {
		sum += m[i][f] * m[i][f]
	}
	return stdmath.Sqrt(sum)
}

// allFinite reports whether every entry of x is finite (no NaN/±Inf from a collapsed pivot).
func allFinite(x [][]float64) bool {
	for i := range x {
		for _, v := range x[i] {
			if stdmath.IsNaN(v) || stdmath.IsInf(v, 0) {
				return false
			}
		}
	}
	return true
}

// buildPlateCoeffs slices the stacked solution x = [λ; a] (per field f) into PlateCoeffs; all
// fields share one copy of the constraint centres and the rFloor.
func buildPlateCoeffs(cs []PlateConstraint, x [][]float64, rFloor float64) []PlateCoeffs {
	centers := append([]PlateConstraint(nil), cs...)
	out := make([]PlateCoeffs, len(x[0]))
	for f := range out {
		out[f] = extractFieldCoeffs(centers, x, f, rFloor)
	}
	return out
}

// extractFieldCoeffs reads field f's λ (top N rows) and quadratic coefficients (last 6 rows).
func extractFieldCoeffs(centers []PlateConstraint, x [][]float64, f int, rFloor float64) PlateCoeffs {
	n := len(centers)
	lambda := make([]float64, n)
	for i := 0; i < n; i++ {
		lambda[i] = x[i][f]
	}
	var poly [6]float64
	for k := 0; k < 6; k++ {
		poly[k] = x[n+k][f]
	}
	return PlateCoeffs{centers: centers, lambda: lambda, poly: poly, rFloor: rFloor}
}

// Eval returns the plate value f(u,v) = Σ λ_j·ψ_j(u,v) + a·[1,u,v,u²,uv,v²], where the
// representer ψ_j = (−1)^{a_j+b_j}·D^{(a_j,b_j)}E((u,v)−P_j) (kit §1/§2).
func (c PlateCoeffs) Eval(u, v float64) float64 {
	sum := polyValue(c.poly, u, v)
	for j, center := range c.centers {
		du, dv := u-center.U, v-center.V
		psi := plateColSign(center.Order) * plateDeriv(center.Order[0], center.Order[1], du, dv, c.rFloor)
		sum += c.lambda[j] * psi
	}
	return sum
}

// EvalGrad returns the plate gradient (∂f/∂u, ∂f/∂v). Each representer term differentiates once
// more in u or v — raising the kernel order by one (still ≤ 2, so plateDeriv covers it).
func (c PlateCoeffs) EvalGrad(u, v float64) (fu, fv float64) {
	fu, fv = polyGrad(c.poly, u, v)
	for j, center := range c.centers {
		du, dv := u-center.U, v-center.V
		sign := c.lambda[j] * plateColSign(center.Order)
		au, bv := center.Order[0], center.Order[1]
		fu += sign * plateDeriv(au+1, bv, du, dv, c.rFloor)
		fv += sign * plateDeriv(au, bv+1, du, dv, c.rFloor)
	}
	return fu, fv
}

// polyValue evaluates the quadratic a·[1,u,v,u²,uv,v²].
func polyValue(p [6]float64, u, v float64) float64 {
	return p[0] + p[1]*u + p[2]*v + p[3]*u*u + p[4]*u*v + p[5]*v*v
}

// polyGrad returns the gradient of the quadratic a·[1,u,v,u²,uv,v²].
func polyGrad(p [6]float64, u, v float64) (pu, pv float64) {
	pu = p[1] + 2*p[3]*u + p[4]*v
	pv = p[2] + p[4]*u + 2*p[5]*v
	return pu, pv
}
