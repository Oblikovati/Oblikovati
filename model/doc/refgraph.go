// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"
	"sort"
)

// AddReference records that this document references the document named childName,
// returning the descriptor. It errors for a standalone document not owned by a
// workspace (there is no graph to record into).
func (d *Document) AddReference(childName string) (*DocumentDescriptor, error) {
	if d.graph == nil {
		return nil, fmt.Errorf("doc: %q is not in a workspace; cannot add a reference", d.fullDocumentName)
	}
	return d.graph.addReference(d, childName), nil
}

// RemoveReference drops this document's reference to childName, reporting whether
// one existed.
func (d *Document) RemoveReference(childName string) bool {
	if d.graph == nil {
		return false
	}
	return d.graph.removeReference(d, childName)
}

// References returns this document's outgoing reference descriptors, in order.
func (d *Document) References() []*DocumentDescriptor {
	if d.graph == nil {
		return nil
	}
	return d.graph.descriptors(d.fullDocumentName)
}

// ReferencedDocuments returns the documents this one references (e.g. an assembly's
// parts), skipping references that cannot be resolved.
func (d *Document) ReferencedDocuments() []*Document {
	if d.graph == nil {
		return nil
	}
	return d.graph.referenced(d.fullDocumentName)
}

// ReferencingDocuments returns the open documents that reference this one (e.g. the
// assemblies a part is used in).
func (d *Document) ReferencingDocuments() []*Document {
	if d.graph == nil {
		return nil
	}
	return d.graph.referencing(d.fullDocumentName)
}

// AllReferencedDocuments returns the transitive closure of documents this one
// references, directly or indirectly.
func (d *Document) AllReferencedDocuments() []*Document {
	if d.graph == nil {
		return nil
	}
	return d.graph.allReferenced(d.fullDocumentName)
}

// DocumentDescriptor records one document's reference to another by full document
// name, so the reference survives even when the referenced document is not loaded
// (lazy resolution). It also carries the topology reference key (M03-F05, a
// placeholder byte slice today) and a needs-update flag for out-of-date references.
//
// A descriptor whose target cannot be resolved is flagged broken — broken
// references are surfaced, never fatal (architecture core/05).
type DocumentDescriptor struct {
	fullDocumentName string
	referenceKey     []byte // topology reference key payload; populated in M03-F05
	needsUpdate      bool
	broken           bool
	resolved         *Document // cache of the last successful resolution
}

// FullDocumentName returns the identity of the referenced document.
func (d *DocumentDescriptor) FullDocumentName() string { return d.fullDocumentName }

// NeedsUpdate reports whether the reference is out of date relative to its target.
func (d *DocumentDescriptor) NeedsUpdate() bool { return d.needsUpdate }

// SetNeedsUpdate marks the reference as needing (or no longer needing) an update.
func (d *DocumentDescriptor) SetNeedsUpdate(needs bool) { d.needsUpdate = needs }

// IsBroken reports whether the last resolution attempt failed to find the target.
func (d *DocumentDescriptor) IsBroken() bool { return d.broken }

// ReferenceKey returns the topology reference-key payload bound to this reference,
// or nil if none. It is populated once persistent identity lands (M03-F05).
func (d *DocumentDescriptor) ReferenceKey() []byte { return d.referenceKey }

// SetReferenceKey binds a topology reference-key payload to this reference.
func (d *DocumentDescriptor) SetReferenceKey(key []byte) { d.referenceKey = key }

// ReferencedDocument returns the resolved target document, or false if it has not
// been (or cannot be) resolved.
func (d *DocumentDescriptor) ReferencedDocument() (*Document, bool) {
	return d.resolved, d.resolved != nil
}

// RefGraph is the directed reference graph among the workspace's documents: an
// assembly references its parts, a drawing references the models it documents. It
// is owned by one [Workspace] and shared by reference from every document in it,
// so a document can answer reference queries about itself (core/02, core/05).
type RefGraph struct {
	ws      *Workspace
	forward map[string][]*DocumentDescriptor // parent name → its references, in order
	reverse map[string]map[string]bool       // child name → set of referencing parents
}

func newRefGraph(ws *Workspace) *RefGraph {
	return &RefGraph{ws: ws, forward: map[string][]*DocumentDescriptor{}, reverse: map[string]map[string]bool{}}
}

// addReference records that parent references the document named childName and
// returns the descriptor. If the child is already open it is counted as referenced
// so an unreferenced-only close leaves it open.
func (g *RefGraph) addReference(parent *Document, childName string) *DocumentDescriptor {
	desc := &DocumentDescriptor{fullDocumentName: childName}
	g.forward[parent.fullDocumentName] = append(g.forward[parent.fullDocumentName], desc)
	if g.reverse[childName] == nil {
		g.reverse[childName] = map[string]bool{}
	}
	g.reverse[childName][parent.fullDocumentName] = true
	if child, ok := g.ws.byName[childName]; ok && child.open {
		child.acquireRef()
	}
	return desc
}

// addReferenceUnique records parent→childName only when no such edge exists yet,
// returning the existing or freshly created descriptor. It is the idempotent form used
// when resolving restored references (#715): a component placed N times, or a reference
// re-resolved across recomputes, must yield exactly one edge so the referenced-by count
// is not inflated (addReference appends unconditionally, by contrast).
func (g *RefGraph) addReferenceUnique(parent *Document, childName string) *DocumentDescriptor {
	for _, desc := range g.forward[parent.fullDocumentName] {
		if desc.fullDocumentName == childName {
			return desc
		}
	}
	return g.addReference(parent, childName)
}

// removeReference drops the edge from parent to childName, if present, and releases
// the child's referenced count.
func (g *RefGraph) removeReference(parent *Document, childName string) bool {
	descs := g.forward[parent.fullDocumentName]
	for i, desc := range descs {
		if desc.fullDocumentName != childName {
			continue
		}
		g.forward[parent.fullDocumentName] = append(descs[:i], descs[i+1:]...)
		delete(g.reverse[childName], parent.fullDocumentName)
		if child, ok := g.ws.byName[childName]; ok && child.open {
			child.releaseRef()
		}
		return true
	}
	return false
}

// repointReference updates the edge parent→oldName to point at newName, fixing
// the reverse index and reference counts — the ReplaceReference repair
// (M03-F07, #608). A missing edge is a no-op: the persisted record is then the
// only thing repaired, which is still correct for a reference whose graph edge
// has not been rebuilt this session.
func (g *RefGraph) repointReference(parent *Document, oldName, newName string) {
	for _, desc := range g.forward[parent.fullDocumentName] {
		if desc.fullDocumentName != oldName {
			continue
		}
		desc.fullDocumentName, desc.resolved, desc.broken = newName, nil, false
		delete(g.reverse[oldName], parent.fullDocumentName)
		if g.reverse[newName] == nil {
			g.reverse[newName] = map[string]bool{}
		}
		g.reverse[newName][parent.fullDocumentName] = true
		g.shiftRefCount(oldName, newName)
		return
	}
}

// shiftRefCount moves one referenced-by count from the old target to the new.
func (g *RefGraph) shiftRefCount(oldName, newName string) {
	if child, ok := g.ws.byName[oldName]; ok && child.open {
		child.releaseRef()
	}
	if child, ok := g.ws.byName[newName]; ok && child.open {
		child.acquireRef()
	}
}

// descriptors returns the reference descriptors of the named parent, in order.
func (g *RefGraph) descriptors(parentName string) []*DocumentDescriptor {
	src := g.forward[parentName]
	out := make([]*DocumentDescriptor, len(src))
	copy(out, src)
	return out
}

// referenced returns the resolved target documents of the named parent, skipping
// any whose target cannot be resolved (those descriptors are flagged broken).
func (g *RefGraph) referenced(parentName string) []*Document {
	out := make([]*Document, 0, len(g.forward[parentName]))
	for _, desc := range g.forward[parentName] {
		if d, ok := g.resolve(desc); ok {
			out = append(out, d)
		}
	}
	return out
}

// referencing returns the open documents that reference the named child, ordered
// by name for determinism.
func (g *RefGraph) referencing(childName string) []*Document {
	names := make([]string, 0, len(g.reverse[childName]))
	for parent := range g.reverse[childName] {
		names = append(names, parent)
	}
	sort.Strings(names)
	out := make([]*Document, 0, len(names))
	for _, name := range names {
		if d, ok := g.ws.byName[name]; ok {
			out = append(out, d)
		}
	}
	return out
}

// allReferenced returns the transitive closure of documents reachable from the
// named parent, in first-seen order, with no duplicates.
func (g *RefGraph) allReferenced(parentName string) []*Document {
	seen := map[string]bool{}
	var out []*Document
	var visit func(name string)
	visit = func(name string) {
		for _, child := range g.referenced(name) {
			if seen[child.fullDocumentName] {
				continue
			}
			seen[child.fullDocumentName] = true
			out = append(out, child)
			visit(child.fullDocumentName)
		}
	}
	visit(parentName)
	return out
}

// resolve binds a descriptor to a live document: an already-open document wins;
// otherwise it is lazily loaded through the workspace store. Failure flags the
// descriptor broken and returns false (never fatal).
func (g *RefGraph) resolve(desc *DocumentDescriptor) (*Document, bool) {
	if d, ok := g.ws.byName[desc.fullDocumentName]; ok && d.open {
		desc.resolved, desc.broken = d, false
		return d, true
	}
	if g.ws.store != nil && g.ws.store.Exists(desc.fullDocumentName) {
		if d, err := g.ws.OpenWithOptions(desc.fullDocumentName, OpenOptions{Visible: false}); err == nil {
			desc.resolved, desc.broken = d, false
			return d, true
		}
	}
	desc.broken = true
	return nil, false
}
