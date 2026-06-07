//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

/*
void obk_head_set_ui_font(const unsigned char* data, int len, float sizePx);
*/
import "C"

import (
	_ "embed"
	"unsafe"
)

// uiFontTTF is the embedded UI typeface: Liberation Sans, a metrically Helvetica/Arial-
// compatible face under the SIL Open Font License (see fonts/LICENSE-LiberationSans.txt).
// It is bundled so the UI renders in a Helvetica-equivalent on every OS, including those
// without Helvetica installed. Real Helvetica is proprietary and cannot be redistributed.
//
//go:embed fonts/LiberationSans-Regular.ttf
var uiFontTTF []byte

// uiFontSizePx is the default UI font size in pixels.
const uiFontSizePx = 16.0

// SetUIFont installs the embedded UI font at sizePx, replacing ImGui's built-in default.
// Call once after the window/ImGui context exists, before the first frame. The C side
// copies the bytes, so the embedded slice is not retained across the call.
func (w *Window) SetUIFont(sizePx float32) {
	if len(uiFontTTF) == 0 {
		return
	}
	C.obk_head_set_ui_font((*C.uchar)(unsafe.Pointer(&uiFontTTF[0])), C.int(len(uiFontTTF)), C.float(sizePx))
}
