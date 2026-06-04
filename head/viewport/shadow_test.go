// SPDX-License-Identifier: GPL-2.0-only

package viewport

import (
	"math"
	"testing"
)

// triVert builds one 16-float vertex at position (x,y,z); the other lanes are irrelevant to
// bounds/shadow framing.
func triVert(x, y, z float32) []float32 {
	return []float32{x, y, z, 0, 0, 1, 1, 1, 1, 1, 0, 0.5, 0, 0, 0, 2}
}

// TestSceneBoundsSpansGeometry checks the bounds cover every vertex of both the shaded and
// occluder streams.
func TestSceneBoundsSpansGeometry(t *testing.T) {
	var m Mesh
	m.TriVerts = append(append(triVert(-1, -2, 0), triVert(3, 0, 1)...), triVert(0, 5, -4)...)
	m.OccVerts = triVert(0, 0, 9)
	min, max, ok := SceneBounds(m)
	if !ok {
		t.Fatal("bounds reported empty for non-empty mesh")
	}
	if min != [3]float32{-1, -2, -4} || max != [3]float32{3, 5, 9} {
		t.Errorf("bounds = %v..%v, want {-1 -2 -4}..{3 5 9}", min, max)
	}
}

// TestSceneBoundsEmpty checks an empty mesh reports ok=false (no shadow frustum to build).
func TestSceneBoundsEmpty(t *testing.T) {
	if _, _, ok := SceneBounds(Mesh{}); ok {
		t.Error("empty mesh should report ok=false")
	}
}

// TestLightMatrixProjectsBoundsIntoClip checks the light matrix maps the bounding box into
// Vulkan clip space: every corner lands in x,y ∈ [-1,1] and z ∈ [0,1], so the whole model is
// inside the shadow frustum (ADR-0026 §6).
func TestLightMatrixProjectsBoundsIntoClip(t *testing.T) {
	min, max := [3]float32{-2, -2, -2}, [3]float32{2, 2, 2}
	lvp := LightMatrix(min, max, [3]float32{0.4, 0.6, 0.8})
	for _, cx := range []float32{min[0], max[0]} {
		for _, cy := range []float32{min[1], max[1]} {
			for _, cz := range []float32{min[2], max[2]} {
				x, y, z, w := apply(lvp, cx, cy, cz)
				if w == 0 {
					t.Fatal("degenerate w")
				}
				x, y, z = x/w, y/w, z/w
				if math.Abs(float64(x)) > 1.001 || math.Abs(float64(y)) > 1.001 || z < -0.001 || z > 1.001 {
					t.Errorf("corner (%g,%g,%g) → clip (%g,%g,%g) outside the frustum", cx, cy, cz, x, y, z)
				}
			}
		}
	}
}

// apply multiplies a column-major 4×4 by (x,y,z,1).
func apply(m [16]float32, x, y, z float32) (float32, float32, float32, float32) {
	return m[0]*x + m[4]*y + m[8]*z + m[12],
		m[1]*x + m[5]*y + m[9]*z + m[13],
		m[2]*x + m[6]*y + m[10]*z + m[14],
		m[3]*x + m[7]*y + m[11]*z + m[15]
}
