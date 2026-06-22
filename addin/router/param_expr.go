// SPDX-License-Identifier: GPL-2.0-only

package router

import "oblikovati.org/model/param"

// quantitySource is anything that owns a parameter table and a unit parser — a part
// or a work-feature host (assembly). Both the concrete *compdef.PartComponentDefinition
// and the router's sketchHost/workHost interfaces satisfy it.
type quantitySource interface {
	Parameters() *param.Parameters
	Units() param.UnitsOfMeasure
}

// resolveQuantity resolves a unit-bearing API string field that may be a PARAMETER
// EXPRESSION — a bare parameter name ("stator_outer_r") or a formula over the part's
// parameters ("bore_r + slot_depth") — OR a plain literal ("23.5 mm"), returning the
// database-unit [param.Quantity] in dimension dim.
//
// It evaluates the field against the part's parameter table first (so geometry can be
// driven by named parameters/formulas), and falls back to the literal unit parser
// when the field is not a parameter expression or evaluates to a different dimension.
// Every geometry/feature handler that takes a unit string MUST go through here rather
// than calling part.Units().Parse directly, so parameter expressions work everywhere —
// the gap was solution-wide, not specific to one method (Oblikovati.API#187).
//
//	resolveQuantity(part, "bore_r + 2 mm", param.Length) // with bore_r = 10 mm → {12 cm? no: 1.2 cm}
func resolveQuantity(part quantitySource, src string, dim param.Unit) (param.Quantity, error) {
	if q, err := part.Parameters().EvaluateExpression(src); err == nil && q.Unit == dim {
		return q, nil
	}
	return part.Units().Parse(src, dim)
}

// literalLength parses src as a pure literal quantity in dim WITHOUT consulting the parameter
// table — the discriminator a LIVE closure needs to tell a constant ("5 mm") from a parameter
// expression ("h", "L - 2*m") that must stay live (a parameter name has no unit, so it fails to
// parse here). ok is false for anything that is not a bare literal. This is a sanctioned
// Units().Parse site (it lives in param_expr.go alongside resolveQuantity's literal fallback — see
// TestRouterUnitStringsGoThroughResolver); callers that need parameter-expression support compose
// it with the auto-parameter path (see modelLengthClosure), they do not bypass the resolver.
func literalLength(host quantitySource, src string, dim param.Unit) (param.Quantity, bool) {
	q, err := host.Units().Parse(src, dim)
	return q, err == nil
}
