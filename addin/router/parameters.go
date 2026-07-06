// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"errors"

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
		Expression: p.Expression(), Value: holder.Units().Format(p.Value()),
	}
	if h := p.Health(); !h.OK() {
		info.Health = h.Reason
	}
	return info
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

// addParameter adds a new user parameter from name + expression (e.g. "4 cm").
func addParameter(_ *app.Session, holder compdef.ParameterHolder, in wire.ParameterSetArgs) (wire.ParameterInfo, error) {
	if err := requireNameExpr(in); err != nil {
		return wire.ParameterInfo{}, err
	}
	p, err := holder.Parameters().AddUserParameter(in.Name, in.Expression)
	if err != nil {
		return wire.ParameterInfo{}, err
	}
	return paramInfo(holder, p), nil
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
