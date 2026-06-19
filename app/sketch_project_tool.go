// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/compdef"
)

// ProjectGeometryTool is the interactive 2D "Project Geometry" command: while editing a
// sketch, click part edges/vertices to project them onto the sketch plane as associative
// reference geometry (they re-derive through recompute via their reference keys, and other
// sketch geometry can be constrained to the projected anchors). Picks arrive as edge/vertex
// handles; Commit projects each onto the active 2D sketch.
type ProjectGeometryTool struct {
	dialogTool
	edges    []EdgeHandle
	vertices []VertexHandle
}

// NewProjectGeometryTool returns a project-geometry tool.
func NewProjectGeometryTool() *ProjectGeometryTool { return &ProjectGeometryTool{} }

// Name implements [Tool].
func (t *ProjectGeometryTool) Name() string { return "Project Geometry" }

// Start restricts the selection to projectable references (edges and vertices).
func (t *ProjectGeometryTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectEdge, SelectVertex))
}

// Pick records a clicked edge or vertex (ignoring other kinds).
func (t *ProjectGeometryTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case EdgeHandle:
		t.edges = append(t.edges, h)
	case VertexHandle:
		t.vertices = append(t.vertices, h)
	}
}

// CanCommit reports whether at least one reference is picked.
func (t *ProjectGeometryTool) CanCommit() bool { return len(t.edges)+len(t.vertices) > 0 }

// Commit projects each picked edge/vertex onto the active 2D sketch as associative
// reference geometry.
func (t *ProjectGeometryTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("project geometry: not editing a 2D sketch")
	}
	if !t.CanCommit() {
		return errors.New("project geometry: pick at least one edge or vertex")
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
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel implements [Tool] (no model change to roll back before commit).

var _ Tool = (*ProjectGeometryTool)(nil)
