// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
)

func TestAllRegionsCounts(t *testing.T) {
	rs := allRegions()
	if len(rs) != 26 {
		t.Fatalf("regions = %d, want 26", len(rs))
	}
	var faces, edges, corners int
	for _, r := range rs {
		switch r.Kind {
		case RegionFace:
			faces++
		case RegionEdge:
			edges++
		case RegionCorner:
			corners++
		}
	}
	if faces != 6 || edges != 12 || corners != 8 {
		t.Errorf("kinds = %d faces / %d edges / %d corners, want 6/12/8", faces, edges, corners)
	}
}

// topDownCamera looks straight down −Z from +Z (so the TOP face faces the viewer), the
// frame the default view uses.
func topDownCamera() scene.Camera {
	return scene.Camera{Eye: math.P3(0, 0, 10), Target: math.P3(0, 0, 0), Up: math.V3(0, 1, 0)}
}

func TestHitTestCenterIsFrontFace(t *testing.T) {
	const radius = 40
	r := HitTest(0, 0, radius, topDownCamera(), doc.IdentityCubeOrient())
	if r == nil || r.Kind != RegionFace || r.Label != "TOP" {
		t.Fatalf("center hit = %+v, want TOP face", r)
	}
}

func TestHitTestCornerZone(t *testing.T) {
	const radius = 40
	// Up-right on screen (−Y px = up) maps to +X,+Y world; on the +Z (TOP) face that is
	// the (1,1,1) corner.
	r := HitTest(0.8*radius, -0.8*radius, radius, topDownCamera(), doc.IdentityCubeOrient())
	if r == nil || r.Kind != RegionCorner || r.X != 1 || r.Y != 1 || r.Z != 1 {
		t.Fatalf("corner hit = %+v, want corner (1,1,1)", r)
	}
}

func TestHitTestEdgeZone(t *testing.T) {
	const radius = 40
	// Straight up on screen (+Y world), centered in X: the top face's back edge (0,1,1).
	r := HitTest(0, -0.8*radius, radius, topDownCamera(), doc.IdentityCubeOrient())
	if r == nil || r.Kind != RegionEdge || r.X != 0 || r.Y != 1 || r.Z != 1 {
		t.Fatalf("edge hit = %+v, want edge (0,1,1)", r)
	}
}

func TestHitTestMissOutside(t *testing.T) {
	if r := HitTest(5*40, 5*40, 40, topDownCamera(), doc.IdentityCubeOrient()); r != nil {
		t.Errorf("far cursor should miss, got %+v", r)
	}
}

func TestSnapCameraPlacesEyeOnRegionSide(t *testing.T) {
	cur := scene.Camera{Eye: math.P3(0, 0, 10), Target: math.P3(0, 0, 0), Up: math.V3(0, 1, 0)}
	top := Region{Z: 1, Kind: RegionFace, Label: "TOP"}
	got := top.SnapCamera(cur, math.P3(0, 0, 0), doc.IdentityCubeOrient())
	if got.Eye != math.P3(0, 0, 10) { // +Z side, distance preserved
		t.Errorf("TOP snap eye = %v, want (0,0,10)", got.Eye)
	}
	right := Region{X: 1, Kind: RegionFace, Label: "RIGHT"}
	got = right.SnapCamera(cur, math.P3(0, 0, 0), doc.IdentityCubeOrient())
	if got.Eye != math.P3(10, 0, 0) {
		t.Errorf("RIGHT snap eye = %v, want (10,0,0)", got.Eye)
	}
}

func TestVisibleFacesFromTopShowsTop(t *testing.T) {
	faces := visibleFaces(topDownCamera(), doc.IdentityCubeOrient(), 40)
	if len(faces) == 0 {
		t.Fatal("no visible faces from top-down view")
	}
	var sawTop bool
	for _, f := range faces {
		if f.region.Label == "TOP" {
			sawTop = true
		}
		if f.region.Label == "BOTTOM" {
			t.Error("BOTTOM should be back-facing from a top-down view")
		}
	}
	if !sawTop {
		t.Error("TOP face should be visible from a top-down view")
	}
}
