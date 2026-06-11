//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/head/icon"
)

// TestPropertyToggleIconsResolve guards that every icon-toggle row of the feature
// property panels names only bundled glyphs — a missing asset degrades the toggle to a
// text button, so a typo'd key fails the build instead of shipping a mixed-looking row.
func TestPropertyToggleIconsResolve(t *testing.T) {
	sets := []propertyToggleSet{
		extrudeDirectionToggles, extrudeExtentToggles, booleanToggles,
		holeSeatToggles, holeTerminationToggles, holeDrillPointToggles,
		{keys: []string{"angle-full"}, tips: []string{"Full"}}, // the revolve full toggle's lone glyph
	}
	for _, set := range sets {
		if len(set.keys) != len(set.tips) {
			t.Errorf("toggle set %v: %d keys but %d tips", set.keys, len(set.keys), len(set.tips))
		}
		for _, key := range set.keys {
			if _, ok := icon.SVG(key); !ok {
				t.Errorf("property toggle references icon %q with no bundled asset (head/icon/assets/%s.svg)", key, key)
			}
		}
	}
}
