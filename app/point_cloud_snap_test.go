// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

// cloudPicker builds a ray picker over a session's attached clouds (no bodies), framed by the
// session camera.
func cloudPicker(s *Session) *RayPicker {
	p := NewRayPicker(s.Camera(), func() []*topo.Body { return nil }).
		WithPointClouds(func() []*pointcloud.PointCloud { return s.PickablePointClouds() })
	p.SetCamera(s.Camera())
	return p
}

// TestPickSnapsToCloudPoint: a click at the screen position of a scan point snaps to it, returning
// a PointCloudPointHandle at the point's model-space location (M17-F06, #645).
func TestPickSnapsToCloudPoint(t *testing.T) {
	s, def := emptyPartSession(t) // camera at (0,0,10) → origin projects to screen centre (100,100)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	if _, err := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(0, 0, 0)}); err != nil {
		t.Fatalf("attach: %v", err)
	}

	sel, ok := cloudPicker(s).Pick(100, 100, NewSelectionFilter())
	if !ok {
		t.Fatal("a click at the scan point's screen position should snap to it")
	}
	h, isCloud := sel.(PointCloudPointHandle)
	if !isCloud {
		t.Fatalf("picked %T, want PointCloudPointHandle", sel)
	}
	if h.Position() != math.P3(0, 0, 0) {
		t.Errorf("snapped location = %v, want the origin point", h.Position())
	}
}

// TestPickMissesHiddenCloud: a hidden cloud's points do not snap, and a click away from any point
// returns no hit (#645).
func TestPickMissesHiddenCloud(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, _ := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(0, 0, 0)})
	pc.SetVisible(false)

	if _, ok := cloudPicker(s).Pick(100, 100, NewSelectionFilter()); ok {
		t.Error("a hidden cloud should not snap")
	}
	pc.SetVisible(true)
	if _, ok := cloudPicker(s).Pick(10, 10, NewSelectionFilter()); ok {
		t.Error("a click far from any scan point should not snap")
	}
}

// TestPointCloudPointHandleKind: the snap handle reports its selection kind (#645).
func TestPointCloudPointHandleKind(t *testing.T) {
	if k := (PointCloudPointHandle{}).SelectionKind(); k != SelectPointCloudPoint {
		t.Errorf("SelectionKind = %v, want SelectPointCloudPoint", k)
	}
}

// TestSelectedCloudPointHighlight: selecting a snapped scan point yields an on-top highlight marker
// at its location; a non-cloud selection yields none (#645).
func TestSelectedCloudPointHighlight(t *testing.T) {
	s, _ := emptyPartSession(t)
	if _, ok := s.SelectedCloudPointHighlight(0.5); ok {
		t.Error("no selection should yield no highlight")
	}
	s.Select(PointCloudPointHandle{Point: math.P3(2, 0, 0)})
	hi, ok := s.SelectedCloudPointHighlight(0.5)
	if !ok || !hi.OnTop || len(hi.Positions) != 6 {
		t.Errorf("highlight = %+v (ok=%v), want an on-top 6-vertex cross", hi, ok)
	}
}
