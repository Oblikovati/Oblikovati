// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/param"
)

// Edit verbs for the Parameters dialog. Each resolves the active part, mutates its
// parameter collection, then recomputes the part so a model parameter's edit flows into
// the features that consume it; a failure surfaces through the session notice (the status
// bar) rather than a panic, matching the rest of the app.

// AddNumericUserParameter adds a numeric user parameter from an expression.
func (s *Session) AddNumericUserParameter(name, expression string) error {
	return s.editParameters(func(ps *param.Parameters) error {
		_, err := ps.AddUserParameter(name, expression)
		return err
	})
}

// AddTextUserParameter adds a text user parameter.
func (s *Session) AddTextUserParameter(name, value string) error {
	return s.editParameters(func(ps *param.Parameters) error {
		_, err := ps.AddTextUserParameter(name, value)
		return err
	})
}

// AddBooleanUserParameter adds a true/false user parameter.
func (s *Session) AddBooleanUserParameter(name string, value bool) error {
	return s.editParameters(func(ps *param.Parameters) error {
		_, err := ps.AddBooleanUserParameter(name, value)
		return err
	})
}

// SetParameterName renames a parameter (dependent expressions re-track by id).
func (s *Session) SetParameterName(id param.ID, name string) error {
	return s.editParameters(func(ps *param.Parameters) error { return ps.Rename(id, name) })
}

// SetParameterEquation sets a parameter's equation: an expression for numeric parameters
// (a list choice when multi-value), or the literal for text parameters.
func (s *Session) SetParameterEquation(id param.ID, equation string) error {
	return s.editParam(id, func(p *param.Parameter) error {
		if p.IsMultiValue() {
			return p.SelectValue(equation)
		}
		if p.IsText() {
			return p.SetText(equation)
		}
		return p.SetExpression(equation)
	})
}

// SetParameterBool sets a true/false parameter's value.
func (s *Session) SetParameterBool(id param.ID, value bool) error {
	return s.editParam(id, func(p *param.Parameter) error { return p.SetBool(value) })
}

// SetParameterComment / SetParameterKey / SetParameterExport set the presentation flags.
func (s *Session) SetParameterComment(id param.ID, comment string) error {
	return s.editParam(id, func(p *param.Parameter) error { p.Comment = comment; return nil })
}

func (s *Session) SetParameterKey(id param.ID, key bool) error {
	return s.editParam(id, func(p *param.Parameter) error { p.IsKey = key; return nil })
}

func (s *Session) SetParameterExport(id param.ID, export bool) error {
	return s.editParam(id, func(p *param.Parameter) error { p.ExposedAsProperty = export; return nil })
}

// SetParameterTolerance sets a numeric parameter's engineering tolerance band
// (stored as a deviation tolerance) and which value within it the model uses.
func (s *Session) SetParameterTolerance(id param.ID, upper, lower float64, kind param.ModelValueType) error {
	return s.editParam(id, func(p *param.Parameter) error {
		if err := p.SetToleranceDeviation(upper, lower); err != nil {
			return err
		}
		return p.SetModelValueType(kind)
	})
}

// SetParameterValueList makes a parameter multi-value; ClearParameterValueList reverts it.
func (s *Session) SetParameterValueList(id param.ID, list []string, allowCustom bool) error {
	return s.editParam(id, func(p *param.Parameter) error { return p.SetExpressionList(list, allowCustom) })
}

func (s *Session) ClearParameterValueList(id param.ID) error {
	return s.editParam(id, func(p *param.Parameter) error { p.ClearExpressionList(); return nil })
}

// CopyParameterToUser duplicates a parameter as a new user parameter.
func (s *Session) CopyParameterToUser(id param.ID) error {
	return s.editParameters(func(ps *param.Parameters) error {
		_, err := ps.CopyToUser(id)
		return err
	})
}

// DeleteParameter removes a parameter (its dependents go sick on the lost reference).
func (s *Session) DeleteParameter(id param.ID) error {
	return s.editParameters(func(ps *param.Parameters) error { return ps.Delete(id) })
}

// AddParameterToGroup / RemoveParameterFromGroup manage a parameter's group
// membership for the head UI's single-name flow: adding creates the group on
// first use (internal name = display name), removing detaches from every group.
func (s *Session) AddParameterToGroup(id param.ID, group string) error {
	return s.editParameters(func(ps *param.Parameters) error {
		if _, ok := ps.GroupByKey(group); !ok {
			if _, err := ps.AddGroup(group, group, ""); err != nil {
				return err
			}
		}
		return ps.AddToGroup(id, group)
	})
}

func (s *Session) RemoveParameterFromGroup(id param.ID) error {
	return s.editParameters(func(ps *param.Parameters) error { return ps.RemoveFromAllGroups(id) })
}

// RenameParameterGroup edits a group's display name; the internal name a group
// is addressed by never changes (M02-F05).
func (s *Session) RenameParameterGroup(key, displayName string) error {
	return s.editParameters(func(ps *param.Parameters) error {
		g, ok := ps.GroupByKey(key)
		if !ok {
			return fmt.Errorf("app: no parameter group named %q", key)
		}
		g.DisplayName = displayName
		return nil
	})
}

// DeleteParameterGroup removes a group and its member parameters (the head
// UI's journey semantics; the wire surface exposes the opt-in cascade).
func (s *Session) DeleteParameterGroup(name string) error {
	return s.editParameters(func(ps *param.Parameters) error { return ps.DeleteGroup(name, true) })
}

// editParam resolves the parameter by id and applies edit, then recomputes.
func (s *Session) editParam(id param.ID, edit func(*param.Parameter) error) error {
	return s.editParameters(func(ps *param.Parameters) error {
		p, ok := ps.ByID(id)
		if !ok {
			return fmt.Errorf("app: no parameter with id %d", id)
		}
		return edit(p)
	})
}

// editParameters runs edit against the active part's parameter collection, records any
// error as the session notice, recomputes the part on success, and returns the error.
func (s *Session) editParameters(edit func(*param.Parameters) error) error {
	part, err := activePart(s)
	if err != nil {
		s.notice = err.Error()
		return err
	}
	if err := edit(part.Parameters()); err != nil {
		s.notice = err.Error()
		return err
	}
	s.notice = ""
	part.Recompute()
	s.recordEdit(part, "Edit Parameters")
	return nil
}
