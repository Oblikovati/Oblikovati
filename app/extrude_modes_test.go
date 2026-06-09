// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// squareRegion adds a 2×2 square sketch to the part and recomputes so its profile is
// detected, returning the sketch.
func squareRegion(def *compdef.PartComponentDefinition) *sketch.Sketch {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-1, -1))
	c1 := sk.Points().Add(math.P2(1, -1))
	c2 := sk.Points().Add(math.P2(1, 1))
	c3 := sk.Points().Add(math.P2(-1, 1))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	def.Recompute()
	return sk
}

func TestExtrudeToolSymmetricExtentFlowsToGeometry(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := squareRegion(def)
	tool := NewExtrudeTool()
	s.StartTool(tool)
	tool.Pick(s, ProfileHandle{Sketch: sk, ProfileIndex: 0})
	tool.SetDirection(feature.SymmetricDir)
	tool.SetDistance(6) // model units (cm)
	if !tool.CanCommit() {
		t.Fatal("tool should be committable with a region and distance")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	bodies := def.SurfaceBodies().All()
	if len(bodies) != 1 {
		t.Fatalf("got %d bodies, want 1", len(bodies))
	}
	// Symmetric distance 6 → the solid spans z ∈ [-3, 3].
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range bodies[0].Vertices() {
		z := float64(v.Point().Z)
		lo, hi = stdmath.Min(lo, z), stdmath.Max(hi, z)
	}
	if stdmath.Abs(lo+3) > 1e-9 || stdmath.Abs(hi-3) > 1e-9 {
		t.Errorf("symmetric extrude z-range = [%g,%g], want [-3,3]", lo, hi)
	}
}

func TestExtrudeToolThroughAllNeedsNoDistance(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := squareRegion(def)
	tool := NewExtrudeTool()
	s.StartTool(tool)
	tool.Pick(s, ProfileHandle{Sketch: sk, ProfileIndex: 0})
	tool.SetExtentType(feature.ThroughAllExtent)
	// Through-all is gauged from the model, so no distance is required to commit.
	if !tool.CanCommit() {
		t.Error("through-all extrude should be committable without a distance")
	}
}
