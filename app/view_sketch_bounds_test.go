// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestModelBoundsIncludesSketches is the regression for issue #1146: Fit/Home framed only solid
// bodies, so a sketch-only part (a DWG/DXF import) had empty model bounds and Fit did nothing.
// modelBounds must now enclose visible 2D and 3D sketch geometry.
func TestModelBoundsIncludesSketches(t *testing.T) {
	s := NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "bounds.opd", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)

	// A 3D sketch line far from the origin (where a body-only bounds would stay empty).
	def.Sketches3D().Add().AddLine3D(math.P3(100, 200, 300), math.P3(140, 260, 360))
	// A 2D sketch on XY with a line.
	sk2d := def.Sketches().Add(sketch.XYPlane())
	sk2d.Lines().AddByTwoPoints(math.P2(-50, -50), math.P2(-10, -20))
	def.Recompute()

	box := s.modelBounds()
	if box.IsEmpty() {
		t.Fatal("modelBounds is empty for a sketch-only part (Fit would not frame the import)")
	}
	for _, p := range []math.Point3{{X: 100, Y: 200, Z: 300}, {X: 140, Y: 260, Z: 360}, {X: -50, Y: -50, Z: 0}} {
		if !box.Contains(p) {
			t.Errorf("modelBounds %v does not contain sketch point %v", box, p)
		}
	}
}
