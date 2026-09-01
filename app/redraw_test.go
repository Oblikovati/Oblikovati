// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/scene"
)

// TestWantsContinuousRedraw pins the #1493 contract that the render-on-demand loop relies on: an
// idle session reports false (so the head may block at ~0% CPU), but a running camera tween
// reports true (so the head keeps ticking and the animation does not freeze mid-transition).
func TestWantsContinuousRedraw(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if s.WantsContinuousRedraw() {
		t.Fatal("a fresh idle session wants continuous redraw; the loop would never block (#1493)")
	}

	target := scene.NewCamera(800, 600)
	s.AnimateCameraTo(target, sketchViewTweenSeconds)
	if !s.CameraAnimating() {
		t.Fatal("AnimateCameraTo did not start a tween; test premise invalid")
	}
	if !s.WantsContinuousRedraw() {
		t.Fatal("a running camera tween must want continuous redraw, else the animation freezes (#1493)")
	}

	s.TickCameraAnimation(sketchViewTweenSeconds) // run the tween to completion
	if s.CameraAnimating() {
		t.Fatal("tween still active after full duration; test premise invalid")
	}
	if s.WantsContinuousRedraw() {
		t.Fatal("session still wants continuous redraw after the tween finished; the loop would never idle (#1493)")
	}
}
