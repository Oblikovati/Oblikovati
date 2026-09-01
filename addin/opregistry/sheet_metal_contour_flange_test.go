// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestSheetMetalContourFlangeApply seeds a sheet-metal wall, adds an open L-profile sketch,
// and sweeps it along a top edge into one merged solid; then checks the error paths.
func TestSheetMetalContourFlangeApply(t *testing.T) {
	t.Parallel()
	s, edge := seedSheetMetalSheet(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	profile := def.Sketches().Add(sketch.XYPlane())
	p0 := profile.Points().Add(math.P2(0, 0))
	p1 := profile.Points().Add(math.P2(1, 0))
	p2 := profile.Points().Add(math.P2(1, 1))
	profile.Lines().Add(p0, p1)
	profile.Lines().Add(p1, p2)

	out, err := applyMap(t, s, "sheetMetalContourFlange", map[string]any{"edge": edge, "profileSketch": 1})
	if err != nil {
		t.Fatalf("contour flange apply: %v", err)
	}
	expectMergedSolid(t, out, "contourFlange")

	// Error paths.
	if _, err := apply(t, sheetMetalProfiledPart(t), "sheetMetalContourFlange", `{"profileSketch":0}`); err == nil {
		t.Error("contour flange without an edge must error")
	}
	if _, err := apply(t, profiledPart(t), "sheetMetalContourFlange", `{"edge":"x","profileSketch":0}`); err == nil {
		t.Error("contour flange on a non-sheet-metal part must error")
	}
	if _, err := applyMap(t, s, "sheetMetalContourFlange", map[string]any{"edge": edge, "profileSketch": 99}); err == nil {
		t.Error("contour flange with an out-of-range profile sketch must error")
	}
}
