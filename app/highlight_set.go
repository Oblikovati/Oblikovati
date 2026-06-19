// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Highlight sets (#157): named, colored emphasis groups — entity references (the face/vertex
// key strings model.referenceKeys/model.selection report) the viewport outlines in the set's
// colour WITHOUT selecting them, so an add-in can guide the user (Inventor HighlightSet). The
// session holds the sets; the viewport resolves and draws them each frame, so they follow the
// geometry across recomputes (a lost reference simply isn't drawn).

// HighlightSet is one named, colored emphasis group.
type HighlightSet struct {
	name  string
	color types.Color
	refs  []string
}

// Name returns the set's name; Color its emphasis colour; Refs its current references.
func (h *HighlightSet) Name() string           { return h.name }
func (h *HighlightSet) Color() types.Color     { return h.color }
func (h *HighlightSet) Refs() []string         { return append([]string(nil), h.refs...) }
func (h *HighlightSet) Count() int             { return len(h.refs) }
func (h *HighlightSet) SetColor(c types.Color) { h.color = c }

// AddItems appends reference strings, ignoring blanks and duplicates.
func (h *HighlightSet) AddItems(refs ...string) {
	for _, r := range refs {
		if r != "" && !containsRef(h.refs, r) {
			h.refs = append(h.refs, r)
		}
	}
}

// HighlightSets is the session's ordered registry of highlight sets.
type HighlightSets struct {
	items  []*HighlightSet
	byName map[string]*HighlightSet
}

// NewHighlightSets returns an empty registry.
func NewHighlightSets() *HighlightSets {
	return &HighlightSets{byName: map[string]*HighlightSet{}}
}

// Create adds a new named set, erroring on an empty or already-used name.
func (hs *HighlightSets) Create(name string, color types.Color) (*HighlightSet, error) {
	if name == "" {
		return nil, fmt.Errorf("highlight set: name must not be empty")
	}
	if _, ok := hs.byName[name]; ok {
		return nil, fmt.Errorf("highlight set %q already exists", name)
	}
	h := &HighlightSet{name: name, color: color}
	hs.items = append(hs.items, h)
	hs.byName[name] = h
	return h, nil
}

// ByName returns the named set, or false.
func (hs *HighlightSets) ByName(name string) (*HighlightSet, bool) {
	h, ok := hs.byName[name]
	return h, ok
}

// Delete removes the named set, reporting whether it existed.
func (hs *HighlightSets) Delete(name string) bool {
	if _, ok := hs.byName[name]; !ok {
		return false
	}
	delete(hs.byName, name)
	for i, h := range hs.items {
		if h.name == name {
			hs.items = append(hs.items[:i], hs.items[i+1:]...)
			break
		}
	}
	return true
}

// All returns the sets in creation order.
func (hs *HighlightSets) All() []*HighlightSet { return append([]*HighlightSet(nil), hs.items...) }

// HighlightSets returns the session's highlight-set registry (lazily created).
func (s *Session) HighlightSets() *HighlightSets {
	if s.highlightSets == nil {
		s.highlightSets = NewHighlightSets()
	}
	return s.highlightSets
}

// ResolveReference resolves a face/vertex reference string (from model.referenceKeys /
// model.selection) to a selectable on the active part's bodies, or false when it does not bind.
// Shared by selection mutation (#157) and highlight-set rendering.
func (s *Session) ResolveReference(ref string) (Selectable, bool) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, false
	}
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return nil, false
	}
	return ResolveRefOnBodies(part.SurfaceBodies().All(), ref)
}

// ResolveRefOnBodies resolves a topology reference string against the given bodies, accepting
// BOTH ref forms an add-in encounters: the "face/…" / "vertex/…" form model.selection reports
// (decoded via feature.*RefKey) and the RAW reference-key bytes model.referenceKeys reports.
// The raw form additionally resolves edges (which the selection form does not round-trip).
func ResolveRefOnBodies(bodies []*topo.Body, ref string) (Selectable, bool) {
	if key, ok := feature.FaceRefKey(feature.WorkRef(ref)); ok {
		if sel, found := findFace(bodies, key); found {
			return sel, true
		}
	}
	if key, ok := feature.VertexRefKey(feature.WorkRef(ref)); ok {
		if sel, found := findVertex(bodies, key); found {
			return sel, true
		}
	}
	// Raw reference-key form (model.referenceKeys): try face, edge, then vertex.
	raw := []byte(ref)
	if sel, found := findFace(bodies, raw); found {
		return sel, true
	}
	if sel, found := findEdge(bodies, raw); found {
		return sel, true
	}
	return findVertex(bodies, raw)
}

// findFace/findEdge/findVertex bind a raw reference key to a selectable across the bodies.
func findFace(bodies []*topo.Body, key []byte) (Selectable, bool) {
	for _, b := range bodies {
		if f, ok := b.FindFaceByKey(key); ok {
			return FaceHandle{Face: f, Body: b}, true
		}
	}
	return nil, false
}

func findEdge(bodies []*topo.Body, key []byte) (Selectable, bool) {
	for _, b := range bodies {
		if e, ok := b.FindEdgeByKey(key); ok {
			return EdgeHandle{Edge: e}, true
		}
	}
	return nil, false
}

func findVertex(bodies []*topo.Body, key []byte) (Selectable, bool) {
	for _, b := range bodies {
		if v, ok := b.FindVertexByKey(key); ok {
			return VertexHandle{Vertex: v}, true
		}
	}
	return nil, false
}

func containsRef(refs []string, r string) bool {
	for _, x := range refs {
		if x == r {
			return true
		}
	}
	return false
}
