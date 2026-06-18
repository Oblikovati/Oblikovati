// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"
	"testing"

	"oblikovati.org/scene"
)

func dist(c scene.Camera) float64 { return float64(c.Eye.DistanceTo(c.Target)) }

func TestApplyNavigationZoomsOnWheelWhenHovered(t *testing.T) {
	cam := scene.NewCamera(800, 600) // eye–target distance 10
	in := ApplyNavigation(cam, NavInput{Hovered: true, Wheel: 1})
	if dist(in) >= 10 {
		t.Errorf("scroll-up over the viewport should zoom in: dist %v, want < 10", dist(in))
	}
	out := ApplyNavigation(cam, NavInput{Hovered: true, Wheel: -1})
	if dist(out) <= 10 {
		t.Errorf("scroll-down should zoom out: dist %v, want > 10", dist(out))
	}
	if ApplyNavigation(cam, NavInput{Wheel: 1}) != cam {
		t.Error("wheel must be ignored when the viewport is not hovered")
	}
}

// TestApplyNavigationWheelZoomsTowardCursor: the wheel zooms toward the cursor (N2) — at the centre
// it is a pure dolly (target fixed); off-centre it pans the target toward the cursor.
func TestApplyNavigationWheelZoomsTowardCursor(t *testing.T) {
	cam := scene.NewCamera(800, 600)
	center := ApplyNavigation(cam, NavInput{Hovered: true, Wheel: 1, CursorX: 400, CursorY: 300})
	if !center.Target.IsEqualTo(cam.Target, 1e-9) {
		t.Errorf("wheel zoom at the centre should keep the target fixed, got %v", center.Target)
	}
	off := ApplyNavigation(cam, NavInput{Hovered: true, Wheel: 1, CursorX: 700, CursorY: 150})
	if off.Target.IsEqualTo(cam.Target, 1e-9) {
		t.Error("wheel zoom off-centre should move the target toward the cursor (zoom-to-cursor)")
	}
}

// TestClassifyOrbitZone covers the Free-Orbit ring zones (#913 N5–N8): inner disc → free, rim
// left/right → yaw, rim top/bottom → pitch, outside → roll, and a degenerate ring → free.
func TestClassifyOrbitZone(t *testing.T) {
	const cx, cy, radius = 400, 300, 200
	cases := []struct {
		name   string
		px, py float64
		want   OrbitZone
	}{
		{"centre", cx, cy, OrbitFree},
		{"left rim", cx - 190, cy, OrbitYaw},
		{"right rim", cx + 190, cy, OrbitYaw},
		{"top rim", cx, cy - 190, OrbitPitch},
		{"bottom rim", cx, cy + 190, OrbitPitch},
		{"outside", cx + 260, cy, OrbitRoll},
	}
	for _, c := range cases {
		if got := classifyOrbitZone(c.px, c.py, cx, cy, radius); got != c.want {
			t.Errorf("%s: zone = %d, want %d", c.name, got, c.want)
		}
	}
	if got := classifyOrbitZone(cx, cy, cx, cy, 0); got != OrbitFree {
		t.Errorf("degenerate ring zone = %d, want OrbitFree", got)
	}
}

// TestApplyOrbitZones: the latched zone selects the rotation — yaw-only keeps the eye in the
// up-plane (Y=0), pitch-only keeps it in the yaw-plane (X=0), roll spins up without moving the eye.
func TestApplyOrbitZones(t *testing.T) {
	base := scene.NewCamera(800, 600) // eye (0,0,10), up +Y, centre (400,300)

	yaw := ApplyNavigation(base, NavInput{Active: true, Modal: NavOrbit, Left: true, DX: 20, DY: 20, OrbitZone: OrbitYaw})
	if stdmath.Abs(float64(yaw.Eye.Y)) > 1e-9 || yaw.Eye.IsEqualTo(base.Eye, 1e-9) {
		t.Errorf("yaw-only orbit eye = %v, want moved with Y=0", yaw.Eye)
	}
	pitch := ApplyNavigation(base, NavInput{Active: true, Modal: NavOrbit, Left: true, DX: 20, DY: 20, OrbitZone: OrbitPitch})
	if stdmath.Abs(float64(pitch.Eye.X)) > 1e-9 || pitch.Eye.IsEqualTo(base.Eye, 1e-9) {
		t.Errorf("pitch-only orbit eye = %v, want moved with X=0", pitch.Eye)
	}
	roll := ApplyNavigation(base, NavInput{Active: true, Modal: NavOrbit, Left: true, DX: 20, CursorX: 400, CursorY: 200, OrbitZone: OrbitRoll})
	if !roll.Eye.IsEqualTo(base.Eye, 1e-9) || roll.Up.IsEqualTo(base.Up, 1e-9) {
		t.Errorf("roll should spin up (got %v) without moving the eye (got %v)", roll.Up, roll.Eye)
	}
}

// TestApplyNavigationConstrainedOrbit: a Constrained-Orbit drag turntables about the model vertical
// and re-levels the view (removes roll), regardless of the drag's start zone (#913 N10).
func TestApplyNavigationConstrainedOrbit(t *testing.T) {
	cam := scene.NewCamera(800, 600).Roll(0.5) // a rolled view
	out := ApplyNavigation(cam, NavInput{Active: true, Constrained: true, DX: 30})
	if out.Eye.IsEqualTo(cam.Eye, 1e-9) {
		t.Error("constrained-orbit drag should move the eye")
	}
	if !out.Up.IsEqualTo(modelUp, 1e-9) {
		t.Errorf("constrained orbit up = %v, want the model vertical %v (re-levelled)", out.Up, modelUp)
	}
}

func TestApplyNavigationPansWithMiddleDrag(t *testing.T) {
	cam := scene.NewCamera(800, 600)
	out := ApplyNavigation(cam, NavInput{Active: true, Middle: true, DX: 50})
	if out.Target.IsEqualTo(cam.Target, 1e-9) {
		t.Error("middle-drag should pan (move the target)")
	}
	if !out.Forward().IsEqualTo(cam.Forward(), 1e-9) {
		t.Error("pan must keep the view direction")
	}
}

func TestApplyNavigationOrbitsWithShiftMiddle(t *testing.T) {
	cam := scene.NewCamera(800, 600)
	o := ApplyNavigation(cam, NavInput{Active: true, Middle: true, Shift: true, DX: 30})
	if o.Eye.IsEqualTo(cam.Eye, 1e-9) {
		t.Error("shift+middle drag should orbit (move the eye)")
	}
	if stdmath.Abs(dist(o)-dist(cam)) > 1e-9 {
		t.Errorf("orbit must preserve distance: %v want %v", dist(o), dist(cam))
	}
}

// Left-drag must NOT navigate — Inventor reserves it for selection / box-select, and
// orbiting on left-drag collided with the sketch editor's left-click select/drag (#916).
func TestApplyNavigationLeftDragDoesNotMoveCamera(t *testing.T) {
	cam := scene.NewCamera(800, 600)
	// A left-drag is signalled only by Active (no Middle); the camera must be untouched.
	if out := ApplyNavigation(cam, NavInput{Active: true, DX: 40, DY: 25}); out != cam {
		t.Error("a plain left-drag (Active, no middle button) must not move the camera")
	}
}

// TestApplyNavigationModalFKeys checks the hold-to-navigate keys drive a left-drag: F2 pans, F4
// orbits (distance preserved), F3 zooms; a modal mode without the left button does nothing (#911).
func TestApplyNavigationModalFKeys(t *testing.T) {
	cam := scene.NewCamera(800, 600)

	if pan := ApplyNavigation(cam, NavInput{Active: true, Modal: NavPan, Left: true, DX: 30}); pan.Target.IsEqualTo(cam.Target, 1e-9) {
		t.Error("F2-hold + left-drag should pan (move the target)")
	}
	orbit := ApplyNavigation(cam, NavInput{Active: true, Modal: NavOrbit, Left: true, DX: 30})
	if orbit.Eye.IsEqualTo(cam.Eye, 1e-9) {
		t.Error("F4-hold + left-drag should orbit (move the eye)")
	}
	if stdmath.Abs(dist(orbit)-dist(cam)) > 1e-9 {
		t.Error("F4 orbit must preserve the eye–target distance")
	}
	if zoom := ApplyNavigation(cam, NavInput{Active: true, Modal: NavZoom, Left: true, DY: 20}); stdmath.Abs(dist(zoom)-dist(cam)) < 1e-9 {
		t.Error("F3-hold + left-drag should zoom (change the distance)")
	}
	if out := ApplyNavigation(cam, NavInput{Active: true, Modal: NavPan, DX: 30}); out != cam {
		t.Error("a modal nav mode without the left button held must not move the camera")
	}
}

func TestApplyNavigationIdleLeavesCameraUnchanged(t *testing.T) {
	cam := scene.NewCamera(800, 600)
	if ApplyNavigation(cam, NavInput{}) != cam {
		t.Error("no input should leave the camera unchanged")
	}
	if ApplyNavigation(cam, NavInput{Active: true, Middle: true}) != cam {
		t.Error("an active drag with zero delta should not move the camera")
	}
}
