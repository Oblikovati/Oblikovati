// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestTwoHolesAreTwoNamespaces is ADR-0043's one-namespace-per-feature-instance rule at the hole
// feature: two drilled holes in one plate are two bore walls with two DISTINCT keys, each resolving
// to exactly one face. The exact drill the feature used to call directly minted brep:drillwall#0 for
// every hole, so a part with two holes carried one key twice and any pick of a bore wall was
// ambiguous (ADR-0061 stage 4).
func TestTwoHolesAreTwoNamespaces(t *testing.T) {
	t.Parallel()
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 4}, {X: 0, Y: 4}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	top := block.Faces()[1].ReferenceKey()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(block)
	holes := NewHoleFeatures(fs)
	c1, c2 := math.P3(3, 2, 2), math.P3(7, 2, 2)
	holes.addHole(&HoleDefinition{PlacementFaceKey: top, Diameter: constFloat(2), Depth: constFloat(3), Type: DrilledHole, Center: &c1})
	holes.addHole(&HoleDefinition{PlacementFaceKey: top, Diameter: constFloat(2), Depth: constFloat(3), Type: DrilledHole, Center: &c2})
	fs.Recompute()
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.ValidSolid() {
		t.Fatalf("two-hole plate is not a valid solid: %+v", r.Issues)
	}
	seen := map[string]int{}
	walls := 0
	for _, f := range body.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			continue
		}
		walls++
		key := string(f.ReferenceKey())
		seen[key]++
		if n := len(body.FacesByKey(f.ReferenceKey())); n != 1 {
			t.Errorf("bore wall key %q resolves to %d faces, want exactly one", key, n)
		}
	}
	if walls != 2 {
		t.Fatalf("two-hole plate has %d bore walls, want 2", walls)
	}
	if len(seen) != 2 {
		t.Errorf("the two bore walls share a key: %v", seen)
	}
}
