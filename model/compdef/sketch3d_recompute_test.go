// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// A parameter that drives a 3D-sketch dimension must move the 3D geometry on recompute, the
// way a 2D-sketch dimension does. This is the regression test for #1566: 3D sketches were not
// re-solved during part recompute (solveSketches was 2D-only), so editing a 3D dimension's
// parameter left the 3D geometry on its pre-edit shape.
func TestSketch3DDimensionParameterDrivesGeometryOnRecompute(t *testing.T) {
	t.Parallel()
	def := compdef.NewPartComponentDefinition()
	span, err := def.Parameters().AddUserParameter("span", "5 cm")
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}

	// A 3D line authored at length 3, dimensioned to follow "span" (5 cm initially).
	sk := def.Sketches3D().Add()
	line := sk.AddLine3D(math.P3(0, 0, 0), math.P3(3, 0, 0))
	if _, err := sk.DimensionConstraints3D().AddDistance(line.StartPoint(), line.EndPoint(), "span"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}

	def.Recompute()
	if l := line.Length(); l < 4.999 || l > 5.001 {
		t.Fatalf("after recompute, line length = %v cm, want ~5 (dimension solved to span)", l)
	}

	// Drive the parameter: the 3D geometry must follow on recompute.
	if err := def.Parameters().SetExpression(span.ID(), "8 cm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	def.RecomputeAfterChange()
	if l := line.Length(); l < 7.999 || l > 8.001 {
		t.Errorf("after span→8, line length = %v cm, want ~8 (3D sketch must re-solve on recompute, #1566)", l)
	}
}
