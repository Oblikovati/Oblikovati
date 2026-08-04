// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
)

// Every shipped theme must draw sketch geometry, dimensions and glyphs legibly against its OWN
// viewport background (#2023). The dark theme shipped sketch geometry at #111111 over a #141414
// background — 1.02:1, near-black on near-black — and the light theme had four tokens under
// 1.7:1, which left sketch editing there with no visible selection, hover or snap feedback.
//
// The cause was mechanical: theme/blendermap.go binds our tokens to Blender slots (sketch
// geometry ← view_3d.wire_edit), and Blender chose those values against ITS mid-grey viewport,
// not against our background. Importing a colour without the background it was chosen for is
// exactly the mistake this gate catches, so it guards the binding as much as the values.

// contrastFloor is the minimum contrast ratio a token needs against the viewport background.
// WCAG 2.1 SC 1.4.11 puts non-text UI components at 3:1; anything rendered as TEXT (a dimension's
// value) takes the 4.5:1 normal-text floor from SC 1.4.3.
type contrastFloor struct {
	token types.ThemeToken
	min   float64
	why   string
}

var viewportContrastFloors = []contrastFloor{
	{types.TokenSketchGeometry, 3.0, "the sketch linework itself"},
	{types.TokenSketchSelected, 3.0, "selection feedback"},
	{types.TokenSketchCandidate, 3.0, "hover/candidate feedback"},
	{types.TokenSnapGlyph, 3.0, "snap glyphs"},
	{types.TokenDimensionDriving, 4.5, "driving dimension text"},
	{types.TokenDimensionDriven, 4.5, "driven dimension text"},
}

// TestViewportTokensAreLegibleInEveryBuiltinTheme is the gate. It runs over Builtins() rather
// than a fixed pair, so a theme added later is covered without touching this test.
func TestViewportTokensAreLegibleInEveryBuiltinTheme(t *testing.T) {
	for _, th := range Builtins() {
		p := th.Palette()
		bg := p.Color(types.TokenViewportBg)
		for _, f := range viewportContrastFloors {
			fg := p.Color(f.token)
			if got := contrastRatio(fg, bg); got < f.min {
				t.Errorf("%s theme: %s (%s) is %.2f:1 against the viewport background %s, want >= %.1f:1 — %s is unreadable",
					th.Name(), f.token, hexOf(fg), got, hexOf(bg), f.min, f.why)
			}
		}
	}
}

// TestSketchGeometryHasHeadroom keeps the primary linework well clear of the bare minimum: it is
// the colour the user stares at for the whole sketching session, not an accent.
func TestSketchGeometryHasHeadroom(t *testing.T) {
	const want = 5.0
	for _, th := range Builtins() {
		p := th.Palette()
		got := contrastRatio(p.Color(types.TokenSketchGeometry), p.Color(types.TokenViewportBg))
		if got < want {
			t.Errorf("%s theme: sketch geometry is %.2f:1, want >= %.1f:1 for the primary linework", th.Name(), got, want)
		}
	}
}

// relativeLuminance is WCAG 2.1's L, on straight (non-premultiplied) sRGB. Alpha is ignored:
// these tokens are drawn over the background, so the opaque colour is the legibility ceiling —
// a translucent one can only be worse, never better.
func relativeLuminance(c Rgba) float64 {
	lin := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}
		return stdmath.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(float64(c.R)) + 0.7152*lin(float64(c.G)) + 0.0722*lin(float64(c.B))
}

// contrastRatio is WCAG 2.1's (L1+0.05)/(L2+0.05), lighter colour first.
func contrastRatio(a, b Rgba) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	return (stdmath.Max(la, lb) + 0.05) / (stdmath.Min(la, lb) + 0.05)
}

// hexOf renders a colour for failure messages, so a break names the offending value.
func hexOf(c Rgba) string {
	const digits = "0123456789abcdef"
	out := []byte("#......")
	for i, v := range []float32{c.R, c.G, c.B} {
		n := int(stdmath.Round(float64(v) * 255))
		n = int(stdmath.Max(0, stdmath.Min(255, float64(n))))
		out[1+i*2], out[2+i*2] = digits[n>>4], digits[n&15]
	}
	return string(out)
}
