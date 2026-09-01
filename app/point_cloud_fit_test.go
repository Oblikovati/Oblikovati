// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

// attachPlanarCloud adds a cloud whose points lie on z = 5 (normal ±Z) to the active part.
func attachPlanarCloud(t *testing.T, def *compdef.PartComponentDefinition) *pointcloud.PointCloud {
	t.Helper()
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pts := []math.Point3{
		math.P3(0, 0, 5), math.P3(2, 0, 5), math.P3(0, 2, 5),
		math.P3(2, 2, 5), math.P3(1, 3, 5), math.P3(-1, 1, 5),
	}
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, pts)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	return pc
}

// TestCreatePointCloudPlaneFitsZPlane: fitting the planar cloud yields a work plane whose normal is
// ±Z and whose origin sits on z = 5.
func TestCreatePointCloudPlaneFitsZPlane(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	attachPlanarCloud(t, def)

	wp, plane, err := s.CreatePointCloudPlane("Scan")
	if err != nil {
		t.Fatalf("CreatePointCloudPlane: %v", err)
	}
	if wp == nil || wp.Name() == "" {
		t.Fatal("expected a named work plane")
	}
	n := plane.Normal()
	if stdmath.Abs(stdmath.Abs(float64(n.Z))-1) > 1e-6 {
		t.Errorf("normal = %v, want ±Z", n)
	}
	if stdmath.Abs(float64(plane.Origin.Z)-5) > 1e-9 {
		t.Errorf("origin Z = %v, want 5", plane.Origin.Z)
	}
}

func TestCreatePointCloudPlaneErrors(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	if _, _, err := s.CreatePointCloudPlane("missing"); err == nil {
		t.Error("want error for an unknown cloud name")
	}
	// A cloud with two points cannot determine a plane.
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	if _, err := def.PointClouds().Add("Tiny", "t.xyz", rid, []math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 1)}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if _, _, err := s.CreatePointCloudPlane("Tiny"); err == nil {
		t.Error("want error fitting a plane to two points")
	}
}

// TestFitSelectedCloudPlaneAndEnable: the command path fits the browser-selected cloud, and the
// enable predicate tracks whether a cloud is selected.
func TestFitSelectedCloudPlaneAndEnable(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	pc := attachPlanarCloud(t, def)

	if canFitPointCloudPlane(s) {
		t.Error("Fit Work Plane should be disabled with nothing selected")
	}
	if _, err := s.FitSelectedCloudPlane(); err == nil {
		t.Error("FitSelectedCloudPlane should error with no cloud selected")
	}

	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})
	if !canFitPointCloudPlane(s) {
		t.Error("Fit Work Plane should be enabled with a cloud selected")
	}
	if got, ok := s.SelectedPointCloud(); !ok || got != pc {
		t.Errorf("SelectedPointCloud = %v,%v want the attached cloud", got, ok)
	}
	wp, err := s.FitSelectedCloudPlane()
	if err != nil || wp == nil {
		t.Fatalf("FitSelectedCloudPlane = %v, %v", wp, err)
	}
}
