//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

/*
#include <stdint.h>

void     obk_viewport_init(void* h, const uint32_t* vert, int vlen, const uint32_t* frag, int flen,
                           const uint32_t* skyVert, int skyVLen, const uint32_t* skyFrag, int skyFLen);
void     obk_viewport_set_skybox(void* h, const float* invVP, int show);
void     obk_viewport_set_shadow(void* h, const float* lightVP, int enabled, float density,
                                 float softness, int castOnDirect, int occludeAmbient);
void     obk_viewport_render(void* h, int slot, int w, int hh, const float* mvp, const float* camPos,
                             const float* triV, int triVC, const uint32_t* triIdx, int triIC,
                             const float* occV, int occVC, const uint32_t* occIdx, int occIC,
                             const float* lineV, int lineVC, const uint32_t* lineIdx, int lineIC,
                             const float* hidV, int hidVC, const uint32_t* hidIdx, int hidIC,
                             const float* topTriV, int topTriVC, const uint32_t* topTriIdx, int topTriIC,
                             const float* topLineV, int topLineVC, const uint32_t* topLineIdx, int topLineIC,
                             int triBiasFirst, const float* clip);
void     obk_viewport_set_clear(void* h, float r, float g, float b);
void     obk_viewport_set_normal_debug(void* h, int on);
void     obk_viewport_set_lighting(void* h, const float* data, int n);
void     obk_viewport_set_environment(void* h, const float* data, const int* dims, int levels,
                                      float rotation, float intensity);
uint64_t obk_viewport_texture(void* h, int slot);
int      obk_viewport_readback(void* h, int slot, unsigned char* out, int cap, int* w, int* hh);
int      obk_window_capture(void* h, unsigned char* out, int cap, int* w, int* hh);

void obk_ig_image(unsigned long long tex, float w, float h);
void obk_ig_content_region_avail(float* w, float* h);
*/
import "C"

import "unsafe"

// InitViewport creates the 3D render pipeline from the embedded compiled SPIR-V. Call
// once after CreateWindow, before rendering.
func (w *Window) InitViewport() {
	C.obk_viewport_init(w.handle,
		(*C.uint32_t)(unsafe.Pointer(&meshVertSPV[0])), C.int(len(meshVertSPV)),
		(*C.uint32_t)(unsafe.Pointer(&meshFragSPV[0])), C.int(len(meshFragSPV)),
		(*C.uint32_t)(unsafe.Pointer(&skyboxVertSPV[0])), C.int(len(skyboxVertSPV)),
		(*C.uint32_t)(unsafe.Pointer(&skyboxFragSPV[0])), C.int(len(skyboxFragSPV)))
}

// SetViewportSkybox enables drawing the HDR environment as the viewport background, passing the
// inverse view-projection (column-major, 16 floats) used to reconstruct view rays. A nil matrix
// or show=false disables the skybox (the themed clear color shows through). Next RenderViewport.
func (w *Window) SetViewportSkybox(invVP []float32, show bool) {
	s := 0
	if show && len(invVP) == 16 {
		s = 1
	}
	if s == 0 {
		C.obk_viewport_set_skybox(w.handle, nil, 0)
		return
	}
	C.obk_viewport_set_skybox(w.handle, floatPtr(invVP), 1)
}

// RenderViewport renders the geometry into the offscreen target at (w,h) using the
// column-major MVP, with camPos (3 floats, world-space camera eye) supplying the PBR view
// vector. Triangle and line vertices are interleaved as 16 floats [pos.xyz, normal.xyz,
// color.rgba, metallic, roughness, emissive.rgb, mode]; indices are 0-based within their own
// vertex array. The topTri/topLine streams are drawn last with the depth test disabled, so
// client-graphics overlay/burn-through geometry stays visible over the model (PBI-067).
// slot selects which offscreen tile target to render into (0 for the single view; 0..3
// for the tiled layouts). Each slot keeps its own image, so tiles show distinct cameras.
func (win *Window) RenderViewport(slot, w, h int, mvp []float32, camPos []float32,
	triVerts []float32, triVCount int, triIdx []uint32,
	occVerts []float32, occVCount int, occIdx []uint32,
	lineVerts []float32, lineVCount int, lineIdx []uint32,
	hidVerts []float32, hidVCount int, hidIdx []uint32,
	topTriVerts []float32, topTriVCount int, topTriIdx []uint32,
	topLineVerts []float32, topLineVCount int, topLineIdx []uint32,
	triBiasFirst int, clip []float32,
) {
	C.obk_viewport_render(win.handle, C.int(slot), C.int(w), C.int(h),
		(*C.float)(unsafe.Pointer(&mvp[0])), floatPtr(camPos),
		floatPtr(triVerts), C.int(triVCount), uint32Ptr(triIdx), C.int(len(triIdx)),
		floatPtr(occVerts), C.int(occVCount), uint32Ptr(occIdx), C.int(len(occIdx)),
		floatPtr(lineVerts), C.int(lineVCount), uint32Ptr(lineIdx), C.int(len(lineIdx)),
		floatPtr(hidVerts), C.int(hidVCount), uint32Ptr(hidIdx), C.int(len(hidIdx)),
		floatPtr(topTriVerts), C.int(topTriVCount), uint32Ptr(topTriIdx), C.int(len(topTriIdx)),
		floatPtr(topLineVerts), C.int(topLineVCount), uint32Ptr(topLineIdx), C.int(len(topLineIdx)),
		C.int(triBiasFirst), floatPtr(clip))
}

// SetViewportClear sets the 3D pass background color (themed); it takes effect on the
// next RenderViewport.
func (w *Window) SetViewportClear(r, g, b float32) {
	C.obk_viewport_set_clear(w.handle, C.float(r), C.float(g), C.float(b))
}

// SaveViewportPNG reads back the current single-view (slot 0) offscreen image and writes it as
// a PNG — a screenshot of exactly what the viewport shows, for inspecting render defects.
func (w *Window) SaveViewportPNG(path string) error {
	return saveViewportPNG(w, path)
}

// SaveWindowPNG reads back the current swapchain image and writes it as a PNG — a screenshot of the
// WHOLE application window (chrome, open dialogs, and the composited viewport), for inspecting UI
// state headlessly. Call after the frame has composited (EndFrame).
func (w *Window) SaveWindowPNG(path string) error {
	return saveWindowPNG(w, path)
}

// SetViewportNormalDebug toggles the normal-debug render: shaded triangles draw front-facing
// green / back-facing red so inverted-winding / flipped-normal triangles stand out. Takes
// effect on the next RenderViewport.
func (w *Window) SetViewportNormalDebug(on bool) {
	v := 0
	if on {
		v = 1
	}
	C.obk_viewport_set_normal_debug(w.handle, C.int(v))
}

// SetViewportLighting uploads the packed scene-lighting UBO (viewport.PackLighting's std140
// float array) to the viewport; it takes effect on the next RenderViewport. An empty slice is
// a no-op (the previously set lighting, or the default headlight, stays in effect).
func (w *Window) SetViewportLighting(ubo []float32) {
	if len(ubo) == 0 {
		return
	}
	C.obk_viewport_set_lighting(w.handle, floatPtr(ubo), C.int(len(ubo)))
}

// SetViewportEnvironment uploads the IBL environment as a CPU mip chain (data = all levels'
// RGBA float32 concatenated; dims = w,h per level, so len(dims) == 2·levels) and enables
// image-based lighting with the given azimuth rotation (radians) and intensity. An empty data
// or dims slice disables IBL (the analytic ambient resumes). Takes effect next RenderViewport.
func (w *Window) SetViewportEnvironment(data []float32, dims []int32, rotation, intensity float32) {
	if len(data) == 0 || len(dims) < 2 {
		C.obk_viewport_set_environment(w.handle, nil, nil, 0, 0, 0)
		return
	}
	C.obk_viewport_set_environment(w.handle, floatPtr(data),
		(*C.int)(unsafe.Pointer(&dims[0])), C.int(len(dims)/2),
		C.float(rotation), C.float(intensity))
}

// SetViewportShadow renders the sun shadow map with the given column-major light-space matrix
// (16 floats) and applies it: castOnDirect darkens direct light (object/ground shadows),
// occludeAmbient attenuates the ambient term in shadowed regions (ambient occlusion). A
// nil/short matrix or enabled=false disables the map. Takes effect next RenderViewport.
func (w *Window) SetViewportShadow(lightVP []float32, enabled bool, density, softness float32,
	castOnDirect, occludeAmbient bool,
) {
	if !enabled || len(lightVP) != 16 {
		C.obk_viewport_set_shadow(w.handle, nil, 0, 0, 0, 0, 0)
		return
	}
	C.obk_viewport_set_shadow(w.handle, floatPtr(lightVP), 1, C.float(density), C.float(softness),
		cBool(castOnDirect), cBool(occludeAmbient))
}

// ViewportTexture returns the ImGui texture handle for tile slot's last rendered frame.
func (w *Window) ViewportTexture(slot int) uint64 {
	return uint64(C.obk_viewport_texture(w.handle, C.int(slot)))
}

// ReadbackViewport copies the offscreen color image to host memory, returning the raw 8-bit
// pixels (the surface's channel order, typically BGRA), width and height, and ok=false if the
// target is not ready. For headless verification/screenshots, not the per-frame path.
func (w *Window) ReadbackViewport(slot int) (pixels []byte, width, height int, ok bool) {
	var cw, ch C.int
	if C.obk_viewport_readback(w.handle, C.int(slot), nil, 0, &cw, &ch) != 0 || cw <= 0 || ch <= 0 {
		return nil, 0, 0, false
	}
	buf := make([]byte, int(cw)*int(ch)*4)
	n := C.obk_viewport_readback(w.handle, C.int(slot), (*C.uchar)(unsafe.Pointer(&buf[0])), C.int(len(buf)),
		&cw, &ch)
	if int(n) != len(buf) {
		return nil, 0, 0, false
	}
	return buf, int(cw), int(ch), true
}

// ReadbackWindow copies the current swapchain image (the WHOLE window: chrome, dialogs, and the
// composited viewport) to host memory, returning the raw 8-bit pixels (the surface's channel order,
// typically BGRA), width and height, and ok=false if the swapchain is not ready. For headless
// verification/screenshots; call after the frame has composited (EndFrame).
func (w *Window) ReadbackWindow() (pixels []byte, width, height int, ok bool) {
	var cw, ch C.int
	if C.obk_window_capture(w.handle, nil, 0, &cw, &ch) != 0 || cw <= 0 || ch <= 0 {
		return nil, 0, 0, false
	}
	buf := make([]byte, int(cw)*int(ch)*4)
	n := C.obk_window_capture(w.handle, (*C.uchar)(unsafe.Pointer(&buf[0])), C.int(len(buf)), &cw, &ch)
	if int(n) != len(buf) {
		return nil, 0, 0, false
	}
	return buf, int(cw), int(ch), true
}

// Image draws a texture (e.g. the viewport) at the given size in the current window.
func Image(tex uint64, w, h float32) {
	C.obk_ig_image(C.ulonglong(tex), C.float(w), C.float(h))
}

// ContentRegionAvail reports the free space (pixels) in the current ImGui window.
func ContentRegionAvail() (w, h float32) {
	var cw, ch C.float
	C.obk_ig_content_region_avail(&cw, &ch)
	return float32(cw), float32(ch)
}

// floatPtr / uint32Ptr return the data pointer of a slice, or nil for an empty slice
// (so the C side gets a null pointer with a zero count rather than a bad address).
func floatPtr(s []float32) *C.float {
	if len(s) == 0 {
		return nil
	}
	return (*C.float)(unsafe.Pointer(&s[0]))
}

func uint32Ptr(s []uint32) *C.uint32_t {
	if len(s) == 0 {
		return nil
	}
	return (*C.uint32_t)(unsafe.Pointer(&s[0]))
}
