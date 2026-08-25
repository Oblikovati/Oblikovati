// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// ProjectGeometryTool is the interactive 2D "Project Geometry" command: while editing a sketch,
// click a part's faces, edges or vertices — or its datum geometry (the origin centre point,
// origin/work axes and planes, in the viewport or the browser tree) — and each click projects that
// reference onto the sketch plane IMMEDIATELY as associative reference geometry (it re-derives
// through recompute via its reference key, and other sketch geometry can be constrained to it). A
// face projects its whole boundary; a work axis/plane becomes a reference line (the axis, or the
// plane↔sketch intersection); a work point or vertex becomes a reference point.
//
// There is no dialog and no OK step (Inventor's Project Geometry): the tool stays armed after each
// pick so the user keeps clicking geometry to project, and Escape or Enter finishes. Each
// projection is its own undo step (#1262, faces #2158).
type ProjectGeometryTool struct {
	dialogTool
	projected bool // at least one reference projected this session, so Enter can finish
}

// NewProjectGeometryTool returns a project-geometry tool.
func NewProjectGeometryTool() *ProjectGeometryTool { return &ProjectGeometryTool{} }

// Name implements [Tool].
func (t *ProjectGeometryTool) Name() string { return "Project Geometry" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *ProjectGeometryTool) Start(*Session) {}

// PicksModelReferences routes this tool's in-sketch viewport clicks to the 3D model hit-test
// (B-rep faces/edges/vertices and datum geometry) instead of the 2D sketch-entity picker. Without
// it the input router short-circuits every in-sketch click to the sketch's own 2D entities, so the
// references to project were unreachable from the viewport and the tool silently did nothing
// (#1496). The browser tree fed picks through SelectBrowserNode, but the 3D view did not.
func (t *ProjectGeometryTool) PicksModelReferences() bool { return true }

// AcceptedKinds declares project-geometry picks projectable references: B-rep faces, edges and
// vertices, plus the part's datum geometry (work points, axes and planes — the Origin folder and
// user work features). A face projects all of its bounding edges (Inventor's behaviour) — the piece
// that made hovering a planar/circular face do nothing at all (#2158).
func (t *ProjectGeometryTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectFace, SelectEdge, SelectVertex, SelectWorkPoint, SelectWorkAxis, SelectWorkPlane}
}

// Pick projects the clicked reference onto the active 2D sketch right away and leaves the tool armed
// for the next pick — Project Geometry has no dialog and no separate OK. Each projection records its
// own undo step. Kinds the tool does not handle, or a pick with no active sketch/part, are ignored.
func (t *ProjectGeometryTool) Pick(s *Session, sel Selectable) {
	sk := s.ActiveSketch()
	if sk == nil {
		return
	}
	part, err := activePart(s)
	if err != nil {
		return
	}
	if projectReference(sk, part, sel) {
		t.projected = true
		s.RecordActiveEdit(t.Name())
	}
}

// projectReference projects one picked reference onto sk, reporting whether it projected anything.
func projectReference(sk *sketch.Sketch, part *compdef.PartComponentDefinition, sel Selectable) bool {
	switch h := sel.(type) {
	case FaceHandle:
		projectFaceEdges(sk, part, h.Face)
	case EdgeHandle:
		sk.ProjectCurve(compdef.NewEdgeRefSource(part, string(h.Edge.ReferenceKey())))
	case VertexHandle:
		sk.ProjectPoint(compdef.NewVertexRefSource(part, string(h.Vertex.ReferenceKey())))
	case WorkPointHandle:
		sk.ProjectPoint(compdef.NewWorkPointRefSource(part, h.Point.Key()))
	case WorkAxisHandle:
		sk.ProjectCurve(compdef.NewWorkAxisRefSource(part, h.Axis.Key()))
	case WorkPlaneHandle:
		if !part.WorkPlaneIntersectsSketch(h.Plane.Key(), sk.Plane()) {
			return false // a plane parallel to the sketch has no intersection line to project
		}
		sk.ProjectCurve(compdef.NewWorkPlaneRefSource(part, h.Plane.Key(), sk.Plane()))
	default:
		return false
	}
	return true
}

// projectFaceEdges projects every bounding edge of a picked face onto sk — Inventor projects a face
// as its whole boundary (e.g. a cylinder's planar end face yields its circular perimeter, #2158).
// Each edge re-derives associatively through its reference key, exactly as a directly picked edge
// does; Face.Edges already returns the distinct edges, so no dedup is needed here.
func projectFaceEdges(sk *sketch.Sketch, part *compdef.PartComponentDefinition, face *topo.Face) {
	for _, e := range face.Edges() {
		sk.ProjectCurve(compdef.NewEdgeRefSource(part, string(e.ReferenceKey())))
	}
}

// CanCommit reports whether Enter can finish the tool — true once at least one reference has been
// projected (the projection itself already happened on each pick, not here).
func (t *ProjectGeometryTool) CanCommit() bool { return t.projected }

// Commit finishes the tool: the projections already happened on each pick, so it is a no-op tear-
// down once anything was projected. It refuses (errors) when nothing was projected, so Enter with an
// empty tool does not silently "succeed" — the same guard the other reference tools hold.
func (t *ProjectGeometryTool) Commit(*Session) error {
	if !t.projected {
		return errors.New("project geometry: pick a face, edge, vertex or datum reference to project")
	}
	return nil
}

var _ Tool = (*ProjectGeometryTool)(nil)
