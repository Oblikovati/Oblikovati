// SPDX-License-Identifier: GPL-2.0-only

package icon

import (
	"image"
	"testing"
)

// fullTestSVG is a solid red square filling its viewBox, so a successful render leaves an
// opaque centre pixel regardless of the output size.
var fullTestSVG = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="#ff0000"/></svg>`)

func TestRenderFullSizeAndContent(t *testing.T) {
	img, err := RenderFull(fullTestSVG, 24)
	if err != nil {
		t.Fatalf("RenderFull: %v", err)
	}
	if got := img.Bounds(); got != image.Rect(0, 0, 24, 24) {
		t.Fatalf("bounds = %v, want 0,0,24,24", got)
	}
	if _, _, _, a := img.At(12, 12).RGBA(); a == 0 {
		t.Fatal("centre pixel is transparent — the square did not draw")
	}
}

func TestRenderFullRejectsNonPositiveSize(t *testing.T) {
	if _, err := RenderFull(fullTestSVG, 0); err == nil {
		t.Fatal("RenderFull(_, 0) should error")
	}
}

func TestRenderFullRejectsInvalidSVG(t *testing.T) {
	if _, err := RenderFull([]byte("<<<not xml>>>"), 16); err == nil {
		t.Fatal("RenderFull on invalid SVG should error")
	}
}
