// SPDX-License-Identifier: GPL-2.0-only

package icon

import "testing"

// sentinelSet is the only paints an asset may use — anything else means the designer
// drifted from the role convention and the glyph would silently lose that element at
// rasterization (it matches no role's pass).
var sentinelSet = map[string]bool{
	sentinelPaint[RoleBackground]: true,
	sentinelPaint[RoleTertiary]:   true,
	sentinelPaint[RoleSecondary]:  true,
	sentinelPaint[RolePrimary]:    true,
}

// TestAssetsConformToColorRoles enforces the icon authoring contract (ADR-0033) on
// every bundled SVG: only role-sentinel paints, a background plate, and primary
// linework — so a non-conforming asset fails the build instead of shipping a glyph
// with invisible or unthemed parts.
func TestAssetsConformToColorRoles(t *testing.T) {
	keys := Keys()
	if len(keys) == 0 {
		t.Fatal("no icons bundled")
	}
	for _, key := range keys {
		assertAssetConforms(t, key)
	}
}

func assertAssetConforms(t *testing.T, key string) {
	t.Helper()
	svg, ok := SVG(key)
	if !ok {
		t.Errorf("%s: listed by Keys() but unreadable", key)
		return
	}
	paints, err := UsedPaints(svg)
	if err != nil {
		t.Errorf("%s: %v", key, err)
		return
	}
	for _, p := range paints {
		if !sentinelSet[p] {
			t.Errorf("%s: uses paint %q outside the role sentinels", key, p)
		}
	}
	masks, err := RasterizeRoles(svg, 32)
	if err != nil {
		t.Errorf("%s: %v", key, err)
		return
	}
	for _, must := range []Role{RoleBackground, RolePrimary} {
		if !masks.hasContent(must) {
			t.Errorf("%s: role %s is empty — every icon needs a plate and primary linework", key, must)
		}
	}
}

// hasContent reports whether any pixel of the role's mask is meaningfully covered.
func (m *RoleMasks) hasContent(r Role) bool {
	for _, v := range m.cover[r] {
		if v > alphaThreshold {
			return true
		}
	}
	return false
}
