// SPDX-License-Identifier: GPL-2.0-only

package app

// Drag-to-create (#2014). A sketch geometry tool accepts both click-click and
// click-drag-release, and they are the SAME path: a drag simply means the second point arrived
// on release instead of on the next press. Tools needing three or more points get the drag for
// points 1→2 and click-click for the rest, with no special-casing.
//
// Before this, creation ran off the viewport's press handler with no release handler at all, so
// a press-drag-release placed one point and the shape appeared on a later, unrelated press —
// the inconsistency the issue reported.

// placementDragSlop is the movement (in viewport pixels) below which a press-release counts as a
// click rather than a drag, so a shaky hand does not create geometry. It mirrors
// orbitPivotClickSlop, the same judgement made for the Free-Orbit set-pivot click.
const placementDragSlop = 4

// sketchPlacement is one in-progress press: where it started, and whether it has moved far
// enough to count as a drag.
type sketchPlacement struct {
	active  bool
	pressX  float64
	pressY  float64
	dragged bool
}

// PlacementActive reports whether a creation press is in progress.
func (s *Session) PlacementActive() bool { return s.placement.active }

// BeginPlacement handles a left press over the sketch plane while a geometry tool is active: it
// places a point and arms the drag. It returns false — so the caller falls through to the normal
// selection path — when no click-consuming tool is active.
func (s *Session) BeginPlacement(px, py float64) bool {
	if !s.toolConsumesPlaneClicks() {
		return false
	}
	s.placement = sketchPlacement{active: true, pressX: px, pressY: py}
	s.sketchClick(px, py)
	return true
}

// toolConsumesPlaneClicks reports whether the active tool places geometry from raw plane points
// (rather than entity picks), which is what makes a press part of a placement.
func (s *Session) toolConsumesPlaneClicks() bool {
	if s.tool == nil {
		return false
	}
	_, ok := s.tool.tool.(PlaneClickTool)
	return ok
}

// UpdatePlacement tracks the cursor while the button is held, promoting the press to a drag once
// it passes the slop.
func (s *Session) UpdatePlacement(px, py float64) {
	if !s.placement.active || s.placement.dragged {
		return
	}
	dx, dy := px-s.placement.pressX, py-s.placement.pressY
	if dx*dx+dy*dy > placementDragSlop*placementDragSlop {
		s.placement.dragged = true
	}
}

// EndPlacement handles the release: a drag places the shape's next point (committing the tool if
// that completes it), while a plain click leaves the tool waiting for the next press.
func (s *Session) EndPlacement(px, py float64) {
	if !s.placement.active {
		return
	}
	s.UpdatePlacement(px, py) // a release can be the first report of movement
	dragged := s.placement.dragged
	s.placement = sketchPlacement{}
	if !dragged {
		return
	}
	s.sketchClick(px, py)
}

// CancelPlacement abandons an in-progress press without placing anything (Escape mid-drag).
func (s *Session) CancelPlacement() { s.placement = sketchPlacement{} }
