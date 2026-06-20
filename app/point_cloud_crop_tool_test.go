// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

func cropTestCloud(t *testing.T, def interface {
	AddResource(doc.Resource) string
	PointClouds() *pointcloud.PointClouds
}) *pointcloud.PointCloud {
	t.Helper()
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	return pc
}

// TestCloudPointsBoxInRect: a full-viewport rectangle encloses every scan point (the crop box is
// the cloud's bounds), and an off-screen rectangle encloses none (#645).
func TestCloudPointsBoxInRect(t *testing.T) {
	s, def := emptyPartSession(t) // 200×200 camera looking down +Z at the XY plane
	pc := cropTestCloud(t, def)

	box, ok := s.cloudPointsBoxInRect(pc, math.P2(0, 0), math.P2(200, 200))
	if !ok {
		t.Fatal("full-viewport rect should enclose the scan points")
	}
	if box.Min != math.P3(0, 0, 0) || box.Max != math.P3(1, 1, 0) {
		t.Errorf("crop box = [%v,%v], want [(0,0,0),(1,1,0)]", box.Min, box.Max)
	}
	if _, ok := s.cloudPointsBoxInRect(pc, math.P2(-80, -80), math.P2(-40, -40)); ok {
		t.Error("an off-screen rect should enclose no points")
	}
}

// TestCropBoxToolFlow: two viewport clicks through the active crop tool add a crop to the cloud;
// the command path requires a selected cloud (#645).
func TestCropBoxToolFlow(t *testing.T) {
	s, def := emptyPartSession(t)
	pc := cropTestCloud(t, def)

	before := pc.Crops().Count()
	s.StartTool(NewCropBoxTool(pc))
	s.Click(0, 0)     // first corner
	s.Click(200, 200) // opposite corner → auto-commits the crop
	if pc.Crops().Count() != before+1 {
		t.Fatalf("crop count = %d, want %d after boxing the whole cloud", pc.Crops().Count(), before+1)
	}

	// Command gating: disabled with nothing selected, enabled with a cloud selected.
	if canCropSelectedCloud(s) {
		t.Error("Crop Box should be disabled with nothing selected")
	}
	if err := s.StartCropSelectedCloud(); err == nil {
		t.Error("StartCropSelectedCloud should error with no cloud selected")
	}
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})
	if !canCropSelectedCloud(s) {
		t.Error("Crop Box should be enabled with a cloud selected")
	}
	if err := s.StartCropSelectedCloud(); err != nil {
		t.Fatalf("StartCropSelectedCloud: %v", err)
	}
}
