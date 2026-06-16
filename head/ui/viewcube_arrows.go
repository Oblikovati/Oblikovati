//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
)

// viewCubeArrowColor / viewCubeArrowHotColor are the arrow widgets' idle and hovered tints.
var (
	viewCubeArrowColor    = [4]float32{0.64, 0.68, 0.74, 0.9} // light gray, like the reference arrows
	viewCubeArrowHotColor = [4]float32{0.42, 0.50, 0.64, 1.0} // muted slate blue (matches the cube hover)
)

// ViewCube arrow widgets (#914 / N20–N21): the four adjacent-face triangles around the cube (step
// to the neighbouring face) and the two upper-right roll arrows (rotate the view 90° CW/CCW). The
// 26-region face/edge/corner snapping lives in viewcube.go; this file adds the arrows' hit-zones,
// actions, and drawing. The arrows are placed outside the cube's 1.6r hit reach (and tested with
// precedence) so they never collide with a face/edge/corner pick.

const (
	viewCubeAdjOff  = 1.7  // adjacent-arrow distance from the cube centre, in radii
	viewCubeRollOff = 1.2  // roll-arrow distance from the cube centre, in radii
	viewCubeArrowR  = 0.30 // arrow hit half-size, in radii
)

// arrowKind identifies which arrow (if any) a hit landed on.
type arrowKind int

const (
	arrowNone arrowKind = iota
	arrowRoll
	arrowAdjacent
)

// cubeArrowHit is a hit on one of the ViewCube arrows.
type cubeArrowHit struct {
	kind arrowKind
	ccw  bool        // roll direction
	dir  AdjacentDir // adjacent direction
}

// arrowZone is one drawable/hit-testable arrow: its screen centre and identity.
type arrowZone struct {
	x, y float32
	hit  cubeArrowHit
}

// viewCubeArrowZones returns the six arrow zones around a cube placement: four adjacent triangles
// (up/down/left/right) and two roll arrows (top-right).
func viewCubeArrowZones(p cubePlacement) []arrowZone {
	adj := p.r * viewCubeAdjOff
	roll := p.r * viewCubeRollOff
	return []arrowZone{
		{p.cx, p.cy - adj, cubeArrowHit{kind: arrowAdjacent, dir: AdjacentUp}},
		{p.cx, p.cy + adj, cubeArrowHit{kind: arrowAdjacent, dir: AdjacentDown}},
		{p.cx - adj, p.cy, cubeArrowHit{kind: arrowAdjacent, dir: AdjacentLeft}},
		{p.cx + adj, p.cy, cubeArrowHit{kind: arrowAdjacent, dir: AdjacentRight}},
		{p.cx + roll*0.62, p.cy - roll*0.95, cubeArrowHit{kind: arrowRoll, ccw: true}},
		{p.cx + roll*0.95, p.cy - roll*0.62, cubeArrowHit{kind: arrowRoll, ccw: false}},
	}
}

// hitViewCubeArrow returns the arrow under the cursor (mx,my), or arrowNone.
func hitViewCubeArrow(mx, my float32, p cubePlacement) cubeArrowHit {
	a := p.r * viewCubeArrowR
	for _, z := range viewCubeArrowZones(p) {
		if mx >= z.x-a && mx <= z.x+a && my >= z.y-a && my <= z.y+a {
			return z.hit
		}
	}
	return cubeArrowHit{kind: arrowNone}
}

// applyViewCubeArrow animates the camera for an arrow click: a roll about the view axis, or a 90°
// step to the neighbouring face.
func applyViewCubeArrow(s arrowSession, a cubeArrowHit, pw, ph int) {
	start := s.Camera()
	start.Width, start.Height = pw, ph
	s.SetCamera(start)
	switch a.kind {
	case arrowRoll:
		s.AnimateCameraTo(rolledView(start, a.ccw), viewCubeSnapSecs)
	case arrowAdjacent:
		s.AnimateCameraTo(adjacentView(start, a.dir, s.CubeOrientation(), s.ViewCubePivot()), viewCubeSnapSecs)
	}
}

// drawViewCubeArrows paints the arrow widgets, brightening the one under the cursor.
func drawViewCubeArrows(p cubePlacement, hovered cubeArrowHit) {
	for _, z := range viewCubeArrowZones(p) {
		on := z.hit.kind == hovered.kind &&
			(z.hit.kind == arrowAdjacent && z.hit.dir == hovered.dir ||
				z.hit.kind == arrowRoll && z.hit.ccw == hovered.ccw)
		col := viewCubeArrowColor
		if on {
			col = viewCubeArrowHotColor
		}
		if z.hit.kind == arrowAdjacent {
			drawAdjacentTriangle(z, p, col)
		} else {
			drawRollArrow(z, p, col)
		}
	}
}

// drawAdjacentTriangle draws a triangle pointing from the arrow position toward the cube centre.
func drawAdjacentTriangle(z arrowZone, p cubePlacement, col [4]float32) {
	h := p.r * viewCubeArrowR * 0.8
	dx, dy := p.cx-z.x, p.cy-z.y
	d := float32(stdmath.Hypot(float64(dx), float64(dy)))
	if d < 1e-3 {
		return
	}
	nx, ny := dx/d, dy/d // toward cube
	px, py := -ny, nx    // perpendicular
	tipX, tipY := z.x+nx*h, z.y+ny*h
	b1x, b1y := z.x-nx*h*0.4+px*h, z.y-ny*h*0.4+py*h
	b2x, b2y := z.x-nx*h*0.4-px*h, z.y-ny*h*0.4-py*h
	native.DrawTriangleFilled(tipX, tipY, b1x, b1y, b2x, b2y, col)
}

// drawRollArrow draws a quarter-circle arc with an arrowhead — the CW/CCW roll control.
func drawRollArrow(z arrowZone, p cubePlacement, col [4]float32) {
	rad := p.r * viewCubeArrowR * 0.9
	start, sweep := -2.2, 1.9 // radians; a ~110° arc near the top
	if !z.ccwOrCW() {
		start, sweep = -0.9, -1.9
	}
	const seg = 6
	var ex, ey, pex, pey float32
	for i := 0; i <= seg; i++ {
		ang := start + sweep*float64(i)/seg
		ex = z.x + rad*float32(stdmath.Cos(ang))
		ey = z.y + rad*float32(stdmath.Sin(ang))
		if i > 0 {
			native.DrawLine(pex, pey, ex, ey, col, 2.0)
		}
		pex, pey = ex, ey
	}
	// Arrowhead at the arc end, tangent to the sweep direction.
	ah := rad * 0.7
	tang := start + sweep
	tx, ty := float32(stdmath.Cos(tang+1.57)), float32(stdmath.Sin(tang+1.57))
	if sweep < 0 {
		tx, ty = -tx, -ty
	}
	nx, ny := float32(stdmath.Cos(tang)), float32(stdmath.Sin(tang))
	native.DrawTriangleFilled(ex+tx*ah, ey+ty*ah, ex-nx*ah*0.6, ey-ny*ah*0.6, ex+nx*ah*0.6, ey+ny*ah*0.6, col)
}

// ccwOrCW reports the roll direction for the arrow (true ⇒ CCW).
func (z arrowZone) ccwOrCW() bool { return z.hit.ccw }

// arrowSession is the session surface applyViewCubeArrow needs (the camera + tween), so the action
// is testable with a small fake.
type arrowSession interface {
	Camera() scene.Camera
	SetCamera(scene.Camera)
	AnimateCameraTo(scene.Camera, float64)
	CubeOrientation() doc.CubeOrient
	ViewCubePivot() math.Point3
}
