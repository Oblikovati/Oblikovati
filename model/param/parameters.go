// SPDX-License-Identifier: GPL-2.0-only

package param

import "fmt"

// Parameters is the owning collection for a document's parameters (contract:
// Parameters / ModelParameters / UserParameters). It assigns stable ids,
// enforces unique names, and is the [Scope] expressions evaluate against. The
// dependency-graph methods (edges, cycle detection, recompute) are added in F04.
type Parameters struct {
	byID   map[ID]*Parameter
	byName map[string]ID
	order  []ID
	nextID ID

	// Dependency graph (F04): edges keyed by stable id. drivenBy[p] is the set
	// of parameters p's expression reads; dependents[p] is the reverse.
	drivenBy   map[ID]idSet
	dependents map[ID]idSet

	// Custom parameter groups (Inventor's CustomParameterGroups): names in creation
	// order plus each parameter's group membership. See group.go.
	groupOrder []string
	groupOf    map[ID]string
}

// idSet is a set of parameter ids.
type idSet map[ID]bool

// NewParameters returns an empty collection.
func NewParameters() *Parameters {
	return &Parameters{
		byID: map[ID]*Parameter{}, byName: map[string]ID{}, nextID: 1,
		drivenBy: map[ID]idSet{}, dependents: map[ID]idSet{},
		groupOf: map[ID]string{},
	}
}

// Count returns the number of parameters.
func (ps *Parameters) Count() int { return len(ps.order) }

// All returns the parameters in insertion order.
func (ps *Parameters) All() []*Parameter {
	out := make([]*Parameter, len(ps.order))
	for i, id := range ps.order {
		out[i] = ps.byID[id]
	}
	return out
}

// ByID returns the parameter with the given id.
func (ps *Parameters) ByID(id ID) (*Parameter, bool) {
	p, ok := ps.byID[id]
	return p, ok
}

// ByName returns the parameter with the given display name.
func (ps *Parameters) ByName(name string) (*Parameter, bool) {
	id, ok := ps.byName[name]
	if !ok {
		return nil, false
	}
	return ps.byID[id], true
}

// ValueOf implements [Scope]: it returns a parameter's current evaluated value.
func (ps *Parameters) ValueOf(id ID) (Quantity, bool) {
	p, ok := ps.byID[id]
	if !ok {
		return Quantity{}, false
	}
	return p.value, true
}

// AddUserParameter adds an editable user parameter defined by an expression.
func (ps *Parameters) AddUserParameter(name, expression string) (*Parameter, error) {
	return ps.addEditable(name, expression, UserParam)
}

// AddModelParameter adds an editable model parameter (a feature dimension).
func (ps *Parameters) AddModelParameter(name, expression string) (*Parameter, error) {
	return ps.addEditable(name, expression, ModelParam)
}

// AddAutoModelParameter adds a model parameter under a freshly minted unique name
// ("d0", "d1", …) — used to back a feature's dimensional argument (extrude depth,
// revolve angle) so the argument joins the dependency graph and may itself
// reference other parameters, exactly like a sketch dimension.
func (ps *Parameters) AddAutoModelParameter(expression string) (*Parameter, error) {
	return ps.AddModelParameter(ps.uniqueName("d"), expression)
}

// uniqueName mints an unused parameter name with the given prefix.
func (ps *Parameters) uniqueName(prefix string) string {
	for i := 0; ; i++ {
		name := fmt.Sprintf("%s%d", prefix, i)
		if _, taken := ps.ByName(name); !taken {
			return name
		}
	}
}

// AddTextUserParameter adds an editable user parameter holding a text literal. Text
// parameters carry no expression and take no part in the dependency graph.
func (ps *Parameters) AddTextUserParameter(name, value string) (*Parameter, error) {
	p, err := ps.add(name, UserParam)
	if err != nil {
		return nil, err
	}
	p.value = Q(0, Text)
	if err := p.SetText(value); err != nil {
		ps.remove(p)
		return nil, err
	}
	ps.onParameterAdded(p)
	return p, nil
}

// AddBooleanUserParameter adds an editable user parameter holding a true/false value.
func (ps *Parameters) AddBooleanUserParameter(name string, value bool) (*Parameter, error) {
	p, err := ps.add(name, UserParam)
	if err != nil {
		return nil, err
	}
	p.value = Q(0, Boolean)
	if err := p.SetBool(value); err != nil {
		ps.remove(p)
		return nil, err
	}
	ps.onParameterAdded(p)
	return p, nil
}

// AddTableParameter adds an editable table parameter (iPart member value).
func (ps *Parameters) AddTableParameter(name, expression string) (*Parameter, error) {
	return ps.addEditable(name, expression, TableParam)
}

// AddReferenceParameter adds a read-only reference parameter holding a measured
// value (driven by geometry, not editable through the expression API).
func (ps *Parameters) AddReferenceParameter(name string, value Quantity) (*Parameter, error) {
	return ps.addReadOnly(name, value, ReferenceParam)
}

// AddDerivedParameter adds a read-only derived parameter (linked from another
// document).
func (ps *Parameters) AddDerivedParameter(name string, value Quantity) (*Parameter, error) {
	return ps.addReadOnly(name, value, DerivedParam)
}

// addEditable creates an editable parameter and sets its expression through the
// dependency graph, so edges are wired and dependents recompute.
func (ps *Parameters) addEditable(name, expression string, kind ParameterKind) (*Parameter, error) {
	p, err := ps.add(name, kind)
	if err != nil {
		return nil, err
	}
	if err := ps.SetExpression(p.id, expression); err != nil {
		ps.remove(p) // a parse error: do not leave a half-constructed parameter
		return nil, err
	}
	ps.onParameterAdded(p)
	return p, nil
}

// addReadOnly creates a read-only parameter with a fixed value.
func (ps *Parameters) addReadOnly(name string, value Quantity, kind ParameterKind) (*Parameter, error) {
	p, err := ps.add(name, kind)
	if err != nil {
		return nil, err
	}
	p.expr, p.value, p.health = constantExpr(value), value, Health{Status: Healthy}
	ps.onParameterAdded(p)
	return p, nil
}

// add allocates a parameter with a unique name and a fresh id.
func (ps *Parameters) add(name string, kind ParameterKind) (*Parameter, error) {
	if name == "" {
		return nil, fmt.Errorf("param: parameter name must not be empty")
	}
	if _, exists := ps.byName[name]; exists {
		return nil, fmt.Errorf("param: a parameter named %q already exists", name)
	}
	p := &Parameter{id: ps.nextID, name: name, kind: kind, Visible: true}
	ps.nextID++
	ps.byID[p.id] = p
	ps.byName[name] = p.id
	ps.order = append(ps.order, p.id)
	return p, nil
}

// Rename changes a parameter's display name, rejecting a clash with another
// parameter. Stable ids mean dependent expressions are unaffected.
func (ps *Parameters) Rename(id ID, newName string) error {
	p, ok := ps.byID[id]
	if !ok {
		return fmt.Errorf("param: no parameter with id %d", id)
	}
	if existing, taken := ps.byName[newName]; taken && existing != id {
		return fmt.Errorf("param: a parameter named %q already exists", newName)
	}
	oldName := p.name
	delete(ps.byName, oldName)
	ps.byName[newName] = id
	p.name = newName
	// References stay bound by stable id; update dependents' display text so it
	// tracks the rename (edges are untouched, so dependents still evaluate).
	for dep := range ps.dependents[id] {
		d := ps.byID[dep]
		d.expr.renameRef(id, newName)
		d.expr.src = renameInSource(d.expr.src, oldName, newName)
	}
	return nil
}

// Delete removes a parameter from the collection.
func (ps *Parameters) Delete(id ID) error {
	p, ok := ps.byID[id]
	if !ok {
		return fmt.Errorf("param: no parameter with id %d", id)
	}
	ps.remove(p)
	return nil
}

// remove deletes a parameter from all indices and detaches its edges; former
// dependents are recomputed and go sick on the now-undefined reference.
func (ps *Parameters) remove(p *Parameter) {
	formerDependents := keysOf(ps.dependents[p.id])
	ps.detachEdges(p.id)
	for d := range ps.dependents[p.id] {
		delete(ps.drivenBy[d], p.id)
	}
	delete(ps.dependents, p.id)
	delete(ps.groupOf, p.id)
	delete(ps.byID, p.id)
	delete(ps.byName, p.name)
	for i, id := range ps.order {
		if id == p.id {
			ps.order = append(ps.order[:i], ps.order[i+1:]...)
			break
		}
	}
	for _, d := range formerDependents {
		if _, ok := ps.byID[d]; ok {
			ps.recomputeFrom(d)
		}
	}
}
