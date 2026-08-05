// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/theme/blenderxml"
)

// Every token must be authored by exactly one binding — direct or derived, never both,
// never neither — so write-back is unambiguous and resolvePalette is complete.
func TestEveryTokenHasExactlyOneBinding(t *testing.T) {
	for _, tok := range types.AllThemeTokens() {
		_, direct := directBindings[tok]
		_, derived := derivedBindings[tok]
		if direct == derived {
			t.Errorf("token %q: direct=%v derived=%v, want exactly one binding", tok, direct, derived)
		}
	}
}

// Direct bindings must target unique attributes: two tokens sharing one Blender slot
// would fight over it on write-back.
func TestDirectBindingPathsUnique(t *testing.T) {
	seen := map[string]Token{}
	for tok, b := range directBindings {
		key := strings.Join(b.path, "/") + "@" + b.attr
		if other, dup := seen[key]; dup {
			t.Errorf("tokens %q and %q both bind %s", tok, other, key)
		}
		seen[key] = tok
	}
}

// Derived tokens must chain off direct ones only — a derived base would make resolve
// order matter. A PICKED binding names no base: it is a colour the product decides, so there is
// nothing to resolve and nothing to order.
func TestDerivedBasesAreDirect(t *testing.T) {
	for tok, d := range derivedBindings {
		if d.pick != nil {
			continue
		}
		if _, ok := directBindings[d.base]; !ok {
			t.Errorf("derived token %q has non-direct base %q", tok, d.base)
		}
	}
}

// TestPickedBindingsIgnoreTheFile: a picked token resolves to its product colour for EVERY theme,
// including a custom one saved before the token existed — deriving it from an unrelated Blender
// field instead made such a theme paint it whatever that field happened to hold (#2032). The
// shade follows the viewport background, so the two shipped themes must NOT agree on it.
func TestPickedBindingsIgnoreTheFile(t *testing.T) {
	seen := map[Rgba]string{}
	for _, th := range Builtins() {
		got := th.Color(types.TokenDimensionSketch)
		if got != sketchDimensionGreen(th.Palette()) {
			t.Errorf("%s theme resolved the sketch dimension to %v, want the picked colour", th.Name(), got)
		}
		if other, dup := seen[got]; dup {
			t.Errorf("%s and %s got the same shade %v — the pick is ignoring the background", th.Name(), other, got)
		}
		seen[got] = th.Name()
	}
}

// Both embedded Blender files must resolve every direct binding — this is the guard
// that a re-export from a future Blender version keeps the attributes the mapping
// depends on.
func TestEmbeddedThemesResolveCompletely(t *testing.T) {
	for name, data := range map[string][]byte{"dark": darkXML, "light": lightXML} {
		doc, err := blenderxml.Parse(data)
		if err != nil {
			t.Fatalf("%s.xml: %v", name, err)
		}
		p, err := resolvePalette(doc)
		if err != nil {
			t.Fatalf("%s.xml: %v", name, err)
		}
		for _, tok := range types.AllThemeTokens() {
			if _, ok := p[tok]; !ok {
				t.Errorf("%s.xml: token %q unresolved", name, tok)
			}
		}
	}
}

// A document missing mapped slots must be rejected loudly, naming what is absent, so a
// hand-trimmed theme file fails at load rather than rendering fallback magenta.
func TestResolvePaletteNamesMissingAttributes(t *testing.T) {
	doc, err := blenderxml.Parse([]byte(`<bpy><Theme></Theme></bpy>`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = resolvePalette(doc)
	if err == nil {
		t.Fatal("resolvePalette on an empty Theme should fail")
	}
	if !strings.Contains(err.Error(), string(types.TokenChromeText)) {
		t.Errorf("error should name the missing token, got: %v", err)
	}
}

func TestDeriveAlphaScaleAndMix(t *testing.T) {
	p := Palette{
		types.TokenChromeText:      {R: 1, G: 1, B: 1, A: 1},
		types.TokenChromeControlBg: {R: 0, G: 0, B: 0, A: 1},
		types.TokenGridMinor:       {R: 0.2, G: 0.2, B: 0.2, A: 0.6},
	}
	dim := derivedBindings[types.TokenChromeTextDisabled].derive(p)
	if dim.A != 0.5 || dim.R != 1 {
		t.Errorf("text_disabled = %+v, want text at half alpha", dim)
	}
	hov := derivedBindings[types.TokenChromeControlHover].derive(p)
	if hov.R != 0.10 || hov.A != 1 {
		t.Errorf("control_hover = %+v, want 10%% toward text with base alpha", hov)
	}
	axis := derivedBindings[types.TokenGridAxis].derive(p)
	if axis.A != 1 { // 0.6 * 3 clamps to 1
		t.Errorf("grid_axis alpha = %v, want clamped 1", axis.A)
	}
}

// Editing a direct token on a custom theme must land in its Blender attribute, so the
// saved body remains a faithful Blender theme.
func TestSetColorWritesBackToDocument(t *testing.T) {
	custom := DefaultDark().Duplicate("My Dark")
	want := Rgba{R: 1, G: 0.5, B: 0, A: 1}
	custom.SetColor(types.TokenChromeAccent, want)
	b := directBindings[types.TokenChromeAccent]
	hex, ok := custom.doc.Find(b.path...).Attr(b.attr)
	if !ok || hex != want.Hex() {
		t.Errorf("accent attribute after SetColor = (%q,%v), want %q", hex, ok, want.Hex())
	}
}
