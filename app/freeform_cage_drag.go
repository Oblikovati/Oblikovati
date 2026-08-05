// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// The free-form cage drag: grab a cage vertex handle, slide it in the camera-facing plane, and
// the body re-subdivides live. One drag is one undo step (#2048).

// cageEditDrag is one in-flight cage-vertex drag.
type cageEditDrag struct {
	vertex  int                   // the grabbed cage vertex index
	feature *feature.PartFeature  // the free-form feature owning the cage (marked dirty per move)
	body    *feature.FreeformBody // the body being edited
	origin  math.Point3           // the drag plane point (the grabbed vertex at start)
	normal  math.Vector3          // the drag plane normal (camera forward at start)
	from    math.Point3           // the world point under the cursor at drag start
	moved   bool                  // a real move happened, so the drag is worth an undo step
	active  bool
}

// CageEditActive reports whether the Edit Freeform Cage tool is the active tool.
func (s *Session) CageEditActive() bool {
	if s.tool == nil {
		return false
	}
	_, ok := s.tool.tool.(*FreeformCageEditTool)
	return ok
}

// CageDragActive reports whether a cage-vertex drag is in progress.
func (s *Session) CageDragActive() bool { return s.cageEdit.active }

// BeginCageDrag starts dragging the cage handle under the cursor pixel. It returns false (no
// drag) when the tool is inactive, the part has no free-form body, or no handle is near.
func (s *Session) BeginCageDrag(px, py float64) bool {
	if !s.CageEditActive() {
		return false
	}
	pf, body, ok := activeFreeformCage(s)
	if !ok {
		return false
	}
	cam := s.Camera()
	o, d := cam.RayThrough(px, py)
	i, ok := nearestCageVertex(body, o, d, cagePickPixels*cam.WorldPerPixel())
	if !ok {
		return false
	}
	at := body.Vertices().Item(i).Point()
	from, ok := rayPlane(o, d, at, cam.Forward())
	if !ok {
		return false
	}
	s.cageEdit = cageEditDrag{vertex: i, feature: pf, body: body, origin: at, normal: cam.Forward(), from: from, active: true}
	if t, ok := s.tool.tool.(*FreeformCageEditTool); ok {
		t.lastVertex = i // the Crease action applies to the handle the user just grabbed
	}
	return true
}

// UpdateCageDrag slides the grabbed cage vertex so the handle tracks the cursor, re-subdividing
// live. The move is applied incrementally, so `from` advances with each step.
func (s *Session) UpdateCageDrag(px, py float64) {
	if !s.cageEdit.active {
		return
	}
	cam := s.Camera()
	o, d := cam.RayThrough(px, py)
	to, ok := rayPlane(o, d, s.cageEdit.origin, s.cageEdit.normal)
	if !ok {
		return
	}
	s.applyCageMove(s.cageEdit.from.VectorTo(to))
}

// applyCageMove translates the grabbed vertex by move and recomputes. It is the
// camera-independent core of [Session.UpdateCageDrag].
func (s *Session) applyCageMove(move math.Vector3) {
	if !s.cageEdit.active || float64(move.Length()) < 1e-12 {
		return
	}
	part, err := activePart(s)
	if err != nil {
		return
	}
	s.cageEdit.body.MoveVertices([]int{s.cageEdit.vertex}, move)
	s.cageEdit.from = s.cageEdit.from.TranslateBy(move)
	s.cageEdit.moved = true
	// The cage lives inside the feature's definition, so the engine cannot see the mutation on
	// its own — without MarkDirty the recompute replays the cached body and the drag is invisible.
	part.Features().MarkDirty(s.cageEdit.feature)
	part.Recompute()
}

// CommitCageDrag ends the drag, recording it as one undo step (nothing when no move happened).
func (s *Session) CommitCageDrag() {
	if s.cageEdit.moved {
		if part, err := activePart(s); err == nil {
			s.recordEdit(part, "Edit Freeform Cage")
		}
	}
	s.cageEdit = cageEditDrag{}
}

// ApplyCageLevel sets the running free-form body's subdivision level and recomputes, recording
// one undo step. It reports false when there is no body to edit.
func (s *Session) ApplyCageLevel(level int) bool {
	pf, body, ok := activeFreeformCage(s)
	if !ok {
		return false
	}
	body.SetLevel(level)
	return s.CommitFeatureEdit(pf) == nil
}

// CreaseCageEdgesAround sets the crease sharpness on every cage edge meeting the given vertex —
// the crease gesture in terms of the handle the user just dragged. It reports false when there
// is no body to edit.
func (s *Session) CreaseCageEdgesAround(vertex int, sharpness float64) bool {
	pf, body, ok := activeFreeformCage(s)
	if !ok {
		return false
	}
	edges := cageEdgesAt(body, vertex)
	if len(edges) == 0 {
		return false
	}
	body.CreaseEdges(edges, math.Clamp(sharpness, 0, 1))
	return s.CommitFeatureEdit(pf) == nil
}

// cageEdgesAt lists the cage edges incident to a vertex.
func cageEdgesAt(body *feature.FreeformBody, vertex int) [][2]int {
	var out [][2]int
	edges := body.Edges()
	for i := 0; i < edges.Count(); i++ {
		if a, b := edges.Item(i).Ends(); a == vertex || b == vertex {
			out = append(out, [2]int{a, b})
		}
	}
	return out
}

// nearestCageVertex returns the cage vertex whose handle the ray passes nearest within tol.
func nearestCageVertex(body *feature.FreeformBody, o math.Point3, d math.Vector3, tol float64) (int, bool) {
	best, found := stdmath.Inf(1), -1
	verts := body.Vertices()
	for i := 0; i < verts.Count(); i++ {
		dist := rayPointDistanceOnly(o, d, verts.Item(i).Point())
		if dist <= tol && dist < best {
			best, found = dist, i
		}
	}
	return found, found >= 0
}

// rayPointDistanceOnly is the perpendicular distance from the forward ray to a point (+Inf
// behind the origin), the hit test the handle picking needs.
func rayPointDistanceOnly(o math.Point3, d math.Vector3, p math.Point3) float64 {
	t := float64(o.VectorTo(p).Dot(d))
	if t <= 0 {
		return stdmath.Inf(1)
	}
	return float64(o.TranslateBy(d.Scale(math.Scalar(t))).DistanceTo(p))
}

// cageOverlayItems draws the cage: a wire for every edge plus a screen-constant cross at every
// vertex, so the handles stay grabbable at any zoom.
func cageOverlayItems(body *feature.FreeformBody, handle float64) []renderer.DrawItem {
	var pos []math.Point3
	var idx []int
	edges := body.Edges()
	verts := body.Vertices()
	for i := 0; i < edges.Count(); i++ {
		a, b := edges.Item(i).Ends()
		idx = append(idx, len(pos), len(pos)+1)
		pos = append(pos, verts.Item(a).Point(), verts.Item(b).Point())
	}
	wire := renderer.DrawItem{Primitive: renderer.Lines, Positions: pos, Indices: idx, Color: cageWireColor, Opacity: 1}
	return []renderer.DrawItem{wire, cageHandleItems(verts, handle)}
}

// cageHandleItems draws each cage vertex as a small axis-aligned cross.
func cageHandleItems(verts *feature.FreeformVertices, h float64) renderer.DrawItem {
	var pos []math.Point3
	var idx []int
	d := math.Scalar(h)
	for i := 0; i < verts.Count(); i++ {
		p := verts.Item(i).Point()
		for _, seg := range [][2]math.Point3{
			{math.P3(p.X-d, p.Y, p.Z), math.P3(p.X+d, p.Y, p.Z)},
			{math.P3(p.X, p.Y-d, p.Z), math.P3(p.X, p.Y+d, p.Z)},
			{math.P3(p.X, p.Y, p.Z-d), math.P3(p.X, p.Y, p.Z+d)},
		} {
			idx = append(idx, len(pos), len(pos)+1)
			pos = append(pos, seg[0], seg[1])
		}
	}
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: pos, Indices: idx, Color: cageHandleColor, Opacity: 1}
}
