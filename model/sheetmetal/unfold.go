// SPDX-License-Identifier: GPL-2.0-only

// Package sheetmetal holds the sheet-metal environment's rule model and the unfold
// math that turns a folded bend into its flat-pattern development (M13-F01). It is
// pure model/geometry support: no document, no feature engine — the compdef layer
// attaches a [Rule] to a part and the bend/flat-pattern features (M13-F02/F04)
// consult the rule's [UnfoldMethod] for each bend's developed length.
package sheetmetal

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/model/param"
)

// UnfoldMethodType and ReliefShape are the canonical Apache-2.0 enums (ADR-0018); the
// sheet-metal model aliases them so call sites read `sheetmetal.ReliefRound`.
type (
	UnfoldMethodType = types.UnfoldMethodType
	ReliefShape      = types.ReliefShape
)

// defaultKFactor is the neutral-axis ratio used when a rule does not specify one — 0.44
// is the standard mild-steel default (the neutral axis sits a little inside the mid-plane).
const defaultKFactor = 0.44

// UnfoldMethod governs how a single bend develops into the flat pattern: its bend
// allowance (the developed neutral-axis arc length the flat must include) and its bend
// deduction (the setback subtracted from the two outside flange lengths). All three
// industry methods are supported — a single K-factor, a per-angle bend table, or a custom
// equation in {thickness, radius, angle}.
type UnfoldMethod struct {
	Type     types.UnfoldMethodType
	KFactor  float64
	Table    *BendTable        // consulted when Type == BendTableUnfold
	equation *compiledEquation // consulted when Type == EquationUnfold
}

// KFactorMethod returns a K-factor unfold method. A non-positive k falls back to the
// standard default so a half-built rule still develops sensibly.
func KFactorMethod(k float64) UnfoldMethod {
	if k <= 0 {
		k = defaultKFactor
	}
	return UnfoldMethod{Type: types.KFactorUnfold, KFactor: k}
}

// BendTableMethod returns a bend-table unfold method over t.
func BendTableMethod(t *BendTable) UnfoldMethod {
	return UnfoldMethod{Type: types.BendTableUnfold, KFactor: defaultKFactor, Table: t}
}

// EquationMethod compiles a custom bend-allowance equation in the variables `t`
// (thickness), `r` (radius) and `a` (bend angle, radians). It returns an error naming
// the offending source if the expression does not parse or references an unknown name.
func EquationMethod(src string) (UnfoldMethod, error) {
	eq, err := compileEquation(src)
	if err != nil {
		return UnfoldMethod{}, err
	}
	return UnfoldMethod{Type: types.EquationUnfold, KFactor: defaultKFactor, equation: eq}, nil
}

// EquationSource returns the authored bend-allowance equation when the method is an
// equation method, else "" — used by persistence to round-trip the formula.
func (m UnfoldMethod) EquationSource() string {
	if m.equation == nil {
		return ""
	}
	return m.equation.src
}

// kFactor returns the method's K-factor, defaulting when unset.
func (m UnfoldMethod) kFactor() float64 {
	if m.KFactor <= 0 {
		return defaultKFactor
	}
	return m.KFactor
}

// BendAllowance returns the developed flat length contributed by one bend. angle is the
// swept bend angle in radians (90° flange ⇒ π/2), radius the inside bend radius, thickness
// the material thickness; all lengths in database units (cm). The K-factor relation
// BA = angle·(radius + K·thickness) is the fallback for every method, so an empty table or
// equation still yields a physically sensible length rather than zero.
func (m UnfoldMethod) BendAllowance(angle, radius, thickness float64) float64 {
	switch m.Type {
	case types.BendTableUnfold:
		if m.Table != nil {
			if ba, ok := m.Table.BendAllowance(angle, radius, thickness); ok {
				return ba
			}
		}
	case types.EquationUnfold:
		if m.equation != nil {
			if ba, ok := m.equation.eval(thickness, radius, angle); ok {
				return ba
			}
		}
	}
	return angle * (radius + m.kFactor()*thickness)
}

// BendDeduction returns the bend deduction (setback) for one bend: the amount by which the
// sum of the two outside flange lengths overshoots the developed length, so a flat built
// from outside dimensions subtracts one deduction per bend. BD = 2·OSSB − BA, where the
// outside setback OSSB = (radius + thickness)·tan(angle/2).
func (m UnfoldMethod) BendDeduction(angle, radius, thickness float64) float64 {
	ossb := (radius + thickness) * math.Tan(angle/2)
	return 2*ossb - m.BendAllowance(angle, radius, thickness)
}

// compiledEquation is a parsed bend-allowance equation bound to the three synthetic
// variables t/r/a, ready to evaluate per bend.
type compiledEquation struct {
	src  string
	expr param.Expr
}

// equation variable ids — synthetic, local to the equation scope (not real parameters).
const (
	eqThicknessID param.ID = 1
	eqRadiusID    param.ID = 2
	eqAngleID     param.ID = 3
)

// eqScope binds {t, r, a} for one bend's evaluation. The variables are exposed as unitless
// scalars (their numeric value in database units — cm for lengths, radians for the angle):
// a bend-allowance formula is raw arithmetic, so passing them dimensionless sidesteps the
// param engine's dimensional checks (which would reject e.g. angle·length) while keeping the
// numeric result correct.
type eqScope struct{ thickness, radius, angle float64 }

func (s eqScope) ValueOf(id param.ID) (param.Quantity, bool) {
	switch id {
	case eqThicknessID:
		return param.Scalar(s.thickness), true
	case eqRadiusID:
		return param.Scalar(s.radius), true
	case eqAngleID:
		return param.Scalar(s.angle), true
	}
	return param.Quantity{}, false
}

// compileEquation parses src and binds its t/r/a references; any other name is rejected so
// a typo fails loudly at rule-set time, not silently at unfold time.
func compileEquation(src string) (*compiledEquation, error) {
	expr, err := param.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal unfold equation %q: %w", src, err)
	}
	unresolved := expr.Bind(func(name string) (param.ID, bool) {
		switch name {
		case "t", "thickness":
			return eqThicknessID, true
		case "r", "radius":
			return eqRadiusID, true
		case "a", "angle":
			return eqAngleID, true
		}
		return 0, false
	})
	if len(unresolved) != 0 {
		return nil, fmt.Errorf("sheet-metal unfold equation %q: unknown variable(s) %v (use t, r, a)", src, unresolved)
	}
	return &compiledEquation{src: src, expr: expr}, nil
}

// eval returns the equation's value for one bend, or ok=false if evaluation fails (a
// dimensional mismatch in the authored formula), letting the caller fall back to K-factor.
func (e *compiledEquation) eval(thickness, radius, angle float64) (float64, bool) {
	q, err := e.expr.Eval(eqScope{thickness, radius, angle})
	if err != nil {
		return 0, false
	}
	return q.Value, true
}
