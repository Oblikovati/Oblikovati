// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// partWithBlock builds a part whose evaluated body is the box [min,max], via a base
// feature, so PlacedBodies has real geometry to flatten.
func partWithBlock(t *testing.T, min, max math.Point3) *PartComponentDefinition {
	t.Helper()
	block, err := brep.SolidBlock(min, max, "p")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	p := NewPartComponentDefinition()
	feature.NewBaseFeatures(p.Features()).AddBase(block)
	p.Recompute()
	return p
}

// TestAssemblyPlacedBodiesFlattensTree checks the flatten: a part nested through a
// sub-assembly is emitted once with the composed world transform, and a directly-placed
// part once at its own transform.
func TestAssemblyPlacedBodiesFlattensTree(t *testing.T) {
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))

	sub := NewAssemblyComponentDefinition()
	sub.Place("p:1", part, math.Translation4(math.V3(5, 0, 0)))

	top := NewAssemblyComponentDefinition()
	top.Place("sub:1", sub, math.Translation4(math.V3(100, 0, 0)))
	top.Place("p:loose", part, math.Identity4())

	placed := top.PlacedBodies()
	if len(placed) != 2 {
		t.Fatalf("placed bodies = %d, want 2 (one nested, one loose)", len(placed))
	}
	var nestedX, looseX float64 = -1, -1
	for _, pb := range placed {
		x := float64(pb.Transform.TransformPoint(math.P3(0, 0, 0)).X)
		if x > 50 {
			nestedX = x
		} else {
			looseX = x
		}
	}
	if nestedX != 105 { // top(+100) ∘ sub-occurrence(+5)
		t.Errorf("nested part world X = %g, want 105 (100+5 composed)", nestedX)
	}
	if looseX != 0 {
		t.Errorf("loose part world X = %g, want 0", looseX)
	}
}

// TestAssemblyPlacedBodiesSkipsSuppressed checks suppressed occurrences contribute no
// bodies.
func TestAssemblyPlacedBodiesSkipsSuppressed(t *testing.T) {
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	top := NewAssemblyComponentDefinition()
	top.Place("keep:1", part, math.Identity4())
	dropped := top.Place("drop:1", part, math.Translation4(math.V3(9, 0, 0)))
	dropped.SetSuppressed(true)

	if got := len(top.PlacedBodies()); got != 1 {
		t.Errorf("placed bodies = %d, want 1 (suppressed occurrence skipped)", got)
	}
}

// TestFlexibleIndependentChildPositions checks the M12-F06 independent solve: a flexible
// sub-assembly occurrence positions its child independently of another placement of the same
// (shared) sub-assembly definition.
func TestFlexibleIndependentChildPositions(t *testing.T) {
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	sub := NewAssemblyComponentDefinition()
	sub.Place("p:1", part, math.Identity4()) // shared default: child at the origin

	top := NewAssemblyComponentDefinition()
	flex := top.Place("sub:1", sub, math.Identity4())             // placement 1, at x=0
	top.Place("sub:2", sub, math.Translation4(math.V3(10, 0, 0))) // placement 2, at x=10

	// Make placement 1 flexible and lift its child by +20 in Z — only this placement changes.
	flex.SetFlexible(true)
	flex.SetChildTransform("p:1", math.Translation4(math.V3(0, 0, 20)))

	placed := top.PlacedBodies()
	if len(placed) != 2 {
		t.Fatalf("placed bodies = %d, want 2", len(placed))
	}
	var flexZ, rigidZ float64 = -1, -1
	for _, pb := range placed {
		p0 := pb.Transform.TransformPoint(math.P3(0, 0, 0))
		if float64(p0.X) > 5 {
			rigidZ = float64(p0.Z) // placement 2 (x=10) — the shared default
		} else {
			flexZ = float64(p0.Z) // placement 1 (x=0) — the flexible override
		}
	}
	if flexZ != 20 {
		t.Errorf("flexible placement child Z = %g, want 20 (independent override)", flexZ)
	}
	if rigidZ != 0 {
		t.Errorf("other placement child Z = %g, want 0 (shared default, unaffected)", rigidZ)
	}
}
