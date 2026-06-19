//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

/*
void obk_head_set_ui_font(const unsigned char* data, int len, float sizePx);
void obk_head_set_mono_font(const unsigned char* data, int len, float sizePx);
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

// monoFontTTF is the embedded fixed-width face for the Script Console code editor: Liberation
// Mono, same SIL Open Font License family as the UI face (see fonts/LICENSE-LiberationMono.txt).
// A monospace face gives the editor aligned glyph columns for the gutter, caret and selection.
//
//go:embed fonts/LiberationMono-Regular.ttf
var monoFontTTF []byte

// uiFontSizePx is the default UI font size in pixels; monoFontSizePx sizes the editor face.
const (
	uiFontSizePx   = 16.0
	monoFontSizePx = 15.0
)

// SetUIFont installs the embedded UI font at sizePx, replacing ImGui's built-in default.
// Call once after the window/ImGui context exists, before the first frame. The C side
// copies the bytes, so the embedded slice is not retained across the call.
func (w *Window) SetUIFont(sizePx float32) {
	if len(uiFontTTF) == 0 {
		return
	}
	C.obk_head_set_ui_font((*C.uchar)(unsafe.Pointer(&uiFontTTF[0])), C.int(len(uiFontTTF)), C.float(sizePx))
}

// SetMonoFont adds the embedded fixed-width editor face to the atlas at sizePx. It must be
// called AFTER SetUIFont (which clears the atlas) and before the first frame; the C side copies
// the bytes, so the embedded slice is not retained.
func (w *Window) SetMonoFont(sizePx float32) {
	if len(monoFontTTF) == 0 {
		return
	}
	C.obk_head_set_mono_font((*C.uchar)(unsafe.Pointer(&monoFontTTF[0])), C.int(len(monoFontTTF)), C.float(sizePx))
}
