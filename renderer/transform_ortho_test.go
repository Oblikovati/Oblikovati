// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"math"
	"testing"

	obkmath "oblikovati/math"
	"oblikovati/scene"
)

func orthoTestCamera() scene.Camera {
	return scene.Camera{
		Eye: obkmath.P3(0, 0, 10), Target: obkmath.P3(0, 0, 0), Up: obkmath.V3(0, 1, 0),
		FOV: math.Pi / 4, Width: 800, Height: 600,
	}
}

// clipW returns the homogeneous w of p under cam's view-projection — depth-dependent for
// perspective, constant 1 for orthographic.
func clipW(cam scene.Camera, p obkmath.Point3) float64 {
	vp := ViewProjection(cam, 0.1, 1000)
	return float64(vp[3])*p.X + float64(vp[7])*p.Y + float64(vp[11])*p.Z + float64(vp[15])
}

// TestPerspectiveWWithDepth confirms the perspective path is unchanged: clip.w tracks
// camera depth (w ≈ -z_eye), so nearer points have smaller w (foreshortening).
func TestPerspectiveWWithDepth(t *testing.T) {
	cam := orthoTestCamera() // Orthographic defaults false
	near := clipW(cam, obkmath.P3(0, 0, 5))
	far := clipW(cam, obkmath.P3(0, 0, -5))
	if !(far > near) {
		t.Fatalf("perspective w should grow with distance from eye: near=%.3f far=%.3f", near, far)
	}
}

// TestOrthographicWConstant confirms the parallel projection: clip.w is 1 regardless of
// depth (no foreshortening), which is the defining property of an orthographic view.
func TestOrthographicWConstant(t *testing.T) {
	cam := orthoTestCamera()
	cam.Orthographic = true
	for _, z := range []float64{5, 0, -5, -50} {
		if w := clipW(cam, obkmath.P3(0, 0, z)); math.Abs(w-1) > 1e-5 {
			t.Errorf("ortho clip.w at z=%.0f = %.5f, want 1", z, w)
		}
	}
}

// TestOrthographicScaleMatchesFOV checks the extent: a point one half-height
// (dist·tan(FOV/2)) above the target maps to the top edge of the viewport (pixel y≈0).
func TestOrthographicScaleMatchesFOV(t *testing.T) {
	cam := orthoTestCamera()
	cam.Orthographic = true
	halfH := 10 * math.Tan(cam.FOV/2) // dist=10
	_, y, ok := Project(cam, 0.1, 1000, obkmath.P3(0, halfH, 0))
	if !ok {
		t.Fatal("ortho point at top edge should project (w=1)")
	}
	if math.Abs(y) > 0.5 { // top edge is pixel y≈0
		t.Errorf("point at +halfH mapped to y=%.2f px, want ≈0 (top edge)", y)
	}
}
