// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
)

// TestMinDistanceBox checks minimum distance on a 4×3×5 cm (40×30×50 mm) block:
//   - the farthest face pair (opposite 40×30 faces) is 50 mm apart;
//   - some face pair touches (adjacent faces share an edge) → 0;
//   - a corner vertex sits on three faces (distance 0) and is 30/40/50 mm from the opposite three;
//   - opposite corners are the space diagonal apart, matching VertexDistanceMm.
func TestMinDistanceBox(t *testing.T) {
	block, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(4, 3, 5), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	q := ops.DefaultQuality()
	faces := block.Faces()

	maxPair, minPair := 0.0, math.Inf(1)
	for i := range faces {
		for j := i + 1; j < len(faces); j++ {
			d := MinDistanceMm(MeasureEntity{Face: faces[i]}, MeasureEntity{Face: faces[j]}, q)
			maxPair = math.Max(maxPair, d)
			minPair = math.Min(minPair, d)
		}
	}
	if math.Abs(maxPair-50) > 1e-6 {
		t.Errorf("max face-pair min-distance = %g mm, want 50 (opposite 40×30 faces)", maxPair)
	}
	if minPair > 1e-6 {
		t.Errorf("min face-pair min-distance = %g mm, want 0 (adjacent faces touch)", minPair)
	}

	// A corner vertex: three faces touch it (0), the opposite three are the box dimensions away.
	v := block.Vertices()[0]
	var gaps []float64
	for _, f := range faces {
		if d := MinDistanceMm(MeasureEntity{Vertex: v}, MeasureEntity{Face: f}, q); d > 1e-6 {
			gaps = append(gaps, d)
		}
	}
	if !sameSet(gaps, []float64{30, 40, 50}) {
		t.Errorf("vertex-to-opposite-face gaps = %v mm, want {30,40,50}", gaps)
	}

	// Vertex-vertex min-distance equals the straight-line vertex distance (space diagonal here).
	var diag float64
	for _, w := range block.Vertices() {
		if d := MinDistanceMm(MeasureEntity{Vertex: v}, MeasureEntity{Vertex: w}, q); d > diag {
			diag = d
		}
	}
	if want := math.Sqrt(40*40 + 30*30 + 50*50); math.Abs(diag-want) > 1e-6 {
		t.Errorf("space diagonal = %g mm, want %g", diag, want)
	}
}

// TestMinDistanceProbeToBody checks the transient-probe distance on the same 4×3×5 cm block: a
// travel segment above the top face clears it by the gap, a segment piercing the block touches (0),
// a lone point measures point-to-body, and an empty probe is +Inf. Results are database units (cm).
func TestMinDistanceProbeToBody(t *testing.T) {
	block, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(4, 3, 5), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	q := ops.DefaultQuality()

	// A horizontal travel segment 5 cm above the top face (z=5).
	above := []gmath.Point3{gmath.P3(0, 1.5, 10), gmath.P3(4, 1.5, 10)}
	if d := MinDistanceProbeToBody(above, block, q); math.Abs(d-5) > 1e-6 {
		t.Errorf("above-box probe distance = %g cm, want 5", d)
	}
	// A segment piercing the block touches it.
	through := []gmath.Point3{gmath.P3(2, 1.5, 2), gmath.P3(2, 1.5, 3)}
	if d := MinDistanceProbeToBody(through, block, q); d > 1e-6 {
		t.Errorf("through-box probe distance = %g cm, want 0", d)
	}
	// A lone point 2 cm beyond the +x face.
	if d := MinDistanceProbeToBody([]gmath.Point3{gmath.P3(6, 1.5, 2.5)}, block, q); math.Abs(d-2) > 1e-6 {
		t.Errorf("point probe distance = %g cm, want 2", d)
	}
	// No probe geometry → +Inf.
	if d := MinDistanceProbeToBody(nil, block, q); !math.IsInf(d, 1) {
		t.Errorf("empty probe distance = %g, want +Inf", d)
	}
}

// sameSet reports whether got contains exactly the wanted values (within tolerance, any order).
func sameSet(got, want []float64) bool {
	if len(got) != len(want) {
		return false
	}
	used := make([]bool, len(got))
	for _, w := range want {
		matched := false
		for i, g := range got {
			if !used[i] && math.Abs(g-w) < 1e-6 {
				used[i], matched = true, true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
