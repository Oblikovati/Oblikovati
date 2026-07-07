// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"oblikovati.org/kernel/exchange/plyfmt"
	"oblikovati.org/math"
)

// PLY point reader (M17-F06, #645): a point cloud needs the vertex positions and any scan colour or
// intensity of a Stanford PLY (the common 3D-scanner export). All the header and record parsing lives
// in the shared kernel/exchange/plyfmt package (#1788); this reader only maps its decoded channels
// onto cloud-local samples, so the byte-level format knowledge is owned in one place.
type plyReader struct{}

// NewPLYReader returns the reader for Stanford .ply scan files.
func NewPLYReader() PointReader { return plyReader{} }

func (plyReader) Extensions() []string { return []string{".ply"} }

// FileUnitMM: PLY carries no unit, so it follows the same declared millimetre convention as the
// unitless mesh formats (STL/OBJ) — the .ply mesh/cloud symmetry test pins this (#1636).
func (plyReader) FileUnitMM() float64 { return 1 }

// ReadSamples decodes the PLY vertex element into cloud-local samples (faces are ignored). The vertex
// layout comes from the header, so a fault is structural: no per-record warnings.
func (plyReader) ReadSamples(data []byte) ([]PointSample, []string, error) {
	doc, err := plyfmt.Parse(data)
	if err != nil {
		return nil, nil, err
	}
	scan, err := doc.Scan()
	if err != nil {
		return nil, nil, err
	}
	return samplesFromChannels(scan.Points, scan.RGB, scan.Intensity), nil, nil
}

// Read returns point-only coordinates for callers that do not need channels.
func (r plyReader) Read(data []byte) ([]math.Point3, []string, error) {
	samples, warns, err := r.ReadSamples(data)
	if err != nil {
		return nil, nil, err
	}
	return pointsOf(samples), warns, nil
}
