//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

/*
#include <stdint.h>

void     obk_viewport_init(void* h, const uint32_t* vert, int vlen, const uint32_t* frag, int flen);
void     obk_viewport_render(void* h, int w, int hh, const float* mvp, const float* camPos,
                             const float* triV, int triVC, const uint32_t* triIdx, int triIC,
                             const float* occV, int occVC, const uint32_t* occIdx, int occIC,
                             const float* lineV, int lineVC, const uint32_t* lineIdx, int lineIC,
                             const float* hidV, int hidVC, const uint32_t* hidIdx, int hidIC);
void     obk_viewport_set_clear(void* h, float r, float g, float b);
uint64_t obk_viewport_texture(void* h);

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
		(*C.uint32_t)(unsafe.Pointer(&meshFragSPV[0])), C.int(len(meshFragSPV)))
}

// RenderViewport renders the geometry into the offscreen target at (w,h) using the
// column-major MVP, with camPos (3 floats, world-space camera eye) supplying the PBR view
// vector. Triangle and line vertices are interleaved as 16 floats [pos.xyz, normal.xyz,
// color.rgba, metallic, roughness, emissive.rgb, mode]; indices are 0-based within their own
// vertex array.
func (win *Window) RenderViewport(w, h int, mvp []float32, camPos []float32,
	triVerts []float32, triVCount int, triIdx []uint32,
	occVerts []float32, occVCount int, occIdx []uint32,
	lineVerts []float32, lineVCount int, lineIdx []uint32,
	hidVerts []float32, hidVCount int, hidIdx []uint32,
) {
	C.obk_viewport_render(win.handle, C.int(w), C.int(h),
		(*C.float)(unsafe.Pointer(&mvp[0])), floatPtr(camPos),
		floatPtr(triVerts), C.int(triVCount), uint32Ptr(triIdx), C.int(len(triIdx)),
		floatPtr(occVerts), C.int(occVCount), uint32Ptr(occIdx), C.int(len(occIdx)),
		floatPtr(lineVerts), C.int(lineVCount), uint32Ptr(lineIdx), C.int(len(lineIdx)),
		floatPtr(hidVerts), C.int(hidVCount), uint32Ptr(hidIdx), C.int(len(hidIdx)))
}

// SetViewportClear sets the 3D pass background color (themed); it takes effect on the
// next RenderViewport.
func (w *Window) SetViewportClear(r, g, b float32) {
	C.obk_viewport_set_clear(w.handle, C.float(r), C.float(g), C.float(b))
}

// ViewportTexture returns the ImGui texture handle for the last rendered frame.
func (w *Window) ViewportTexture() uint64 { return uint64(C.obk_viewport_texture(w.handle)) }

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
