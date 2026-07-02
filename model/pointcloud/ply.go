// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"oblikovati.org/kernel/exchange/plyfmt"
	"oblikovati.org/math"
)

// PLY point reader (M17-F06, #645): a point cloud needs only the vertex positions of a Stanford
// PLY (the common 3D-scanner export), so this reader delegates the format parsing to the shared
// kernel/exchange/plyfmt package and takes its vertices — the same parser the mesh-exchange
// importer uses for the full mesh, so there is one PLY decoder.
type plyReader struct{}

// NewPLYReader returns the reader for Stanford .ply scan files.
func NewPLYReader() PointReader { return plyReader{} }

func (plyReader) Extensions() []string { return []string{".ply"} }

// FileUnitMM: PLY carries no unit, so it follows the same declared millimetre convention as the
// unitless mesh formats (STL/OBJ) — the .ply mesh/cloud symmetry test pins this (#1636).
func (plyReader) FileUnitMM() float64 { return 1 }

// Read decodes the PLY's vertex positions into cloud-local points (faces are ignored).
func (plyReader) Read(data []byte) ([]math.Point3, error) {
	doc, err := plyfmt.Parse(data)
	if err != nil {
		return nil, err
	}
	return doc.Vertices()
}
