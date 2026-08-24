// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"image"
	"image/color"
	"math/rand"
	"testing"
)

func gradientImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(255 * (x + y) / (w + h))
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return img
}

func TestSSIMIdenticalImagesScoreOne(t *testing.T) {
	img := gradientImage(64, 64)
	if got := ssim(img, img); got < 0.999 {
		t.Errorf("ssim(img, img) = %v, want ~1", got)
	}
}

func TestSSIMRandomNoiseVsConstantScoresLow(t *testing.T) {
	const w, h = 64, 64
	rng := rand.New(rand.NewSource(1))
	noise := image.NewRGBA(image.Rect(0, 0, w, h))
	flat := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			noise.Set(x, y, color.RGBA{uint8(rng.Intn(256)), uint8(rng.Intn(256)), uint8(rng.Intn(256)), 255})
			flat.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	got := ssim(noise, flat)
	if got > 0.5 {
		t.Errorf("ssim(random noise, flat gray) = %v, want a low score (structurally unrelated)", got)
	}
}

func TestSSIMSlightlyPerturbedImageScoresHigh(t *testing.T) {
	const w, h = 64, 64
	base := gradientImage(w, h)
	perturbed := image.NewRGBA(image.Rect(0, 0, w, h))
	rng := rand.New(rand.NewSource(2))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := base.At(x, y).RGBA()
			jitter := int32(rng.Intn(7) - 3) // +/- ~1% noise
			perturbed.Set(x, y, color.RGBA{
				clampU8(int32(r>>8) + jitter), clampU8(int32(g>>8) + jitter), clampU8(int32(b>>8) + jitter), uint8(a >> 8),
			})
		}
	}
	got := ssim(base, perturbed)
	if got < 0.9 {
		t.Errorf("ssim(gradient, slightly perturbed gradient) = %v, want >= 0.9 (small noise should score high)", got)
	}
}

func clampU8(v int32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func TestSSIMPanicsOnMismatchedSize(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("ssim on mismatched image sizes did not panic")
		}
	}()
	ssim(gradientImage(32, 32), gradientImage(64, 64))
}
