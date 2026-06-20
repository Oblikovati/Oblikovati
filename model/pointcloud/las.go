// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"oblikovati.org/kernel/exchange/lasfmt"
	"oblikovati.org/math"
)

// LAS point reader (M17-F06, #645): the ASPRS LAS format is the standard interchange for LiDAR
// point data (airborne and terrestrial surveys). A point cloud needs only the XYZ positions, so
// this reader delegates the header/record parsing to the shared kernel/exchange/lasfmt package and
// takes its vertices. The compressed LAZ variant is not handled here.
type lasReader struct{}

// NewLASReader returns the reader for ASPRS .las scan files.
func NewLASReader() PointReader { return lasReader{} }

func (lasReader) Extensions() []string { return []string{".las"} }

// Read decodes the LAS point records into cloud-local points (intensity/returns are ignored).
func (lasReader) Read(data []byte) ([]math.Point3, error) {
	doc, err := lasfmt.Parse(data)
	if err != nil {
		return nil, err
	}
	return doc.Vertices()
}
