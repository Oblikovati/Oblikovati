// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
)

// TestMeasureBox checks edge length, face area and vertex distance against the analytic box
// (4×3×5 cm = 40×30×50 mm): total edge length 4(40+30+50) = 480 mm; total face area
// 2(40·30+40·50+30·50) = 9400 mm²; the space diagonal between opposite corners is √(40²+30²+50²).
func TestMeasureBox(t *testing.T) {
	block, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(4, 3, 5), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	q := ops.DefaultQuality()

	var totalEdge float64
	for _, e := range block.Edges() {
		totalEdge += EdgeLengthMm(e, q)
	}
	approx(t, "total edge length", totalEdge, 480)

	var totalArea float64
	for _, f := range block.Faces() {
		totalArea += FaceAreaMm2(f, q)
	}
	approx(t, "total face area", totalArea, 9400)

	verts := block.Vertices()
	var maxDist float64
	for i := range verts {
		for j := i + 1; j < len(verts); j++ {
			if d := VertexDistanceMm(verts[i], verts[j]); d > maxDist {
				maxDist = d
			}
		}
	}
	approx(t, "space diagonal", maxDist, math.Sqrt(40*40+30*30+50*50))
}
