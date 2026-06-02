// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// SelectionKind classifies what is selected — Inventor's selection-filter
// categories. A [SelectionFilter] restricts which kinds a pick accepts.
type SelectionKind uint8

const (
	SelectFace SelectionKind = iota
	SelectEdge
	SelectVertex
	SelectBody
	SelectFeature
	SelectProfile
	SelectSketchEntity
	SelectWorkPlane
)

// Selectable is anything the selection set can hold. Concrete handles wrap the
// underlying model entity so the kernel/model packages stay free of UI types;
// callers type-assert the concrete handle to reach the entity (no untyped any).
type Selectable interface {
	SelectionKind() SelectionKind
}

// FaceHandle / EdgeHandle / VertexHandle / BodyHandle wrap picked topology.
type (
	FaceHandle   struct{ Face *topo.Face }
	EdgeHandle   struct{ Edge *topo.Edge }
	VertexHandle struct{ Vertex *topo.Vertex }
	BodyHandle   struct{ Body *topo.Body }
)

func (FaceHandle) SelectionKind() SelectionKind   { return SelectFace }
func (EdgeHandle) SelectionKind() SelectionKind   { return SelectEdge }
func (VertexHandle) SelectionKind() SelectionKind { return SelectVertex }
func (BodyHandle) SelectionKind() SelectionKind   { return SelectBody }

// FeatureHandle wraps a picked feature (browser or graphics).
type FeatureHandle struct{ Feature *feature.PartFeature }

func (FeatureHandle) SelectionKind() SelectionKind { return SelectFeature }

// ProfileHandle wraps a sketch profile (the input an extrude/revolve consumes).
type ProfileHandle struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
}

func (ProfileHandle) SelectionKind() SelectionKind { return SelectProfile }

// SketchEntityHandle wraps a picked sketch entity.
type SketchEntityHandle struct{ Entity sketch.Entity }

func (SketchEntityHandle) SelectionKind() SelectionKind { return SelectSketchEntity }

// WorkPlaneHandle wraps a picked origin/work plane (selected to host a new sketch).
type WorkPlaneHandle struct{ Plane *feature.WorkPlane }

func (WorkPlaneHandle) SelectionKind() SelectionKind { return SelectWorkPlane }

// SelectionFilter restricts which selection kinds a pick accepts — Inventor's
// SelectionFilterEnum. A zero filter (no kinds) accepts everything.
type SelectionFilter struct {
	kinds map[SelectionKind]bool
}

// NewSelectionFilter accepts only the given kinds; with none it accepts all.
func NewSelectionFilter(kinds ...SelectionKind) *SelectionFilter {
	f := &SelectionFilter{kinds: map[SelectionKind]bool{}}
	for _, k := range kinds {
		f.kinds[k] = true
	}
	return f
}

// Accepts reports whether the filter admits a kind (an empty filter admits all).
func (f *SelectionFilter) Accepts(k SelectionKind) bool {
	if f == nil || len(f.kinds) == 0 {
		return true
	}
	return f.kinds[k]
}

// Selection is the current selection set — Inventor's SelectSet. It is an ordered,
// de-duplicated list of selectables with an active filter.
type Selection struct {
	items  []Selectable
	filter *SelectionFilter
}

// NewSelection returns an empty selection that accepts all kinds.
func NewSelection() *Selection {
	return &Selection{filter: NewSelectionFilter()}
}

// SetFilter restricts subsequent picks/additions to the given kinds.
func (s *Selection) SetFilter(f *SelectionFilter) { s.filter = f }

// Filter returns the active selection filter.
func (s *Selection) Filter() *SelectionFilter { return s.filter }

// Add appends a selectable if the filter accepts its kind, reporting success.
func (s *Selection) Add(sel Selectable) bool {
	if !s.filter.Accepts(sel.SelectionKind()) {
		return false
	}
	s.items = append(s.items, sel)
	return true
}

// Clear empties the selection.
func (s *Selection) Clear() { s.items = nil }

// Count returns the number of selected items; Items returns a snapshot.
func (s *Selection) Count() int { return len(s.items) }

func (s *Selection) Items() []Selectable {
	out := make([]Selectable, len(s.items))
	copy(out, s.items)
	return out
}

// First returns the first selected item, or nil.
func (s *Selection) First() Selectable {
	if len(s.items) == 0 {
		return nil
	}
	return s.items[0]
}
