// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// curvedFaces counts the faces whose surface is still analytic-curved.
func curvedFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			n++
		}
	}
	return n
}

// TestPatternedCutKeepsAnalyticCylinders is the regression for #3463: a pattern must not facet
// its operands.
//
// The size assertion in TestAnalyticPatternedCutDoesNotExplode (≤2000 edges) did NOT catch this —
// the faceted result was 194 planar faces / 576 edges and passed it comfortably. Only the face
// KIND shows the loss, and the loss is what matters downstream: a fillet, chamfer, thread or STEP
// export against a 32-gon prism is working on facets where the model has a cylinder.
//
// Five Ø6 bores round a Ø60 disc is 6 cylinder walls (the disc's own + five bores) and 2 planar
// caps — exactly what the kernel produces when it is handed the analytic operands.
func TestPatternedCutKeepsAnalyticCylinders(t *testing.T) {
	t.Parallel()
	body := discThenCutPatterned(t)
	if got := curvedFaces(body); got != 6 {
		t.Errorf("patterned disc has %d curved faces of %d total, want 6 (disc wall + 5 bores) — "+
			"the pattern faceted its operands (#3463)", got, len(body.Faces()))
	}
	if got := len(body.Faces()); got != 8 {
		t.Errorf("patterned disc has %d faces, want 8 — a faceted result runs to ~194", got)
	}
}

// TestPatternedDrilledHoleKeepsAnalyticBores is the same regression through the HOLE feature,
// which is where the root cause lived: the hole cut with the exact drill but RECORDED a 32-gon
// prism as its replay tool, so the pattern re-applied a different solid than the hole removed.
// Three Ø2 bores in a block is 3 cylinder walls and 6 planar faces.
func TestPatternedDrilledHoleKeepsAnalyticBores(t *testing.T) {
	t.Parallel()
	corners := []math.Point2{{X: -8, Y: -2}, {X: 8, Y: -2}, {X: 8, Y: 2}, {X: -8, Y: 2}}
	fs := NewPartFeatures(nil)
	block := buildPrism(corners, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(block.Faces()[1].ReferenceKey(),
		func() float64 { return 2 }, func() float64 { return 3 })
	NewPatternFeatures(fs).AddRectangular([]ID{hole.ID()},
		func() int { return 3 }, func() int { return 1 }, math.V3(3, 0, 0), noStep)
	fs.Recompute()
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.ValidSolid() {
		t.Fatalf("patterned drilled holes are not a valid solid: %+v", r.Issues)
	}
	if got := curvedFaces(body); got != 3 {
		t.Errorf("patterned block has %d curved faces of %d total, want 3 bores — the recorded "+
			"drill tool went back to a faceted prism (#3463)", got, len(body.Faces()))
	}
}

// TestHoleReplayToolIsTheSolidTheHoleRemoved pins the root cause directly: the tool a hole records
// for replay must be the same shape it cut with. A prism is not a cylinder, and a pattern that
// replays a prism removes a different volume at every occurrence but the first.
func TestHoleReplayToolIsTheSolidTheHoleRemoved(t *testing.T) {
	t.Parallel()
	corners := []math.Point2{{X: -8, Y: -2}, {X: 8, Y: -2}, {X: 8, Y: 2}, {X: -8, Y: 2}}
	fs := NewPartFeatures(nil)
	block := buildPrism(corners, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(block.Faces()[1].ReferenceKey(),
		func() float64 { return 2 }, func() float64 { return 3 })
	fs.Recompute()
	tool, _, ok := fs.sourceTool(hole.ID())
	if !ok || tool == nil {
		t.Fatal("hole recorded no replay tool")
	}
	if got := curvedFaces(tool); got != 1 {
		t.Errorf("hole replay tool has %d curved faces of %d, want 1 cylinder wall — a faceted "+
			"tool replays a different solid than the hole cut (#3463)", got, len(tool.Faces()))
	}
}
