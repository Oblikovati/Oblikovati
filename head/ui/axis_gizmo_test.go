// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"math"
	"testing"

	"oblikovati.org/scene"

	gmath "oblikovati.org/math"
)

// frontCamera looks straight down −Z (Inventor's default), so +X points screen-right,
// +Y screen-up, and +Z straight at the viewer.
func frontCamera() scene.Camera {
	cam := scene.NewCamera(800, 600)
	cam.Eye = gmath.P3(0, 0, 10)
	cam.Target = gmath.P3(0, 0, 0)
	cam.Up = gmath.V3(0, 1, 0)
	return cam
}

func findArrow(arrows []axisArrow, label string) (axisArrow, bool) {
	for _, a := range arrows {
		if a.label == label {
			return a, true
		}
	}
	return axisArrow{}, false
}

// A front view must put +X to the right (+tipX), +Y up (−tipY because screen y is down),
// and leave Z with no screen extent (it points at the camera).
func TestAxisTriadFrontViewScreenDirections(t *testing.T) {
	arrows := axisTriad(frontCamera(), 30)

	x, _ := findArrow(arrows, "X")
	if x.tipX <= 0 || math.Abs(float64(x.tipY)) > 1e-3 {
		t.Fatalf("X arrow: got tip (%.3f,%.3f), want +x, ~0 y", x.tipX, x.tipY)
	}
	y, _ := findArrow(arrows, "Y")
	if y.tipY >= 0 || math.Abs(float64(y.tipX)) > 1e-3 {
		t.Fatalf("Y arrow: got tip (%.3f,%.3f), want -y (up on screen), ~0 x", y.tipX, y.tipY)
	}
	z, _ := findArrow(arrows, "Z")
	if math.Hypot(float64(z.tipX), float64(z.tipY)) > 1e-3 {
		t.Fatalf("Z arrow: got tip (%.3f,%.3f), want ~origin (points at viewer)", z.tipX, z.tipY)
	}
}

// The tip offset must scale with the requested radius (the gizmo is screen-constant size).
func TestAxisTriadRadiusScalesTip(t *testing.T) {
	x30, _ := findArrow(axisTriad(frontCamera(), 30), "X")
	x60, _ := findArrow(axisTriad(frontCamera(), 60), "X")
	if math.Abs(float64(x60.tipX-2*x30.tipX)) > 1e-3 {
		t.Fatalf("doubling radius should double tipX: got %.3f then %.3f", x30.tipX, x60.tipX)
	}
}

// Painter order: the arrow pointing most toward the viewer (most-negative depth) is drawn
// last, so it lands on top. On the front view that is +Z.
func TestAxisTriadSortedBackToFront(t *testing.T) {
	arrows := axisTriad(frontCamera(), 30)
	for i := 1; i < len(arrows); i++ {
		if arrows[i-1].depth < arrows[i].depth {
			t.Fatalf("arrows not sorted far→near: %v", arrows)
		}
	}
	if arrows[len(arrows)-1].label != "Z" {
		t.Fatalf("front view should draw Z (toward viewer) last, got %q", arrows[len(arrows)-1].label)
	}
}
