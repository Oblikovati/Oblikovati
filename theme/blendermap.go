// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"fmt"
	stdmath "math"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/theme/blenderxml"
)

// This file is the curated bridge between the Blender theme XML schema and the frozen
// semantic token vocabulary (ADR-0032). Each token resolves from exactly one Blender
// attribute (direct) or from another token's resolved color (derived) — never both —
// so write-back is unambiguous: editing a direct token updates its source attribute.
//
// Derived tokens mirror what Blender computes at draw time rather than stores: widget
// hover/press brightening, disabled-text dimming, and grid emphasis. They keep the
// palette complete when a plain Blender theme export (which has no hover slots) loads.

// Element paths shared by several bindings, relative to the <bpy> document root.
var (
	pathUI      = []string{"Theme", "user_interface", "ThemeUserInterface"}
	pathView3D  = []string{"Theme", "view_3d", "ThemeView3D"}
	pathV3Space = []string{"Theme", "view_3d", "ThemeView3D", "space", "ThemeSpaceGradient"}
	pathTopbar  = []string{"Theme", "topbar", "ThemeTopBar", "space", "ThemeSpaceGeneric"}
)

// wcol returns the path of one widget-color group under user_interface.
func wcol(group string) []string {
	return append(append([]string(nil), pathUI...), group, "ThemeWidgetColors")
}

// directBinding ties a token to the Blender attribute that authors it. Opaque forces
// alpha to 1 on load for colors Blender authors translucent but that paint solid
// surfaces here (e.g. the accent drives title bars).
type directBinding struct {
	path   []string
	attr   string
	opaque bool
}

// directBindings maps every directly-authored token to its Blender attribute. Paths are
// unique across entries (asserted by TestDirectBindingPathsUnique) so a token edit
// writes back to exactly one attribute.
var directBindings = map[Token]directBinding{
	// Chrome ← user_interface widget/panel slots.
	types.TokenChromeWindowBg:     {pathUI, "editor_border", true},
	types.TokenChromePanelBg:      {pathUI, "panel_back", false},
	types.TokenChromeHeaderBg:     {pathUI, "panel_header", false},
	types.TokenChromePopupBg:      {wcol("wcol_menu_back"), "inner", false},
	types.TokenChromeMenuBarBg:    {pathTopbar, "header", false},
	types.TokenChromeText:         {wcol("wcol_regular"), "text", false},
	types.TokenChromeBorder:       {wcol("wcol_regular"), "outline", false},
	types.TokenChromeControlBg:    {wcol("wcol_text"), "inner", false},
	types.TokenChromeButton:       {wcol("wcol_tool"), "inner", false},
	types.TokenChromeButtonActive: {wcol("wcol_tool"), "inner_sel", false},
	types.TokenChromeAccent:       {wcol("wcol_regular"), "inner_sel", true},
	types.TokenChromeScrollbar:    {wcol("wcol_scroll"), "item", false},
	// Danger reads the console's error-line red — the only legible per-theme red the
	// Blender document authors (wcol_state's `error` is a dark fill, not a text/outline).
	types.TokenChromeDanger: {[]string{"Theme", "console", "ThemeConsole"}, "line_error", true},
	// Viewport 2D ← view_3d editor slots (sketch == Blender edit-mode geometry).
	types.TokenViewportBg:       {append(append([]string(nil), pathV3Space...), "gradients", "ThemeGradientColors"), "high_gradient", true},
	types.TokenGridMinor:        {pathView3D, "grid", false},
	types.TokenSketchGeometry:   {pathView3D, "wire_edit", false},
	types.TokenSketchSelected:   {pathView3D, "edge_select", false},
	types.TokenSketchCandidate:  {pathView3D, "editmesh_active", false},
	types.TokenSketchPreview:    {pathView3D, "transform", false},
	types.TokenSnapGlyph:        {pathView3D, "vertex_select", false},
	types.TokenDimensionDriving: {pathV3Space, "text", false},
	// Blender's edge-length measurement colour: the same job — a length read off the geometry
	// while you work — so a theme recoloured in Blender carries the intent across.
	types.TokenDimensionDriven:      {append(append([]string(nil), pathUI...), "wcol_state", "ThemeWidgetStateColors"), "inner_driven", false},
	types.TokenViewportActiveBorder: {pathUI, "editor_outline_active", false},
	// Gizmos 3D ← Blender's gizmo and object-state slots.
	types.TokenPlaneFaint:         {pathUI, "gizmo_primary", false},
	types.TokenPlaneHover:         {pathUI, "gizmo_hi", false},
	types.TokenPlaneSelected:      {pathView3D, "object_active", false},
	types.TokenPlaneFill:          {pathView3D, "face", false},
	types.TokenSelectionHighlight: {pathView3D, "object_selected", false},
	// Icons ← Blender's own icon category colors (ThemeUserInterface icon_* slots):
	// the neutral collection gray is the main linework, the object color the accent,
	// the modifier color the detail layer; the plate reuses the translucent sub-panel
	// backdrop, the closest thing Blender has to an "behind a glyph" surface.
	types.TokenIconPrimary:    {pathUI, "icon_collection", false},
	types.TokenIconSecondary:  {pathUI, "icon_object", false},
	types.TokenIconTertiary:   {pathUI, "icon_modifier", false},
	types.TokenIconBackground: {pathUI, "panel_sub_back", false},
}

// derivedBinding computes a token from an already-resolved base token: mix blends the
// base toward TokenChromeText by the given fraction (Blender's hover brightening —
// toward light text on dark themes, dark text on light ones), alphaScale multiplies the
// base alpha (Blender's disabled dimming and single-`grid`-slot emphasis).
type derivedBinding struct {
	base       Token
	mix        float32
	alphaScale float32
	// pick computes the colour from the already-resolved direct tokens. It is for a colour the
	// PRODUCT decides rather than the theme file — one Blender has no slot that means.
	pick func(Palette) Rgba
}

// derivedBindings lists every token Blender has no storage slot for, in dependency
// order on directBindings (a derived base is always direct, never derived).
var derivedBindings = map[Token]derivedBinding{
	types.TokenChromeTextDisabled:  {base: types.TokenChromeText, alphaScale: 0.5},
	types.TokenChromeControlHover:  {base: types.TokenChromeControlBg, mix: 0.10},
	types.TokenChromeControlActive: {base: types.TokenChromeControlBg, mix: 0.18},
	types.TokenChromeButtonHover:   {base: types.TokenChromeButton, mix: 0.10},
	types.TokenGridMajor:           {base: types.TokenGridMinor, alphaScale: 2},
	types.TokenGridAxis:            {base: types.TokenGridMinor, alphaScale: 3},
	// The in-place dimension is drafting green in every CAD package, and Blender has no slot
	// that means it. Deriving it from an unrelated Blender field made an EXISTING custom theme
	// resolve it to whatever that field happened to hold (a near-black brown, #2034) — a token
	// snapshot only carries the tokens that existed when it was saved.
	types.TokenDimensionSketch: {pick: sketchDimensionGreen},
}

// sketchDimensionGreen is the sketch-dimension colour: drafting green, in the shade that stays
// legible on the viewport background it will be drawn over.
//
// ONE fixed green cannot serve both shipped themes — bright enough for the dark viewport is
// 1.31:1 on the light one, which the contrast guard rejects outright. So the shade follows the
// background, which is what makes it a derived token rather than a constant.
func sketchDimensionGreen(p Palette) Rgba {
	if relativeLuminance(p.Color(types.TokenViewportBg)) > 0.2 {
		return Rgba{R: 0.02, G: 0.36, B: 0.13, A: 1} // dark green, for a light viewport
	}
	return Rgba{R: 0.24, G: 0.85, B: 0.40, A: 1} // bright green, for a dark viewport
}

// relativeLuminance is WCAG 2.1's L on straight sRGB — how light a colour reads, which is what
// decides which shade contrasts against it.
func relativeLuminance(c Rgba) float64 {
	lin := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}
		return stdmath.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(float64(c.R)) + 0.7152*lin(float64(c.G)) + 0.0722*lin(float64(c.B))
}

// resolvePalette extracts a complete palette from a Blender theme document: direct
// tokens from their attributes, then derived tokens from the direct results. It errors,
// naming every missing attribute, when the document lacks a mapped slot — a foreign
// theme either yields a complete palette or is rejected loudly.
func resolvePalette(doc *blenderxml.Node) (Palette, error) {
	p := make(Palette, len(directBindings)+len(derivedBindings))
	var missing []string
	for tok, b := range directBindings {
		c, err := resolveDirect(doc, b)
		if err != nil {
			missing = append(missing, fmt.Sprintf("%s (%v)", tok, err))
			continue
		}
		p[tok] = c
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("theme: blender document is missing %s", strings.Join(missing, "; "))
	}
	for tok, d := range derivedBindings {
		p[tok] = d.derive(p)
	}
	return p, nil
}

// resolveDirect reads one binding's color out of the document.
func resolveDirect(doc *blenderxml.Node, b directBinding) (Rgba, error) {
	elem := doc.Find(b.path...)
	if elem == nil {
		return Rgba{}, fmt.Errorf("element %s not found", strings.Join(b.path, "/"))
	}
	hex, ok := elem.Attr(b.attr)
	if !ok {
		return Rgba{}, fmt.Errorf("attribute %s@%s not found", strings.Join(b.path, "/"), b.attr)
	}
	c, err := types.ParseHex(hex)
	if err != nil {
		return Rgba{}, err
	}
	if b.opaque {
		c.A = 1
	}
	return c, nil
}

// derive computes the binding's color from the resolved direct tokens.
func (d derivedBinding) derive(p Palette) Rgba {
	if d.pick != nil {
		return d.pick(p)
	}
	c := p.Color(d.base)
	if d.alphaScale != 0 {
		c.A = math.Clamp(c.A*d.alphaScale, 0, 1)
		return c
	}
	return mixRgb(c, p.Color(types.TokenChromeText), d.mix)
}

// mixRgb blends a toward b's hue by t, keeping a's alpha — the widget stays a surface,
// it just shifts toward the text color the way Blender brightens hovered widgets.
func mixRgb(a, b Rgba, t float32) Rgba {
	lerp := func(x, y float32) float32 { return x + (y-x)*t }
	return Rgba{R: lerp(a.R, b.R), G: lerp(a.G, b.G), B: lerp(a.B, b.B), A: a.A}
}

// writeBackColor pushes one token's color into its Blender attribute, so the saved
// document stays a faithful Blender theme after edits. Derived tokens have no slot of
// their own (their color persists via the token snapshot, see blenderfile.go) — for
// them this is a no-op.
func writeBackColor(doc *blenderxml.Node, tok Token, c Rgba) {
	b, ok := directBindings[tok]
	if !ok {
		return
	}
	if elem := doc.Find(b.path...); elem != nil {
		elem.SetAttr(b.attr, c.Hex())
	}
}
