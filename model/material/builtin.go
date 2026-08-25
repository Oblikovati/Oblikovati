// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"fmt"

	"oblikovati.org/api/types"
)

// seedBuiltins fills a fresh library with the shipped, read-only catalog: the neutral
// default appearance (always present, the resolver's last resort) followed by the
// embedded category catalogs (metals, steels, aluminium, plastics, woods, composites,
// ADR-0024). A malformed embedded file is a build defect, so it panics with the
// offending file — caught by the package tests, never reached at runtime.
func seedBuiltins(l *Library) {
	l.AddAppearance(defaultAppearance())
	if err := loadCatalog(l); err != nil {
		panic(fmt.Sprintf("material: built-in catalog: %v", err))
	}
}

// mustColor parses an sRGB hex color literal, panicking with the offending value on a
// malformed constant (a programming error caught by the package tests, not a runtime
// condition). Used by legacy-shaped fixtures/migration code — the current AppearanceSpec
// carries linear ACEScg Color3 values, not sRGB hex.
func mustColor(s string) Rgba {
	c, err := types.ParseHex(s)
	if err != nil {
		panic(fmt.Sprintf("material: malformed built-in color %q: %v", s, err))
	}
	return c
}
