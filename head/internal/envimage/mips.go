// SPDX-License-Identifier: GPL-2.0-only

package envimage

// MipChain builds a box-filtered mip pyramid from a base equirect down to 1×1. The Realistic
// shader samples higher mips for rougher reflections (and a high mip approximates the diffuse
// irradiance), so the blur is precomputed here rather than relying on GPU blit-format support
// (some drivers lack linear-blit for wide float formats). Generating it in pure Go keeps it
// unit-testable (ADR-0014).
func MipChain(base Equirect) []Equirect {
	levels := []Equirect{base}
	cur := base
	for cur.W > 1 || cur.H > 1 {
		cur = downsample(cur)
		levels = append(levels, cur)
	}
	return levels
}

// downsample halves each dimension (min 1) by averaging the covered 2×2 source block, clamping
// at edges so odd dimensions are handled without sampling out of bounds.
func downsample(src Equirect) Equirect {
	w := max1(src.W / 2)
	h := max1(src.H / 2)
	dst := newEquirect(w, h)
	for y := range h {
		for x := range w {
			r, g, b := avg2x2(src, x*2, y*2)
			dst.set(x, y, r, g, b)
		}
	}
	return dst
}

// avg2x2 averages the 2×2 source block whose top-left is (sx,sy), clamping to the source bounds.
func avg2x2(src Equirect, sx, sy int) (float32, float32, float32) {
	var r, g, b float32
	for dy := range 2 {
		for dx := range 2 {
			pr, pg, pb := src.At(clampi(sx+dx, src.W-1), clampi(sy+dy, src.H-1))
			r, g, b = r+pr, g+pg, b+pb
		}
	}
	return r / 4, g / 4, b / 4
}

// Upload is a mip chain flattened for the cgo bridge: all levels' RGBA float32 concatenated,
// plus the per-level dimensions (w,h interleaved) so the native side computes copy offsets.
type Upload struct {
	Data []float32
	Dims []int32 // 2 per level: w0,h0,w1,h1,…
}

// Flatten concatenates a mip chain into a single [Upload] for the GPU.
func Flatten(chain []Equirect) Upload {
	var u Upload
	for _, lvl := range chain {
		u.Data = append(u.Data, lvl.Pixels...)
		u.Dims = append(u.Dims, int32(lvl.W), int32(lvl.H))
	}
	return u
}

func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}

func clampi(v, hi int) int {
	if v < 0 {
		return 0
	}
	if v > hi {
		return hi
	}
	return v
}
