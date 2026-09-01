// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
)

// TestFeatureScaleWarning covers the ADR-0042 Phase 2 span-ceiling diagnostic (#1249): an empty
// part never warns, a part with extent flags a feature far below its resolution and accepts a
// resolvable one.
func TestFeatureScaleWarning(t *testing.T) {
	t.Parallel()
	empty := NewPartComponentDefinition()
	if w := empty.FeatureScaleWarning(1e-30); w != "" {
		t.Errorf("empty part should not warn, got %q", w)
	}
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1000, 1000, 1000))
	if w := part.FeatureScaleWarning(1e-9); w == "" {
		t.Error("a feature far below the model resolution should warn")
	}
	if w := part.FeatureScaleWarning(100); w != "" {
		t.Errorf("a resolvable feature should not warn, got %q", w)
	}
}
