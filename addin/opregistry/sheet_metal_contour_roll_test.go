// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// addRollProfile adds a Y-axis centerline (line 0) and a vertical profile at x=radius (line 1)
// to the part, returning the sketch index.
func addRollProfile(def *compdef.PartComponentDefinition, radius, height float64) int {
	s := def.Sketches().Add(sketch.XYPlane())
	a0 := s.Points().Add(math.P2(0, 0))
	a1 := s.Points().Add(math.P2(0, 1))
	axis := s.Lines().Add(a0, a1)
	axis.SetCenterline(true)
	p0 := s.Points().Add(math.P2(math.Scalar(radius), 0))
	p1 := s.Points().Add(math.P2(math.Scalar(radius), math.Scalar(height)))
	s.Lines().Add(p0, p1)
	return def.Sketches().Count() - 1
}

// TestSheetMetalContourRollApply rolls a profile into a tube on a sheet-metal part and
// confirms one healthy solid; then checks the error paths.
func TestSheetMetalContourRollApply(t *testing.T) {
	t.Parallel()
	s := sheetMetalProfiledPart(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	idx := addRollProfile(def, 2, 3)

	out, err := apply(t, s, "sheetMetalContourRoll", `{"profileSketch":1,"axisLine":0,"angle":"360 deg"}`)
	if err != nil {
		t.Fatalf("contour roll apply: %v", err)
	}
	expectMergedSolid(t, out, "contourRoll")
	_ = idx

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalContourRoll", `{"profileSketch":0,"axisLine":0}`); err == nil {
		t.Error("contour roll on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalContourRoll", `{"profileSketch":99,"axisLine":0}`); err == nil {
		t.Error("contour roll with an out-of-range profile must error")
	}
	if _, err := apply(t, s, "sheetMetalContourRoll", `{"profileSketch":1,"axisLine":0,"angle":"bad"}`); err == nil {
		t.Error("contour roll with a bad angle must error")
	}
}
