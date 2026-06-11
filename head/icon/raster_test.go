// SPDX-License-Identifier: GPL-2.0-only

package icon

import (
	"image"
	"sort"
	"testing"
)

// quadrantSVG paints one role per quadrant of the viewBox (filled squares, no
// stroke), so each role's mask and composed color can be checked at a known pixel.
const quadrantSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" fill="none" stroke="none">
  <rect x="0" y="0" width="5" height="5" fill="#00ff00"/>
  <rect x="5" y="0" width="5" height="5" fill="#0000ff"/>
  <rect x="0" y="5" width="5" height="5" fill="#ff0000"/>
  <rect x="5" y="5" width="5" height="5" fill="#000000"/>
</svg>`

// quadrantCenter returns the mask index of the centre of each role's quadrant for a
// px-sized mask of quadrantSVG (background TL, tertiary TR, secondary BL, primary BR).
func quadrantCenter(px int, r Role) int {
	q := px / 4
	switch r {
	case RoleBackground:
		return q*px + q
	case RoleTertiary:
		return q*px + 3*q
	case RoleSecondary:
		return 3*q*px + q
	default:
		return 3*q*px + 3*q
	}
}

func TestRasterizeRolesSplitsBySentinel(t *testing.T) {
	masks, err := RasterizeRoles([]byte(quadrantSVG), 16)
	if err != nil {
		t.Fatalf("RasterizeRoles: %v", err)
	}
	for r := Role(0); r < RoleCount; r++ {
		for other := Role(0); other < RoleCount; other++ {
			cov := masks.cover[other][quadrantCenter(16, r)]
			if other == r && cov < 200 {
				t.Errorf("role %s mask misses its own quadrant (coverage %d)", r, cov)
			}
			if other != r && cov > 30 {
				t.Errorf("role %s mask bleeds into %s's quadrant (coverage %d)", other, r, cov)
			}
		}
	}
}

// All role passes must share one content box: a mask-only-knows-its-own-shapes crop
// would scale each layer differently and the colors would drift apart.
func TestRoleLayersStayRegistered(t *testing.T) {
	masks, err := RasterizeRoles([]byte(quadrantSVG), 16)
	if err != nil {
		t.Fatalf("RasterizeRoles: %v", err)
	}
	// The four quadrants tile the full content box; with shared registration the
	// primary quadrant must stay in the bottom-right and NOT be rescaled to fill the
	// output (its own bounds would be just its quadrant).
	if cov := masks.cover[RolePrimary][quadrantCenter(16, RoleBackground)]; cov > 30 {
		t.Errorf("primary layer rescaled into the background quadrant (coverage %d): bounds not shared", cov)
	}
}

func TestComposeColorsEachRole(t *testing.T) {
	masks, err := RasterizeRoles([]byte(quadrantSVG), 16)
	if err != nil {
		t.Fatalf("RasterizeRoles: %v", err)
	}
	colors := RoleColors{}
	colors[RoleBackground] = [4]float32{0, 0, 0, 1}
	colors[RoleTertiary] = [4]float32{0, 0, 1, 1}
	colors[RoleSecondary] = [4]float32{0, 1, 0, 1}
	colors[RolePrimary] = [4]float32{1, 0, 0, 1}
	img := masks.Compose(colors)
	checks := []struct {
		role Role
		want [3]byte
	}{
		{RoleBackground, [3]byte{0, 0, 0}},
		{RoleTertiary, [3]byte{0, 0, 255}},
		{RoleSecondary, [3]byte{0, 255, 0}},
		{RolePrimary, [3]byte{255, 0, 0}},
	}
	for _, c := range checks {
		i := quadrantCenter(16, c.role) * 4
		got := [3]byte{img.Pix[i], img.Pix[i+1], img.Pix[i+2]}
		if got != c.want {
			t.Errorf("composed %s pixel = %v, want %v", c.role, got, c.want)
		}
		if img.Pix[i+3] < 200 {
			t.Errorf("composed %s pixel alpha = %d, want opaque", c.role, img.Pix[i+3])
		}
	}
}

// A translucent role color must keep its alpha in the composite (the background plate
// theme color is translucent by design).
func TestComposeKeepsTranslucentAlpha(t *testing.T) {
	masks, err := RasterizeRoles([]byte(quadrantSVG), 16)
	if err != nil {
		t.Fatalf("RasterizeRoles: %v", err)
	}
	colors := RoleColors{}
	colors[RoleBackground] = [4]float32{1, 1, 1, 0.25}
	img := masks.Compose(colors)
	a := img.Pix[quadrantCenter(16, RoleBackground)*4+3]
	if a < 48 || a > 80 { // ~0.25 of full coverage
		t.Errorf("translucent background alpha = %d, want ~64", a)
	}
}

func TestRasterizeNormalizesGlyphSize(t *testing.T) {
	px := 64
	want := int(contentFraction * float64(px))
	for _, key := range []string{"extrude", "slot", "centerline", "move-face", "combine"} {
		svg, ok := SVG(key)
		if !ok {
			t.Fatalf("bundled icon %q missing", key)
		}
		masks, err := RasterizeRoles(svg, px)
		if err != nil {
			t.Fatalf("RasterizeRoles(%q): %v", key, err)
		}
		b := unionBounds(masks, px)
		longest := b.Dx()
		if b.Dy() > longest {
			longest = b.Dy()
		}
		if diff := longest - want; diff < -5 || diff > 5 {
			t.Errorf("%q glyph longest side = %d px, want ~%d (normalized to contentFraction)", key, longest, want)
		}
	}
}

// unionBounds is the tight box of every role's coverage in a mask set.
func unionBounds(m *RoleMasks, px int) image.Rectangle {
	minX, minY, maxX, maxY := px, px, -1, -1
	for r := Role(0); r < RoleCount; r++ {
		for i, v := range m.cover[r] {
			if v <= alphaThreshold {
				continue
			}
			x, y := i%px, i/px
			minX, maxX = min(minX, x), max(maxX, x)
			minY, maxY = min(minY, y), max(maxY, y)
		}
	}
	if maxX < 0 {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

func TestRasterizeRejectsNonPositivePx(t *testing.T) {
	if _, err := RasterizeRoles([]byte(quadrantSVG), 0); err == nil {
		t.Error("RasterizeRoles(px=0) returned nil error, want failure")
	}
}

func TestRasterizeRejectsBadSVG(t *testing.T) {
	// Malformed XML (unbalanced angle brackets) must fail the tokenizer; oksvg
	// tolerates non-SVG plain text, so a stricter input is needed to prove errors
	// propagate.
	if _, err := RasterizeRoles([]byte("<svg><<<"), 16); err == nil {
		t.Error("RasterizeRoles(malformed XML) returned nil error, want parse failure")
	}
}

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
		if _, err := RasterizeRoles(svg, 32); err != nil {
			t.Errorf("RasterizeRoles(%q): %v", key, err)
		}
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

func TestUsedPaintsNormalizesAndInherits(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" fill="none" stroke="#000">
	  <line x1="0" y1="0" x2="9" y2="9"/>
	  <g stroke="#F00"><path d="M0 9 L9 0"/></g>
	  <circle cx="5" cy="5" r="2" stroke="none" fill="blue"/>
	</svg>`
	paints, err := UsedPaints([]byte(svg))
	if err != nil {
		t.Fatalf("UsedPaints: %v", err)
	}
	sort.Strings(paints)
	want := []string{"#000000", "#0000ff", "#ff0000"}
	if len(paints) != len(want) {
		t.Fatalf("paints = %v, want %v", paints, want)
	}
	for i := range want {
		if paints[i] != want[i] {
			t.Fatalf("paints = %v, want %v", paints, want)
		}
	}
}
