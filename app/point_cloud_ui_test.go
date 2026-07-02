// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
)

// findChild returns the first child node with the given label, or nil.
func findChild(n BrowserNode, label string) *BrowserNode {
	for i := range n.Children {
		if n.Children[i].Label == label {
			return &n.Children[i]
		}
	}
	return nil
}

// TestBrowserListsPointClouds: an attached cloud appears under a Point Clouds folder, a hidden one
// tagged, and the node carries a selectable handle (#645).
func TestBrowserListsPointClouds(t *testing.T) {
	s, def := emptyPartSession(t)
	if findChild(BuildBrowser(s), "Point Clouds") != nil {
		t.Fatal("a part with no clouds should have no Point Clouds folder")
	}
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, _ := def.PointClouds().Add("RoomScan", "r.xyz", rid, []math.Point3{math.P3(0, 0, 0)})

	folder := findChild(BuildBrowser(s), "Point Clouds")
	if folder == nil || len(folder.Children) != 1 || folder.Children[0].Label != "RoomScan" {
		t.Fatalf("Point Clouds folder = %+v, want one RoomScan child", folder)
	}
	if _, ok := folder.Children[0].Select.(PointCloudHandle); !ok {
		t.Error("cloud node should carry a PointCloudHandle")
	}

	pc.SetVisible(false)
	folder = findChild(BuildBrowser(s), "Point Clouds")
	if folder.Children[0].Label != "RoomScan  (hidden)" {
		t.Errorf("hidden cloud label = %q, want the (hidden) tag", folder.Children[0].Label)
	}
}

// TestPointCloudMenuVisibilityAndDelete: the right-click menu toggles visibility and deletes the
// cloud (#645).
func TestPointCloudMenuVisibilityAndDelete(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, _ := def.PointClouds().Add("C", "c.xyz", rid, nil)
	node := BrowserNode{Kind: "pointCloud", Select: PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc}}

	menu := BrowserMenu(s, node)
	if len(menu) != 2 || menu[0].Label != "Visibility" || menu[1].Label != "Delete" {
		t.Fatalf("menu = %+v, want Visibility + Delete", menu)
	}
	_ = menu[0].Invoke(s)
	if pc.Visible() {
		t.Error("Visibility should have hidden the cloud")
	}
	_ = menu[1].Invoke(s)
	if def.PointClouds().Count() != 0 {
		t.Errorf("Delete left %d clouds, want 0", def.PointClouds().Count())
	}
}

// TestAttachPointCloudAndRequest: AttachPointCloud reads a scan into the active part, and the
// import-request flag is one-shot (#645).
func TestAttachPointCloudAndRequest(t *testing.T) {
	s, def := emptyPartSession(t)
	path := filepath.Join(t.TempDir(), "scan.xyz")
	if err := os.WriteFile(path, []byte("0 0 0\n1 1 1\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	pc, err := s.AttachPointCloud("Scan", path)
	if err != nil || pc.TotalPointCount() != 2 {
		t.Fatalf("AttachPointCloud = %v pts, err %v", pc, err)
	}
	if def.PointClouds().Count() != 1 {
		t.Errorf("active part has %d clouds, want 1", def.PointClouds().Count())
	}
	if _, err := s.AttachPointCloud("X", "/no/such/file.xyz"); err == nil {
		t.Error("AttachPointCloud of a missing file should fail")
	}

	if s.TakeImportPointCloudRequest() {
		t.Error("no request pending initially")
	}
	s.RequestImportPointCloud()
	if !s.TakeImportPointCloudRequest() || s.TakeImportPointCloudRequest() {
		t.Error("import request should be true once, then cleared")
	}
}

// TestBrowserNestsCropsWithMenu: a cloud's crops appear as child nodes (inactive ones tagged),
// and the crop right-click menu toggles active and deletes (#645).
func TestBrowserNestsCropsWithMenu(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, _ := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(0, 0, 0)})
	crop := pc.AddCrop(math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1)))

	cloudNode := findChild(*findChild(BuildBrowser(s), "Point Clouds"), "Scan")
	if cloudNode == nil || len(cloudNode.Children) != 1 || cloudNode.Children[0].Label != "Crop1" {
		t.Fatalf("cloud node children = %+v, want one Crop1", cloudNode)
	}

	menu := BrowserMenu(s, cloudNode.Children[0])
	if len(menu) != 2 || menu[0].Label != "Deactivate" {
		t.Fatalf("crop menu = %+v, want Deactivate + Delete", menu)
	}
	_ = menu[0].Invoke(s)
	if crop.Active() {
		t.Error("Deactivate should have turned the crop off")
	}
	// Now the menu's toggle reads Activate, and the node is tagged inactive.
	cloudNode = findChild(*findChild(BuildBrowser(s), "Point Clouds"), "Scan")
	if cloudNode.Children[0].Label != "Crop1  (inactive)" {
		t.Errorf("inactive crop label = %q, want the (inactive) tag", cloudNode.Children[0].Label)
	}
	if BrowserMenu(s, cloudNode.Children[0])[0].Label != "Activate" {
		t.Error("an inactive crop's menu should offer Activate")
	}
	_ = menu[1].Invoke(s)
	if pc.Crops().Count() != 0 {
		t.Errorf("Delete left %d crops, want 0", pc.Crops().Count())
	}
}

// TestPointCloudCropMenuWrongHandle: the crop menu rejects a non-crop handle (#645).
func TestPointCloudCropMenuWrongHandle(t *testing.T) {
	if pointCloudCropMenu(BodyHandle{}) != nil {
		t.Error("pointCloudCropMenu with a non-crop handle should be nil")
	}
	if k := (PointCloudCropHandle{}).SelectionKind(); k != SelectPointCloud {
		t.Errorf("crop handle SelectionKind = %v, want SelectPointCloud", k)
	}
}

// TestPointCloudHandleKind: the browser handle reports the point-cloud selection kind (#645).
func TestPointCloudHandleKind(t *testing.T) {
	if k := (PointCloudHandle{}).SelectionKind(); k != SelectPointCloud {
		t.Errorf("SelectionKind = %v, want SelectPointCloud", k)
	}
}

// TestPointCloudEdgeCasesWithoutPart: with no active part the render and attach paths return empty
// / error rather than panicking, and the menu rejects a wrong handle (#645).
func TestPointCloudEdgeCasesWithoutPart(t *testing.T) {
	s := NewSession() // no active document
	if len(s.PointCloudItems(s.Camera(), 0.5)) != 0 {
		t.Error("no active part should yield no point-cloud items")
	}
	if _, err := s.AttachPointCloud("X", "/tmp/whatever.xyz"); err == nil {
		t.Error("AttachPointCloud with no active part should fail")
	}
	if pointCloudMenu(BodyHandle{}) != nil {
		t.Error("pointCloudMenu with a non-cloud handle should be nil")
	}
}

// TestImportPointCloudCommandArmsDialog: executing the Import Point Cloud command arms the file
// dialog request (#645).
func TestImportPointCloudCommandArmsDialog(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := s.Execute("PointCloud.Import"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !s.TakeImportPointCloudRequest() {
		t.Error("Import Point Cloud command should arm the import request")
	}
}

// TestAttachPointCloudEmptyName mints a unique name when none is given (#645).
func TestAttachPointCloudEmptyName(t *testing.T) {
	s, _ := emptyPartSession(t)
	path := filepath.Join(t.TempDir(), "s.xyz")
	_ = os.WriteFile(path, []byte("0 0 0\n"), 0o600)
	pc, err := s.AttachPointCloud("", path)
	if err != nil || pc.Name() != "Cloud1" {
		t.Errorf("AttachPointCloud(empty name) = %v, err %v; want Cloud1", pc, err)
	}

	if _, err := s.AttachPointCloud("X", ""); err == nil {
		t.Error("AttachPointCloud with an empty file name should fail")
	}
	bad := filepath.Join(t.TempDir(), "scan.las") // exists but no reader for .las
	_ = os.WriteFile(bad, []byte("binary"), 0o600)
	if _, err := s.AttachPointCloud("Y", bad); err == nil {
		t.Error("AttachPointCloud of an unsupported scan format should fail")
	}
}
