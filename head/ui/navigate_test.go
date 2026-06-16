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

func TestApplyNavigationIdleLeavesCameraUnchanged(t *testing.T) {
	cam := scene.NewCamera(800, 600)
	if ApplyNavigation(cam, NavInput{}) != cam {
		t.Error("no input should leave the camera unchanged")
	}
	if ApplyNavigation(cam, NavInput{Active: true, Middle: true}) != cam {
		t.Error("an active drag with zero delta should not move the camera")
	}
}
