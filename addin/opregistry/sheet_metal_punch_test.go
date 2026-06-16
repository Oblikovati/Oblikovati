// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestSheetMetalPunchApply seeds a sheet-metal wall, punches two holes from one sketch, and
// confirms one healthy solid; then checks the error paths.
func TestSheetMetalPunchApply(t *testing.T) {
	s := sheetMetalProfiledPart(t)
	if _, err := apply(t, s, "sheetMetalFace", `{"sketchIndex":0}`); err != nil {
		t.Fatalf("seed face: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	holes := def.Sketches().Add(sketch.XYPlane())
	for _, c := range []math.Point2{math.P2(1, 1), math.P2(3, 2)} {
		a := holes.Points().Add(math.P2(c.X-0.3, c.Y-0.3))
		b := holes.Points().Add(math.P2(c.X+0.3, c.Y-0.3))
		d := holes.Points().Add(math.P2(c.X+0.3, c.Y+0.3))
		e := holes.Points().Add(math.P2(c.X-0.3, c.Y+0.3))
		holes.Lines().Add(a, b)
		holes.Lines().Add(b, d)
		holes.Lines().Add(d, e)
		holes.Lines().Add(e, a)
	}

	out, err := apply(t, s, "sheetMetalPunch", `{"sketchIndex":1}`)
	if err != nil {
		t.Fatalf("punch apply: %v", err)
	}
	expectMergedSolid(t, out, "punch")

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalPunch", `{"sketchIndex":0}`); err == nil {
		t.Error("punch on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalPunch", `{"sketchIndex":99}`); err == nil {
		t.Error("punch with an out-of-range sketch must error")
	}
	if _, err := apply(t, s, "sheetMetalPunch", `{"sketchIndex":1,"depth":"bad"}`); err == nil {
		t.Error("punch with a bad depth must error")
	}
}
