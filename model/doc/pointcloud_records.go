// SPDX-License-Identifier: GPL-2.0-only

package doc

// Point-cloud persistence bridge (M17-F06, #645). A point cloud is document-level input (like
// resources, not part of the recipe), so it round-trips through a neutral record the content
// exposes — mirroring [ResourceBearer]. The scan's points are NOT in the record: they live once
// in the resource table (addressed by ResourceID) and the cloud re-decodes them on restore, so
// the .obk stays small.

// PointCloudRecord is the serialisable form of one attached scan: its metadata and the id of the
// resource holding its bytes. Transform is the 16 cells of the cloud→model placement matrix in
// row-major order.
type PointCloudRecord struct {
	Name        string
	Source      string
	ResourceID  string
	Visible     bool
	DisplayMode string
	Scale       float64
	Transform   [16]float64
	MaxPoints   int
	Crops       []PointCloudCropRecord
}

// PointCloudCropRecord is the serialisable form of one crop volume (#645): its name, whether it
// is active, and its model-space box as min/max corners.
type PointCloudCropRecord struct {
	Name   string
	Active bool
	Min    [3]float64
	Max    [3]float64
}

// PointCloudBearer is the optional interface content implements when it owns attached point
// clouds (#645). The store reads it on save and restores it on open AFTER the resource table is
// in place (so the cloud can re-decode its points). Like [ResourceBearer], it is optional.
type PointCloudBearer interface {
	Content
	// PointCloudRecords returns the document's attached-cloud records (save path).
	PointCloudRecords() []PointCloudRecord
	// SetPointCloudRecords reconstructs the clouds from their records, reading each cloud's
	// points from the already-restored resource table (load path).
	SetPointCloudRecords([]PointCloudRecord)
}
