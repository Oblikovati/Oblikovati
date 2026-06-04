//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/head/internal/envimage"
	"github.com/Oblikovati/oblikovati/head/viewport"
	"github.com/Oblikovati/oblikovati/renderer"
)

// RunSmoke opens the real window and renders up to maxFrames Dear ImGui frames, then
// tears everything down — proving the native stack (window, Vulkan device, swapchain,
// ImGui backends) and the Go↔C frame/widget bindings work on this machine. Returns 0
// on success, non-zero if window creation failed. maxFrames bounds the loop so the
// check cannot hang in automation.
func RunSmoke(maxFrames int) int {
	w, err := CreateWindow(1280, 720, "Oblikovati (smoke)")
	if err != nil {
		return 1
	}
	defer w.Destroy()
	for i := 0; i < maxFrames && !w.ShouldClose(); i++ {
		w.BeginFrame()
		if Begin("Oblikovati") {
			Text(fmt.Sprintf("native head smoke: frame %d", i))
		}
		End()
		w.EndFrame(0.10, 0.10, 0.12)
	}
	return 0
}

// RunViewportSmoke additionally exercises the 3D viewport stack: it initializes the offscreen
// pipeline, sets the scene lighting + an HDR environment (so the descriptor set, scene UBO, and
// environment image upload all execute), and renders a PBR triangle for maxFrames. With the
// validation layer force-enabled it surfaces any Vulkan API misuse in the lighting/IBL path
// (ADR-0026). Returns 0 on success, non-zero if window creation failed.
func RunViewportSmoke(maxFrames int) int {
	w, err := CreateWindow(1280, 720, "Oblikovati (viewport smoke)")
	if err != nil {
		return 1
	}
	defer w.Destroy()
	w.InitViewport()
	w.SetViewportLighting(viewport.PackLighting(renderer.SceneLightingFor(renderer.LightingOutdoors)))
	if img, ok, _ := envimage.Resolve(renderer.Environment{Preset: renderer.EnvOutdoors, Intensity: 1}); ok {
		u := envimage.Flatten(envimage.MipChain(img))
		w.SetViewportEnvironment(u.Data, u.Dims, 0, 1)
	}
	tri, idx := smokeTriangle()
	for i := 0; i < maxFrames && !w.ShouldClose(); i++ {
		w.BeginFrame()
		w.RenderViewport(1280, 720, identity4x4(), []float32{0, 0, 3},
			tri, 3, idx, nil, 0, nil, nil, 0, nil, nil, 0, nil)
		w.EndFrame(0.10, 0.10, 0.12)
	}
	return 0
}

// smokeTriangle returns one metallic PBR triangle in the 16-float vertex layout (mode 2 =
// Realistic) so the smoke exercises the lit + IBL shader path, not just the clear.
func smokeTriangle() ([]float32, []uint32) {
	v := func(x, y float32) []float32 {
		return []float32{x, y, 0, 0, 0, 1, 0.8, 0.8, 0.85, 1, 0.9, 0.2, 0, 0, 0, 2}
	}
	verts := append(append(v(-0.5, -0.5), v(0.5, -0.5)...), v(0, 0.5)...)
	return verts, []uint32{0, 1, 2}
}

// identity4x4 is the column-major identity MVP for the smoke (the triangle is already in NDC).
func identity4x4() []float32 {
	return []float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}
