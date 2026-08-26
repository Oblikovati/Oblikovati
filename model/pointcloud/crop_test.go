// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"testing"

	"oblikovati.org/math"
)

// gridCloud returns a cloud of points at integer x in [0,9], y=z=0.
func gridCloud() *PointCloud {
	var pts []math.Point3
	for i := range 10 {
		pts = append(pts, math.P3(math.Scalar(i), 0, 0))
	}
	return New("scan", "", "", pts)
}

// TestCropLimitsDisplayedPoints: an active crop limits the displayed (model-space) points to those
// inside its box; toggling it off restores the full set.
func TestCropLimitsDisplayedPoints(t *testing.T) {
	pc := gridCloud()
	if pc.DisplayedPointCount() != 10 {
		t.Fatalf("uncropped displayed count = %d, want 10", pc.DisplayedPointCount())
	}

	crop := pc.AddCrop(math.NewBox(math.P3(-0.5, -1, -1), math.P3(3.5, 1, 1))) // x in [0,3]
	if pc.DisplayedPointCount() != 4 {
		t.Errorf("cropped displayed count = %d, want 4 (x 0..3)", pc.DisplayedPointCount())
	}
	for _, p := range pc.DisplayedPoints() {
		if p.X < 0 || p.X > 3 {
			t.Errorf("displayed point %v outside the crop", p)
		}
	}

	crop.SetActive(false)
	if pc.DisplayedPointCount() != 10 {
		t.Errorf("after deactivating the crop, displayed = %d, want 10", pc.DisplayedPointCount())
	}
}

// TestCropsUnion: two active crops admit the union of their boxes.
func TestCropsUnion(t *testing.T) {
	pc := gridCloud()
	pc.AddCrop(math.NewBox(math.P3(-0.5, -1, -1), math.P3(1.5, 1, 1))) // x 0,1
	pc.AddCrop(math.NewBox(math.P3(7.5, -1, -1), math.P3(9.5, 1, 1)))  // x 8,9
	if pc.DisplayedPointCount() != 4 {
		t.Errorf("two-crop union displayed = %d, want 4 (x 0,1,8,9)", pc.DisplayedPointCount())
	}
}

// TestCropBudgetOrder: the display budget applies after cropping, so the cap counts cropped points.
func TestCropBudgetOrder(t *testing.T) {
	pc := gridCloud()
	pc.AddCrop(math.NewBox(math.P3(-0.5, -1, -1), math.P3(5.5, 1, 1))) // x 0..5 → 6 points
	pc.SetMaximumPointCount(3)
	if pc.DisplayedPointCount() != 3 || len(pc.DisplayedPoints()) != 3 {
		t.Errorf("budgeted cropped count = %d / %d, want 3", pc.DisplayedPointCount(), len(pc.DisplayedPoints()))
	}
}

// TestCropAccessors: the crop's box/active accessors and setters round-trip, and Item resolves it.
func TestCropAccessors(t *testing.T) {
	pc := gridCloud()
	box := math.NewBox(math.P3(1, 2, 3), math.P3(4, 5, 6))
	c := pc.AddCrop(box)
	if c.Box() != box || !c.Active() {
		t.Fatalf("new crop = box %+v active %v, want %+v / true", c.Box(), c.Active(), box)
	}
	c.SetBox(math.EmptyBox())
	c.SetActive(false)
	if c.Active() || !c.Box().IsEmpty() {
		t.Error("SetBox/SetActive did not take")
	}
	if pc.Crops().Item(0) != c {
		t.Error("Item(0) should return the added crop")
	}
}

// TestCropCollection: add/byName/remove/names + unique-name minting and bad-name rejection.
func TestCropCollection(t *testing.T) {
	pc := gridCloud()
	c1 := pc.AddCrop(math.EmptyBox())
	c2 := pc.AddCrop(math.EmptyBox())
	if c1.Name() != "Crop1" || c2.Name() != "Crop2" {
		t.Errorf("crop names = %q/%q, want Crop1/Crop2", c1.Name(), c2.Name())
	}
	crops := pc.Crops()
	if crops.Count() != 2 || len(crops.Names()) != 2 {
		t.Errorf("crops Count/Names mismatch: %d", crops.Count())
	}
	if got, ok := crops.ByName("Crop2"); !ok || got != c2 {
		t.Error("ByName(Crop2) failed")
	}
	if _, ok := crops.Add("", math.EmptyBox()); ok {
		t.Error("Add with empty name should fail")
	}
	if _, ok := crops.Add("Crop1", math.EmptyBox()); ok {
		t.Error("Add with duplicate name should fail")
	}
	if !crops.Remove("Crop1") || crops.Remove("Crop1") {
		t.Error("Remove(Crop1) should be true once then false")
	}
}
