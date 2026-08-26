//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import "testing"

// TestUpdateTextureRoundTrip is a smoke test for M45-F05 PBI-350's progressive
// Realistic-mode texture presentation: create a texture, update its pixels in place
// several times (as repeated accumulation frames would), and destroy it — proving the
// in-place update path works without crashing or leaking, independent of any read-back
// mechanism (there is none for a plain sampled texture, unlike the viewport render
// target's ReadbackViewport — full pixel-correctness is covered where the accumulated
// image is actually composited into the viewport).
func TestUpdateTextureRoundTrip(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (texture update test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	const width, height = 8, 8
	red := make([]byte, width*height*4)
	for i := range width * height {
		red[i*4], red[i*4+1], red[i*4+2], red[i*4+3] = 255, 0, 0, 255
	}
	tex := w.CreateTexture(red, width, height)
	if tex == 0 {
		t.Fatal("CreateTexture returned 0")
	}
	defer w.DestroyTexture(tex)

	blue := make([]byte, width*height*4)
	for i := range width * height {
		blue[i*4], blue[i*4+1], blue[i*4+2], blue[i*4+3] = 0, 0, 255, 255
	}
	for i := range 5 {
		if !w.UpdateTexture(tex, blue, width, height) {
			t.Fatalf("UpdateTexture failed on iteration %d", i)
		}
	}

	if w.UpdateTexture(0, blue, width, height) {
		t.Error("UpdateTexture with a zero handle reported success")
	}
	if w.UpdateTexture(tex, blue, width, height+1) {
		t.Error("UpdateTexture with a buffer too short for the claimed size reported success")
	}
}
