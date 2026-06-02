// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"fmt"

	"github.com/Oblikovati/api/types"
)

// DefaultDark is the shipped dark theme. Its viewport/overlay/icon colors reproduce the
// values the head hardcoded before theming (so the default look is unchanged); the
// chrome colors are a coherent dark Dear ImGui palette. Built fresh on each call so the
// library holds an independent copy.
func DefaultDark() *Theme { return New("Dark", KindDark, mustPalette(darkHex)) }

// DefaultLight is the shipped light theme — the same token set tuned for a light shell.
func DefaultLight() *Theme { return New("Light", KindLight, mustPalette(lightHex)) }

// Builtins returns the two shipped themes, dark first (the out-of-the-box default).
func Builtins() []*Theme { return []*Theme{DefaultDark(), DefaultLight()} }

// darkHex reproduces the pre-theming hardcoded overlay/icon colors (head/ui) and pairs
// them with a dark chrome palette. See ADR-0021 for the token meanings.
var darkHex = map[string]string{
	// Chrome
	string(types.TokenChromeWindowBg):      "#1e2127ff",
	string(types.TokenChromePanelBg):       "#262a31ff",
	string(types.TokenChromePopupBg):       "#1a1c22ff",
	string(types.TokenChromeMenuBarBg):     "#15171cff",
	string(types.TokenChromeHeaderBg):      "#2a2f38ff",
	string(types.TokenChromeText):          "#d9dce1ff",
	string(types.TokenChromeTextDisabled):  "#6b7280ff",
	string(types.TokenChromeBorder):        "#3a3f49ff",
	string(types.TokenChromeControlBg):     "#262a31ff",
	string(types.TokenChromeControlHover):  "#2f343cff",
	string(types.TokenChromeControlActive): "#3a414bff",
	string(types.TokenChromeButton):        "#2a2f38ff",
	string(types.TokenChromeButtonHover):   "#353b45ff",
	string(types.TokenChromeButtonActive):  "#454d59ff",
	string(types.TokenChromeAccent):        "#3fb6ffff",
	string(types.TokenChromeScrollbar):     "#3a3f49ff",
	// Viewport 2D (pre-theming head values)
	string(types.TokenViewportBg):       "#1a1c21ff", // viewport.cpp clear {0.10,0.11,0.13}
	string(types.TokenGridMinor):        "#454a54ff", // grid_overlay.go gridMinorColor
	string(types.TokenGridMajor):        "#666e7aff", // grid_overlay.go gridMajorColor
	string(types.TokenGridAxis):         "#8c99adff", // grid_overlay.go gridAxisColor
	string(types.TokenSketchGeometry):   "#ffc733ff", // sketch_overlay.go sketchColor
	string(types.TokenSketchSelected):   "#4de6ffff", // sketch_overlay.go sketchSelectedColor
	string(types.TokenSketchCandidate):  "#66ff73ff", // sketch_overlay.go sketchCandidateColor
	string(types.TokenSketchPreview):    "#f2f2ffff", // chrome.go previewColor
	string(types.TokenDimensionDriving): "#73d980ff", // dimension_overlay.go dimensionColor
	string(types.TokenDimensionDriven):  "#8cb3f2ff", // dimension_overlay.go dimensionDrivenColor
	string(types.TokenSnapGlyph):        "#ffffffff", // snap_glyph.go snapGlyphColor
	// Gizmos 3D (pre-theming head values)
	string(types.TokenPlaneFaint):         "#738099ff", // plane_overlay.go faintPlaneColor
	string(types.TokenPlaneHover):         "#ffb333ff", // plane_overlay.go hoverPlaneColor
	string(types.TokenPlaneSelected):      "#4dd9ffff", // plane_overlay.go selectedPlaneColor
	string(types.TokenSelectionHighlight): "#4dd9ffff", // highlight.go selectionHighlight
	// Icons (pre-theming head values)
	string(types.TokenIconTint):     "#d9dee6ff", // icon_cache.go iconTint
	string(types.TokenIconDisabled): "#6b7280ff",
}

// lightHex is a light-shell palette over the same token set.
var lightHex = map[string]string{
	// Chrome
	string(types.TokenChromeWindowBg):      "#f3f4f6ff",
	string(types.TokenChromePanelBg):       "#ffffffff",
	string(types.TokenChromePopupBg):       "#ffffffff",
	string(types.TokenChromeMenuBarBg):     "#e6e8ebff",
	string(types.TokenChromeHeaderBg):      "#dfe3e8ff",
	string(types.TokenChromeText):          "#1c1f24ff",
	string(types.TokenChromeTextDisabled):  "#9aa0a8ff",
	string(types.TokenChromeBorder):        "#c7ccd3ff",
	string(types.TokenChromeControlBg):     "#ffffffff",
	string(types.TokenChromeControlHover):  "#eceef1ff",
	string(types.TokenChromeControlActive): "#dde1e6ff",
	string(types.TokenChromeButton):        "#e6e8ebff",
	string(types.TokenChromeButtonHover):   "#dde1e6ff",
	string(types.TokenChromeButtonActive):  "#cfd4daff",
	string(types.TokenChromeAccent):        "#1f7fd6ff",
	string(types.TokenChromeScrollbar):     "#c7ccd3ff",
	// Viewport 2D
	string(types.TokenViewportBg):       "#e8eaedff",
	string(types.TokenGridMinor):        "#c2c7cfff",
	string(types.TokenGridMajor):        "#a8aeb8ff",
	string(types.TokenGridAxis):         "#7f8794ff",
	string(types.TokenSketchGeometry):   "#c77f00ff",
	string(types.TokenSketchSelected):   "#0a84c7ff",
	string(types.TokenSketchCandidate):  "#1faa3aff",
	string(types.TokenSketchPreview):    "#5a5a99ff",
	string(types.TokenDimensionDriving): "#2f9e45ff",
	string(types.TokenDimensionDriven):  "#3f66c7ff",
	string(types.TokenSnapGlyph):        "#1c1f24ff",
	// Gizmos 3D
	string(types.TokenPlaneFaint):         "#8c99adff",
	string(types.TokenPlaneHover):         "#d98a00ff",
	string(types.TokenPlaneSelected):      "#0a84c7ff",
	string(types.TokenSelectionHighlight): "#0a84c7ff",
	// Icons
	string(types.TokenIconTint):     "#3a3f49ff",
	string(types.TokenIconDisabled): "#aeb4bdff",
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
