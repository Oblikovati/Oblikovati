// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/kernel/exchange/dxf"
	"oblikovati.org/model/sketch"
)

// ExportDXF encodes a 2D sketch's geometry as an ASCII DXF file of the given version: the
// inverse of ImportDXF. The sketch→drawing conversion is shared with every drawing format
// (sketch_to_drawing.go); only the dxf.Encode call is DXF-specific. It returns the bytes and
// the number of curves written. Coordinates are written in database units (cm).
//
//	data, n, err := exchange.ExportDXF(sk, types.DXFR2018)
func ExportDXF(sk *sketch.Sketch, version types.DXFVersion) ([]byte, int, error) {
	dr := &drawing.Drawing{Entities: sketchToDrawing(sk), Units: drawing.INSCentimetres}
	data, err := dxf.Encode(dr, dxfVersion(version))
	return data, len(dr.Entities), err
}

// ExportDXFFile writes ExportDXF's output to path, returning the number of curves written.
func ExportDXFFile(sk *sketch.Sketch, path string, version types.DXFVersion) (int, error) {
	data, n, err := ExportDXF(sk, version)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return 0, fmt.Errorf("export dxf: write %q: %w", path, err)
	}
	return n, nil
}

// dxfVersion maps the public DXFVersion enum to the codec's version, defaulting to R2000.
func dxfVersion(v types.DXFVersion) dxf.Version {
	if v.Normalized() == types.DXFR2018 {
		return dxf.R2018
	}
	return dxf.R2000
}
