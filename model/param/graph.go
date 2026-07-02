// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	"slices"
)

// This file is the parameter dependency graph (F04): the same dirty-propagation
// machinery the feature engine will reuse (architecture core/04). Edges are
// keyed by stable id, cycles are rejected at edit time (the parameter goes
// sick, never a crash), and a change recomputes exactly its transitive
// dependents in dependency order.

// CycleError reports that an edit would make a parameter depend on itself.
type CycleError struct {
	ID   ID
	Name string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("param: expression for %q (id %d) forms a dependency cycle", e.Name, e.ID)
}

// SetExpression re-authors a parameter's expression and reflows the graph: it
// rejects cycles (marking the parameter sick), rewires edges, and recomputes
// the parameter and its transitive dependents. Malformed source returns a parse
// error with the edit not applied.
func (ps *Parameters) SetExpression(id ID, src string) error {
	p, ok := ps.byID[id]
	if !ok {
		return fmt.Errorf(errNoParameter, id)
	}
	if !p.kind.Editable() {
		return fmt.Errorf(errReadOnly, p.kind, p.name)
	}
	e, err := Parse(src)
	if err != nil {
		return err
	}
	p.expr = e
	return ps.rebind(id)
}

// SetValue assigns a constant value through the graph, recomputing dependents.
func (ps *Parameters) SetValue(id ID, value Quantity) error {
	p, ok := ps.byID[id]
	if !ok {
		return fmt.Errorf(errNoParameter, id)
	}
	if !p.kind.Editable() {
		return fmt.Errorf(errReadOnly, p.kind, p.name)
	}
	p.expr = constantExpr(value)
	return ps.rebind(id)
}

// DrivenBy returns the parameters this one reads, in insertion order.
func (ps *Parameters) DrivenBy(id ID) []ID { return ps.orderedSet(ps.drivenBy[id]) }

// Dependents returns the parameters that read this one, in insertion order.
func (ps *Parameters) Dependents(id ID) []ID { return ps.orderedSet(ps.dependents[id]) }

// InUse reports whether the parameter currently drives anything: another
// parameter reads it, or it is a model parameter (owned by a feature/dimension,
// which consumes its model value by construction).
func (ps *Parameters) InUse(id ID) bool {
	if len(ps.dependents[id]) > 0 {
		return true
	}
	p, ok := ps.byID[id]
	return ok && p.kind == ModelParam
}

// rebind resolves the parameter's references to ids, rejects a resulting cycle,
// rewires its edges, and recomputes it together with its dependents.
func (ps *Parameters) rebind(id ID) error {
	p := ps.byID[id]
	p.expr.Bind(ps.resolver())
	drivers := p.expr.boundRefs()
	if ps.wouldCycle(id, drivers) {
		ps.detachEdges(id)
		p.health = Health{Status: Failed, Reason: "cyclic reference"}
		return &CycleError{ID: id, Name: p.name}
	}
	ps.detachEdges(id)
	ps.attachEdges(id, drivers)
	ps.recomputeFrom(id)
	return nil
}

// onParameterAdded resolves any existing parameter that referenced the new
// parameter's name but could not bind before it existed.
func (ps *Parameters) onParameterAdded(p *Parameter) {
	for _, other := range ps.All() {
		if other.id != p.id && slices.Contains(other.expr.References(), p.name) {
			_ = ps.rebind(other.id) // a re-add cannot introduce a new cycle here
		}
	}
}

// resolver maps a reference name to a stable id for binding.
func (ps *Parameters) resolver() func(string) (ID, bool) {
	return func(name string) (ID, bool) {
		id, ok := ps.byName[name]
		return id, ok
	}
}

// wouldCycle reports whether making id depend on drivers creates a cycle —
// i.e. id already (transitively) drives one of the proposed drivers.
func (ps *Parameters) wouldCycle(id ID, drivers []ID) bool {
	closure := ps.dependentsClosure(id)
	for _, d := range drivers {
		if d == id || closure[d] {
			return true
		}
	}
	return false
}

// dependentsClosure returns every parameter that transitively depends on id
// (id excluded).
func (ps *Parameters) dependentsClosure(id ID) idSet {
	closure := idSet{}
	var visit func(ID)
	visit = func(n ID) {
		for dep := range ps.dependents[n] {
			if !closure[dep] {
				closure[dep] = true
				visit(dep)
			}
		}
	}
	visit(id)
	return closure
}

// detachEdges removes id's outgoing dependency edges.
func (ps *Parameters) detachEdges(id ID) {
	for d := range ps.drivenBy[id] {
		delete(ps.dependents[d], id)
	}
	delete(ps.drivenBy, id)
}

// attachEdges records that id depends on each driver.
func (ps *Parameters) attachEdges(id ID, drivers []ID) {
	if len(drivers) == 0 {
		return
	}
	ps.drivenBy[id] = idSet{}
	for _, d := range drivers {
		ps.drivenBy[id][d] = true
		if ps.dependents[d] == nil {
			ps.dependents[d] = idSet{}
		}
		ps.dependents[d][id] = true
	}
}

// recomputeFrom re-evaluates id and its transitive dependents in dependency
// order — and nothing else. The reflowed set is exactly the parameters this edit
// can change, so it is recorded for the feature engine's targeted invalidation
// (Oblikovati#1414): a later DrainChanged hands it to the edit seam.
func (ps *Parameters) recomputeFrom(id ID) {
	dirty := ps.dependentsClosure(id)
	dirty[id] = true
	for _, n := range ps.topoOrder(dirty) {
		ps.markChanged(n)
		ps.evaluate(ps.byID[n])
	}
}

// topoOrder orders a dirty set so each parameter follows the dependencies it has
// within the set (drivers first). The graph is acyclic, so this terminates.
func (ps *Parameters) topoOrder(set idSet) []ID {
	var order []ID
	visited := idSet{}
	var visit func(ID)
	visit = func(n ID) {
		if visited[n] {
			return
		}
		visited[n] = true
		for d := range ps.drivenBy[n] {
			if set[d] {
				visit(d)
			}
		}
		order = append(order, n)
	}
	for n := range set {
		visit(n)
	}
	return order
}

// evaluate recomputes one parameter against the collection, updating its value
// and health. Undefined references and dimensional errors become sick health.
func (ps *Parameters) evaluate(p *Parameter) {
	if p.expr.root == nil {
		return
	}
	if !hasRefs(p.expr.root) {
		p.reevaluateConstant()
		return
	}
	if unresolved := p.expr.Bind(ps.resolver()); len(unresolved) > 0 {
		p.health = Health{Status: Failed, Reason: "undefined reference: " + unresolved[0]}
		return
	}
	q, err := p.expr.Eval(ps)
	if err != nil {
		p.health = Health{Status: Failed, Reason: err.Error()}
		return
	}
	p.value, p.health = q, Health{Status: Healthy}
}

// orderedSet returns the set's ids in the collection's insertion order.
func (ps *Parameters) orderedSet(set idSet) []ID {
	var out []ID
	for _, id := range ps.order {
		if set[id] {
			out = append(out, id)
		}
	}
	return out
}

// keysOf returns the keys of an id set (order unspecified).
func keysOf(set idSet) []ID {
	out := make([]ID, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}
