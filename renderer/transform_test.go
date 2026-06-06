// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
	"oblikovati/scene"
)

// project applies a column-major view-projection to a world point, returning NDC
// (clip.xyz / clip.w) and the clip-space w (w > 0 ⇒ in front of the camera).
func project(vp [16]float32, p math.Point3) (ndc [3]float64, w float64) {
	v := [4]float64{p.X, p.Y, p.Z, 1}
	var clip [4]float64
	for r := 0; r < 4; r++ {
		clip[r] = float64(vp[0*4+r])*v[0] + float64(vp[1*4+r])*v[1] +
			float64(vp[2*4+r])*v[2] + float64(vp[3*4+r])*v[3]
	}
	w = clip[3]
	return [3]float64{clip[0] / w, clip[1] / w, clip[2] / w}, w
}

func testCamera() scene.Camera {
	c := scene.NewCamera(800, 600)
	c.Eye = math.P3(0, 0, 10)
	c.Target = math.P3(0, 0, 0)
	return c
}

func TestTargetProjectsToScreenCenter(t *testing.T) {
	vp := ViewProjection(testCamera(), 0.1, 1000)
	ndc, w := project(vp, math.P3(0, 0, 0))
	if w <= 0 {
		t.Fatalf("target should be in front of the camera, w=%g", w)
	}
	if stdmath.Abs(ndc[0]) > 1e-5 || stdmath.Abs(ndc[1]) > 1e-5 {
		t.Errorf("target NDC = (%g, %g), want screen center (0,0)", ndc[0], ndc[1])
	}
	if ndc[2] < 0 || ndc[2] > 1 {
		t.Errorf("depth = %g, want within Vulkan [0,1]", ndc[2])
	}
}

func TestPerspectiveYIsFlippedForVulkan(t *testing.T) {
	vp := ViewProjection(testCamera(), 0.1, 1000)
	// A point above the target (world +Y) must land in the UPPER half of the screen,
	// which in Vulkan's y-down clip space is negative NDC y.
	up, _ := project(vp, math.P3(0, 1, 0))
	if up[1] >= 0 {
		t.Errorf("world +Y projected to NDC y=%g, want negative (upper screen, y-down)", up[1])
	}
	// World +X is to the right → positive NDC x.
	right, _ := project(vp, math.P3(1, 0, 0))
	if right[0] <= 0 {
		t.Errorf("world +X projected to NDC x=%g, want positive (right)", right[0])
	}
}

func TestNearerIsCloserInDepth(t *testing.T) {
	vp := ViewProjection(testCamera(), 0.1, 1000)
	near, _ := project(vp, math.P3(0, 0, 5)) // closer to the eye at z=10
	far, _ := project(vp, math.P3(0, 0, -5)) // farther away
	if !(near[2] < far[2]) {
		t.Errorf("depth not monotonic: near=%g should be < far=%g", near[2], far[2])
	}
}

func TestAspectRatioScalesX(t *testing.T) {
	wide := scene.NewCamera(1600, 400) // aspect 4:1
	wide.Eye, wide.Target = math.P3(0, 0, 10), math.P3(0, 0, 0)
	tall := scene.NewCamera(400, 1600) // aspect 1:4
	tall.Eye, tall.Target = math.P3(0, 0, 10), math.P3(0, 0, 0)
	p := math.P3(1, 0, 0)
	wx, _ := project(ViewProjection(wide, 0.1, 1000), p)
	tx, _ := project(ViewProjection(tall, 0.1, 1000), p)
	// Same world X spans less NDC on a wider viewport.
	if !(wx[0] < tx[0]) {
		t.Errorf("aspect not applied: wide NDC x=%g should be < tall NDC x=%g", wx[0], tx[0])
	}
}

// TestProjectionTranslationInvariance is a metamorphic oracle: shifting the eye,
// target and point by the same offset leaves the projected NDC unchanged.
func TestProjectionTranslationInvariance(t *testing.T) {
	base := ViewProjection(testCamera(), 0.1, 1000)
	baseNDC, _ := project(base, math.P3(0.5, 0.3, 0))

	off := math.V3(7, -4, 2)
	cam := testCamera()
	cam.Eye = cam.Eye.TranslateBy(off)
	cam.Target = cam.Target.TranslateBy(off)
	movedNDC, _ := project(ViewProjection(cam, 0.1, 1000), math.P3(0.5, 0.3, 0).TranslateBy(off))

	for i := 0; i < 3; i++ {
		if stdmath.Abs(baseNDC[i]-movedNDC[i]) > 1e-5 {
			t.Errorf("translation changed NDC[%d]: %g vs %g", i, baseNDC[i], movedNDC[i])
		}
	}
}

func TestProjectTargetIsScreenCenter(t *testing.T) {
	cam := testCamera() // 800×600, looking at the origin
	x, y, ok := Project(cam, 0.1, 1000, math.P3(0, 0, 0))
	if !ok {
		t.Fatal("the target is in front of the camera, want ok")
	}
	if stdmath.Abs(x-400) > 1e-3 || stdmath.Abs(y-300) > 1e-3 {
		t.Errorf("target projected to pixel (%g, %g), want screen center (400, 300)", x, y)
	}
}

func TestProjectPlacesWorldUpInUpperHalf(t *testing.T) {
	cam := testCamera()
	_, y, ok := Project(cam, 0.1, 1000, math.P3(0, 1, 0)) // above the target
	if !ok || y >= 300 {
		t.Errorf("world +Y projected to pixel y=%g (ok=%v), want upper half (< 300)", y, ok)
	}
}

func TestProjectBehindCameraNotOk(t *testing.T) {
	cam := testCamera() // eye at z=10 looking toward −z
	if _, _, ok := Project(cam, 0.1, 1000, math.P3(0, 0, 20)); ok {
		t.Error("a point behind the camera should report ok=false")
	}
}
