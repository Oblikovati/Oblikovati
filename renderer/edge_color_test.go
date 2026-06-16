// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestSetEdgeColorColorsEdges checks the display-settings edge-color override (M16-F07 #643)
// is applied to the wireframe line items of a Shaded-with-Edges draw list.
func TestSetEdgeColorColorsEdges(t *testing.T) {
	red := [4]float32{1, 0, 0, 1}
	SetEdgeColor(red)
	defer SetEdgeColor(DefaultEdgeColor())
	if EdgeColor() != red {
		t.Fatalf("EdgeColor() = %v, want red after SetEdgeColor", EdgeColor())
	}
	list := BuildDrawListStyled([]*topo.Body{box(2, math.V3(0, 0, 0))}, frontCamera(), ops.DefaultQuality(), nil, ShadedWithEdges)
	var lines int
	for _, it := range list.Items {
		if it.Primitive == Lines {
			lines++
			if it.Color != red {
				t.Errorf("edge line color = %v, want red", it.Color)
			}
		}
	}
	if lines == 0 {
		t.Error("Shaded-with-Edges produced no edge line items")
	}
}
