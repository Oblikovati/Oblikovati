//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

/*
#include <stdint.h>

uint64_t obk_create_texture(void* h, const unsigned char* rgba, int w, int hh);
int      obk_update_texture(void* h, uint64_t tex, const unsigned char* rgba, int w, int hh);
void     obk_destroy_texture(void* h, uint64_t tex);

int obk_ig_image_button(const char* id, unsigned long long tex, float w, float h,
                        float r, float g, float b, float a);
*/
import "C"

import "unsafe"

// CreateTexture uploads a width×height RGBA8 bitmap (row-major, 4 bytes/pixel) into a
// device-local Vulkan image and returns its ImGui texture handle for ImageButton/Image,
// or 0 on failure (e.g. a bad size or a full descriptor pool). The pixels are copied,
// so the caller may reuse rgba afterwards. Pair every success with DestroyTexture.
func (w *Window) CreateTexture(rgba []byte, width, height int) uint64 {
	if len(rgba) < width*height*4 || width <= 0 || height <= 0 {
		return 0
	}
	return uint64(C.obk_create_texture(w.handle,
		(*C.uchar)(unsafe.Pointer(&rgba[0])), C.int(width), C.int(height)))
}

// UpdateTexture re-uploads a width×height RGBA8 bitmap into an EXISTING texture (from
// CreateTexture) in place — cheaper than destroying and recreating one every frame, for
// a texture whose pixel content changes but not its size (the progressive Realistic-
// mode path tracer's accumulated image, M45-F05 PBI-350). width/height must match the
// size tex was created at. Reports whether the update succeeded.
func (w *Window) UpdateTexture(tex uint64, rgba []byte, width, height int) bool {
	if len(rgba) < width*height*4 || width <= 0 || height <= 0 || tex == 0 {
		return false
	}
	return C.obk_update_texture(w.handle, C.uint64_t(tex),
		(*C.uchar)(unsafe.Pointer(&rgba[0])), C.int(width), C.int(height)) == 0
}

// DestroyTexture frees a texture created by CreateTexture (no-op on a zero handle).
func (w *Window) DestroyTexture(tex uint64) {
	C.obk_destroy_texture(w.handle, C.uint64_t(tex))
}

// ImageButton draws a clickable icon of size w×h for the texture, tinted by the RGBA
// color so the same monochrome glyph recolors per theme. id must be unique per button
// (use the command id). Returns true on the frame it is clicked.
func ImageButton(id string, tex uint64, w, h float32, tint [4]float32) bool {
	c, free := cstr(id)
	defer free()
	return C.obk_ig_image_button(c, C.ulonglong(tex), C.float(w), C.float(h),
		C.float(tint[0]), C.float(tint[1]), C.float(tint[2]), C.float(tint[3])) != 0
}
