//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"oblikovati.org/head/internal/envimage"
	"oblikovati.org/head/viewport"
	"oblikovati.org/math"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
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
	verts, idx, min, max := smokeScene()
	configureViewportSmokeLighting(w, min, max)
	cam := smokeCamera()
	mvp := renderer.ViewProjection(cam, 0.1, 100)
	eye := []float32{float32(cam.Eye.X), float32(cam.Eye.Y), float32(cam.Eye.Z)}
	if inv, ok := viewport.Invert4x4(mvp); ok {
		w.SetViewportSkybox(inv[:], true)
	}

	for i := 0; i < maxFrames && !w.ShouldClose(); i++ {
		w.BeginFrame()
		w.RenderViewport(0, 1280, 720, mvp[:], eye, verts, len(verts)/16, idx,
			nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil, len(idx))
		w.EndFrame(0.10, 0.10, 0.12)
	}
	if path := os.Getenv("OBK_SMOKE_PNG"); path != "" {
		if err := saveViewportPNG(w, path); err != nil {
			fmt.Fprintf(os.Stderr, "smoke: save png: %v\n", err)
		}
	}
	return 0
}

func configureViewportSmokeLighting(w *Window, min, max [3]float32) {
	lightDir := [3]float32{0.5, 0.85, 0.4}
	rig := renderer.SceneLightingFor(renderer.LightingSun)
	rig.Lights[0].Direction = lightDir
	w.SetViewportLighting(viewport.PackLighting(rig))
	if img, ok, _ := envimage.Resolve(renderer.Environment{Preset: renderer.EnvOutdoors, Intensity: 1}); ok {
		u := envimage.Flatten(envimage.MipChain(img))
		w.SetViewportEnvironment(u.Data, u.Dims, 0, 1)
	}
	lvp := viewport.LightMatrix(min, max, lightDir)
	w.SetViewportShadow(lvp[:], true, 0.6, 0.3, true, true) // cast on direct + occlude ambient
}

func smokeCamera() scene.Camera {
	cam := scene.NewCamera(1280, 720)
	cam.Eye, cam.Target, cam.Up = math.P3(3.5, 3, 5), math.P3(0, 0.4, 0), math.V3(0, 1, 0)
	return cam
}

// saveViewportPNG reads the offscreen color image back and writes it as a PNG (swapping the
// surface's BGRA byte order to RGBA), so the rendered lighting/IBL/shadow result can be
// inspected headlessly.
func saveViewportPNG(w *Window, path string) error {
	px, width, height, ok := w.ReadbackViewport(0)
	if !ok {
		return fmt.Errorf("readback unavailable")
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i+4 <= len(px); i += 4 {
		img.Pix[i+0], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = px[i+2], px[i+1], px[i+0], 255
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// smokeScene builds a metallic cube sitting on a large matte ground quad (16-float PBR
// vertices, mode 2 = Realistic), returning the geometry plus the cube's bounds (so the shadow
// frustum frames the caster, not the wide ground).
func smokeScene() (verts []float32, idx []uint32, min, max [3]float32) {
	var base uint32
	box(&verts, &idx, &base, -0.6, 0.6, 0.0, 1.2, -0.6, 0.6, [3]float32{0.55, 0.6, 0.95}, 0.1, 0.35)
	quad(&verts, &idx, &base,
		[4][3]float32{{-6, 0, -6}, {6, 0, -6}, {6, 0, 6}, {-6, 0, 6}},
		[3]float32{0, 1, 0}, [3]float32{0.6, 0.6, 0.62}, 0, 0.95)
	return verts, idx, [3]float32{-0.6, 0, -0.6}, [3]float32{0.6, 1.2, 0.6}
}

// box appends the six faces of an axis-aligned box as PBR quads.
func box(v *[]float32, idx *[]uint32, base *uint32, x0, x1, y0, y1, z0, z1 float32,
	col [3]float32, metal, rough float32,
) {
	quad(v, idx, base, [4][3]float32{{x0, y0, z1}, {x1, y0, z1}, {x1, y1, z1}, {x0, y1, z1}}, [3]float32{0, 0, 1}, col, metal, rough)
	quad(v, idx, base, [4][3]float32{{x1, y0, z0}, {x0, y0, z0}, {x0, y1, z0}, {x1, y1, z0}}, [3]float32{0, 0, -1}, col, metal, rough)
	quad(v, idx, base, [4][3]float32{{x1, y0, z1}, {x1, y0, z0}, {x1, y1, z0}, {x1, y1, z1}}, [3]float32{1, 0, 0}, col, metal, rough)
	quad(v, idx, base, [4][3]float32{{x0, y0, z0}, {x0, y0, z1}, {x0, y1, z1}, {x0, y1, z0}}, [3]float32{-1, 0, 0}, col, metal, rough)
	quad(v, idx, base, [4][3]float32{{x0, y1, z1}, {x1, y1, z1}, {x1, y1, z0}, {x0, y1, z0}}, [3]float32{0, 1, 0}, col, metal, rough)
	quad(v, idx, base, [4][3]float32{{x0, y0, z0}, {x1, y0, z0}, {x1, y0, z1}, {x0, y0, z1}}, [3]float32{0, -1, 0}, col, metal, rough)
}

// quad appends one PBR quad (4 corners CCW + a normal) as two triangles in the 16-float layout.
func quad(v *[]float32, idx *[]uint32, base *uint32, c [4][3]float32, n, col [3]float32,
	metal, rough float32,
) {
	for _, p := range c {
		*v = append(*v, p[0], p[1], p[2], n[0], n[1], n[2], col[0], col[1], col[2], 1, metal, rough, 0, 0, 0, 2)
	}
	*idx = append(*idx, *base+0, *base+1, *base+2, *base+0, *base+2, *base+3)
	*base += 4
}
