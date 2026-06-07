// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"path/filepath"
	"testing"

	"oblikovati/api/types"
	"oblikovati/math"
	"oblikovati/model/doc"
	"oblikovati/persistence/yamlcodec"
)

// TestViewsRoundTripThroughPackage checks the views section survives a save → open through
// the .obk file (cameras, active index, and layout).
func TestViewsRoundTripThroughPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.obk")
	p := NewPackage()
	if err := p.SetManifest(Manifest{SchemaVersion: CurrentSchemaVersion, DocumentType: 1, DisplayName: "Part1"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	p.SetViews(&yamlcodec.ViewsSection{
		Views: []yamlcodec.ViewFrame{
			{Name: "Front", Eye: [3]float64{0, 0, 10}, Target: [3]float64{0, 0, 0}, Up: [3]float64{0, 1, 0}, FOV: 0.78},
			{Name: "Iso", Eye: [3]float64{5, 6, 7}, Target: [3]float64{1, 2, 3}, Up: [3]float64{0, 1, 0}, FOV: 0.6},
		},
		Active: 1, Layout: int32(types.LayoutFour),
	})
	if err := p.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := OpenPackage(path)
	if err != nil {
		t.Fatalf("OpenPackage: %v", err)
	}
	vs := got.Views()
	if vs == nil || len(vs.Views) != 2 || vs.Active != 1 || vs.Layout != int32(types.LayoutFour) {
		t.Fatalf("views section = %+v, want 2 views active=1 layout=four", vs)
	}
	if vs.Views[1].Name != "Iso" || vs.Views[1].Eye != [3]float64{5, 6, 7} {
		t.Errorf("view[1] = %+v, want Iso eye[5 6 7]", vs.Views[1])
	}
}

// TestV2DocumentMigratesToV3WithoutViews checks a pre-views (v2) file opens, migrates to
// the current version, and simply carries no views section.
func TestV2DocumentMigratesToV3WithoutViews(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.obk")
	p := NewPackage()
	if err := p.SetManifest(Manifest{SchemaVersion: 2, DocumentType: 1, DisplayName: "Old"}); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	if err := p.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := OpenPackage(path) // OpenPackage runs the migration pipeline
	if err != nil {
		t.Fatalf("OpenPackage: %v", err)
	}
	m, _ := got.Manifest()
	if m.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("schema = v%d, want migrated to v%d", m.SchemaVersion, CurrentSchemaVersion)
	}
	if got.Views() != nil {
		t.Errorf("a v2 document should carry no views section, got %+v", got.Views())
	}
}

// TestStoreViewsHelpersRoundTrip checks the store's document↔section projection: a
// document's views project to the section and restore into another document (framed).
func TestStoreViewsHelpersRoundTrip(t *testing.T) {
	src, err := doc.Restore(doc.Part, "a.obk", "A")
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	src.Views().Active().Name = "Front"
	src.Views().Add(doc.DefaultView("Iso"))
	src.Views().Active().Eye = math.P3(9, 9, 9)
	src.Views().SetLayout(types.LayoutTwoH)

	dst, err := doc.Restore(doc.Part, "b.obk", "B")
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restoreViews(dst, viewsSection(src))

	if dst.Views().Count() != 2 || dst.Views().Layout() != types.LayoutTwoH {
		t.Fatalf("restored views=%d layout=%v, want 2 / twoH", dst.Views().Count(), dst.Views().Layout())
	}
	got := dst.Views().All()[1]
	if got.Eye != math.P3(9, 9, 9) || !got.Framed {
		t.Errorf("restored view[1] = %+v, want eye(9,9,9) framed", got)
	}
}
