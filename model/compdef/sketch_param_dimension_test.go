// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestParameterDrivenDimensionResolvesAfterRestore guards the part restore path against a
// regression where a restored sketch kept its own empty parameter set, so a dimension
// expression that references a user parameter ("width") resolved to 0 and collapsed the
// geometry. The fix shares the part's parameter DAG into every sketch at creation
// (compdef wires it via Sketches.ShareParameters), mirroring the live and assembly paths.
func TestParameterDrivenDimensionResolvesAfterRestore(t *testing.T) {
	src := compdef.NewPartComponentDefinition()
	if _, err := src.Parameters().AddUserParameter("width", "40 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	s := src.Sketches().Add(sketch.XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(4, 0)) // 4 cm == 40 mm
	if _, err := s.DimensionConstraints().AddDistance(a, b, "width"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}

	model, err := src.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	dst := compdef.NewPartComponentDefinition()
	if err := dst.ApplyRecipe(model); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}

	got := dst.Sketches().Item(0).DimensionConstraints().Item(0)
	// 40 mm resolves to 4 db units; a regression makes "width" unknown → ~0.
	if measured := got.Measured(); measured < 3.999 || measured > 4.001 {
		t.Errorf("restored param-driven dimension measured %v, want ~4 (40 mm)", measured)
	}
}
