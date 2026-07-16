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
// input order), the quadratic polynomial coefficients poly (basis 1,û,v̂,û²,ûv̂,v̂²), and the
// constraint centres plus the r→0 floor rFloor needed to re-evaluate the representers. The
// centres, poly and lambda live in the NORMALIZED unit-diameter frame û=(u−u0)/scale,
// v̂=(v−v0)/scale (see [plateDomainFrame]); Eval / EvalGrad normalize their world (u,v) input
// before evaluating. Evaluate the field anywhere in Ω with Eval / EvalGrad.
type PlateCoeffs struct {
	centers       []PlateConstraint
	lambda        []float64
	poly          [6]float64
	rFloor        float64
	u0, v0, scale float64 // the non-dimensionalizing frame; see plateDomainFrame
}

// normalize maps a world Ω coordinate into the unit-diameter frame the coefficients were solved
// in (û=(u−u0)/scale, v̂=(v−v0)/scale). The inverse of the assembly-time similarity transform.
func (c PlateCoeffs) normalize(u, v float64) (uh, vh float64) {
	return (u - c.u0) / c.scale, (v - c.v0) / c.scale
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
	frame, err := newPlateDomainFrame(cs)
	if err != nil {
		return nil, err
	}
	// The Resolution — hence the acceptance threshold weld·‖b‖ and the r→0 floor — is measured
	// against the WORLD domain diameter (frame.scale), a relative tolerance unchanged by the
	// non-dimensionalization; the normalized floor is res.Weld()² = rFloor_world/scale² because
	// the normalized squared radius R̂ = R_world/scale² (advisory §4 step 4).
	res := ResolutionForSize(frame.scale)
	rFloor := res.Weld() * res.Weld()
	ncs := frame.normalizeConstraints(cs)
	nvals := frame.normalizeValues(cs, values)
	a := assemblePlateMatrix(ncs, rFloor)
	b := assemblePlateRHS(ncs, nvals)
	x, err := solvePlateSystem(a, b, res.Weld())
	if err != nil {
		return nil, err
	}
	return buildPlateCoeffs(ncs, x, rFloor, frame), nil
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

// plateRFloor is the removable-singularity guard R_floor = (res.Weld·L)² (kit §1) in WORLD
// coordinates. The solver runs on normalized coordinates and uses res.Weld()² (= this / L²)
// instead; this world form is kept for the assembly-symmetry self-check, which assembles the
// matrix directly on world constraints without the normalizing frame.
func plateRFloor(res Resolution) float64 {
	w := res.Weld() * res.Size()
	return w * w
}

// plateDomainFrame is the similarity transform that non-dimensionalizes the bordered TPS system
// to unit-diameter coordinates û=(u−u0)/scale, v̂=(v−v0)/scale, where (u0,v0) is the constraint
// (u,v) bounding-box centre and scale is the domain diameter. In these O(1) coordinates the RBF
// K-block (whose entries otherwise span L⁴logL … L²logL across G0/G1 rows) and the polynomial
// border (otherwise 1:L:L²) are all O(1)-conditioned, so the dense Gauss-Jordan solve no longer
// loses the 10–100×-smaller G1-derivative rows at model scale — the P4a G1 divergence
// (residual ~1e11 at the r=5 corner). Under the moment conditions Pᵀλ=0 the transform preserves
// the interpolant EXACTLY in exact arithmetic (Duchon dilation-invariance; advisory §3), so it
// re-conditions the matrix without smoothing the rails (watertight-preserving).
type plateDomainFrame struct {
	u0, v0, scale float64
}

// newPlateDomainFrame derives the non-dimensionalizing frame from the constraints' (u,v) extent:
// (u0,v0) is the bounding-box centre (centering keeps û∈[−½,½], minimizing the monomial
// magnitudes) and scale is the domain diameter (max pairwise distance). Rejects an all-coincident
// set, whose zero diameter would divide by zero (carrying the measured diameter and the required
// minimum).
func newPlateDomainFrame(cs []PlateConstraint) (plateDomainFrame, error) {
	minU, maxU, minV, maxV := cs[0].U, cs[0].U, cs[0].V, cs[0].V
	for _, c := range cs {
		minU, maxU = stdmath.Min(minU, c.U), stdmath.Max(maxU, c.U)
		minV, maxV = stdmath.Min(minV, c.V), stdmath.Max(maxV, c.V)
	}
	scale := plateDomainDiameter(cs)
	if scale <= 0 {
		return plateDomainFrame{}, fmt.Errorf(
			"geom: plate domain degenerate: all %d constraints coincide in (u,v) (diameter %.6g, need > 0)",
			len(cs), scale)
	}
	return plateDomainFrame{u0: (minU + maxU) / 2, v0: (minV + maxV) / 2, scale: scale}, nil
}

// normalize maps a world (u,v) into the unit-diameter frame û=(u−u0)/scale, v̂=(v−v0)/scale.
func (f plateDomainFrame) normalize(u, v float64) (uh, vh float64) {
	return (u - f.u0) / f.scale, (v - f.v0) / f.scale
}

// normalizeConstraints rebuilds the constraint list in the unit-diameter frame — same Order and
// Value, only (U,V) transformed. The kernel formulas then receive O(1) Δû,Δv̂ unchanged.
func (f plateDomainFrame) normalizeConstraints(cs []PlateConstraint) []PlateConstraint {
	out := make([]PlateConstraint, len(cs))
	for i, c := range cs {
		u, v := f.normalize(c.U, c.V)
		out[i] = PlateConstraint{U: u, V: v, Order: c.Order, Value: c.Value}
	}
	return out
}

// normalizeValues scales each RHS target by scale^{order}: a G0 value is scale-free (order 0), a
// first-derivative target carries one inverse length so its normalized value is ×scale
// (∂f/∂û = scale·∂f/∂u; advisory §3.2). This is the ONLY change to the value packing and the
// crux that balances the G0 and G1 rows.
func (f plateDomainFrame) normalizeValues(cs []PlateConstraint, values [][]float64) [][]float64 {
	out := make([][]float64, len(values))
	for fld, vf := range values {
		out[fld] = make([]float64, len(vf))
		for i, v := range vf {
			out[fld][i] = v * f.orderScale(cs[i].Order)
		}
	}
	return out
}

// orderScale is scale^{a+b} for a constraint of derivative order (a,b) — 1 for a G0 value,
// scale for a first-derivative target.
func (f plateDomainFrame) orderScale(order [2]int) float64 {
	return stdmath.Pow(f.scale, float64(order[0]+order[1]))
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
func buildPlateCoeffs(ncs []PlateConstraint, x [][]float64, rFloor float64, frame plateDomainFrame) []PlateCoeffs {
	centers := append([]PlateConstraint(nil), ncs...)
	out := make([]PlateCoeffs, len(x[0]))
	for f := range out {
		out[f] = extractFieldCoeffs(centers, x, f, rFloor, frame)
	}
	return out
}

// extractFieldCoeffs reads field f's λ (top N rows) and quadratic coefficients (last 6 rows),
// stamping the normalizing frame so Eval/EvalGrad can map world (u,v) into the solved frame.
func extractFieldCoeffs(centers []PlateConstraint, x [][]float64, f int, rFloor float64, frame plateDomainFrame) PlateCoeffs {
	n := len(centers)
	lambda := make([]float64, n)
	for i := 0; i < n; i++ {
		lambda[i] = x[i][f]
	}
	var poly [6]float64
	for k := 0; k < 6; k++ {
		poly[k] = x[n+k][f]
	}
	return PlateCoeffs{centers: centers, lambda: lambda, poly: poly, rFloor: rFloor, u0: frame.u0, v0: frame.v0, scale: frame.scale}
}

// Eval returns the plate value f(u,v) = Σ λ_j·ψ_j(û,v̂) + a·[1,û,v̂,û²,ûv̂,v̂²], where (û,v̂) is
// the world input normalized into the solve frame and ψ_j = (−1)^{a_j+b_j}·D^{(a_j,b_j)}E((û,v̂)
// −P̂_j) (kit §1/§2). The field VALUE is scale-invariant (f=f̂), so the result is returned
// unchanged — the normalization touches only the coordinates fed to the representers/poly
// (advisory §3.2).
func (c PlateCoeffs) Eval(u, v float64) float64 {
	uh, vh := c.normalize(u, v)
	sum := polyValue(c.poly, uh, vh)
	for j, center := range c.centers {
		du, dv := uh-center.U, vh-center.V
		psi := plateColSign(center.Order) * plateDeriv(center.Order[0], center.Order[1], du, dv, c.rFloor)
		sum += c.lambda[j] * psi
	}
	return sum
}

// EvalGrad returns the world plate gradient (∂f/∂u, ∂f/∂v). It evaluates the NORMALIZED gradient
// (∂f/∂û, ∂f/∂v̂) at the normalized input, then applies the chain rule ∂f/∂u = (1/scale)·∂f/∂û
// (advisory §3.2, the one place output is rescaled). Each representer term differentiates once
// more in û or v̂ — raising the kernel order by one (still ≤ 2, so plateDeriv covers it).
func (c PlateCoeffs) EvalGrad(u, v float64) (fu, fv float64) {
	uh, vh := c.normalize(u, v)
	fu, fv = polyGrad(c.poly, uh, vh)
	for j, center := range c.centers {
		du, dv := uh-center.U, vh-center.V
		sign := c.lambda[j] * plateColSign(center.Order)
		au, bv := center.Order[0], center.Order[1]
		fu += sign * plateDeriv(au+1, bv, du, dv, c.rFloor)
		fv += sign * plateDeriv(au, bv+1, du, dv, c.rFloor)
	}
	return fu / c.scale, fv / c.scale
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
