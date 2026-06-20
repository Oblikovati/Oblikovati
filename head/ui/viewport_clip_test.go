//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// TestViewportClipFarKeepsDefaultForSmallScenes: an ordinary model close to the camera keeps
// the fixed far plane, so nothing about normal scenes changes.
func TestViewportClipFarKeepsDefaultForSmallScenes(t *testing.T) {
	cam := scene.Camera{Eye: math.P3(40, 40, 40), Target: math.P3(0, 0, 0), Up: math.V3(0, 0, 1), FOV: 0.8}
	if got := viewportClipFar(cam, [3]float32{-10, -10, -10}, [3]float32{10, 10, 10}, true); got != viewportFar {
		t.Errorf("small scene far = %v, want fixed %v", got, viewportFar)
	}
	if got := viewportClipFar(cam, [3]float32{}, [3]float32{}, false); got != viewportFar {
		t.Errorf("no-geometry far = %v, want fixed %v", got, viewportFar)
	}
}

// TestViewportClipFarEnclosesLargeDrawing: a DWG-scale drawing (~100k units) viewed from far
// enough that the fixed 5000 far plane would clip it must get a far plane beyond the farthest
// geometry — the zoom-out-hides-the-sketch bug. The far plane must exceed the camera's
// distance to the farthest corner.
func TestViewportClipFarEnclosesLargeDrawing(t *testing.T) {
	// A 100k-unit-wide drawing centred at the origin, camera pulled back 150k units on Z.
	mn, mx := [3]float32{-50000, -50000, 0}, [3]float32{50000, 50000, 0}
	cam := scene.Camera{Eye: math.P3(0, 0, 150000), Target: math.P3(0, 0, 0), Up: math.V3(0, 1, 0), FOV: 0.8}

	far := viewportClipFar(cam, mn, mx, true)
	if far <= viewportFar {
		t.Fatalf("large drawing far = %v, expected it to extend past the fixed %v", far, viewportFar)
	}
	// The farthest geometry corner is at distance sqrt(50000^2+50000^2+150000^2) from the eye;
	// the far plane must enclose it or the sketch clips on zoom-out (regression guard).
	farthest := cam.Eye.DistanceTo(math.P3(50000, 50000, 0))
	if far < farthest {
		t.Errorf("far plane %v does not enclose farthest corner at %v", far, farthest)
	}
}
