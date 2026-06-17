// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"

	"oblikovati.org/kernel/exchange/dxf"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// ImportDXFFile reads a .dxf file and imports it into part on the chosen plane (2D) or as a
// Sketch3D — the DXF counterpart of ImportDWGFile.
func ImportDXFFile(part *compdef.PartComponentDefinition, path string, plane sketch.Plane) (SketchImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SketchImportResult{}, fmt.Errorf("import dxf: read %q: %w", path, err)
	}
	return ImportDXF(part, data, plane)
}

// ImportDXF decodes DXF bytes and adds their geometry to part. A planar drawing becomes a 2D
// Sketch on plane; a non-planar drawing becomes a Sketch3D and plane is ignored. The decode
// and sketch conversion are shared with the other drawing formats (sketch_from_drawing.go);
// only the dxf.Decode call is DXF-specific.
//
// Example:
//
//	res, err := exchange.ImportDXF(part, data, workPlane.Plane())
func ImportDXF(part *compdef.PartComponentDefinition, data []byte, plane sketch.Plane) (SketchImportResult, error) {
	dr, warns, err := dxf.Decode(data)
	if err != nil {
		return SketchImportResult{}, fmt.Errorf("import dxf: %w", err)
	}
	imp := importDrawing(part, dr, plane)
	return SketchImportResult{Is3D: imp.is3D, EntityCount: imp.entityCount, Warnings: append(warns, imp.warnings...)}, nil
}
