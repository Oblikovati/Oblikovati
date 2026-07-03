// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"testing"

	"oblikovati.org/kernel/exchange"

	"oblikovati.org/math"
)

// TestCollectionAddListByNameRemove: the collection adds uniquely-named clouds, enumerates them,
// resolves by name, and removes them.
func TestCollectionAddListByNameRemove(t *testing.T) {
	c := NewPointClouds()
	if c.Count() != 0 {
		t.Fatalf("fresh collection Count = %d, want 0", c.Count())
	}

	a, err := c.Add("A", "a.xyz", "r1", []math.Point3{math.P3(0, 0, 0)})
	if err != nil {
		t.Fatalf("Add A: %v", err)
	}
	if _, err := c.Add("B", "b.xyz", "r2", nil); err != nil {
		t.Fatalf("Add B: %v", err)
	}
	if c.Count() != 2 || c.Item(0) != a {
		t.Errorf("Count = %d, Item(0) mismatch", c.Count())
	}
	if names := c.Names(); len(names) != 2 || names[0] != "A" || names[1] != "B" {
		t.Errorf("Names = %v, want [A B]", names)
	}
	if got, ok := c.ByName("B"); !ok || got.Name() != "B" {
		t.Errorf("ByName(B) = (%v, %v)", got, ok)
	}
	if _, ok := c.ByName("missing"); ok {
		t.Error("ByName(missing) should be false")
	}
	if !c.Remove("A") || c.Remove("A") {
		t.Error("Remove(A) should be true once, then false")
	}
	if c.Count() != 1 {
		t.Errorf("after remove Count = %d, want 1", c.Count())
	}
}

// TestCollectionRejectsBadNames: an empty name and a duplicate name are rejected.
func TestCollectionRejectsBadNames(t *testing.T) {
	c := NewPointClouds()
	if _, err := c.Add("", "", "", nil); err == nil {
		t.Error("Add with an empty name should fail")
	}
	if _, err := c.Add("Dup", "", "", nil); err != nil {
		t.Fatalf("Add Dup: %v", err)
	}
	if _, err := c.Add("Dup", "", "", nil); err == nil {
		t.Error("Add with a duplicate name should fail")
	}
}

// TestCollectionAppendAndUniqueName: Append re-attaches a built cloud (rejecting nil/duplicates),
// and UniqueName avoids existing names.
func TestCollectionAppendAndUniqueName(t *testing.T) {
	c := NewPointClouds()
	if err := c.Append(nil); err == nil {
		t.Error("Append(nil) should fail")
	}
	if err := c.Append(New("Cloud1", "", "", nil)); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := c.Append(New("Cloud1", "", "", nil)); err == nil {
		t.Error("Append of a duplicate name should fail")
	}
	if got := c.UniqueName("Cloud"); got != "Cloud2" {
		t.Errorf("UniqueName(Cloud) = %q, want Cloud2 (Cloud1 taken)", got)
	}
}

// TestCloudScalarAccessors: the simple metadata accessors round-trip, and the degenerate-placement
// and budget edge cases behave.
func TestCloudScalarAccessors(t *testing.T) {
	pc := New("Scan", "s.xyz", "rid", []math.Point3{math.P3(1, 2, 3), math.P3(4, 5, 6)})
	pc.SetName("Renamed")
	pc.SetVisible(false)
	if pc.Name() != "Renamed" || pc.Visible() {
		t.Errorf("name/visible = %q/%v, want Renamed/false", pc.Name(), pc.Visible())
	}
	if len(pc.CloudPoints()) != 2 || pc.CloudRangeBox().IsEmpty() {
		t.Error("CloudPoints / CloudRangeBox not reported")
	}
	pc.SetMaximumPointCount(-5) // clamps to 0 (unbounded)
	if pc.MaximumPointCount() != 0 || pc.DisplayedPointCount() != 2 {
		t.Errorf("negative budget: max %d displayed %d, want 0/2", pc.MaximumPointCount(), pc.DisplayedPointCount())
	}
}

// TestFromModelSpaceDegenerate: a non-invertible placement (zero scale) reports ok=false.
func TestFromModelSpaceDegenerate(t *testing.T) {
	pc := New("s", "", "", nil)
	pc.scale = 0 // force a degenerate placement (SetScale would reject this)
	if _, ok := pc.FromModelSpace(math.P3(1, 1, 1)); ok {
		t.Error("FromModelSpace with zero scale should report ok=false")
	}
}

// TestReadScanDispatch: ReadScan routes by extension and errors on an unknown one.
func TestReadScanDispatch(t *testing.T) {
	pts, _, err := ReadScan("room.PTS", []byte("1 2 3\n"), exchange.TranslationOptions{})
	if err != nil || len(pts) != 1 {
		t.Fatalf("ReadScan(.PTS) = %v points, err %v", len(pts), err)
	}
	if _, _, err := ReadScan("room.bin", []byte("binary"), exchange.TranslationOptions{}); err == nil {
		t.Error("ReadScan of an unregistered extension should fail")
	}
}

// TestIsScanFileAndExtensions: scan extensions are recognized (any case) and non-scan ones are not
// (#645).
func TestIsScanFileAndExtensions(t *testing.T) {
	for _, p := range []string{"room.ply", "scan.XYZ", "a.pts", "b.asc", "c.txt", "part.E57", "survey.las"} {
		if !IsScanFile(p) {
			t.Errorf("IsScanFile(%q) = false, want true", p)
		}
	}
	for _, p := range []string{"part.stl", "model.step", "draw.dxf", "noext"} {
		if IsScanFile(p) {
			t.Errorf("IsScanFile(%q) = true, want false", p)
		}
	}
	if exts := ScanExtensions(); len(exts) < 5 {
		t.Errorf("ScanExtensions = %v, want at least the ascii + ply set", exts)
	}
}
