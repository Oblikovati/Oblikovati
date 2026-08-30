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
// curved (analytic-sphere) bodies — the self-aligning bearing's 16 balls — must stay cheap per frame.
// Since M48/C3 the pick resolves against the ANALYTIC surfaces (RayCastFaces → analytic ray∩surface +
// trim classification, a broad-phase box cull skipping the balls the ray misses) and no longer
// tessellates a face at all, so the per-frame cost is bounded by a few small slices per candidate. A
// regression that reintroduces per-frame face tessellation allocates ~150/op PER curved body and
// trips this ceiling.
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

	// One warm-up frame, then measure the steady-state per-frame pick cost.
	if _, ok := picker.Pick(200, 200, filter); !ok {
		t.Fatal("warm-up pick missed the ball complement")
	}
	const ceiling = 64 // analytic pick: a few small slices per candidate; re-tessellation would be ~150 × curved bodies
	avg := testing.AllocsPerRun(100, func() {
		picker.Pick(200, 200, filter)
	})
	if avg > ceiling {
		t.Fatalf("RayPicker.Pick over %d curved bodies allocates %.0f/op (ceiling %d): the analytic pick "+
			"is re-tessellating the balls every frame instead of resolving against the surface", balls, avg, ceiling)
	}
}
