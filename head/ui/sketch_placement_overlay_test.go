//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The overlay must keep real geometry and construction geometry apart, so a centre rectangle's
// diagonals draw dashed while its four edges draw solid (#2014).
func TestPlacementOverlaySplitsConstruction(t *testing.T) {
	curves := sketch.RecipeCurves(sketch.CenterRectangleRecipe(math.P2(0, 0), math.P2(5, 4)))
	solid, construction := splitRecipeGeometry(curves)
	if len(solid) != 4 {
		t.Errorf("solid curves = %d, want 4 edges", len(solid))
	}
	if len(construction) != 2 {
		t.Errorf("construction curves = %d, want 2 diagonals", len(construction))
	}
}

// A shape with no construction geometry yields none — the split must not invent any.
func TestPlacementOverlayPlainRectangleHasNoConstruction(t *testing.T) {
	curves := sketch.RecipeCurves(sketch.RectangleRecipe(math.P2(0, 0), math.P2(10, 8)))
	solid, construction := splitRecipeGeometry(curves)
	if len(solid) != 4 || len(construction) != 0 {
		t.Errorf("solid = %d, construction = %d; want 4 and 0", len(solid), len(construction))
	}
}
