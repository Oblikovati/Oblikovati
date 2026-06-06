// SPDX-License-Identifier: GPL-2.0-only

// This file has no cgo build tag: projecting the world axes onto the screen plane is
// pure Go (like navigate.go), so the triad geometry is unit-tested without the native
// stack. axis_gizmo_draw.go (cgo) anchors the result in the viewport corner and paints
// it with the ImGui draw list.
package ui

import (
	"sort"

	"oblikovati/math"
	"oblikovati/scene"
)

// Axis-triad colors follow the universal CAD convention (X red, Y green, Z blue) and are
// deliberately theme-independent — users read axis identity by color across every CAD app.
var (
	axisColorX = [4]float32{0.90, 0.23, 0.27, 1}
	axisColorY = [4]float32{0.42, 0.74, 0.28, 1}
	axisColorZ = [4]float32{0.24, 0.50, 0.92, 1}
)

// axisArrow is one world axis projected onto the gizmo's 2D screen plane: tipX/tipY is
// the arrow tip's pixel offset from the gizmo center (y grows downward), depth is the
// axis' camera-forward component (larger ⇒ pointing away from the viewer) used to paint
// the triad back-to-front, and label/color identify the axis.
type axisArrow struct {
	tipX, tipY float32
	depth      float64
	color      [4]float32
	label      string
}

// axisTriad projects the +X/+Y/+Z world axes onto the camera's screen plane, scaled to
// radius pixels, and returns them sorted back-to-front so a painter draws the arrow
// nearest the viewer last (on top). Only the camera's ORIENTATION is used, so the triad
// spins to match the view but never pans or scales with zoom — it stays pinned wherever
// the caller anchors the center. Example: axisTriad(session.Camera(), 30).
func axisTriad(cam scene.Camera, radius float32) []axisArrow {
	fwd := normVec(cam.Forward())
	right := normVec(fwd.Cross(cam.Up))
	up := right.Cross(fwd) // forward ⟂ right and both unit ⟹ already unit
	arrows := []axisArrow{
		projectAxis(math.V3(1, 0, 0), right, up, fwd, radius, axisColorX, "X"),
		projectAxis(math.V3(0, 1, 0), right, up, fwd, radius, axisColorY, "Y"),
		projectAxis(math.V3(0, 0, 1), right, up, fwd, radius, axisColorZ, "Z"),
	}
	sort.SliceStable(arrows, func(i, j int) bool { return arrows[i].depth > arrows[j].depth })
	return arrows
}

// projectAxis maps one world axis onto the (right, up) screen basis: screen x grows
// rightward, screen y grows downward (hence the negated up component, matching
// renderer.Project's Vulkan y-down convention). depth is the axis' forward component.
func projectAxis(axis, right, up, fwd math.Vector3, radius float32, color [4]float32, label string) axisArrow {
	return axisArrow{
		tipX:  float32(axis.Dot(right)) * radius,
		tipY:  float32(-axis.Dot(up)) * radius,
		depth: axis.Dot(fwd),
		color: color,
		label: label,
	}
}

// normVec returns v scaled to unit length, or v unchanged when it is zero-length (no
// direction to normalize).
func normVec(v math.Vector3) math.Vector3 {
	l := v.Length()
	if l == 0 {
		return v
	}
	return v.Scale(1 / l)
}
