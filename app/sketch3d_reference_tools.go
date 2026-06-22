// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// IncludeGeometry3DTool is the interactive 3D-sketch "Include Geometry" command: while
// editing a 3D sketch, click part edges/vertices to include them as associative reference
// geometry (an edge as a reference polyline, a vertex as a constrainable anchor point).
// They re-derive through recompute via their reference keys.
type IncludeGeometry3DTool struct {
	dialogTool
	edges    []EdgeHandle
	vertices []VertexHandle
}

// NewIncludeGeometry3DTool returns a 3D include-geometry tool.
func NewIncludeGeometry3DTool() *IncludeGeometry3DTool { return &IncludeGeometry3DTool{} }

// Name implements [Tool].
func (t *IncludeGeometry3DTool) Name() string { return "Include Geometry" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *IncludeGeometry3DTool) Start(*Session) {}

// AcceptedKinds declares include-geometry picks includable references: edges and vertices.
func (t *IncludeGeometry3DTool) AcceptedKinds() []SelectionKind {
	return []SelectionKind{SelectEdge, SelectVertex}
}

// Picks reports the picked edges and vertices for the unified highlight.
func (t *IncludeGeometry3DTool) Picks() []Selectable {
	picks := edgeSelectables(t.edges)
	for _, v := range t.vertices {
		picks = append(picks, v)
	}
	return picks
}

// Pick records a clicked edge or vertex (ignoring other kinds).
func (t *IncludeGeometry3DTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case EdgeHandle:
		t.edges = append(t.edges, h)
	case VertexHandle:
		t.vertices = append(t.vertices, h)
	}
}

// CanCommit reports whether at least one reference is picked.
func (t *IncludeGeometry3DTool) CanCommit() bool { return len(t.edges)+len(t.vertices) > 0 }

// Commit includes each picked edge/vertex into the active 3D sketch as associative
// reference geometry.
func (t *IncludeGeometry3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("include geometry: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("include geometry: pick at least one edge or vertex")
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	for _, e := range t.edges {
		sk.IncludeCurve3D(compdef.NewEdgeRefSource(part, string(e.Edge.ReferenceKey())))
	}
	for _, v := range t.vertices {
		sk.IncludePoint3D(compdef.NewVertexRefSource(part, string(v.Vertex.ReferenceKey())))
	}
	return nil
}

// Cancel implements [Tool].

// SurfaceCurve3DTool is the interactive 3D-sketch surface-derived-curve command: while
// editing a 3D sketch, pick faces and commit an intersection curve (two faces) or a
// silhouette curve (one face, for the current view direction) onto the sketch. The curve is
// associative — it re-evaluates against the rebuilt B-rep on recompute via the face keys.
type SurfaceCurve3DTool struct {
	dialogTool
	faces      []FaceHandle
	silhouette bool
	viewDir    math.Vector3
}

// NewSurfaceCurve3DTool returns a surface-curve tool defaulting to an intersection of two
// faces, with a +Z silhouette view direction if switched to silhouette mode.
func NewSurfaceCurve3DTool() *SurfaceCurve3DTool {
	return &SurfaceCurve3DTool{viewDir: math.V3(0, 0, 1)}
}

// Name implements [Tool].
func (t *SurfaceCurve3DTool) Name() string { return "Surface Curve" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *SurfaceCurve3DTool) Start(*Session) {}

// AcceptedKinds declares surface-curve picks faces (the surfaces to intersect).
func (t *SurfaceCurve3DTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports the picked faces for the unified highlight.
func (t *SurfaceCurve3DTool) Picks() []Selectable { return faceSelectables(t.faces) }

// Pick records a clicked face.
func (t *SurfaceCurve3DTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok {
		t.faces = append(t.faces, f)
	}
}

// SetSilhouette switches between intersection (false) and silhouette (true) mode;
// SetViewDir sets the silhouette view direction.
func (t *SurfaceCurve3DTool) SetSilhouette(on bool)     { t.silhouette = on }
func (t *SurfaceCurve3DTool) SetViewDir(v math.Vector3) { t.viewDir = v }

// CanCommit reports whether the picked face count matches the mode (1 silhouette, 2
// intersection).
func (t *SurfaceCurve3DTool) CanCommit() bool {
	if t.silhouette {
		return len(t.faces) == 1
	}
	return len(t.faces) == 2
}

// Commit adds the surface-derived curve to the active 3D sketch.
func (t *SurfaceCurve3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("surface curve: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("surface curve: pick 2 faces (intersection) or 1 face (silhouette)")
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if t.silhouette {
		src := compdef.NewFaceRefSource(part, string(t.faces[0].Face.ReferenceKey()))
		sk.AddSilhouetteCurve3DRef(src, t.viewDir, geom.SurfaceGrid{})
	} else {
		a := compdef.NewFaceRefSource(part, string(t.faces[0].Face.ReferenceKey()))
		b := compdef.NewFaceRefSource(part, string(t.faces[1].Face.ReferenceKey()))
		sk.AddIntersectionCurve3DRef(a, b, geom.SurfaceGrid{})
	}
	return nil
}

// Cancel implements [Tool].

var (
	_ Tool = (*IncludeGeometry3DTool)(nil)
	_ Tool = (*SurfaceCurve3DTool)(nil)
)
