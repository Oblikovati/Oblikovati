// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"fmt"

	"oblikovati.org/api/types"
)

// DefaultDark is the shipped dark theme. Its colors follow Autodesk Inventor's dark
// design spec (../../inventor-color.md): the cool blue-gray "Light Logic" surface stack,
// the #0696D7 brand accent, cyan pre-highlight/selection, green under-constrained sketch
// geometry and translucent-orange work planes. Built fresh on each call so the library
// holds an independent copy.
func DefaultDark() *Theme { return New("Dark", KindDark, mustPalette(darkHex)) }

// DefaultLight is the shipped light theme — the same Inventor token set tuned for the
// neutral-gray light shell (white/#F5F5F5/#D9D9D9 surfaces).
func DefaultLight() *Theme { return New("Light", KindLight, mustPalette(lightHex)) }

// Builtins returns the two shipped themes, dark first (the out-of-the-box default).
func Builtins() []*Theme { return []*Theme{DefaultDark(), DefaultLight()} }

// autodeskBlue500 is Inventor's primary brand accent (inventor-color.md §2): the
// "On/Active/Selected/Focus" color shared by both themes (active tab, selection,
// checkmark, slider grab, focus). The secondary #38ABDF outline accent is not a separate
// token here — TokenChromeBorder drives every panel separator, so it stays neutral.
const autodeskBlue500 = "#0696d7ff"

// darkHex maps every token to Inventor's dark theme (inventor-color.md). Surfaces use the
// three "Light Logic" levels (L3 base #222933 → L2 mid #3B4453 → L1 highest #454F61);
// text/icons sit at #A2A6B0. See ADR-0021 for the token meanings.
var darkHex = map[string]string{
	// Chrome — Light Logic surface stack (§1 Dark Theme, §6 summary)
	string(types.TokenChromeWindowBg):      "#222933ff", // L3 base: app frame / empty canvas
	string(types.TokenChromeMenuBarBg):     "#222933ff", // L3: top frame bar
	string(types.TokenChromePanelBg):       "#3b4453ff", // L2 mid: property panels / dialogs
	string(types.TokenChromeHeaderBg):      "#3b4453ff", // L2: inactive title / tree header
	string(types.TokenChromePopupBg):       "#454f61ff", // L1 highest: flyouts / floating panels
	string(types.TokenChromeText):          "#a2a6b0ff", // default text/icon
	string(types.TokenChromeTextDisabled):  "#6c7382ff", // muted blue-gray
	string(types.TokenChromeBorder):        "#4a5466ff", // subtle separator above the surfaces
	string(types.TokenChromeControlBg):     "#2c333fff", // inset field well (darker than panel)
	string(types.TokenChromeControlHover):  "#3b4453ff",
	string(types.TokenChromeControlActive): "#454f61ff",
	string(types.TokenChromeButton):        "#3b4453ff", // L2: buttons / inactive tabs
	string(types.TokenChromeButtonHover):   "#454f61ff",
	string(types.TokenChromeButtonActive):  "#515c70ff",
	string(types.TokenChromeAccent):        autodeskBlue500, // On/Active/Selected/Focus (§2)
	string(types.TokenChromeScrollbar):     "#3b4453ff",
	// Viewport 2D (§4 Dark canvas, §5 interactive/constraint states)
	string(types.TokenViewportBg):           "#1e232bff", // "Dark" scheme solid canvas
	string(types.TokenGridMinor):            "#3a424eff",
	string(types.TokenGridMajor):            "#55606eff",
	string(types.TokenGridAxis):             "#8c99adff",
	string(types.TokenSketchGeometry):       "#5fb158ff", // under-constrained green (§5.2)
	string(types.TokenSketchSelected):       "#00bfffff", // selection deep cyan (§5.1)
	string(types.TokenSketchCandidate):      "#00ffffff", // pre-highlight bright cyan (§5.1)
	string(types.TokenSketchPreview):        "#d1deeeff", // focus light (§2)
	string(types.TokenDimensionDriving):     "#c5cad3ff", // legible neutral annotation
	string(types.TokenDimensionDriven):      "#8d96a3ff", // driven/reference indigo gray (§5.2)
	string(types.TokenSnapGlyph):            "#ffffffff",
	string(types.TokenViewportActiveBorder): "#11151cff", // a touch darker than the dark canvas (#1e232b)
	// Gizmos 3D (§5.3 reference infrastructure)
	string(types.TokenPlaneFaint):         "#e37a22ff", // translucent-orange work plane border
	string(types.TokenPlaneHover):         "#f09d4fff", // brighter surface orange on hover
	string(types.TokenPlaneSelected):      "#00bfffff", // selection cyan
	string(types.TokenPlaneFill):          "#e37a224d", // orange at ~30% opacity
	string(types.TokenSelectionHighlight): "#00bfffff", // picked-element cyan (§5.1)
	// Icons (§3 semantic palette — gray main body on dark)
	string(types.TokenIconTint):     "#a2a6b0ff",
	string(types.TokenIconDisabled): "#6c7382ff",
}

// lightHex maps every token to Inventor's neutral-gray light theme (inventor-color.md §1
// Light Theme): white foreground (L1) over #F5F5F5 panels (L2) over the #D9D9D9 frame
// edge (L3). Viewport cyans/greens are toned a step darker so they stay visible on the
// light canvas.
var lightHex = map[string]string{
	// Chrome — neutral-gray Light Logic stack
	string(types.TokenChromeWindowBg):      "#d9d9d9ff", // L3 base: frame edge
	string(types.TokenChromeMenuBarBg):     "#f5f5f5ff", // L2: top toolbar
	string(types.TokenChromePanelBg):       "#f5f5f5ff", // L2 mid: panels / dialogs
	string(types.TokenChromeHeaderBg):      "#e9ebedff", // inactive title / tree header
	string(types.TokenChromePopupBg):       "#ffffffff", // L1 highest: flyouts / foreground
	string(types.TokenChromeText):          "#2b2f36ff", // dark body text (legible)
	string(types.TokenChromeTextDisabled):  "#9aa0a8ff",
	string(types.TokenChromeBorder):        "#c7ccd3ff",
	string(types.TokenChromeControlBg):     "#ffffffff", // L1: text inputs / foreground fields
	string(types.TokenChromeControlHover):  "#f0f1f3ff",
	string(types.TokenChromeControlActive): "#e4e6e9ff",
	string(types.TokenChromeButton):        "#f5f5f5ff", // L2: buttons / inactive tabs
	string(types.TokenChromeButtonHover):   "#e9ebedff",
	string(types.TokenChromeButtonActive):  "#dcdee1ff",
	string(types.TokenChromeAccent):        autodeskBlue500, // primary accent (§2)
	string(types.TokenChromeScrollbar):     "#c7ccd3ff",
	// Viewport 2D
	string(types.TokenViewportBg):           "#d4dce4ff", // light blue-gray "Sky"-style canvas
	string(types.TokenGridMinor):            "#c2c7cfff",
	string(types.TokenGridMajor):            "#a8aeb8ff",
	string(types.TokenGridAxis):             "#7f8794ff",
	string(types.TokenSketchGeometry):       "#4f9e48ff", // under-constrained green (toned)
	string(types.TokenSketchSelected):       "#0098ccff", // selection cyan (toned for light)
	string(types.TokenSketchCandidate):      "#00b3b3ff", // pre-highlight cyan (toned for light)
	string(types.TokenSketchPreview):        "#5a6b85ff",
	string(types.TokenDimensionDriving):     "#404652ff",
	string(types.TokenDimensionDriven):      "#8d96a3ff", // driven/reference indigo gray (§5.2)
	string(types.TokenSnapGlyph):            "#1c1f24ff",
	string(types.TokenViewportActiveBorder): "#b4bec9ff", // a touch darker than the light canvas (#d4dce4)
	// Gizmos 3D
	string(types.TokenPlaneFaint):         "#e37a22ff", // work plane orange
	string(types.TokenPlaneHover):         "#e08a2aff",
	string(types.TokenPlaneSelected):      "#0098ccff", // selection cyan
	string(types.TokenPlaneFill):          "#e37a224d", // orange at ~30% opacity
	string(types.TokenSelectionHighlight): "#0098ccff",
	// Icons (§3 — gray icon linework on light)
	string(types.TokenIconTint):     "#666b73ff",
	string(types.TokenIconDisabled): "#b0b5bdff",
}

// mustPalette builds a palette from a built-in hex map, panicking with the offending
// color if a literal is malformed — these maps are authored constants, so a bad value
// is a programming error caught by [TestDefaultsComplete], not a runtime condition.
func mustPalette(hex map[string]string) Palette {
	p, err := NewPalette(hex)
	if err != nil {
		panic(fmt.Sprintf("theme: malformed built-in palette: %v", err))
	}
	return p
}
