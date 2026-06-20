// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
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
// resource is missing or undecodable.
func (d *PartComponentDefinition) scanPoints(rec doc.PointCloudRecord) []math.Point3 {
	res, ok := d.resources[rec.ResourceID]
	if !ok {
		return nil
	}
	points, err := pointcloud.ReadScan(rec.Source, res.Value)
	if err != nil {
		return nil
	}
	return points
}

// recordOfCloud captures a cloud's metadata + placement (its points stay in the resource table).
func recordOfCloud(pc *pointcloud.PointCloud) doc.PointCloudRecord {
	return doc.PointCloudRecord{
		Name:       pc.Name(),
		Source:     pc.SourceFullFileName(),
		ResourceID: pc.ResourceID(),
		Visible:    pc.Visible(),
		Scale:      pc.Scale(),
		Transform:  cellsToFloats(pc.Transform().Cells()),
		MaxPoints:  pc.MaximumPointCount(),
	}
}

// cloudFromRecord rebuilds a cloud from its record and freshly decoded points.
func cloudFromRecord(rec doc.PointCloudRecord, points []math.Point3) *pointcloud.PointCloud {
	pc := pointcloud.New(rec.Name, rec.Source, rec.ResourceID, points)
	pc.SetVisible(rec.Visible)
	pc.SetScale(rec.Scale) // rejected for a corrupt non-positive value, leaving the default 1
	pc.SetTransform(math.Matrix4FromCells(floatsToCells(rec.Transform)))
	pc.SetMaximumPointCount(rec.MaxPoints)
	return pc
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
