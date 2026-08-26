// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// Interactive NURBS control-point editing (M36-F03) — the Class-A styling gesture: while the Edit
// Control Points tool is active the running surface body's control net is drawn as draggable
// handles, and a left-drag on a handle slides it in the camera-facing plane, re-evaluating the
// limit surface live (and its reflections, once F12 lands). The dialog options choose the region
// (single CV, row, column), a soft falloff radius, and optional X-symmetry. Each drag is recorded
// as one [feature.ControlPointEditFeature], so undo reverts a single edit. Mirrors the built-in
// point-cloud move drag (cloud_move_tool.go).

const (
	cvModeSingle = iota
	cvModeRow
	cvModeColumn
)

// cvPickPixels is the handle hit radius (a CV within this many pixels of the cursor ray grabs).
const cvPickPixels = 12

// cvHandlePixels is the on-screen size of a control-point handle marker.
const cvHandlePixels = 7

// cvHandleColor / cvNetColor style the control net overlay (orange handles, dim grid).
var (
	cvHandleColor = [4]float32{1, 0.55, 0, 1}
	cvNetColor    = [4]float32{1, 0.55, 0, 0.5}
)

// ControlPointEditTool arms interactive control-net editing of the running surface body. It is
// drag-driven (no clicks/OK of its own); its presence lets the viewport route a left-drag to
// moving a control point, and it draws the net handles via [ControlPointEditTool.Preview].
type ControlPointEditTool struct {
	dialogTool
	radius   float64 // falloff radius in model units (0 = move only the picked region, rigidly)
	falloff  int     // geom.Falloff
	mode     int     // cvModeSingle | cvModeRow | cvModeColumn
	symmetry bool    // also drive the X-mirrored control points with the mirrored move
}

// NewControlPointEditTool returns the CV-edit tool defaulting to single-CV, smooth falloff.
func NewControlPointEditTool() *ControlPointEditTool {
	return &ControlPointEditTool{falloff: int(geom.FalloffSmooth)}
}

// Name implements [Tool].
func (t *ControlPointEditTool) Name() string { return "Edit Control Points" }

// Prompt guides the input.
func (t *ControlPointEditTool) Prompt(*Session) string {
	return "Drag a control-point handle to shape the surface; set region/falloff/symmetry options."
}

// CanCommit is false — the tool commits per drag, not via OK.
func (t *ControlPointEditTool) CanCommit() bool { return false }

// Commit is a no-op (drag-driven).
func (t *ControlPointEditTool) Commit(*Session) error { return nil }

// Params exposes the region, falloff and symmetry options for the generic dialog.
func (t *ControlPointEditTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{{Label: "Falloff Radius", Get: func() float64 { return t.radius }, Set: func(v float64) { t.radius = v }}},
		Bools:  []BoolParam{{Label: "Symmetry (X)", Get: func() bool { return t.symmetry }, Set: func(v bool) { t.symmetry = v }}},
		Choices: []ChoiceParam{
			{Label: "Region", Options: []string{"Single", "Row", "Column"}, Get: func() int { return t.mode }, Set: func(v int) { t.mode = v }},
			{Label: "Falloff", Options: []string{"Constant", "Linear", "Smooth"}, Get: func() int { return t.falloff }, Set: func(v int) { t.falloff = v }},
		},
	}
}

// Preview draws the editable control net (handles + grid) over the running surface body.
func (t *ControlPointEditTool) Preview(s *Session) []renderer.DrawItem {
	surf, ok := activeEditableSurface(s)
	if !ok {
		return nil
	}
	return controlNetItems(surf, s.Camera())
}

// cvEditDrag is one in-flight control-point drag.
type cvEditDrag struct {
	feature  *feature.PartFeature // the edit feature, created lazily on first real move (undo per drag)
	cvU, cvV int                  // the grabbed control-point grid index
	origin   math.Point3          // the drag plane point (the grabbed CV at start)
	normal   math.Vector3         // the drag plane normal (camera forward at start)
	from     math.Point3          // the world point under the cursor at drag start
	surf     geom.BSplineSurface  // the face surface at drag start (drivers/positions are read from it)
	active   bool
}

// CVEditActive reports whether the Edit Control Points tool is the active tool.
func (s *Session) CVEditActive() bool {
	if s.tool == nil {
		return false
	}
	_, ok := s.tool.tool.(*ControlPointEditTool)
	return ok
}

// CVDragActive reports whether a control-point drag is in progress.
func (s *Session) CVDragActive() bool { return s.cvEdit.active }

// BeginCVDrag starts dragging the control-point handle under the cursor pixel. It returns false
// (no drag) when the tool is inactive, the running body has no NURBS face, or no handle is near.
func (s *Session) BeginCVDrag(px, py float64) bool {
	if !s.CVEditActive() {
		return false
	}
	surf, ok := activeEditableSurface(s)
	if !ok {
		return false
	}
	cam := s.Camera()
	o, d := cam.RayThrough(px, py)
	i, j, ok := nearestControlPoint(surf, o, d, cvPickPixels*cam.WorldPerPixel())
	if !ok {
		return false
	}
	from, ok := rayPlane(o, d, surf.Ctrl[i][j], cam.Forward())
	if !ok {
		return false
	}
	s.cvEdit = cvEditDrag{cvU: i, cvV: j, origin: surf.Ctrl[i][j], normal: cam.Forward(), from: from, surf: surf, active: true}
	return true
}

// UpdateCVDrag slides the grabbed control point (and its region/symmetry partners) so the handle
// tracks the cursor, re-evaluating the surface live. The edit feature is created on the first real
// move so a click without a drag leaves no feature behind.
func (s *Session) UpdateCVDrag(px, py float64) {
	if !s.cvEdit.active {
		return
	}
	cam := s.Camera()
	o, d := cam.RayThrough(px, py)
	to, ok := rayPlane(o, d, s.cvEdit.origin, s.cvEdit.normal)
	if !ok {
		return
	}
	s.applyCVMove(s.cvEdit.from.VectorTo(to))
}

// applyCVMove drives the in-progress edit by a world-space move: it creates the edit feature on
// the first real move, sets its displacements (region/falloff/symmetry), and recomputes live. It
// is the camera-independent core of [Session.UpdateCVDrag].
func (s *Session) applyCVMove(move math.Vector3) {
	part, err := activePart(s)
	if err != nil {
		return
	}
	if s.cvEdit.feature == nil {
		if move.Length() < 1e-9 {
			return
		}
		s.cvEdit.feature = feature.NewControlPointEditFeatures(part.Features()).Add(nil)
	}
	s.cvEdit.feature.Definition().(*feature.ControlPointEditFeature).Definition().Deltas = s.cvEditDeltas(move)
	part.Recompute()
}

// CommitCVDrag ends the drag, recording the edit as one undo step (nothing when no move happened).
func (s *Session) CommitCVDrag() {
	if s.cvEdit.feature != nil {
		if part, err := activePart(s); err == nil {
			s.recordEdit(part, "Edit Control Points")
		}
	}
	s.cvEdit = cvEditDrag{}
}

// cvEditDeltas computes the per-control-point displacements for the current drag move, applying
// the tool's region (drivers), falloff and optional X-symmetry against the drag-start surface.
func (s *Session) cvEditDeltas(move math.Vector3) []geom.ControlPointDelta {
	t := s.tool.tool.(*ControlPointEditTool)
	surf := s.cvEdit.surf
	drivers := editDrivers(surf, s.cvEdit.cvU, s.cvEdit.cvV, t.mode)
	deltas := surf.FalloffDeltas(drivers, move, t.radius, geom.Falloff(t.falloff))
	if t.symmetry {
		if md := mirrorDrivers(drivers, len(surf.Ctrl)); !sameDrivers(drivers, md) {
			deltas = append(deltas, surf.FalloffDeltas(md, mirrorMove(move), t.radius, geom.Falloff(t.falloff))...)
		}
	}
	return deltas
}

// activeEditableSurface returns the first NURBS face surface of the active part's most recent
// surface body — the net the CV-edit tool shapes.
func activeEditableSurface(s *Session) (geom.BSplineSurface, bool) {
	part, err := activePart(s)
	if err != nil {
		return geom.BSplineSurface{}, false
	}
	bodies := part.SurfaceBodies()
	if bodies.Count() == 0 {
		return geom.BSplineSurface{}, false
	}
	for _, f := range bodies.Item(bodies.Count() - 1).Faces() {
		if bs, ok := f.Geometry().(geom.BSplineSurface); ok {
			return bs, true
		}
	}
	return geom.BSplineSurface{}, false
}

// nearestControlPoint returns the control index whose point is closest to the ray, within tol.
func nearestControlPoint(surf geom.BSplineSurface, o math.Point3, d math.Vector3, tol float64) (int, int, bool) {
	best, bi, bj := tol, -1, -1
	for i := range surf.Ctrl {
		for j := range surf.Ctrl[i] {
			if dist := float64(o.VectorTo(surf.Ctrl[i][j]).Cross(d).Length()); dist <= best {
				best, bi, bj = dist, i, j
			}
		}
	}
	return bi, bj, bi >= 0
}

// editDrivers returns the control indices a drag drives: the single CV, its whole V-row, or its
// whole U-column, per the tool mode.
func editDrivers(surf geom.BSplineSurface, i, j, mode int) [][2]int {
	switch mode {
	case cvModeRow:
		out := make([][2]int, len(surf.Ctrl[i]))
		for v := range surf.Ctrl[i] {
			out[v] = [2]int{i, v}
		}
		return out
	case cvModeColumn:
		out := make([][2]int, len(surf.Ctrl))
		for u := range surf.Ctrl {
			out[u] = [2]int{u, j}
		}
		return out
	default:
		return [][2]int{{i, j}}
	}
}

// mirrorDrivers reflects driver indices across the net's U-midline (u → nu−1−u).
func mirrorDrivers(drivers [][2]int, nu int) [][2]int {
	out := make([][2]int, len(drivers))
	for k, d := range drivers {
		out[k] = [2]int{nu - 1 - d[0], d[1]}
	}
	return out
}

// mirrorMove flips a move across the world X axis (the symmetry plane).
func mirrorMove(move math.Vector3) math.Vector3 { return math.V3(-move.X, move.Y, move.Z) }

// sameDrivers reports whether two driver sets are identical (so symmetry on a self-symmetric
// selection does not double-apply).
func sameDrivers(a, b [][2]int) bool {
	for k := range a {
		if a[k] != b[k] {
			return false
		}
	}
	return true
}

// controlNetItems builds the handle markers and the connecting grid lines for a control net.
func controlNetItems(surf geom.BSplineSurface, cam scene.Camera) []renderer.DrawItem {
	var pts []math.Point3
	for i := range surf.Ctrl {
		pts = append(pts, surf.Ctrl[i]...)
	}
	items := []renderer.DrawItem{controlNetLines(surf)}
	if m := renderer.PointMarkers(pts, cvHandlePixels*cam.WorldPerPixel(), cvHandleColor, 0); m != nil {
		items = append(items, *m)
	}
	return items
}

// controlNetLines builds the line item connecting each control point to its U/V grid neighbours.
func controlNetLines(surf geom.BSplineSurface) renderer.DrawItem {
	nu, nv := len(surf.Ctrl), len(surf.Ctrl[0])
	pos := make([]math.Point3, 0, nu*nv)
	for i := range surf.Ctrl {
		pos = append(pos, surf.Ctrl[i]...)
	}
	idx := func(i, j int) int { return i*nv + j }
	var indices []int
	for i := range nu {
		for j := range nv {
			if i+1 < nu {
				indices = append(indices, idx(i, j), idx(i+1, j))
			}
			if j+1 < nv {
				indices = append(indices, idx(i, j), idx(i, j+1))
			}
		}
	}
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: pos, Indices: indices, Color: cvNetColor}
}
