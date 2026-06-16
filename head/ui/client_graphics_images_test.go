//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/model/clientgraphics"
)

// TestDecodeImageRGBA writes a small PNG and checks it decodes to tightly-packed RGBA bytes.
func TestDecodeImageRGBA(t *testing.T) {
	path := filepath.Join(t.TempDir(), "swatch.png")
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	f.Close()

	pix, w, h, err := decodeImageRGBA(path)
	if err != nil {
		t.Fatalf("decodeImageRGBA: %v", err)
	}
	if w != 2 || h != 3 || len(pix) != 2*3*4 {
		t.Errorf("decoded %dx%d (%d bytes), want 2x3 (24 bytes)", w, h, len(pix))
	}
	if _, _, _, err := decodeImageRGBA(filepath.Join(t.TempDir(), "missing.png")); err == nil {
		t.Error("decoding a missing file should error")
	}
}

// TestBillboardPixels converts model dimensions to pixels via world-per-pixel, with a fallback
// for a degenerate size.
func TestBillboardPixels(t *testing.T) {
	w, h := billboardPixels(clientgraphics.ImageBillboard{Width: 4, Height: 2}, 0.1)
	if w != 40 || h != 20 {
		t.Errorf("billboardPixels(4x2 @0.1) = %vx%v, want 40x20", w, h)
	}
	if dw, dh := billboardPixels(clientgraphics.ImageBillboard{}, 0.1); dw != 64 || dh != 64 {
		t.Errorf("degenerate billboard = %vx%v, want the 64x64 fallback", dw, dh)
	}
}
