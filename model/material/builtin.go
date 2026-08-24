// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"fmt"

	"oblikovati.org/api/types"
)

// DefaultAppearanceID is the neutral appearance applied when a body has no material or
// appearance assigned — it reproduces the renderer's pre-materials default gray, so an
// un-assigned model looks exactly as it did before this subsystem.
const DefaultAppearanceID = "default"

// seedBuiltins fills a fresh library with the shipped, read-only catalog: the neutral
// default appearance (always present, the resolver's last resort) followed by the
// embedded category catalogs (metals, steels, aluminium, plastics, woods, composites,
// ADR-0024). A malformed embedded file is a build defect, so it panics with the
// offending file — caught by the package tests, never reached at runtime.
func seedBuiltins(l *Library) {
	l.AddAppearance(defaultAppearance())
	l.AddOpenPBRAppearance(defaultOpenPBRAppearance())
	if err := loadCatalog(l); err != nil {
		panic(fmt.Sprintf("material: built-in catalog: %v", err))
	}
}

// defaultAppearance is the neutral gray fallback, kept in code (not the YAML catalog) so
// DefaultAppearanceID is guaranteed present even if a catalog file is hand-edited.
func defaultAppearance() *Appearance {
	return NewAppearance(DefaultAppearanceID, SourceBuiltin, AppearanceSpec{
		DisplayName: "Default", Albedo: mustColor("#b3b8bdff"), Metallic: 0, Roughness: 0.6,
		Emissive: mustColor("#000000ff"), Opacity: 1,
	})
}

// mustColor parses an authored built-in color literal, panicking with the offending
// value on a malformed constant (a programming error caught by the package tests, not a
// runtime condition).
func mustColor(s string) Rgba {
	c, err := types.ParseHex(s)
	if err != nil {
		panic(fmt.Sprintf("material: malformed built-in color %q: %v", s, err))
	}
	return c
}
