// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/pointcloud"
)

// Point-cloud dispatch integration tests (M40 audit S12, #1646): scan formats route through the
// same model/exchange vertical as every other import — FormatFromPath recognises them, the
// body dispatch redirects them with guidance, and record-level defects warn-and-continue like
// the DWG decoder instead of sinking the whole file.

// TestFormatFromPathRoutesPointCloudFormats: the scanner formats resolve to their api/types
// constants (case-insensitively), classified as point-cloud imports.
func TestFormatFromPathRoutesPointCloudFormats(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]types.ExchangeFormat{
		"scan.ply": types.FormatPLY, "survey.E57": types.FormatE57, "lidar.las": types.FormatLAS,
	} {
		got, ok := FormatFromPath(path)
		if !ok || got != want {
			t.Errorf("FormatFromPath(%q) = (%q, %v), want (%q, true)", path, got, ok, want)
		}
		if !got.IsPointCloud() {
			t.Errorf("FormatFromPath(%q) = %q, which is not IsPointCloud", path, got)
		}
	}
}

// asciiScanExtensions are the registered scan extensions that CANNOT yet appear in
// FormatFromPath: the ASCII XYZ/PTS family has no types.ExchangeFormat constant in api/types
// (an Apache-2.0 sibling-repo addition, #1646); .txt additionally is far too generic to claim
// in a format-from-extension dispatch. Keep in sync with the api/types exchange formats.
var asciiScanExtensions = map[string]bool{".xyz": true, ".pts": true, ".asc": true, ".txt": true}

// TestEveryScanExtensionIsRouted is registry-driven (#1646): every extension a registered
// PointReader handles must either resolve through FormatFromPath as a point-cloud format or be
// on the documented ASCII allow-list above — so a new reader cannot ship unrouted.
func TestEveryScanExtensionIsRouted(t *testing.T) {
	t.Parallel()
	for _, ext := range pointcloud.ScanExtensions() {
		if asciiScanExtensions[ext] {
			continue
		}
		format, ok := FormatFromPath("scan" + ext)
		if !ok {
			t.Errorf("registered scan extension %q is not routed by FormatFromPath; add it or extend the documented allow-list", ext)
			continue
		}
		if !format.IsPointCloud() {
			t.Errorf("FormatFromPath(%q) = %q, want a point-cloud format", ext, format)
		}
	}
}

// TestImportRejectsPointCloudFormatWithGuidance: the body-import dispatch names the point-cloud
// attach path instead of failing obscurely, mirroring the sketch-format guard.
func TestImportRejectsPointCloudFormatWithGuidance(t *testing.T) {
	t.Parallel()
	part := compdef.NewPartComponentDefinition()
	_, err := Import(part, "survey.las", types.FormatLAS)
	if err == nil || !strings.Contains(err.Error(), "point cloud") {
		t.Fatalf("Import(las) error = %v, want a point-cloud-attach redirect", err)
	}
}

// TestImportPointCloudWarnsAndContinues: one malformed record produces one warning naming the
// line and value, the good points still import, and the scan bytes are embedded (#1646).
func TestImportPointCloudWarnsAndContinues(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "scan.xyz")
	if err := os.WriteFile(path, []byte("0 0 0\n10 zero 0\n10 0 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	part := compdef.NewPartComponentDefinition()
	pc, res, err := ImportPointCloud(part, "Scan", path)
	if err != nil {
		t.Fatalf("ImportPointCloud: %v", err)
	}
	if pc.TotalPointCount() != 2 || res.PointCount != 2 {
		t.Errorf("imported %d/%d points, want 2 (malformed record skipped)", pc.TotalPointCount(), res.PointCount)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "line 2") || !strings.Contains(res.Warnings[0], "10 zero 0") {
		t.Errorf("Warnings = %v, want one naming line 2 and its content", res.Warnings)
	}
	if pc.ResourceID() == "" {
		t.Error("scan bytes should be embedded as a document resource (ADR-0031)")
	}
}

// TestImportPointCloudFailsWhenNothingDecodes: a scan with zero decodable records is an error
// naming the offending input, not an empty cloud.
func TestImportPointCloudFailsWhenNothingDecodes(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "scan.xyz")
	if err := os.WriteFile(path, []byte("not a point\nalso bad\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	part := compdef.NewPartComponentDefinition()
	if _, _, err := ImportPointCloud(part, "Scan", path); err == nil || !strings.Contains(err.Error(), "not a point") {
		t.Fatalf("ImportPointCloud of an undecodable scan = %v, want an error naming the bad record", err)
	}
}

// TestPLYAlwaysRoutesToPointCloud pins the documented .ply rule (#1646): PLY is a point-cloud
// format by contract (api/types FormatPLY), never a mesh — the same answer from every entry
// point (FormatFromPath, the scan-file predicate, and the mesh translator's rejection).
func TestPLYAlwaysRoutesToPointCloud(t *testing.T) {
	t.Parallel()
	format, ok := FormatFromPath("scan.ply")
	if !ok || !format.IsPointCloud() || format.IsMesh() {
		t.Errorf("FormatFromPath(.ply) = (%q, %v), want the point-cloud PLY format", format, ok)
	}
	if !pointcloud.IsScanFile("scan.ply") {
		t.Error("IsScanFile(.ply) = false, want true (PLY is scan data)")
	}
	if (MeshExchange{}).CanImport(types.FormatPLY) {
		t.Error("MeshExchange.CanImport(ply) = true, want false — .ply never imports as a mesh")
	}
}
