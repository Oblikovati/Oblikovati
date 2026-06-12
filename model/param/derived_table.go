// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	"slices"
)

// Derived parameter tables (the reference API's DerivedParameterTable,
// M02-F06, Oblikovati#605): the connection objects that link parameters from
// another document into this one. A table records the source document and the
// chosen linked subset, and owns the resulting read-only [DerivedParam]
// parameters. The collection knows nothing about documents — callers resolve
// the source into [SourceParameterValue]s (app.Session does, through the
// workspace reference graph) and feed them to the sync methods, so the model
// stays document-free and headless-testable.

// SourceParameterValue is one linkable source parameter as the resolver sees
// it: its name and its current evaluated quantity (database units). Only
// numeric parameters link — a derived parameter is a value, not a text.
type SourceParameterValue struct {
	Name  string
	Value Quantity
}

// DerivedParameterTable is one cross-document link. The id is stable for the
// document's life (persisted, so undo restores keep wire handles valid).
type DerivedParameterTable struct {
	id             int
	sourceDocument string
	linked         []string
	produced       map[string]ID // linked source name → derived parameter id
	health         Health
	ownedByFeature bool
}

// ID returns the table's stable identity.
func (t *DerivedParameterTable) ID() int { return t.id }

// SourceDocument returns the linked document's full document name.
func (t *DerivedParameterTable) SourceDocument() string { return t.sourceDocument }

// Linked returns the linked source-parameter names, in link order.
func (t *DerivedParameterTable) Linked() []string { return append([]string(nil), t.linked...) }

// Health returns the table's link health: healthy when every link resolves.
func (t *DerivedParameterTable) Health() Health { return t.health }

// OwnedByFeature reports whether a derived component owns this table — such a
// table dies with its component, never through DeleteDerivedTable.
func (t *DerivedParameterTable) OwnedByFeature() bool { return t.ownedByFeature }

// MarkOwnedByFeature hands the table to a derived component (M08-F04).
func (t *DerivedParameterTable) MarkOwnedByFeature() { t.ownedByFeature = true }

// DerivedTables returns the tables in creation order.
func (ps *Parameters) DerivedTables() []*DerivedParameterTable {
	return append([]*DerivedParameterTable(nil), ps.derivedTables...)
}

// DerivedTableByID returns the table with the given id.
func (ps *Parameters) DerivedTableByID(id int) (*DerivedParameterTable, bool) {
	for _, t := range ps.derivedTables {
		if t.id == id {
			return t, true
		}
	}
	return nil, false
}

// AddDerivedTable links the named source parameters into this collection,
// creating one read-only derived parameter per linked name. Every linked name
// must exist in source; a clash with an existing local parameter rejects the
// add.
func (ps *Parameters) AddDerivedTable(sourceDocument string, linked []string, source []SourceParameterValue) (*DerivedParameterTable, error) {
	if sourceDocument == "" {
		return nil, fmt.Errorf("param: derived table needs a source document")
	}
	t := &DerivedParameterTable{
		id: ps.nextTableID, sourceDocument: sourceDocument, produced: map[string]ID{},
	}
	if err := ps.linkDerived(t, linked, source); err != nil {
		return nil, err
	}
	ps.nextTableID++
	ps.derivedTables = append(ps.derivedTables, t)
	return t, nil
}

// SetDerivedTableLinked replaces the table's linked subset: newly linked
// names gain derived parameters, unlinked ones lose theirs, kept ones update.
func (ps *Parameters) SetDerivedTableLinked(id int, linked []string, source []SourceParameterValue) error {
	t, ok := ps.DerivedTableByID(id)
	if !ok {
		return fmt.Errorf("param: no derived table with id %d", id)
	}
	for name, pid := range t.produced {
		if slices.Contains(linked, name) {
			continue
		}
		if p, exists := ps.byID[pid]; exists {
			ps.remove(p)
		}
		delete(t.produced, name)
	}
	return ps.linkDerived(t, linked, source)
}

// linkDerived points the table at the given subset, creating missing derived
// parameters and refreshing the values of kept ones.
func (ps *Parameters) linkDerived(t *DerivedParameterTable, linked []string, source []SourceParameterValue) error {
	values := sourceByName(source)
	for _, name := range linked {
		q, ok := values[name]
		if !ok {
			return fmt.Errorf("param: source document %q has no linkable parameter %q", t.sourceDocument, name)
		}
		if err := ps.produceDerived(t, name, q); err != nil {
			return err
		}
	}
	t.linked = append([]string(nil), linked...)
	t.health = Health{Status: Healthy}
	return nil
}

// produceDerived creates or refreshes the derived parameter for one link.
func (ps *Parameters) produceDerived(t *DerivedParameterTable, name string, q Quantity) error {
	if pid, exists := t.produced[name]; exists {
		ps.refreshDerivedValue(pid, q)
		return nil
	}
	p, err := ps.AddDerivedParameter(name, q)
	if err != nil {
		return fmt.Errorf("param: link %q: %w", name, err)
	}
	t.produced[name] = p.ID()
	return nil
}

// SyncDerivedTable refreshes the table from the source's current parameters.
// sourceOK=false means the source document itself is unreachable; a missing
// linked name turns its derived parameter sick instead of deleting it, so the
// model keeps computing on the last known value.
func (ps *Parameters) SyncDerivedTable(id int, source []SourceParameterValue, sourceOK bool) error {
	t, ok := ps.DerivedTableByID(id)
	if !ok {
		return fmt.Errorf("param: no derived table with id %d", id)
	}
	if !sourceOK {
		ps.sickenTable(t, "source document "+t.sourceDocument+" is unavailable")
		return nil
	}
	values := sourceByName(source)
	broken := ps.syncLinks(t, values)
	if len(broken) > 0 {
		t.health = Health{Status: Failed, Reason: fmt.Sprintf("removed at source: %v", broken)}
		return nil
	}
	t.health = Health{Status: Healthy}
	return nil
}

// syncLinks refreshes every link, returning the names no longer at the source.
func (ps *Parameters) syncLinks(t *DerivedParameterTable, values map[string]Quantity) []string {
	var broken []string
	for _, name := range t.linked {
		pid, produced := t.produced[name]
		if !produced {
			continue
		}
		q, ok := values[name]
		if !ok {
			broken = append(broken, name)
			ps.sickenDerived(pid, "removed at source document "+t.sourceDocument)
			continue
		}
		ps.refreshDerivedValue(pid, q)
	}
	return broken
}

// RestoreDerivedTable re-creates a persisted table, reconnecting its produced
// derived parameters by name (they restore with the parameter list itself).
// A linked name without a matching derived parameter stays unproduced until
// the next sync re-links it.
func (ps *Parameters) RestoreDerivedTable(id int, sourceDocument string, linked []string, ownedByFeature bool) error {
	if _, exists := ps.DerivedTableByID(id); exists {
		return fmt.Errorf("param: a derived table with id %d already exists", id)
	}
	t := &DerivedParameterTable{
		id: id, sourceDocument: sourceDocument, linked: append([]string(nil), linked...),
		produced: map[string]ID{}, ownedByFeature: ownedByFeature,
		health: Health{Status: Healthy},
	}
	for _, name := range linked {
		if p, ok := ps.ByName(name); ok && p.Kind() == DerivedParam {
			t.produced[name] = p.ID()
		}
	}
	if id >= ps.nextTableID {
		ps.nextTableID = id + 1
	}
	ps.derivedTables = append(ps.derivedTables, t)
	return nil
}

// DeleteDerivedTable removes a table and its derived parameters. A table
// owned by a derived component is deleted through its component, never here.
func (ps *Parameters) DeleteDerivedTable(id int) error {
	t, ok := ps.DerivedTableByID(id)
	if !ok {
		return fmt.Errorf("param: no derived table with id %d", id)
	}
	if t.ownedByFeature {
		return fmt.Errorf("param: derived table %d is owned by a derived component; delete the component instead", id)
	}
	for _, pid := range t.produced {
		if p, exists := ps.byID[pid]; exists {
			ps.remove(p)
		}
	}
	for i, other := range ps.derivedTables {
		if other.id == id {
			ps.derivedTables = append(ps.derivedTables[:i], ps.derivedTables[i+1:]...)
			break
		}
	}
	return nil
}

// refreshDerivedValue updates one read-only derived parameter in place and
// recomputes whatever reads it.
func (ps *Parameters) refreshDerivedValue(id ID, q Quantity) {
	p, ok := ps.byID[id]
	if !ok {
		return
	}
	p.expr, p.value, p.health = constantExpr(q), q, Health{Status: Healthy}
	ps.recomputeFrom(id)
}

// sickenDerived fails one derived parameter, keeping its last value so the
// geometry it drives stays computable.
func (ps *Parameters) sickenDerived(id ID, reason string) {
	if p, ok := ps.byID[id]; ok {
		p.health = Health{Status: Failed, Reason: reason}
	}
}

// sickenTable fails the table and every parameter it produced.
func (ps *Parameters) sickenTable(t *DerivedParameterTable, reason string) {
	t.health = Health{Status: Failed, Reason: reason}
	for _, pid := range t.produced {
		ps.sickenDerived(pid, reason)
	}
}

// sourceByName indexes the resolver's view for link lookups.
func sourceByName(source []SourceParameterValue) map[string]Quantity {
	out := make(map[string]Quantity, len(source))
	for _, s := range source {
		out[s.Name] = s.Value
	}
	return out
}
