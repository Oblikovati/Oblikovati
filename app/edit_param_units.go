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
