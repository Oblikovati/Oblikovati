// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestCreatePointCloudPointSnapsToScan: placing a cloud work point at a query near a scan point
// snaps it onto the nearest scan point (on z = 5) and keeps a healthy, cloud-anchored datum — the
// wire-reachable AddByCloudPoint path (#1842).
func TestCreatePointCloudPointSnapsToScan(t *testing.T) {
	s, def := emptyPartSession(t)
	attachPlanarCloud(t, def)

	wp, err := s.CreatePointCloudPoint("Scan", math.P3(2, 2, 4)) // nearest scan point is (2,2,5)
	if err != nil {
		t.Fatalf("CreatePointCloudPoint: %v", err)
	}
	if !wp.Health().OK() {
		t.Fatalf("cloud work point sick: %+v", wp.Health())
	}
	if got := wp.Point(); !got.IsEqualTo(math.P3(2, 2, 5), 1e-9) {
		t.Errorf("snapped point = %v, want (2,2,5)", got)
	}
	if stdmath.Abs(float64(wp.Point().Z)-5) > 1e-9 {
		t.Errorf("point Z = %v, want 5 (the scan plane)", wp.Point().Z)
	}
}

// TestCreatePointCloudPointErrors: an unknown cloud name is a clean error, not a panic.
func TestCreatePointCloudPointErrors(t *testing.T) {
	s, _ := emptyPartSession(t)
	if _, err := s.CreatePointCloudPoint("missing", math.P3(0, 0, 0)); err == nil {
		t.Error("want error for an unknown cloud name")
	}
}
