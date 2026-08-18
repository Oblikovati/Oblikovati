// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// addLProfile adds an open L-profile (out w, up h) on the given plane to the part and returns
// its sketch index.
func addLProfile(def *compdef.PartComponentDefinition, plane sketch.Plane, w, h float64) int {
	s := def.Sketches().Add(plane)
	p0 := s.Points().Add(math.P2(0, 0))
	p1 := s.Points().Add(math.P2(math.Scalar(w), 0))
	p2 := s.Points().Add(math.P2(math.Scalar(w), math.Scalar(h)))
	s.Lines().Add(p0, p1)
	s.Lines().Add(p1, p2)
	return def.Sketches().Count() - 1
}

// TestSheetMetalLoftedFlangeApply lofts a wall between two open profiles on a sheet-metal part
// and confirms one healthy solid; then checks the error paths.
func TestSheetMetalLoftedFlangeApply(t *testing.T) {
	s := sheetMetalProfiledPart(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	planeB, _ := sketch.NewPlane(math.P3(0, 0, 3), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	a := addLProfile(def, sketch.XYPlane(), 1, 1)
	b := addLProfile(def, planeB, 2, 2)

	out, err := applyMap(t, s, "sheetMetalLoftedFlange", map[string]any{"profileA": a, "profileB": b})
	if err != nil {
		t.Fatalf("lofted flange apply: %v", err)
	}
	expectMergedSolid(t, out, "loftedFlange")

	// A press-brake output with a facet distance plumbs through to a healthy faceted wall.
	if _, err := applyMap(t, s, "sheetMetalLoftedFlange", map[string]any{
		"profileA": a, "profileB": b, "outputType": "pressBrakeFacetDistance", "facetTolerance": "1 mm",
	}); err != nil {
		t.Fatalf("press-brake lofted flange apply: %v", err)
	}

	// Error paths.
	if _, err := apply(t, profiledPart(t), "sheetMetalLoftedFlange", `{"profileA":0,"profileB":0}`); err == nil {
		t.Error("lofted flange on a non-sheet-metal part must error")
	}
	if _, err := apply(t, s, "sheetMetalLoftedFlange", `{"profileA":99,"profileB":0}`); err == nil {
		t.Error("lofted flange with an out-of-range profile must error")
	}
	if _, err := applyMap(t, s, "sheetMetalLoftedFlange", map[string]any{
		"profileA": a, "profileB": b, "outputType": "handHammered",
	}); err == nil {
		t.Error("lofted flange with an unknown outputType must error")
	}
}
