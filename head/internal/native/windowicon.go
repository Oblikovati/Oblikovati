//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

/*
#include <stdlib.h>
void obk_head_set_icon(void* h, int count, const int* sizes, const unsigned char* const* pixels);
*/
import "C"

import (
	"image"
	"unsafe"
)

// SetIcon sets the window and taskbar icon from one or more candidate images; the
// window manager picks the resolution it needs (so pass a range, e.g. 16…256). Each
// image must be a tightly packed square RGBA8 bitmap (image.NewRGBA output). On macOS
// this is a no-op — the dock uses the .app bundle icon (set by package-macos.sh).
//
//	win.SetIcon(img16, img32, img48, img256)
func (w *Window) SetIcon(imgs ...*image.RGBA) {
	if len(imgs) == 0 {
		return
	}
	sizes := make([]C.int, len(imgs))
	ptrs := make([]*C.uchar, len(imgs))
	defer func() {
		for _, p := range ptrs {
			if p != nil {
				C.free(unsafe.Pointer(p))
			}
		}
	}()
	for i, im := range imgs {
		sizes[i] = C.int(im.Bounds().Dx())
		ptrs[i] = (*C.uchar)(C.CBytes(im.Pix)) // copied to C memory; GLFW copies again, then we free
	}
	C.obk_head_set_icon(w.handle, C.int(len(imgs)), &sizes[0], (**C.uchar)(unsafe.Pointer(&ptrs[0])))
}
