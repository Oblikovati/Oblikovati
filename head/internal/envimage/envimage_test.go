// SPDX-License-Identifier: GPL-2.0-only

package envimage

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati/renderer"
)

// TestResolvePresetsProduceImages checks every non-None preset resolves to a correctly-sized,
// 2:1 equirect with finite, non-negative pixels.
func TestResolvePresetsProduceImages(t *testing.T) {
	for _, opt := range renderer.EnvironmentGallery() {
		if opt.Preset == renderer.EnvNone {
			continue
		}
		img, ok, err := Resolve(renderer.Environment{Preset: opt.Preset})
		if err != nil || !ok {
			t.Fatalf("Resolve(%v) ok=%v err=%v", opt.Preset, ok, err)
		}
		if img.W != 2*img.H || img.W != presetW {
			t.Errorf("%v size = %dx%d, want %dx%d (2:1)", opt.Preset, img.W, img.H, presetW, presetH)
		}
		if len(img.Pixels) != img.W*img.H*4 {
			t.Fatalf("%v has %d floats, want %d", opt.Preset, len(img.Pixels), img.W*img.H*4)
		}
		for i, p := range img.Pixels {
			if p < 0 || math.IsNaN(float64(p)) {
				t.Fatalf("%v pixel %d = %g (want finite ≥ 0)", opt.Preset, i, p)
			}
		}
	}
}

// TestResolveNoneIsInactive checks EnvNone with no file resolves to ok=false (no IBL).
func TestResolveNoneIsInactive(t *testing.T) {
	if _, ok, err := Resolve(renderer.Environment{Preset: renderer.EnvNone}); ok || err != nil {
		t.Errorf("Resolve(None) = ok %v err %v, want inactive", ok, err)
	}
}

// TestOutdoorsTopBrighterThanGround sanity-checks the gradient: the zenith row is brighter than
// the nadir row (so the IBL has a recognizable up-light).
func TestOutdoorsTopBrighterThanGround(t *testing.T) {
	img := presetEquirect(renderer.EnvOutdoors)
	_, tg, tb := img.At(0, 0)
	_, bg, bb := img.At(0, img.H-1)
	if tg+tb <= bg+bb {
		t.Errorf("outdoors zenith (%g,%g) not brighter than nadir (%g,%g)", tg, tb, bg, bb)
	}
}

// TestDecodeHDRRoundTrip writes a tiny flat RGBE file and decodes it, checking the shared
// exponent expands correctly (a mid-grey pixel and a black pixel).
func TestDecodeHDRRoundTrip(t *testing.T) {
	// 2×1 image: pixel0 RGBE (128,128,128,128) → 0.5·ldexp(1,128-136)=… ; pixel1 black (e=0).
	hdr := []byte("#?RADIANCE\nFORMAT=32-bit_rle_rgbe\n\n-Y 1 +X 2\n")
	hdr = append(hdr, 128, 128, 128, 128, 0, 0, 0, 0) // flat scanline (w<8 ⇒ never RLE)
	path := filepath.Join(t.TempDir(), "tiny.hdr")
	if err := os.WriteFile(path, hdr, 0o644); err != nil {
		t.Fatal(err)
	}
	img, err := DecodeHDR(path)
	if err != nil {
		t.Fatalf("DecodeHDR: %v", err)
	}
	if img.W != 2 || img.H != 1 {
		t.Fatalf("decoded size %dx%d, want 2x1", img.W, img.H)
	}
	r0, g0, b0 := img.At(0, 0)
	if r0 <= 0 || r0 != g0 || g0 != b0 {
		t.Errorf("pixel0 = (%g,%g,%g), want equal positive channels", r0, g0, b0)
	}
	if r1, _, _ := img.At(1, 0); r1 != 0 {
		t.Errorf("pixel1 (e=0) = %g, want 0", r1)
	}
}

// TestResolveFileDecodes checks Resolve routes a FilePath through the HDR decoder.
func TestResolveFileDecodes(t *testing.T) {
	hdr := []byte("#?RADIANCE\n\n-Y 1 +X 2\n")
	hdr = append(hdr, 200, 100, 50, 130, 200, 100, 50, 130)
	path := filepath.Join(t.TempDir(), "env.hdr")
	if err := os.WriteFile(path, hdr, 0o644); err != nil {
		t.Fatal(err)
	}
	img, ok, err := Resolve(renderer.Environment{FilePath: path})
	if err != nil || !ok || img.W != 2 {
		t.Fatalf("Resolve(file) ok=%v err=%v w=%d", ok, err, img.W)
	}
}
