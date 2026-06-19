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
