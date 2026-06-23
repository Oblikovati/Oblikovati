//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"image"
	"testing"
)

// TestSetIcon exercises the GLFW window-icon path: it opens a real window (skipped when
// no display/Vulkan is available, e.g. a GPU-less local run) and sets multiple candidate
// sizes plus the empty no-op case, verifying the cgo marshaling + free path doesn't panic.
func TestSetIcon(t *testing.T) {
	win, err := CreateWindow(320, 240, "obk-icon-test")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()

	win.SetIcon() // empty: early return, must be a no-op
	win.SetIcon(solidRGBA(16), solidRGBA(32), solidRGBA(48))
}

// solidRGBA returns an opaque-white px×px RGBA image.
func solidRGBA(px int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, px, px))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	return img
}
