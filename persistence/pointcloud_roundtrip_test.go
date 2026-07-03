// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

// TestPointCloudSurvivesStoreRoundTrip: an attached scan's metadata, placement, and points
// round-trip through a save/open cycle — the points re-decoded from the embedded resource bytes,
// not re-read from the original file (M17-F06, #645).
func TestPointCloudSurvivesStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scan.obk")
	store := NewPackageStore()

	ws := doc.NewWorkspace(store, contentset.Default())
	d, err := compdef.AddPart(ws, path, true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := d.Content().(*compdef.PartComponentDefinition)

	scan := []byte("0 0 0\n1 2 3\n4 5 6\n")
	rid := def.AddResource(doc.Resource{Type: "PointCloudScan", Encoding: doc.EncodingUTF8, Value: scan, Origin: "room.xyz"})
	points, _, err := pointcloud.ReadScan("room.xyz", scan, exchange.TranslationOptions{TargetUnitMM: def.WorkingUnitMM()})
	if err != nil {
		t.Fatalf("decode scan: %v", err)
	}
	pc, err := def.PointClouds().Add("Cloud1", "room.xyz", rid, points)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	pc.SetVisible(false)
	pc.SetScale(2.5)
	pc.SetMaximumPointCount(2)
	pc.SetTransform(translation4(10, 0, 0))
	crop := pc.AddCrop(math.NewBox(math.P3(0, 0, 0), math.P3(5, 5, 5)))
	crop.SetActive(false)

	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reopened, err := doc.NewWorkspace(store, contentset.Default()).Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	rdef := reopened.Content().(*compdef.PartComponentDefinition)
	got, ok := rdef.PointClouds().ByName("Cloud1")
	if !ok {
		t.Fatalf("reopened part has no Cloud1; names=%v", rdef.PointClouds().Names())
	}
	if got.Visible() || got.Scale() != 2.5 || got.MaximumPointCount() != 2 {
		t.Errorf("reopened cloud = visible %v scale %v max %d, want false/2.5/2", got.Visible(), got.Scale(), got.MaximumPointCount())
	}
	if got.TotalPointCount() != 3 {
		t.Errorf("reopened total points = %d, want 3 (re-decoded from the embedded resource)", got.TotalPointCount())
	}
	if got.Transform() != translation4(10, 0, 0) {
		t.Errorf("reopened transform = %+v, want a +10x translation", got.Transform())
	}
	if got.SourceFullFileName() != "room.xyz" || got.ResourceID() != rid {
		t.Errorf("reopened source/resource = %q/%q, want room.xyz/%q", got.SourceFullFileName(), got.ResourceID(), rid)
	}
	rc, ok := got.Crops().ByName("Crop1")
	if !ok || rc.Active() || rc.Box().Max != math.P3(5, 5, 5) {
		t.Errorf("reopened crop = %+v (ok=%v), want an inactive Crop1 to (5,5,5)", rc, ok)
	}
}

// translation4 builds a pure-translation 4×4 matrix (cells[3,7,11] hold the translation column).
func translation4(x, y, z math.Scalar) math.Matrix4 {
	cells := math.Identity4().Cells()
	cells[3], cells[7], cells[11] = x, y, z
	return math.Matrix4FromCells(cells)
}
