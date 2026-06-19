// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
)

// Shared unit plumbing for the scalar fields of the feature- and work-plane-edit dialogs:
// the model exposes [feature.EditableParam]s in database units, while the dialogs read and
// write them in the document's preferred unit. Both accessor families (EditFeatureParam* in
// feature_edit.go, EditPlaneScalar* in work_plane_edit.go) convert through these helpers.

// paramUnitName returns the display unit name for an editable param ("mm", "deg", …), or ""
// for a unitless field.
func (s *Session) paramUnitName(p feature.EditableParam) string {
	switch p.Unit {
	case param.Length:
		return s.LengthUnitName()
	case param.Angle:
		return s.AngleUnitName()
	default:
		return ""
	}
}

// paramDisplayValue returns p's value converted to the document's preferred unit.
func (s *Session) paramDisplayValue(p feature.EditableParam) float64 {
	if p.Unit == param.Unitless {
		return p.Get()
	}
	return s.DocumentUnits().ToPreferred(param.Q(p.Get(), p.Unit))
}

// setParamDisplayValue sets p from a value given in the document's preferred unit.
func (s *Session) setParamDisplayValue(p feature.EditableParam, value float64) {
	if p.Unit == param.Unitless {
		p.Set(value)
		return
	}
	p.Set(s.DocumentUnits().FromPreferred(value, p.Unit).Value)
}

// setParamText sets p from a unit-bearing STRING that may be a parameter expression
// (a bare parameter name "d0" or a formula "d0 + 5 mm") or a literal ("12 mm"). It
// evaluates the text against the active part's parameter table first — so feature
// dialogs accept parameter expressions just like the sketch-dimension editor — and
// falls back to the document's literal unit parser otherwise. Mirrors the API
// router's resolveQuantity so the UI and the API resolve fields the same way
// (Oblikovati.API#187; the gap was solution-wide on the UI side too).
func (s *Session) setParamText(p feature.EditableParam, text string) error {
	q, err := s.evalParamText(text, p.Unit)
	if err != nil {
		return err
	}
	p.Set(q.Value) // EvaluateExpression and Parse both return database units
	return nil
}

// evalParamText resolves text to a database-unit quantity in dimension dim: the
// active part's parameter evaluator first (names + formulas), then a literal parse.
func (s *Session) evalParamText(text string, dim param.Unit) (param.Quantity, error) {
	if part, err := activePart(s); err == nil {
		if q, e := part.Parameters().EvaluateExpression(text); e == nil && q.Unit == dim {
			return q, nil
		}
	}
	return s.DocumentUnits().Parse(text, dim)
}
