// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/model/sketch"
)

// edgeHandleAnchors captures each picked edge's midpoint, keyed by its reference key, at
// SELECTION time (the edge still exists) — the mint-time anchor a dress-up feature persists
// for the geometric recovery tier (ADR-0043 P6b). The midpoint uses the edge's endpoints, the
// same representative point the resolver compares siblings by, so a captured anchor and a live
// sibling's anchor are computed identically. Returns nil when nothing anchorable was picked.
func edgeHandleAnchors(edges []EdgeHandle) map[string]math.Point3 {
	anchors := make(map[string]math.Point3, len(edges))
	for _, h := range edges {
		s, e := h.Edge.StartVertex(), h.Edge.EndVertex()
		if s == nil || e == nil {
			continue
		}
		anchors[string(h.Edge.ReferenceKey())] = s.Point().Midpoint(e.Point())
	}
	if len(anchors) == 0 {
		return nil
	}
	return anchors
}

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
	SelectSketch
	SelectWorkPoint
	SelectWorkAxis
	SelectPath
	SelectOccurrence
	SelectConstraint
	SelectJoint
	SelectRepresentation
	SelectModelState
	SelectDrawingView
	SelectPointCloud
	SelectPointCloudPoint
	SelectMeshFace         // a facet of a placed mesh reference (#1776)
	SelectSketchDimension  // a 2D-sketch dimension, selected by its value label (#2015)
	SelectSketchConstraint // a 2D-sketch geometric constraint, selected by its Show-Constraints marker
)

// Selectable is anything the selection set can hold. Concrete handles wrap the
// underlying model entity so the kernel/model packages stay free of UI types;
// callers type-assert the concrete handle to reach the entity (no untyped any).
type Selectable interface {
	SelectionKind() SelectionKind
}

// FaceHandle / EdgeHandle / VertexHandle / BodyHandle wrap picked topology. FaceHandle
// also carries the owning Body so a face pick can light up that body's browser node.
type (
	FaceHandle struct {
		Face *topo.Face
		Body *topo.Body
	}
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

// PathHandle wraps a sketch path chain (the rail a sweep follows) by its index among
// the sketch's detected paths. The sweep resolves it to a 3D rail via the sketch plane.
type PathHandle struct {
	Sketch    *sketch.Sketch
	PathIndex int
}

func (PathHandle) SelectionKind() SelectionKind { return SelectPath }

// SketchEntityHandle wraps a picked sketch entity.
type SketchEntityHandle struct{ Entity sketch.Entity }

func (SketchEntityHandle) SelectionKind() SelectionKind { return SelectSketchEntity }

// SketchDimensionHandle wraps a picked sketch dimension. Dimensions are selected by clicking
// their value label, which is the only part of a dimension with a hit area — the head already
// hit-tests it to open the value editor on a double-click (#2015).
type SketchDimensionHandle struct{ Dim *sketch.DimensionConstraint }

func (SketchDimensionHandle) SelectionKind() SelectionKind { return SelectSketchDimension }

// SketchConstraintHandle wraps a picked geometric constraint. Constraints are selected by clicking
// the marker Show Constraints draws for them, never by clicking the geometry they relate — that
// selects the curve.
type SketchConstraintHandle struct{ Constraint sketch.Constraint }

func (SketchConstraintHandle) SelectionKind() SelectionKind { return SelectSketchConstraint }

// OccurrenceHandle wraps a placed component occurrence (selected in the assembly browser or, once
// viewport occurrence picking lands in #769, by clicking its body). It is the input the assembly
// occurrence operations (ground/suppress/delete, and the replication commands in #765) consume.
type OccurrenceHandle struct{ Occurrence *occurrence.Occurrence }

func (OccurrenceHandle) SelectionKind() SelectionKind { return SelectOccurrence }

// AssemblyFeatureHandle wraps a committed assembly machining feature (selected in the assembly
// browser), the input the edit/suppress/delete actions consume (#766). It wraps the program
// wrapper, not the bare feature, so those actions reach the suppression/name/id state.
type AssemblyFeatureHandle struct{ Feature *compdef.AssemblyFeature }

func (AssemblyFeatureHandle) SelectionKind() SelectionKind { return SelectFeature }

// AssemblyConstraintHandle wraps an assembly relationship (mate/flush/insert/…) selected in
// the assembly browser's Constraints folder, the input the suppress/delete actions consume
// (M12-F01). It wraps the constraint interface so those actions reach its id/suppression.
type AssemblyConstraintHandle struct{ Constraint assembly.Constraint }

func (AssemblyConstraintHandle) SelectionKind() SelectionKind { return SelectConstraint }

// AssemblyJointHandle wraps an assembly joint (rigid/rotational/…) selected in the assembly
// browser's Joints folder, the input the suppress/delete/flip actions consume (M12-F02).
type AssemblyJointHandle struct{ Joint assembly.Joint }

func (AssemblyJointHandle) SelectionKind() SelectionKind { return SelectJoint }

// RepresentationHandle wraps a representation selected in the assembly browser's
// Representations folders (M12-F04): its family and id, so Activate/Delete dispatch to the
// right collection.
type RepresentationHandle struct {
	Family types.RepresentationKind
	ID     uint64
	Name   string
}

func (RepresentationHandle) SelectionKind() SelectionKind { return SelectRepresentation }

// ModelStateHandle wraps a model state selected in the assembly browser's Model States folder
// (M12-F04) — the input the Activate/Delete actions consume.
type ModelStateHandle struct {
	ID   uint64
	Name string
}

func (ModelStateHandle) SelectionKind() SelectionKind { return SelectModelState }

// SketchHandle wraps a whole sketch (selected in the browser), as opposed to one of its
// entities — the input for Edit Sketch and for highlighting the sketch in the view.
type SketchHandle struct{ Sketch *sketch.Sketch }

func (SketchHandle) SelectionKind() SelectionKind { return SelectSketch }

// Sketch3DHandle wraps a 3D sketch (selected from its browser node). A 3D sketch is a sketch
// for selection purposes, so it shares SelectSketch — the concrete handle distinguishes it.
type Sketch3DHandle struct{ Sketch3D *sketch.Sketch3D }

func (Sketch3DHandle) SelectionKind() SelectionKind { return SelectSketch }

// WorkPlaneHandle wraps a picked origin/work plane (selected to host a new sketch).
type WorkPlaneHandle struct{ Plane *feature.WorkPlane }

func (WorkPlaneHandle) SelectionKind() SelectionKind { return SelectWorkPlane }

// WorkPointHandle / WorkAxisHandle wrap a picked datum point / axis — the reference
// inputs for the work-plane constructors that build on points or lines (three points,
// normal to a curve, two lines).
type (
	WorkPointHandle struct{ Point *feature.WorkPoint }
	WorkAxisHandle  struct{ Axis *feature.WorkAxis }
)

func (WorkPointHandle) SelectionKind() SelectionKind { return SelectWorkPoint }
func (WorkAxisHandle) SelectionKind() SelectionKind  { return SelectWorkAxis }

// SelectionFilter restricts which selection kinds a pick accepts — Inventor's
// SelectionFilterEnum. A zero filter (no kinds) accepts everything; a blocking
// filter (every kind deactivated via the Selection Filter window, #1222) accepts
// nothing so a click selects nothing at all.
type SelectionFilter struct {
	kinds    map[SelectionKind]bool
	blockAll bool
}

// NewSelectionFilter accepts only the given kinds; with none it accepts all.
func NewSelectionFilter(kinds ...SelectionKind) *SelectionFilter {
	f := &SelectionFilter{kinds: map[SelectionKind]bool{}}
	for _, k := range kinds {
		f.kinds[k] = true
	}
	return f
}

// newBlockingFilter accepts no kind at all — the "Deselect All" state of the
// Selection Filter window (#1222), distinct from the empty accept-all filter.
func newBlockingFilter() *SelectionFilter { return &SelectionFilter{blockAll: true} }

// Accepts reports whether the filter admits a kind (an empty filter admits all,
// a blocking filter admits none).
func (f *SelectionFilter) Accepts(k SelectionKind) bool {
	if f == nil {
		return true
	}
	if f.blockAll {
		return false
	}
	if len(f.kinds) == 0 {
		return true
	}
	return f.kinds[k]
}

// IsRestricted reports whether the filter limits the accepted kinds (i.e. it is not the
// accept-everything default) — so a caller can tell an explicitly-set filter from the default.
func (f *SelectionFilter) IsRestricted() bool {
	return f != nil && (f.blockAll || len(f.kinds) > 0)
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

// Add appends a selectable if the filter accepts its kind and it is not already in the
// set, reporting whether the set changed. De-duplication honours the SelectSet contract
// (the docstring above) — re-picking the same entity must not create a duplicate.
func (s *Selection) Add(sel Selectable) bool {
	if !s.filter.Accepts(sel.SelectionKind()) || s.Contains(sel) {
		return false
	}
	s.items = append(s.items, sel)
	return true
}

// Contains reports whether sel is already in the set. Selectable handles are comparable
// value structs wrapping the underlying entity pointer/id, so identity is plain equality.
func (s *Selection) Contains(sel Selectable) bool {
	for _, it := range s.items {
		if it == sel {
			return true
		}
	}
	return false
}

// Remove drops sel from the set, reporting whether it was present.
func (s *Selection) Remove(sel Selectable) bool {
	for i, it := range s.items {
		if it == sel {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return true
		}
	}
	return false
}

// Toggle flips sel's membership — removing it when already selected, otherwise adding it
// (subject to the filter). This is Inventor's Shift/Ctrl+click behaviour (GUID-B8F6E805):
// clicking an already-selected object removes just that object, leaving the rest. Returns
// whether the set changed.
func (s *Selection) Toggle(sel Selectable) bool {
	if s.Remove(sel) {
		return true
	}
	return s.Add(sel)
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

// References returns, parallel to Items, the work-feature reference for each selected
// entity — a datum plane/axis/point key, or a face/vertex/body reference — the string an
// automation client passes to a work-plane constructor or a body-scoped operation. Entities
// with no such reference (a sketch entity) yield an empty string, so the slice stays
// index-aligned. A whole-body pick reports a body/<key> reference (#1492).
func (s *Selection) References() []string {
	refs := make([]string, len(s.items))
	for i, it := range s.items {
		refs[i] = string(selectionRef(it))
	}
	return refs
}

// selectionRef maps a selectable to its work-feature reference (empty when it has none).
func selectionRef(it Selectable) feature.WorkRef {
	switch h := it.(type) {
	case WorkPlaneHandle:
		return h.Plane.Key()
	case WorkAxisHandle:
		return h.Axis.Key()
	case WorkPointHandle:
		return h.Point.Key()
	case FaceHandle:
		return feature.FaceRef(h.Face.ReferenceKey())
	case VertexHandle:
		return feature.VertexRef(h.Vertex.ReferenceKey())
	case BodyHandle:
		return feature.BodyRef(h.Body.ReferenceKey())
	default:
		return ""
	}
}
