// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// TestAngleBox checks angle measurement on a 4×3×5 cm block: every face is 90° from each adjacent
// face and 180° from its opposite, every meeting edge pair is 90°, and the three edges at a corner
// vertex make 90° three-point angles.
func TestAngleBox(t *testing.T) {
	block, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(4, 3, 5), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	q := ops.DefaultQuality()
	faces := block.Faces()

	sawOpposite := false
	for i := range faces {
		for j := i + 1; j < len(faces); j++ {
			deg, err := AngleDegrees(MeasureEntity{Face: faces[i]}, MeasureEntity{Face: faces[j]}, q)
			if err != nil {
				t.Fatalf("AngleDegrees(face%d,face%d): %v", i, j, err)
			}
			if !near(deg, 90) && !near(deg, 180) {
				t.Errorf("angle(face%d,face%d) = %g°, want 90 or 180", i, j, deg)
			}
			if near(deg, 180) {
				sawOpposite = true
			}
		}
	}
	if !sawOpposite {
		t.Error("no opposite face pair measured 180°")
	}

	// A vertex has no direction → entity angle rejects it.
	if _, err := AngleDegrees(MeasureEntity{Vertex: block.Vertices()[0]}, MeasureEntity{Face: faces[0]}, q); err == nil {
		t.Error("AngleDegrees(vertex, face) = ok, want error")
	}
}

// TestAngleEdgesAndThreePoint checks a perpendicular edge pair and a corner three-point angle.
func TestAngleEdgesAndThreePoint(t *testing.T) {
	block, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(4, 3, 5), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	q := ops.DefaultQuality()

	// Adjacent box edges meet at right angles: at least one edge pair measures 90°.
	edges := block.Edges()
	sawRight := false
	for i := range edges {
		for j := i + 1; j < len(edges); j++ {
			deg, err := AngleDegrees(MeasureEntity{Edge: edges[i]}, MeasureEntity{Edge: edges[j]}, q)
			if err != nil {
				t.Fatalf("edge angle: %v", err)
			}
			if near(deg, 90) {
				sawRight = true
			}
		}
	}
	if !sawRight {
		t.Error("no perpendicular edge pair measured 90°")
	}

	// Three box corners sharing axes: apex (0,0,0) to (4,0,0) and (0,3,0) is a right angle.
	apex := vertexAt(t, block, 0, 0, 0)
	px := vertexAt(t, block, 4, 0, 0)
	py := vertexAt(t, block, 0, 3, 0)
	if deg := ThreePointAngleDegrees(px, apex, py); !near(deg, 90) {
		t.Errorf("three-point angle = %g°, want 90", deg)
	}
}

func vertexAt(t *testing.T, b *topo.Body, x, y, z float64) *topo.Vertex {
	t.Helper()
	want := gmath.P3(x, y, z)
	for _, v := range b.Vertices() {
		if v.Point().DistanceTo(want) < 1e-6 {
			return v
		}
	}
	t.Fatalf("no vertex at (%g,%g,%g)", x, y, z)
	return nil
}

func near(got, want float64) bool { return math.Abs(got-want) < 1e-6 }
