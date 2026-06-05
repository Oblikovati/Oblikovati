// SPDX-License-Identifier: GPL-2.0-only

package viewport

import (
	"testing"

	"github.com/Oblikovati/oblikovati/renderer"
)

// TestPackLightingLayout pins the std140 layout the scene UBO depends on: the header floats and
// each light's three vec4s land at the offsets mesh.frag reads. A layout drift here would
// silently mis-light the scene, so this is the contract test for the Go↔GLSL boundary.
func TestPackLightingLayout(t *testing.T) {
	l := renderer.SceneLighting{
		Ambience: 0.3, Brightness: 1.2, Exposure: 0.9,
		Lights: []renderer.SceneLight{
			{
				Kind: renderer.PointLight, Direction: [3]float32{0.1, 0.2, 0.3},
				Color: [3]float32{0.4, 0.5, 0.6}, Intensity: 2, Position: [3]float32{7, 8, 9}, On: true,
			},
		},
	}
	got := PackLighting(l)
	if len(got) != SceneUBOFloats {
		t.Fatalf("packed %d floats, want %d", len(got), SceneUBOFloats)
	}
	want := []struct {
		idx int
		val float32
	}{
		{0, 0.3}, {1, 1.2}, {2, 0.9}, {3, 1}, // header: ambience, brightness, exposure, count
		{4, 0.1}, {5, 0.2}, {6, 0.3}, {7, float32(renderer.PointLight)}, // dir + kind
		{8, 0.4}, {9, 0.5}, {10, 0.6}, {11, 2}, // color + intensity
		{12, 7}, {13, 8}, {14, 9}, {15, 1}, // pos + on
	}
	for _, w := range want {
		if got[w.idx] != w.val {
			t.Errorf("float[%d] = %g, want %g", w.idx, got[w.idx], w.val)
		}
	}
}

// TestPackLightingDropsOffLightsAndDefaults checks Off lights are excluded from the count and a
// zero Brightness/Exposure falls back to 1 (so a partial rig is not black).
func TestPackLightingDropsOffLightsAndDefaults(t *testing.T) {
	l := renderer.SceneLighting{
		Lights: []renderer.SceneLight{
			{Kind: renderer.DirectionalLight, On: true},
			{Kind: renderer.DirectionalLight, On: false},
		},
	}
	got := PackLighting(l)
	if got[3] != 1 {
		t.Errorf("light count = %g, want 1 (the Off light dropped)", got[3])
	}
	if got[1] != 1 || got[2] != 1 {
		t.Errorf("brightness/exposure = %g/%g, want 1/1 fallback", got[1], got[2])
	}
}

// TestPackLightingClampsToMax checks more than MaxSceneLights On lights do not overflow the UBO
// float array.
func TestPackLightingClampsToMax(t *testing.T) {
	var l renderer.SceneLighting
	for i := 0; i < renderer.MaxSceneLights+5; i++ {
		l.Lights = append(l.Lights, renderer.SceneLight{Kind: renderer.DirectionalLight, On: true})
	}
	got := PackLighting(l)
	if len(got) != SceneUBOFloats {
		t.Fatalf("packed %d floats, want %d", len(got), SceneUBOFloats)
	}
	if got[3] != float32(renderer.MaxSceneLights) {
		t.Errorf("light count = %g, want %d (clamped)", got[3], renderer.MaxSceneLights)
	}
}
