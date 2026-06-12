//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// The move/rotate triad and manipulator handles (M05-F13): the head renders the
// session's gizmo state as on-top overlay geometry, hit-tests it in screen space,
// and routes pointer rays to the session's drag math.

// triadPixelLen is the axis length in pixels; rings sit at 3/4 of it, the plane
// quads at ~1/3. gizmoHitPx is the screen-space grab radius.
const (
	triadPixelLen = 90
	gizmoHitPx    = 9.0
)

// triadAxisColors is x/y/z in the conventional RGB.
var triadAxisColors = [3][4]float32{{0.9, 0.2, 0.2, 1}, {0.2, 0.8, 0.2, 1}, {0.25, 0.45, 1, 1}}

// triadOverlay appends the triad's geometry to the overlay list.
func triadOverlay(s *app.Session, cam scene.Camera, list renderer.DrawList) renderer.DrawList {
	spec := s.TriadSpec()
	if !spec.Visible {
		return list
	}
	worldLen := triadPixelLen * cam.WorldPerPixel()
	pos := s.TriadPosition()
	for i, axis := range triadWorldAxes(s) {
		if seg := types.TriadSegment(uint8(types.TriadXAxis) + uint8(i)); s.TriadAllows(seg) {
			list.Items = append(list.Items, gizmoLine(pos, pos.TranslateBy(axis.Scale(worldLen)), triadAxisColors[i]))
		}
		if seg := types.TriadSegment(uint8(types.TriadXRing) + uint8(i)); s.TriadAllows(seg) {
			list.Items = append(list.Items, gizmoRing(pos, axis, worldLen*0.75, triadAxisColors[i]))
		}
	}
	return list
}

// triadWorldAxes resolves the spec's axes as unit world vectors.
func triadWorldAxes(s *app.Session) [3]math.Vector3 {
	spec := s.TriadSpec()
	axes := [3]math.Vector3{math.V3(1, 0, 0), math.V3(0, 1, 0), math.V3(0, 0, 1)}
	for i, a := range []*types.UnitVector{spec.AxisX, spec.AxisY, spec.AxisZ} {
		if a != nil {
			axes[i] = math.V3(a.X, a.Y, a.Z)
		}
	}
	return axes
}

// gizmoLine is one on-top axis line.
func gizmoLine(a, b math.Point3, color [4]float32) renderer.DrawItem {
	return renderer.DrawItem{
		Primitive: renderer.Lines, Positions: []math.Point3{a, b},
		Indices: []int{0, 1}, Color: color, OnTop: true,
	}
}

// gizmoRing is a rotation ring: a 24-segment circle around axis at radius.
func gizmoRing(center math.Point3, axis math.Vector3, radius float64, color [4]float32) renderer.DrawItem {
	u, v := planeBasis(axis)
	const segments = 24
	pos := make([]math.Point3, segments)
	idx := make([]int, 0, segments*2)
	for i := 0; i < segments; i++ {
		ang := 2 * stdmath.Pi * float64(i) / segments
		offset := u.Scale(radius * stdmath.Cos(ang)).Add(v.Scale(radius * stdmath.Sin(ang)))
		pos[i] = center.TranslateBy(offset)
		idx = append(idx, i, (i+1)%segments)
	}
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: pos, Indices: idx, Color: color, OnTop: true}
}

// planeBasis returns two unit vectors spanning the plane normal to axis.
func planeBasis(axis math.Vector3) (math.Vector3, math.Vector3) {
	ref := math.V3(0, 1, 0)
	if stdmath.Abs(float64(axis.Dot(ref))) > 0.9 {
		ref = math.V3(1, 0, 0)
	}
	u := axis.Cross(ref)
	u = u.Scale(1 / float64(u.Length()))
	return u, axis.Cross(u).Scale(-1)
}

// manipulatorOverlay appends every declared handle as a small on-top marker.
func manipulatorOverlay(s *app.Session, cam scene.Camera, list renderer.DrawList) renderer.DrawList {
	size := 5 * cam.WorldPerPixel()
	for _, handles := range s.Manipulators().Handles() {
		for _, h := range handles {
			p := math.P3(h.Position.X, h.Position.Y, h.Position.Z)
			list.Items = append(list.Items, gizmoMarker(p, cam, size))
		}
	}
	return list
}

// gizmoMarker is a camera-facing quad outline marking a handle hotspot.
func gizmoMarker(p math.Point3, cam scene.Camera, half float64) renderer.DrawItem {
	forward := unitV(cam.Eye.VectorTo(cam.Target))
	right := unitV(forward.Cross(cam.Up)).Scale(half)
	up := unitV(right.Cross(forward)).Scale(half)
	pos := []math.Point3{
		p.TranslateBy(right.Add(up)), p.TranslateBy(right.Sub(up).Scale(1)),
		p.TranslateBy(right.Scale(-1).Sub(up)), p.TranslateBy(up.Sub(right)),
	}
	return renderer.DrawItem{
		Primitive: renderer.Lines, Positions: pos,
		Indices: []int{0, 1, 1, 2, 2, 3, 3, 0}, Color: [4]float32{1, 0.8, 0.2, 1}, OnTop: true,
	}
}

// unitV normalizes (zero-safe).
func unitV(v math.Vector3) math.Vector3 {
	l := float64(v.Length())
	if l == 0 {
		return v
	}
	return v.Scale(1 / l)
}

// routeGizmoInput drives the triad/manipulator gestures from the viewport pointer.
// It reports whether the gizmo consumed the pointer this frame (the caller then
// skips picking).
func routeGizmoInput(s *app.Session, cam scene.Camera) bool {
	if s.TriadDragging() || s.ManipulatorDragging() {
		continueGizmoDrag(s, cam)
		return true
	}
	if !native.IsItemHovered() {
		return false
	}
	mx, my := viewportCursor()
	if seg, hit := triadHitTest(s, cam, mx, my); hit {
		s.HoverTriadSegment(seg, true)
		if native.IsItemClicked(native.MouseLeft) {
			rayO, rayD := cam.RayThrough(mx, my)
			snapTol := gizmoHitPx * cam.WorldPerPixel()
			_ = s.BeginTriadDrag(seg, rayO, rayD, snapTol, native.KeyShift(), native.KeyCtrl())
		}
		return true
	}
	s.HoverTriadSegment(types.TriadOrigin, false)
	if gizmo, handle, hit := manipulatorHitTest(s, cam, mx, my); hit && native.IsItemClicked(native.MouseLeft) {
		rayO, rayD := cam.RayThrough(mx, my)
		forward := unitV(cam.Eye.VectorTo(cam.Target))
		snapTol := gizmoHitPx * cam.WorldPerPixel()
		_ = s.BeginManipulatorDrag(gizmo, handle, forward.Scale(-1), rayO, rayD, snapTol, native.KeyShift(), native.KeyCtrl())
		return true
	}
	return false
}

// continueGizmoDrag advances or ends the in-flight gesture with the pointer ray.
func continueGizmoDrag(s *app.Session, cam scene.Camera) {
	mx, my := viewportCursor()
	rayO, rayD := cam.RayThrough(mx, my)
	shift, ctrl := native.KeyShift(), native.KeyCtrl()
	if native.MouseDown(native.MouseLeft) {
		if s.TriadDragging() {
			_ = s.DragTriad(rayO, rayD, shift, ctrl)
		} else {
			_ = s.DragManipulator(rayO, rayD, shift, ctrl)
		}
		return
	}
	if s.TriadDragging() {
		_ = s.EndTriadDrag(rayO, rayD, shift, ctrl)
	} else {
		_ = s.EndManipulatorDrag(rayO, rayD, shift, ctrl)
	}
}

// triadHitTest finds the triad segment under the cursor in screen space.
func triadHitTest(s *app.Session, cam scene.Camera, mx, my float64) (types.TriadSegment, bool) {
	if !s.TriadSpec().Visible {
		return 0, false
	}
	worldLen := triadPixelLen * cam.WorldPerPixel()
	pos := s.TriadPosition()
	axes := triadWorldAxes(s)
	best, bestDist := types.TriadSegment(0), gizmoHitPx
	found := false
	probe := func(seg types.TriadSegment, p math.Point3) {
		if !s.TriadAllows(seg) {
			return
		}
		if d, ok := screenDistance(cam, p, mx, my); ok && d < bestDist {
			best, bestDist, found = seg, d, true
		}
	}
	for i, axis := range axes {
		for _, t := range []float64{0.4, 0.7, 1.0} {
			probe(types.TriadSegment(uint8(types.TriadXAxis)+uint8(i)), pos.TranslateBy(axis.Scale(worldLen*t)))
		}
		u, v := planeBasis(axis)
		ringSeg := types.TriadSegment(uint8(types.TriadXRing) + uint8(i))
		for k := 0; k < 16; k++ {
			ang := 2 * stdmath.Pi * float64(k) / 16
			offset := u.Scale(0.75 * worldLen * stdmath.Cos(ang)).Add(v.Scale(0.75 * worldLen * stdmath.Sin(ang)))
			probe(ringSeg, pos.TranslateBy(offset))
		}
	}
	probe(types.TriadOrigin, pos)
	return best, found
}

// manipulatorHitTest finds the handle under the cursor.
func manipulatorHitTest(s *app.Session, cam scene.Camera, mx, my float64) (string, string, bool) {
	for gizmo, handles := range s.Manipulators().Handles() {
		for _, h := range handles {
			radius := h.RadiusPx
			if radius <= 0 {
				radius = gizmoHitPx
			}
			p := math.P3(h.Position.X, h.Position.Y, h.Position.Z)
			if d, ok := screenDistance(cam, p, mx, my); ok && d <= radius {
				return gizmo, h.ID, true
			}
		}
	}
	return "", "", false
}

// screenDistance projects p and returns its pixel distance to the cursor.
func screenDistance(cam scene.Camera, p math.Point3, mx, my float64) (float64, bool) {
	sx, sy, ok := renderer.Project(cam, viewportNear, viewportFar, p)
	if !ok {
		return 0, false
	}
	dx, dy := sx-mx, sy-my
	return stdmath.Sqrt(dx*dx + dy*dy), true
}
