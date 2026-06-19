// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// TestValidateDefinitionFlagsCurvelessEdge covers the "edge has no curve" issue: an edge with
// in-range vertices but a nil curve is reported as a definition problem.
func TestValidateDefinitionFlagsCurvelessEdge(t *testing.T) {
	def := SurfaceBodyDefinition{
		Vertices: []VertexDefinition{{Position: math.P3(0, 0, 0)}, {Position: math.P3(1, 0, 0)}},
		Edges:    []EdgeDefinition{{StartVertex: 0, EndVertex: 1, Curve: nil}},
	}
	if issues := validateDefinitionIndices(def); len(issues) == 0 {
		t.Error("an edge with no curve should be reported as a definition issue")
	}
}
