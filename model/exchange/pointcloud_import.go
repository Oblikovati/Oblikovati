// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

// Point-cloud import vertical (M40 audit S12, #1646): scan files flow through the same
// model/exchange seam as every other format — file units scale into the document working unit
// (#1636), record-level defects warn-and-continue like the DWG decoder, and the scan bytes are
// embedded in the document resource table (ADR-0031) so the .obk stays self-contained. The
// session attach path (app.AttachPointCloud) and the pointClouds.attach wire method both land
// here, so cross-cutting exchange improvements reach clouds by construction.

// PointCloudImportResult reports what a scan import produced: the decoded point count and any
// non-fatal per-record warnings (the same slot the drawing importers fill).
type PointCloudImportResult struct {
	PointCount int
	Warnings   []string
}

// ImportPointCloud reads the scan file at path, embeds its bytes as a document resource, decodes
// its points into the part's working unit, and attaches the cloud under name (a unique name is
// minted when name is empty). Malformed records are skipped with a warning naming the record;
// only a scan with zero decodable points errors.
//
// Example:
//
//	pc, res, err := exchange.ImportPointCloud(part, "", "survey.las")
func ImportPointCloud(part *compdef.PartComponentDefinition, name, path string) (*pointcloud.PointCloud, PointCloudImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, PointCloudImportResult{}, fmt.Errorf("import scan: read %q: %w", path, err)
	}
	points, warns, err := pointcloud.ReadScan(path, data, exchange.TranslationOptions{TargetUnitMM: workingUnitMM(part)})
	if err != nil {
		return nil, PointCloudImportResult{}, err
	}
	if name == "" {
		name = part.PointClouds().UniqueName("Cloud")
	}
	rid := part.AddResource(doc.Resource{Type: "PointCloudScan", Encoding: doc.EncodingUTF8, Value: data, Origin: path})
	pc, err := part.PointClouds().Add(name, path, rid, points)
	if err != nil {
		return nil, PointCloudImportResult{}, err
	}
	return pc, PointCloudImportResult{PointCount: len(points), Warnings: warns}, nil
}
