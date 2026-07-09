// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// paramInfo marshals a model parameter into the wire DTO: its authored expression
// and its evaluated value formatted in the document's display units.
func paramInfo(holder compdef.ParameterHolder, p *param.Parameter) wire.ParameterInfo {
	info := wire.ParameterInfo{
		Name: p.Name(), Kind: p.Kind().String(),
		Expression: p.Expression(), Value: formatParamValue(holder, p),
	}
	if h := p.Health(); !h.OK() {
		info.Health = h.Reason
	}
	return info
}

// formatParamValue renders a parameter's evaluated value for the wire: the literal string for a
// text parameter, "true"/"false" for a boolean, and the unit-formatted quantity otherwise (#1845).
func formatParamValue(holder compdef.ParameterHolder, p *param.Parameter) string {
	switch {
	case p.IsText():
		return p.Text()
	case p.IsBoolean():
		return strconv.FormatBool(p.Bool())
	default:
		return holder.Units().Format(p.Value())
	}
}

// listParameters returns the active part's or assembly's parameters.
func listParameters(_ *app.Session, holder compdef.ParameterHolder) (wire.ListParametersResult, error) {
	ps := holder.Parameters().All()
	out := make([]wire.ParameterInfo, len(ps))
	for i, p := range ps {
		out[i] = paramInfo(holder, p)
	}
	return wire.ListParametersResult{Parameters: out}, nil
}

// getParameter returns one parameter by name.
func getParameter(_ *app.Session, holder compdef.ParameterHolder, in wire.ParameterNameArgs) (wire.ParameterInfo, error) {
	p, ok := holder.Parameters().ByName(in.Name)
	if !ok {
		return wire.ParameterInfo{}, errors.New("parameters.get: no parameter named " + in.Name)
	}
	return paramInfo(holder, p), nil
}

// addParameter adds a new parameter. ValueType selects numeric (default) / text / boolean and
// Kind selects the user (default) / model table — Inventor's non-numeric and model-parameter
// creation (#1845). A numeric parameter takes a unit-bearing expression (e.g. "4 cm").
func addParameter(_ *app.Session, holder compdef.ParameterHolder, in wire.ParameterSetArgs) (wire.ParameterInfo, error) {
	p, err := addParameterOfKind(holder, in)
	if err != nil {
		return wire.ParameterInfo{}, err
	}
	return paramInfo(holder, p), nil
}

// addParameterOfKind dispatches to the model's typed creators from the wire ValueType/Kind
// discriminators. Text values may be empty; boolean values parse "true"/"false". #1845.
func addParameterOfKind(holder compdef.ParameterHolder, in wire.ParameterSetArgs) (*param.Parameter, error) {
	if in.Name == "" {
		return nil, errors.New("parameters: name is required")
	}
	switch strings.ToLower(strings.TrimSpace(in.ValueType)) {
	case "", "numeric":
		if in.Expression == "" {
			return nil, errors.New("parameters: expression is required for a numeric parameter")
		}
		if strings.EqualFold(strings.TrimSpace(in.Kind), "model") {
			return holder.Parameters().AddModelParameter(in.Name, in.Expression)
		}
		return holder.Parameters().AddUserParameter(in.Name, in.Expression)
	case "text":
		return holder.Parameters().AddTextUserParameter(in.Name, in.Expression)
	case "boolean", "bool":
		b, err := strconv.ParseBool(strings.TrimSpace(in.Expression))
		if err != nil {
			return nil, fmt.Errorf("parameters: boolean parameter %q needs expression \"true\" or \"false\", got %q", in.Name, in.Expression)
		}
		return holder.Parameters().AddBooleanUserParameter(in.Name, b)
	default:
		return nil, fmt.Errorf("parameters: unknown valueType %q (want numeric|text|boolean)", in.ValueType)
	}
}

// renameParameter renames a parameter, keeping its identity so dependent expressions stay bound
// and are rewritten to the new name (#1847).
func renameParameter(s *app.Session, holder compdef.ParameterHolder, in wire.ParameterRenameArgs) (wire.ParameterInfo, error) {
	if in.Name == "" || in.NewName == "" {
		return wire.ParameterInfo{}, errors.New("parameters.rename: name and newName are required")
	}
	p, ok := holder.Parameters().ByName(in.Name)
	if !ok {
		return wire.ParameterInfo{}, errors.New("parameters.rename: no parameter named " + in.Name)
	}
	if err := holder.Parameters().Rename(p.ID(), in.NewName); err != nil {
		return wire.ParameterInfo{}, err
	}
	info := paramInfo(holder, p)
	emitParameterChanged(s, info) // #148: relay the rename to subscribing add-ins
	return info, nil
}

// convertParameter changes a parameter's category (user/model/reference) in place, preserving its
// identity, expression and dependency edges; converting to reference makes it read-only. A
// built-in/auto or derived parameter is refused by the model (#1850).
func convertParameter(s *app.Session, holder compdef.ParameterHolder, in wire.ParameterConvertArgs) (wire.ParameterInfo, error) {
	if in.Name == "" {
		return wire.ParameterInfo{}, errors.New("parameters.convert: name is required")
	}
	p, ok := holder.Parameters().ByName(in.Name)
	if !ok {
		return wire.ParameterInfo{}, errors.New("parameters.convert: no parameter named " + in.Name)
	}
	target, err := parseConvertKind(in.TargetKind)
	if err != nil {
		return wire.ParameterInfo{}, err
	}
	if err := holder.Parameters().Convert(p.ID(), target); err != nil {
		return wire.ParameterInfo{}, err
	}
	info := paramInfo(holder, p)
	emitParameterChanged(s, info) // relay the kind change to subscribing add-ins
	return info, nil
}

// parseConvertKind maps the wire targetKind spelling to a model parameter kind — only the three
// Inventor-convertible categories (ConvertTo{User,Model,Reference}) are accepted here.
func parseConvertKind(kind string) (param.ParameterKind, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "user":
		return param.UserParam, nil
	case "model":
		return param.ModelParam, nil
	case "reference":
		return param.ReferenceParam, nil
	default:
		return 0, fmt.Errorf("parameters.convert: unknown targetKind %q (want user|model|reference)", kind)
	}
}

// setParameter changes an existing parameter's expression and recomputes so any
// driven features update.
func setParameter(s *app.Session, holder compdef.ParameterHolder, in wire.ParameterSetArgs) (wire.ParameterInfo, error) {
	if err := requireNameExpr(in); err != nil {
		return wire.ParameterInfo{}, err
	}
	p, ok := holder.Parameters().ByName(in.Name)
	if !ok {
		return wire.ParameterInfo{}, errors.New("parameters.set: no parameter named " + in.Name)
	}
	// A bare number means the parameter's own display unit (7 → 7 mm for a Length parameter),
	// not the raw database unit — qualify before setting, matching the dimension/feature paths
	// and the head, so the wire and UI resolve typed values identically (#1783).
	expr := p.QualifyAuthored(in.Expression, p.Unit(), holder.Units())
	// Edit through the Parameters graph (not p.SetExpression, which only updates
	// this parameter): the graph rewires edges and recomputes transitive
	// dependents, so a dimension like "od/2" follows when od changes.
	if err := holder.Parameters().SetExpression(p.ID(), expr); err != nil {
		return wire.ParameterInfo{}, err
	}
	// A parameter edit can change any feature's live inputs (sketch dimensions,
	// value closures), which the engine does not track as dependencies, so force a
	// full parametric rebuild — the shared seam every edit path uses (#1413).
	holder.RecomputeAfterChange()
	info := paramInfo(holder, p)
	emitParameterChanged(s, info) // #148 granular parameter-change notification
	return info, nil
}

// emitParameterChanged publishes the granular parameter-change event on the session bus (#148),
// carrying the parameter's new state for relay to subscribing add-ins.
func emitParameterChanged(s *app.Session, info wire.ParameterInfo) {
	event.Emit(s.Events(), event.After, app.ParameterChanged{
		Document:   s.ActiveDocument().ID(),
		Name:       info.Name,
		Kind:       info.Kind,
		Expression: info.Expression,
		Value:      info.Value,
	})
}

// requireNameExpr rejects a parameter add/set missing either field, before touching the model
// (shared by add and set — the validation holderAndSetArgs used to carry, #1649).
func requireNameExpr(in wire.ParameterSetArgs) error {
	if in.Name == "" || in.Expression == "" {
		return errors.New("parameters: name and expression are required")
	}
	return nil
}
