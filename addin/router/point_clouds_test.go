// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// writeScan writes a small ASCII scan file and returns its path.
func writeScan(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scan.xyz")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write scan: %v", err)
	}
	return path
}

// TestPointCloudAttachListGetDelete: the full attach → list → get → delete flow over the wire,
// with the cloud's point count and default state surfaced (M17-F06, #645).
func TestPointCloudAttachListGetDelete(t *testing.T) {
	r, s := emptyPartSession(t)
	path := writeScan(t, "0 0 0\n1 0 0\n2 0 0\n3 0 0\n")

	var attached wire.PointCloudInfo
	call(t, r, s, "pointClouds.attach", mustJSON(t, wire.AttachPointCloudArgs{FullFileName: path}), &attached)
	if attached.Name != "Cloud1" || attached.TotalPointCount != 4 || !attached.Visible || attached.Scale != 1 {
		t.Fatalf("attached = %+v, want Cloud1/4 pts/visible/scale 1", attached)
	}

	var list wire.ListPointCloudsResult
	call(t, r, s, "pointClouds.list", `{}`, &list)
	if len(list.PointClouds) != 1 || list.PointClouds[0].Name != "Cloud1" {
		t.Errorf("list = %+v, want one Cloud1", list.PointClouds)
	}

	var got wire.PointCloudInfo
	call(t, r, s, "pointClouds.get", `{"name":"Cloud1"}`, &got)
	if got.TotalPointCount != 4 {
		t.Errorf("get TotalPointCount = %d, want 4", got.TotalPointCount)
	}

	var del wire.DeletePointCloudResult
	call(t, r, s, "pointClouds.delete", `{"name":"Cloud1"}`, &del)
	if !del.Deleted {
		t.Error("delete should report Deleted=true")
	}
	call(t, r, s, "pointClouds.list", `{}`, &list)
	if len(list.PointClouds) != 0 {
		t.Errorf("after delete, list = %+v, want empty", list.PointClouds)
	}
}

// TestPointCloudPlacementAndBudget: setScale/setDensity/setVisible/setTransform mutate the cloud,
// and the space-conversion methods round-trip a point through the placement (#645).
func TestPointCloudPlacementAndBudget(t *testing.T) {
	r, s := emptyPartSession(t)
	path := writeScan(t, "0 0 0\n1 1 1\n2 2 2\n3 3 3\n4 4 4\n")
	call(t, r, s, "pointClouds.attach", mustJSON(t, wire.AttachPointCloudArgs{Name: "Scan", FullFileName: path}), &wire.PointCloudInfo{})

	var info wire.PointCloudInfo
	call(t, r, s, "pointClouds.setScale", `{"name":"Scan","scale":2}`, &info)
	if info.Scale != 2 {
		t.Errorf("scale = %v, want 2", info.Scale)
	}
	call(t, r, s, "pointClouds.setDensity", `{"name":"Scan","maximumPointCount":2}`, &info)
	if info.MaximumPointCount != 2 || info.DisplayedPointCount != 2 {
		t.Errorf("budget = max %d displayed %d, want 2/2", info.MaximumPointCount, info.DisplayedPointCount)
	}
	call(t, r, s, "pointClouds.setVisible", `{"name":"Scan","visible":false}`, &info)
	if info.Visible {
		t.Error("setVisible(false) should hide the cloud")
	}

	move := types.TranslationMatrix(types.Vector{X: 10, Y: 0, Z: 0})
	call(t, r, s, "pointClouds.setTransform", mustJSON(t, wire.SetPointCloudTransformArgs{Name: "Scan", Transform: move}), &info)

	// A cloud point (1,1,1) maps to model space scaled (×2) then translated (+10x) → (12,2,2).
	var sp wire.PointCloudSpaceResult
	call(t, r, s, "pointClouds.toModelSpace", mustJSON(t, wire.PointCloudSpaceArgs{Name: "Scan", Point: types.Point{X: 1, Y: 1, Z: 1}}), &sp)
	if !sp.OK || sp.Point != (types.Point{X: 12, Y: 2, Z: 2}) {
		t.Errorf("toModelSpace = %+v (ok=%v), want (12,2,2)", sp.Point, sp.OK)
	}
	call(t, r, s, "pointClouds.fromModelSpace", mustJSON(t, wire.PointCloudSpaceArgs{Name: "Scan", Point: sp.Point}), &sp)
	if !sp.OK || sp.Point != (types.Point{X: 1, Y: 1, Z: 1}) {
		t.Errorf("fromModelSpace round-trip = %+v (ok=%v), want (1,1,1)", sp.Point, sp.OK)
	}
}

// TestPointCloudCropLifecycle: add a crop, see it limit the displayed count, toggle it off and on,
// list it, and delete it — all over the wire (#645).
func TestPointCloudCropLifecycle(t *testing.T) {
	r, s := emptyPartSession(t)
	path := writeScan(t, "0 0 0\n1 0 0\n2 0 0\n3 0 0\n4 0 0\n5 0 0\n")
	var info wire.PointCloudInfo
	call(t, r, s, "pointClouds.attach", mustJSON(t, wire.AttachPointCloudArgs{Name: "Scan", FullFileName: path}), &info)
	if info.DisplayedPointCount != 6 {
		t.Fatalf("attached displayed count = %d, want 6", info.DisplayedPointCount)
	}

	// Crop to x in [0,2] → 3 points displayed.
	crop := wire.AddPointCloudCropArgs{Cloud: "Scan", Min: types.Point{X: -0.5, Y: -1, Z: -1}, Max: types.Point{X: 2.5, Y: 1, Z: 1}}
	var ci wire.PointCloudCropInfo
	call(t, r, s, "pointClouds.addCrop", mustJSON(t, crop), &ci)
	if ci.Crop != "Crop1" || !ci.Active {
		t.Fatalf("addCrop = %+v, want active Crop1", ci)
	}
	call(t, r, s, "pointClouds.get", `{"name":"Scan"}`, &info)
	if info.DisplayedPointCount != 3 {
		t.Errorf("cropped displayed count = %d, want 3 (x 0..2)", info.DisplayedPointCount)
	}

	// Deactivating restores the full set.
	call(t, r, s, "pointClouds.setCropActive", `{"cloud":"Scan","crop":"Crop1","active":false}`, &ci)
	call(t, r, s, "pointClouds.get", `{"name":"Scan"}`, &info)
	if info.DisplayedPointCount != 6 {
		t.Errorf("after deactivating crop, displayed = %d, want 6", info.DisplayedPointCount)
	}

	var list wire.ListPointCloudCropsResult
	call(t, r, s, "pointClouds.listCrops", `{"cloud":"Scan"}`, &list)
	if len(list.Crops) != 1 || list.Crops[0].Active {
		t.Errorf("listCrops = %+v, want one inactive crop", list.Crops)
	}

	var del wire.DeletePointCloudCropResult
	call(t, r, s, "pointClouds.deleteCrop", `{"cloud":"Scan","crop":"Crop1"}`, &del)
	call(t, r, s, "pointClouds.listCrops", `{"cloud":"Scan"}`, &list)
	if !del.Deleted || len(list.Crops) != 0 {
		t.Errorf("after delete: deleted=%v crops=%+v, want deleted/empty", del.Deleted, list.Crops)
	}
}

// TestPointCloudCropErrors: crop ops on a missing cloud or crop error (#645).
func TestPointCloudCropErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "pointClouds.addCrop", []byte(`{"cloud":"nope","min":{"x":0,"y":0,"z":0},"max":{"x":1,"y":1,"z":1}}`)); err == nil {
		t.Error("addCrop on a missing cloud should fail")
	}
	path := writeScan(t, "0 0 0\n")
	call(t, r, s, "pointClouds.attach", mustJSON(t, wire.AttachPointCloudArgs{Name: "S", FullFileName: path}), &wire.PointCloudInfo{})
	if _, err := r.Handle(s, "pointClouds.setCropActive", []byte(`{"cloud":"S","crop":"nope","active":true}`)); err == nil {
		t.Error("setCropActive on a missing crop should fail")
	}
	for _, m := range []string{"pointClouds.listCrops", "pointClouds.deleteCrop"} {
		if _, err := r.Handle(s, m, []byte(`{"cloud":"nope","crop":"x"}`)); err == nil {
			t.Errorf("%s on a missing cloud should fail", m)
		}
	}
}

// TestPointCloudMissingNameErrors: every name-keyed operation errors when no cloud has the name,
// exercising the not-found branches of each handler (#645).
func TestPointCloudMissingNameErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	ops := []struct {
		method string
		args   string
	}{
		{"pointClouds.get", `{"name":"nope"}`},
		{"pointClouds.setVisible", `{"name":"nope","visible":true}`},
		{"pointClouds.setScale", `{"name":"nope","scale":2}`},
		{"pointClouds.setDensity", `{"name":"nope","maximumPointCount":1}`},
		{"pointClouds.setTransform", `{"name":"nope","transform":{"cells":[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}}`},
		{"pointClouds.toModelSpace", `{"name":"nope","point":{"x":0,"y":0,"z":0}}`},
		{"pointClouds.fromModelSpace", `{"name":"nope","point":{"x":0,"y":0,"z":0}}`},
	}
	for _, op := range ops {
		if _, err := r.Handle(s, op.method, []byte(op.args)); err == nil {
			t.Errorf("%s on a missing cloud should fail", op.method)
		}
	}
	// delete of a missing cloud is not an error — it reports Deleted=false.
	var del wire.DeletePointCloudResult
	call(t, r, s, "pointClouds.delete", `{"name":"nope"}`, &del)
	if del.Deleted {
		t.Error("delete of a missing cloud should report Deleted=false")
	}
}

// TestPointCloudBadJSON: a malformed request body is rejected by the decode guard (#645).
func TestPointCloudBadJSON(t *testing.T) {
	r, s := emptyPartSession(t)
	for _, m := range []string{"pointClouds.attach", "pointClouds.get", "pointClouds.setScale", "pointClouds.toModelSpace"} {
		if _, err := r.Handle(s, m, []byte(`{`)); err == nil {
			t.Errorf("%s with malformed JSON should fail", m)
		}
	}
}

// TestPointCloudUnreadableScan: attaching a file whose extension has no reader fails (#645).
func TestPointCloudUnreadableScan(t *testing.T) {
	r, s := emptyPartSession(t)
	path := filepath.Join(t.TempDir(), "scan.las")
	if err := os.WriteFile(path, []byte("binary"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := r.Handle(s, "pointClouds.attach", []byte(mustJSON(t, wire.AttachPointCloudArgs{FullFileName: path}))); err == nil {
		t.Error("attach of an unsupported scan format should fail")
	}
}

// TestPointCloudAttachErrors: a missing file and a non-positive scale are rejected (#645).
func TestPointCloudAttachErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "pointClouds.attach", []byte(`{"fullFileName":"/no/such/scan.xyz"}`)); err == nil {
		t.Error("attach of a missing file should fail")
	}
	path := writeScan(t, "0 0 0\n")
	call(t, r, s, "pointClouds.attach", mustJSON(t, wire.AttachPointCloudArgs{Name: "S", FullFileName: path}), &wire.PointCloudInfo{})
	if _, err := r.Handle(s, "pointClouds.setScale", []byte(`{"name":"S","scale":0}`)); err == nil {
		t.Error("setScale(0) should fail")
	}
}

// TestPointCloudFitPlane: fitPlane over the wire fits a work plane to a planar (z = 5) scan and
// reports the new plane's name, origin (centroid), and unit normal (#645).
func TestPointCloudFitPlane(t *testing.T) {
	r, s := emptyPartSession(t)
	path := writeScan(t, "0 0 5\n2 0 5\n0 2 5\n2 2 5\n1 3 5\n-1 1 5\n")
	var info wire.PointCloudInfo
	call(t, r, s, "pointClouds.attach", mustJSON(t, wire.AttachPointCloudArgs{Name: "Scan", FullFileName: path}), &info)

	var res wire.FitPointCloudPlaneResult
	call(t, r, s, "pointClouds.fitPlane", `{"cloud":"Scan"}`, &res)
	if res.WorkPlane == "" {
		t.Fatal("fitPlane should return a work plane name")
	}
	if stdmath.Abs(res.Origin.Z-5) > 1e-9 {
		t.Errorf("origin Z = %v, want 5", res.Origin.Z)
	}
	if stdmath.Abs(stdmath.Abs(res.Normal.Z)-1) > 1e-6 {
		t.Errorf("normal = %+v, want ±Z", res.Normal)
	}
	if _, err := r.Handle(s, "pointClouds.fitPlane", []byte(`{"cloud":"nope"}`)); err == nil {
		t.Error("fitPlane on an unknown cloud should error")
	}
}

// TestPointCloudNearestPoint: nearestPoint snaps a query onto the cloud's closest scan point and
// reports the distance; an unknown cloud errors (#645).
func TestPointCloudNearestPoint(t *testing.T) {
	r, s := emptyPartSession(t)
	path := writeScan(t, "0 0 0\n2 0 0\n0 2 0\n")
	call(t, r, s, "pointClouds.attach", mustJSON(t, wire.AttachPointCloudArgs{Name: "Scan", FullFileName: path}), &wire.PointCloudInfo{})

	var res wire.NearestPointResult
	call(t, r, s, "pointClouds.nearestPoint", mustJSON(t, wire.NearestPointArgs{Cloud: "Scan", Point: types.Point{X: 1.9, Y: 0.1, Z: 0}}), &res)
	if !res.Found || res.Point != (types.Point{X: 2, Y: 0, Z: 0}) {
		t.Errorf("nearest = %+v (found=%v), want (2,0,0)", res.Point, res.Found)
	}
	if stdmath.Abs(res.Distance-stdmath.Hypot(0.1, 0.1)) > 1e-9 {
		t.Errorf("distance = %v, want hypot(0.1,0.1)", res.Distance)
	}
	if _, err := r.Handle(s, "pointClouds.nearestPoint", []byte(`{"cloud":"nope","point":{"x":0,"y":0,"z":0}}`)); err == nil {
		t.Error("nearestPoint on an unknown cloud should error")
	}
}
