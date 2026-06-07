// SPDX-License-Identifier: GPL-2.0-only

package icon

import "testing"

// tinySVG is a filled square covering the whole viewBox, so a correct rasterization
// leaves every pixel fully covered.
const tinySVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect x="0" y="0" width="10" height="10" fill="#000"/></svg>`

func TestRasterizeProducesTintableMask(t *testing.T) {
	img, err := Rasterize([]byte(tinySVG), 16)
	if err != nil {
		t.Fatalf("Rasterize: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 16 || b.Dy() != 16 {
		t.Fatalf("size = %dx%d, want 16x16", b.Dx(), b.Dy())
	}
	// A fully covered pixel must be opaque white (RGB forced to 255, alpha = coverage).
	off := img.PixOffset(8, 8)
	px := img.Pix[off : off+4]
	if px[0] != 255 || px[1] != 255 || px[2] != 255 {
		t.Errorf("center RGB = %v, want forced white [255 255 255]", px[:3])
	}
	if px[3] == 0 {
		t.Error("center alpha = 0, want coverage > 0")
	}
}

// TestRasterizeNormalizesGlyphSize checks that glyphs with very different internal
// margins (a tall block, a wide stadium, a thin centreline, a small offset face) all
// rasterize to the same visual size: each one's longest drawn side fills ~contentFraction
// of the output. This is the regression guard for "some icons are too small".
func TestRasterizeNormalizesGlyphSize(t *testing.T) {
	px := 64
	want := int(contentFraction * float64(px))
	for _, key := range []string{"extrude", "slot", "centerline", "move-face", "combine"} {
		svg, ok := SVG(key)
		if !ok {
			t.Fatalf("bundled icon %q missing", key)
		}
		img, err := Rasterize(svg, px)
		if err != nil {
			t.Fatalf("Rasterize(%q): %v", key, err)
		}
		b := alphaBounds(img)
		longest := b.Dx()
		if b.Dy() > longest {
			longest = b.Dy()
		}
		if diff := longest - want; diff < -5 || diff > 5 {
			t.Errorf("%q glyph longest side = %d px, want ~%d (normalized to contentFraction)", key, longest, want)
		}
	}
}

func TestRasterizeRejectsNonPositivePx(t *testing.T) {
	if _, err := Rasterize([]byte(tinySVG), 0); err == nil {
		t.Error("Rasterize(px=0) returned nil error, want failure")
	}
}

func TestRasterizeRejectsBadSVG(t *testing.T) {
	// Malformed XML (unbalanced angle brackets) must fail the tokenizer; oksvg
	// tolerates non-SVG plain text, so a stricter input is needed to prove errors
	// propagate.
	if _, err := Rasterize([]byte("<svg><<<"), 16); err == nil {
		t.Error("Rasterize(malformed XML) returned nil error, want parse failure")
	}
}

func TestSVGLookup(t *testing.T) {
	if _, ok := SVG("extrude"); !ok {
		t.Error(`SVG("extrude") not found, want bundled`)
	}
	if _, ok := SVG("does-not-exist"); ok {
		t.Error(`SVG("does-not-exist") found, want false`)
	}
}

// TestEveryBundledIconRasterizes guards that all shipped glyphs are valid SVG the
// rasterizer accepts (a malformed asset would otherwise only surface at runtime as a
// silently-missing icon).
func TestEveryBundledIconRasterizes(t *testing.T) {
	keys := Keys()
	if len(keys) == 0 {
		t.Fatal("no icons bundled")
	}
	for _, key := range keys {
		svg, ok := SVG(key)
		if !ok {
			t.Errorf("Keys() listed %q but SVG() did not find it", key)
			continue
		}
		if _, err := Rasterize(svg, 32); err != nil {
			t.Errorf("Rasterize(%q): %v", key, err)
		}
	}
}
