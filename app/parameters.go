// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"

	"oblikovati.org/model/param"
)

// The Manage ▸ Parameters dialog is driven through the session: the head asks for
// presentation-ready rows (strings/bools only — the units and formatting stay here, as
// with the sketch-grid preference) and calls the edit verbs in parameters_edit.go, which
// mutate the active part's parameters and recompute. State is the open flag plus the
// part's own [param.Parameters]; there is no separate dialog model to keep in sync.

// OpenParameters / CloseParameters / ParametersOpen drive the dialog's visibility.
func (s *Session) OpenParameters()      { s.paramsDialogOpen = true }
func (s *Session) CloseParameters()     { s.paramsDialogOpen = false }
func (s *Session) ParametersOpen() bool { return s.paramsDialogOpen }

// ParameterRow is one parameter rendered for the dialog: identity plus the already-
// formatted text the table cells show. Value/Equation are in the document's display units.
type ParameterRow struct {
	ID         param.ID
	Name       string
	UnitName   string // "mm", "deg", "Text", "Boolean", …
	ValueType  string // "numeric" | "text" | "boolean"
	Equation   string // expression, quoted text, or true/false
	Value      string // formatted nominal value
	Comment    string
	Group      string
	Options    []string // dropdown choices when multi-value (list + current custom value)
	Tolerance  string   // formatted band, "" when none
	IsKey      bool
	Export     bool
	Editable   bool
	MultiValue bool
	Healthy    bool
	Health     string // reason when not healthy
}

// ParameterRows returns the active part's or assembly's parameters split into the Model and
// User tables (Inventor's two lists), each filtered by the search query (matched across name,
// equation, and comment). With no parameter-holding active document both slices are empty.
func (s *Session) ParameterRows(filter string) (model, user []ParameterRow) {
	holder, err := s.activeParameterHolder()
	if err != nil {
		return nil, nil
	}
	q := strings.ToLower(strings.TrimSpace(filter))
	for _, p := range holder.Parameters().All() {
		row := s.parameterRow(holder.Parameters(), p)
		if !matchParameter(row, q) {
			continue
		}
		if p.Kind() == param.UserParam {
			user = append(user, row)
		} else {
			model = append(model, row)
		}
	}
	return model, user
}

// parameterRow projects one parameter into its presentation row.
func (s *Session) parameterRow(ps *param.Parameters, p *param.Parameter) ParameterRow {
	group := firstGroupDisplayName(ps, p.ID())
	row := ParameterRow{
		ID: p.ID(), Name: p.Name(), ValueType: valueTypeName(p), Equation: p.Expression(),
		Comment: p.Comment, Group: group, IsKey: p.IsKey, Export: p.ExposedAsProperty,
		Editable: p.Kind().Editable(), MultiValue: p.IsMultiValue(),
		Healthy: p.Health().OK(), Health: p.Health().Reason,
	}
	s.fillRowValue(&row, p)
	if p.IsMultiValue() {
		row.Options = multiValueOptions(p)
	}
	return row
}

// firstGroupDisplayName names the row's group column: the display name of the
// parameter's first group (creation order) — the head UI shows one group per
// row even though the model allows several (M02-F05).
func firstGroupDisplayName(ps *param.Parameters, id param.ID) string {
	keys := ps.GroupsOf(id)
	if len(keys) == 0 {
		return ""
	}
	if g, ok := ps.GroupByKey(keys[0]); ok {
		return g.DisplayName
	}
	return keys[0]
}

// fillRowValue formats the unit name, displayed value, and tolerance for a row, branching
// on the parameter's value flavor.
func (s *Session) fillRowValue(row *ParameterRow, p *param.Parameter) {
	switch {
	case p.IsText():
		row.UnitName, row.Value = "Text", p.Text()
	case p.IsBoolean():
		row.UnitName, row.Value = "Boolean", p.Expression()
	default:
		u := s.DocumentUnits()
		row.UnitName, row.Value = u.PreferredName(p.Unit()), u.Format(p.Value())
		if t := p.Tolerance(); t != (param.Tolerance{}) {
			row.Tolerance = u.Format(param.Q(t.Upper, p.Unit())) + " / " + u.Format(param.Q(t.Lower, p.Unit()))
		}
	}
}

// valueTypeName names a parameter's value flavor for the row.
func valueTypeName(p *param.Parameter) string {
	switch {
	case p.IsText():
		return "text"
	case p.IsBoolean():
		return "boolean"
	default:
		return "numeric"
	}
}

// multiValueOptions returns the dropdown choices for a multi-value parameter: its fixed
// list plus the current value when that value is a custom one outside the list.
func multiValueOptions(p *param.Parameter) []string {
	list := p.ExpressionList()
	current := p.Expression()
	for _, v := range list {
		if v == current {
			return list
		}
	}
	if p.AllowsCustomValue() {
		return append(list, current)
	}
	return list
}

// matchParameter reports whether a row matches the lowercased search query (empty matches
// all), searched across the name, equation, and comment — Inventor's parameter search.
func matchParameter(row ParameterRow, query string) bool {
	if query == "" {
		return true
	}
	hay := strings.ToLower(row.Name + "\x00" + row.Equation + "\x00" + row.Comment)
	return strings.Contains(hay, query)
}
