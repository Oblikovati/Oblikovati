// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"testing"

	"oblikovati.org/math"
)

// TestPointMarkersBuildsCrosses: each point becomes a 3-axis cross (6 vertices, 3 segments), and
// the marker spans the requested size on each axis.
func TestPointMarkersBuildsCrosses(t *testing.T) {
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(10, 0, 0)}
	item := PointMarkers(pts, 2, PointCloudColor, 7)
	if item == nil {
		t.Fatal("PointMarkers returned nil for non-empty points")
	}
	if item.Primitive != Lines || item.ObjectID != 7 {
		t.Errorf("primitive/id = %v/%d, want Lines/7", item.Primitive, item.ObjectID)
	}
	if len(item.Positions) != 12 || len(item.Indices) != 12 {
		t.Fatalf("positions/indices = %d/%d, want 12/12 (2 points × 6)", len(item.Positions), len(item.Indices))
	}
	// First point's X-axis segment spans ±1 about the origin (size 2).
	if item.Positions[0] != math.P3(-1, 0, 0) || item.Positions[1] != math.P3(1, 0, 0) {
		t.Errorf("x-axis segment = %v..%v, want (-1,0,0)..(1,0,0)", item.Positions[0], item.Positions[1])
	}
}

// TestPointMarkersColoredRepeatsPerPointColors checks the helper repeats each per-point color
// across the six vertices that make up the marker cross.
func TestPointMarkersColoredRepeatsPerPointColors(t *testing.T) {
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}
	cols := [][4]float32{{1, 0, 0, 1}, {0, 1, 0, 1}}
	item := PointMarkersColored(pts, 2, cols, 7)
	if item == nil {
		t.Fatal("PointMarkersColored returned nil for non-empty points")
	}
	if len(item.Colors) != 12 {
		t.Fatalf("color count = %d, want 12", len(item.Colors))
	}
	for i := 0; i < 6; i++ {
		if item.Colors[i] != cols[0] {
			t.Fatalf("first point color %d = %v, want %v", i, item.Colors[i], cols[0])
		}
	}
	for i := 6; i < 12; i++ {
		if item.Colors[i] != cols[1] {
			t.Fatalf("second point color %d = %v, want %v", i, item.Colors[i], cols[1])
		}
	}
}

// TestPointMarkersEmptyAndDegenerate: no points or a non-positive size yields no item.
func TestPointMarkersEmptyAndDegenerate(t *testing.T) {
	if PointMarkers(nil, 2, PointCloudColor, 0) != nil {
		t.Error("no points should yield nil")
	}
	if PointMarkers([]math.Point3{math.P3(0, 0, 0)}, 0, PointCloudColor, 0) != nil {
		t.Error("non-positive size should yield nil")
	}
}
