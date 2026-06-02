// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/scene"
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

func TestApplyNavigationOrbitsWithShiftMiddleAndLeftDrag(t *testing.T) {
	cam := scene.NewCamera(800, 600)
	for name, in := range map[string]NavInput{
		"shift+middle": {Active: true, Middle: true, Shift: true, DX: 30},
		"left":         {Active: true, Left: true, DX: 30},
	} {
		o := ApplyNavigation(cam, in)
		if o.Eye.IsEqualTo(cam.Eye, 1e-9) {
			t.Errorf("%s drag should orbit (move the eye)", name)
		}
		if stdmath.Abs(dist(o)-dist(cam)) > 1e-9 {
			t.Errorf("%s orbit must preserve distance: %v want %v", name, dist(o), dist(cam))
		}
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
