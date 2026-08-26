// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// TestRayPickerCurvedSceneStaysCheapPerFrame is the integration-level fence for the recurring
// hover-pick starvation: the viewport runs RayPicker.Pick EVERY frame while orbiting, and a scene of
// curved (analytic-sphere) bodies — the self-aligning bearing's 16 balls — must not re-tessellate
// every face every frame. This drives the whole pick chain (Pick → nearestFace → RayCastFaces), so a
// regression that bypasses the face pick-tessellation memo (in the picker, not just kernel/ops) also
// trips it. A re-tessellating pick allocates ~150/op PER curved body; the memoized path is bounded.
func TestRayPickerCurvedSceneStaysCheapPerFrame(t *testing.T) {
	const balls = 16
	bodies := make([]*topo.Body, 0, balls)
	for i := range balls {
		// Stagger the balls along x so several sit under the pick ray's neighbourhood, like the
		// bearing's ball complement — the picker still ray-tests them all every frame.
		b, err := brep.SolidSphere(math.V3(float64(i)*0.1, 0, 0).AsPoint(), 1.0, "ball")
		if err != nil {
			t.Fatalf("ball %d: %v", i, err)
		}
		bodies = append(bodies, b)
	}
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(0, 0, 20)
	cam.Target = math.P3(0, 0, 0)
	cam.Up = math.V3(0, 1, 0)
	picker := NewRayPicker(cam, func() []*topo.Body { return bodies })
	filter := NewSelectionFilter(SelectFace)

	// Warm every ball's memo (one orbit frame), then measure the steady-state per-frame pick cost.
	if _, ok := picker.Pick(200, 200, filter); !ok {
		t.Fatal("warm-up pick missed the ball complement")
	}
	const ceiling = 64 // memoized: small per-body slices; re-tessellation would be ~150 × curved bodies
	avg := testing.AllocsPerRun(100, func() {
		picker.Pick(200, 200, filter)
	})
	if avg > ceiling {
		t.Fatalf("RayPicker.Pick over %d curved bodies allocates %.0f/op after warm-up (ceiling %d): the "+
			"pick-tessellation memo is not holding — every orbit frame re-tessellates the balls", balls, avg, ceiling)
	}
}
