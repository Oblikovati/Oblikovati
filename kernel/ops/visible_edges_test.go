// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/math"
)

// A box has only sharp (90°) edges, so tangent-edge suppression must keep every one.
func TestVisibleEdgesKeepsSharpBoxEdges(t *testing.T) {
	m := subd.Box(2, 2, 2)
	b := subd.ToBody(m, "box")
	got := len(ops.VisibleEdges(b, ops.DefaultQuality(), ops.DefaultCreaseAngle()))
	if want := len(b.Edges()); got != want {
		t.Errorf("box visible edges = %d, want all %d (no sharp edge should be suppressed)", got, want)
	}
}

// A solid cylinder's vertical seam is a parametric seam internal to the smooth side face — it
// must be suppressed — while the two rim circles (side meets cap at 90°) are kept.
func TestVisibleEdgesSuppressesCylinderSeam(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	total := len(cyl.Edges())
	got := len(ops.VisibleEdges(cyl, ops.DefaultQuality(), ops.DefaultCreaseAngle()))
	if got >= total {
		t.Errorf("cylinder visible edges = %d, want < %d (the seam should be suppressed)", got, total)
	}
	if got < 2 {
		t.Errorf("cylinder visible edges = %d, want the two rim circles kept", got)
	}
}
