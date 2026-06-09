// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"errors"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// paramInfo marshals a model parameter into the wire DTO: its authored expression
// and its evaluated value formatted in the document's display units.
func paramInfo(part *compdef.PartComponentDefinition, p *param.Parameter) wire.ParameterInfo {
	info := wire.ParameterInfo{
		Name: p.Name(), Kind: p.Kind().String(),
		Expression: p.Expression(), Value: part.Units().Format(p.Value()),
	}
	if h := p.Health(); !h.OK() {
		info.Health = h.Reason
	}
	return info
}

// listParameters returns the active part's parameters.
func listParameters(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	ps := part.Parameters().All()
	out := make([]wire.ParameterInfo, len(ps))
	for i, p := range ps {
		out[i] = paramInfo(part, p)
	}
	return json.Marshal(wire.ListParametersResult{Parameters: out})
}

// getParameter returns one parameter by name.
func getParameter(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.ParameterNameArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	p, ok := part.Parameters().ByName(in.Name)
	if !ok {
		return nil, errors.New("parameters.get: no parameter named " + in.Name)
	}
	return json.Marshal(paramInfo(part, p))
}

// addParameter adds a new user parameter from name + expression (e.g. "4 cm").
func addParameter(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	part, in, err := partAndSetArgs(s, args)
	if err != nil {
		return nil, err
	}
	p, err := part.Parameters().AddUserParameter(in.Name, in.Expression)
	if err != nil {
		return nil, err
	}
	return json.Marshal(paramInfo(part, p))
}

// setParameter changes an existing parameter's expression and recomputes so any
// driven features update.
func setParameter(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	part, in, err := partAndSetArgs(s, args)
	if err != nil {
		return nil, err
	}
	p, ok := part.Parameters().ByName(in.Name)
	if !ok {
		return nil, errors.New("parameters.set: no parameter named " + in.Name)
	}
	// Edit through the Parameters graph (not p.SetExpression, which only updates
	// this parameter): the graph rewires edges and recomputes transitive
	// dependents, so a dimension like "od/2" follows when od changes.
	if err := part.Parameters().SetExpression(p.ID(), in.Expression); err != nil {
		return nil, err
	}
	// A parameter edit can change any feature's live inputs (sketch dimensions,
	// value closures), which the engine does not track as dependencies, so force a
	// full parametric rebuild.
	part.Features().MarkAllDirty()
	part.Recompute()
	return json.Marshal(paramInfo(part, p))
}

// partAndSetArgs decodes name+expression and resolves the active part, validating
// both fields are present (shared by add and set).
func partAndSetArgs(s *app.Session, args json.RawMessage) (*compdef.PartComponentDefinition, wire.ParameterSetArgs, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, wire.ParameterSetArgs{}, err
	}
	var in wire.ParameterSetArgs
	if err := decode(args, &in); err != nil {
		return nil, wire.ParameterSetArgs{}, err
	}
	if in.Name == "" || in.Expression == "" {
		return nil, wire.ParameterSetArgs{}, errors.New("parameters: name and expression are required")
	}
	return part, in, nil
}
