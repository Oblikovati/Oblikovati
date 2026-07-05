// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

// Point-cloud persistence (M17-F06, #645): the part implements doc.PointCloudBearer so the store
// round-trips attached scans. Records carry only metadata + the resource id of the scan bytes;
// the points are re-decoded from the already-restored resource table on open (the .obk embeds the
// scan once, in resources). See model/doc/pointcloud_records.go.

// var assertion: the part is a point-cloud bearer.
var _ doc.PointCloudBearer = (*PartComponentDefinition)(nil)

// PointCloudRecords serialises the part's attached clouds (save path).
func (d *PartComponentDefinition) PointCloudRecords() []doc.PointCloudRecord {
	clouds := d.pointClouds
	out := make([]doc.PointCloudRecord, 0, clouds.Count())
	for i := 0; i < clouds.Count(); i++ {
		out = append(out, recordOfCloud(clouds.Item(i)))
	}
	return out
}

// SetPointCloudRecords rebuilds the clouds from their records (load path), re-decoding each
// cloud's points from the resource table (already restored before this runs). A record whose
// resource or scan is unreadable still rebuilds the cloud — with no points — so its metadata and
// placement are not lost.
func (d *PartComponentDefinition) SetPointCloudRecords(records []doc.PointCloudRecord) {
	d.pointClouds = pointcloud.NewPointClouds()
	for _, rec := range records {
		_ = d.pointClouds.Append(cloudFromRecord(rec, d.scanPoints(rec)))
	}
}

// scanPoints re-decodes a record's points from the embedded resource bytes, or nil when the
// resource is missing or undecodable. The scan re-scales into the document's CURRENT working
// unit (the same options the attach path used), so restored clouds match the attach scale (#1636).
func (d *PartComponentDefinition) scanPoints(rec doc.PointCloudRecord) []pointcloud.PointSample {
	res, ok := d.resources[rec.ResourceID]
	if !ok {
		return nil
	}
	// Restore is non-interactive, so per-record warnings are not surfaced here; the attach path
	// already reported them (#1646).
	samples, _, err := pointcloud.ReadScanSamples(rec.Source, res.Value, exchange.TranslationOptions{TargetUnitMM: d.WorkingUnitMM()})
	if err != nil {
		return nil
	}
	return samples
}

// recordOfCloud captures a cloud's metadata + placement + crops (its points stay in the resource
// table).
func recordOfCloud(pc *pointcloud.PointCloud) doc.PointCloudRecord {
	return doc.PointCloudRecord{
		Name:        pc.Name(),
		Source:      pc.SourceFullFileName(),
		ResourceID:  pc.ResourceID(),
		Visible:     pc.Visible(),
		DisplayMode: pc.DisplayMode().String(),
		Scale:       pc.Scale(),
		Transform:   cellsToFloats(pc.Transform().Cells()),
		MaxPoints:   pc.MaximumPointCount(),
		Crops:       cropRecords(pc.Crops()),
	}
}

// cloudFromRecord rebuilds a cloud from its record and freshly decoded points, restoring its crops.
func cloudFromRecord(rec doc.PointCloudRecord, samples []pointcloud.PointSample) *pointcloud.PointCloud {
	pc := pointcloud.NewWithSamples(rec.Name, rec.Source, rec.ResourceID, samples)
	if mode := strings.ToLower(strings.TrimSpace(rec.DisplayMode)); mode != "" {
		_ = pc.SetDisplayMode(types.PointCloudDisplayMode(mode))
	}
	pc.SetVisible(rec.Visible)
	pc.SetScale(rec.Scale) // rejected for a corrupt non-positive value, leaving the default 1
	pc.SetTransform(math.Matrix4FromCells(floatsToCells(rec.Transform)))
	pc.SetMaximumPointCount(rec.MaxPoints)
	restoreCrops(pc, rec.Crops)
	return pc
}

// cropRecords serialises a cloud's crop volumes.
func cropRecords(crops *pointcloud.PointCloudCrops) []doc.PointCloudCropRecord {
	out := make([]doc.PointCloudCropRecord, 0, crops.Count())
	for i := 0; i < crops.Count(); i++ {
		c := crops.Item(i)
		out = append(out, doc.PointCloudCropRecord{
			Name: c.Name(), Active: c.Active(),
			Min: pointToFloats(c.Box().Min), Max: pointToFloats(c.Box().Max),
		})
	}
	return out
}

// restoreCrops rebuilds a cloud's crops from their records (each named crop with its box + active
// state).
func restoreCrops(pc *pointcloud.PointCloud, records []doc.PointCloudCropRecord) {
	for _, r := range records {
		box := math.NewBox(floatsToPoint(r.Min), floatsToPoint(r.Max))
		if c, ok := pc.Crops().Add(r.Name, box); ok {
			c.SetActive(r.Active)
		}
	}
}

// pointToFloats / floatsToPoint bridge a Point3 and the record's plain float64 triple.
func pointToFloats(p math.Point3) [3]float64 {
	return [3]float64{float64(p.X), float64(p.Y), float64(p.Z)}
}

func floatsToPoint(f [3]float64) math.Point3 {
	return math.P3(math.Scalar(f[0]), math.Scalar(f[1]), math.Scalar(f[2]))
}

// cellsToFloats / floatsToCells bridge the matrix cell array between math.Scalar and the record's
// plain float64s.
func cellsToFloats(c [16]math.Scalar) [16]float64 {
	var out [16]float64
	for i, v := range c {
		out[i] = float64(v)
	}
	return out
}

func floatsToCells(f [16]float64) [16]math.Scalar {
	var out [16]math.Scalar
	for i, v := range f {
		out[i] = math.Scalar(v)
	}
	return out
}
