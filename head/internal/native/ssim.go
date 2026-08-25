// SPDX-License-Identifier: GPL-2.0-only

package native

import "image"

// ssim (M45-F05 PBI-353, ADR-0053): a windowed structural-similarity index between two
// equal-sized images, on ITU-R 601 luma in [0,1] — the perceptual metric
// architecture/testing/00-renderer-oracle-pipeline.md's Tier-4 Blender oracle calls for
// ("SSIM + FLIP"). This implements SSIM only (a uniform 8x8 window, not FLIP's full
// perceptual color-difference model) — a real, standard, well-understood metric, not a
// placeholder; FLIP is a documented follow-up if SSIM alone proves too permissive for
// some future golden. No external dependency, pure Go, so this and its test run
// anywhere `go test` does.
//
// Returns 1 for identical images, trending toward 0 (and below, for anti-correlated
// images) as structural similarity degrades. Panics if the images differ in size — a
// caller error, not a runtime condition to recover from.
func ssim(a, b image.Image) float64 {
	bounds := a.Bounds()
	if b.Bounds() != bounds {
		panic("native: ssim: images have different bounds")
	}
	w, h := bounds.Dx(), bounds.Dy()
	la, lb := lumaGrid(a, w, h), lumaGrid(b, w, h)

	const window = 8
	const c1, c2 = 0.01 * 0.01, 0.03 * 0.03 // C1=(0.01*L)^2, C2=(0.03*L)^2, L=1 (normalized luma)
	var sum float64
	windows := 0
	for y := 0; y+window <= h; y += window {
		for x := 0; x+window <= w; x += window {
			sum += ssimWindow(la, lb, w, x, y, window, c1, c2)
			windows++
		}
	}
	if windows == 0 {
		// Image smaller than one window: treat the whole image as a single window.
		return ssimWindow(la, lb, w, 0, 0, min(w, h), c1, c2)
	}
	return sum / float64(windows)
}

// ssimWindow computes SSIM over one window×window block starting at (x0,y0).
func ssimWindow(la, lb []float64, stride, x0, y0, window int, c1, c2 float64) float64 {
	n := float64(window * window)
	var sumA, sumB float64
	for y := y0; y < y0+window; y++ {
		for x := x0; x < x0+window; x++ {
			i := y*stride + x
			sumA += la[i]
			sumB += lb[i]
		}
	}
	muA, muB := sumA/n, sumB/n

	var varA, varB, covAB float64
	for y := y0; y < y0+window; y++ {
		for x := x0; x < x0+window; x++ {
			i := y*stride + x
			da, db := la[i]-muA, lb[i]-muB
			varA += da * da
			varB += db * db
			covAB += da * db
		}
	}
	varA, varB, covAB = varA/n, varB/n, covAB/n

	num := (2*muA*muB + c1) * (2*covAB + c2)
	den := (muA*muA + muB*muB + c1) * (varA + varB + c2)
	return num / den
}

// lumaGrid extracts ITU-R 601 luma in [0,1] for every pixel, row-major.
func lumaGrid(img image.Image, w, h int) []float64 {
	out := make([]float64, w*h)
	b := img.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			out[y*w+x] = (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 65535
		}
	}
	return out
}
