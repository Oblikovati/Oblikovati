// SPDX-License-Identifier: GPL-2.0-only

package envimage

import "testing"

// TestMipChainReachesOnePixel checks the pyramid halves down to exactly 1×1 with the expected
// level count for a 256×128 base (max dim 256 ⇒ log2 = 8 ⇒ 9 levels).
func TestMipChainReachesOnePixel(t *testing.T) {
	chain := MipChain(presetEquirect(1)) // EnvStudio
	last := chain[len(chain)-1]
	if last.W != 1 || last.H != 1 {
		t.Errorf("last mip = %dx%d, want 1x1", last.W, last.H)
	}
	if len(chain) != 9 {
		t.Errorf("256x128 base produced %d levels, want 9", len(chain))
	}
	for i := 1; i < len(chain); i++ {
		if chain[i].W > chain[i-1].W || chain[i].H > chain[i-1].H {
			t.Errorf("level %d (%dx%d) not smaller than %d (%dx%d)",
				i, chain[i].W, chain[i].H, i-1, chain[i-1].W, chain[i-1].H)
		}
	}
}

// TestDownsampleAveragesEnergy checks a uniform image downsamples to the same uniform value
// (box filter preserves a constant), guarding the averaging math.
func TestDownsampleAveragesEnergy(t *testing.T) {
	src := newEquirect(4, 4)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.set(x, y, 0.5, 0.25, 0.75)
		}
	}
	d := downsample(src)
	r, g, b := d.At(0, 0)
	if r != 0.5 || g != 0.25 || b != 0.75 {
		t.Errorf("downsampled uniform = (%g,%g,%g), want (0.5,0.25,0.75)", r, g, b)
	}
}

// TestFlattenConcatenatesLevels checks Flatten lays out every level's pixels and dims so the
// native upload can walk them by offset.
func TestFlattenConcatenatesLevels(t *testing.T) {
	chain := MipChain(presetEquirect(1))
	u := Flatten(chain)
	if len(u.Dims) != 2*len(chain) {
		t.Fatalf("dims len = %d, want %d", len(u.Dims), 2*len(chain))
	}
	total := 0
	for _, lvl := range chain {
		total += lvl.W * lvl.H * 4
	}
	if len(u.Data) != total {
		t.Errorf("flattened data = %d floats, want %d", len(u.Data), total)
	}
}
