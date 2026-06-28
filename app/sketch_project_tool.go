// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// ProjectGeometryTool is the interactive 2D "Project Geometry" command: while editing a
// sketch, click part edges/vertices — or the part's datum geometry (the origin centre point,
// origin/work axes and planes, in the viewport or the browser tree) — to project them onto the
// sketch plane as associative reference geometry (they re-derive through recompute via their
// reference keys, and other sketch geometry can be constrained to the projected anchors). A
// projected work axis/plane becomes a reference line (the axis, or the plane↔sketch
// intersection); a work point becomes a reference point. Commit projects each onto the active
// 2D sketch (#1262).
type ProjectGeometryTool struct {
	dialogTool
	edges      []EdgeHandle
	vertices   []VertexHandle
	workPoints []WorkPointHandle
	workAxes   []WorkAxisHandle
	workPlanes []WorkPlaneHandle
}

// NewProjectGeometryTool returns a project-geometry tool.
func NewProjectGeometryTool() *ProjectGeometryTool { return &ProjectGeometryTool{} }

// Name implements [Tool].
func (t *ProjectGeometryTool) Name() string { return "Project Geometry" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *ProjectGeometryTool) Start(*Session) {}

// PicksModelReferences routes this tool's in-sketch viewport clicks to the 3D model hit-test
// (B-rep edges/vertices and datum geometry) instead of the 2D sketch-entity picker. Without it
// the input router short-circuits every in-sketch click to the sketch's own 2D entities, so the
// references to project were unreachable from the viewport and the tool silently did nothing
// (#1496). The browser tree fed picks through SelectBrowserNode, but the 3D view did not.
func (t *ProjectGeometryTool) PicksModelReferences() bool { return true }

// AcceptedKinds declares project-geometry picks projectable references: B-rep edges and
// vertices, plus the part's datum geometry (work points, axes and planes — the Origin folder
// and user work features).
func (t *ProjectGeometryTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectEdge, SelectVertex, SelectWorkPoint, SelectWorkAxis, SelectWorkPlane}
}

// Picks reports every picked reference for the unified highlight.
func (t *ProjectGeometryTool) Picks() []Selectable {
	picks := append(edgeSelectables(t.edges), vertexSelectables(t.vertices)...)
	for _, h := range t.workPoints {
		picks = append(picks, h)
	}
	for _, h := range t.workAxes {
		picks = append(picks, h)
	}
	for _, h := range t.workPlanes {
		picks = append(picks, h)
	}
	return picks
}

// Pick records a clicked edge, vertex or datum reference (ignoring other kinds).
func (t *ProjectGeometryTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case EdgeHandle:
		t.edges = append(t.edges, h)
	case VertexHandle:
		t.vertices = append(t.vertices, h)
	case WorkPointHandle:
		t.workPoints = append(t.workPoints, h)
	case WorkAxisHandle:
		t.workAxes = append(t.workAxes, h)
	case WorkPlaneHandle:
		t.workPlanes = append(t.workPlanes, h)
	}
}

// CanCommit reports whether at least one reference is picked.
func (t *ProjectGeometryTool) CanCommit() bool {
	return len(t.edges)+len(t.vertices)+len(t.workPoints)+len(t.workAxes)+len(t.workPlanes) > 0
}

// Commit projects each picked edge/vertex onto the active 2D sketch as associative
// reference geometry.
func (t *ProjectGeometryTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("project geometry: not editing a 2D sketch")
	}
	if !t.CanCommit() {
		return errors.New("project geometry: pick at least one edge, vertex or datum reference")
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	for _, e := range t.edges {
		sk.ProjectCurve(compdef.NewEdgeRefSource(part, string(e.Edge.ReferenceKey())))
	}
	for _, v := range t.vertices {
		sk.ProjectPoint(compdef.NewVertexRefSource(part, string(v.Vertex.ReferenceKey())))
	}
	t.projectDatums(sk, part)
	return nil
}

// projectDatums projects the picked datum geometry onto sk: the origin/work points as
// reference points, the axes and the (non-parallel) planes as reference lines (#1262).
func (t *ProjectGeometryTool) projectDatums(sk *sketch.Sketch, part *compdef.PartComponentDefinition) {
	for _, p := range t.workPoints {
		sk.ProjectPoint(compdef.NewWorkPointRefSource(part, p.Point.Key()))
	}
	for _, a := range t.workAxes {
		sk.ProjectCurve(compdef.NewWorkAxisRefSource(part, a.Axis.Key()))
	}
	for _, p := range t.workPlanes {
		if !part.WorkPlaneIntersectsSketch(p.Plane.Key(), sk.Plane()) {
			continue // a plane parallel to the sketch has no intersection line to project
		}
		sk.ProjectCurve(compdef.NewWorkPlaneRefSource(part, p.Plane.Key(), sk.Plane()))
	}
}

// Cancel implements [Tool] (no model change to roll back before commit).

var _ Tool = (*ProjectGeometryTool)(nil)
