// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
)

// TestCreateWorkPointAtSelectedCloudPoint: with a snapped scan point selected, the command adds a
// datum point at that location; with nothing selected it is disabled and errors (#645).
func TestCreateWorkPointAtSelectedCloudPoint(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(3, 4, 5)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}

	if canWorkPointAtCloudPoint(s) {
		t.Error("Work Point should be disabled with nothing selected")
	}
	if _, err := s.CreateWorkPointAtSelectedCloudPoint(); err == nil {
		t.Error("expected an error with no scan point selected")
	}

	s.Select(PointCloudPointHandle{Cloud: pc, Point: math.P3(3, 4, 5)})
	if !canWorkPointAtCloudPoint(s) {
		t.Error("Work Point should be enabled with a scan point selected")
	}
	if got, ok := s.SelectedCloudPoint(); !ok || got != math.P3(3, 4, 5) {
		t.Errorf("SelectedCloudPoint = %v,%v want (3,4,5)", got, ok)
	}

	before := def.WorkPoints().Count()
	wp, err := s.CreateWorkPointAtSelectedCloudPoint()
	if err != nil {
		t.Fatalf("CreateWorkPointAtSelectedCloudPoint: %v", err)
	}
	if def.WorkPoints().Count() != before+1 {
		t.Errorf("work point count = %d, want %d", def.WorkPoints().Count(), before+1)
	}
	if wp.Point() != math.P3(3, 4, 5) {
		t.Errorf("work point at %v, want the snapped scan point (3,4,5)", wp.Point())
	}
}
