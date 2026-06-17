// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"

	"oblikovati.org/kernel/exchange/dwg"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// ImportDWGFile reads a .dwg file and imports it into part on the chosen plane (2D) or as a
// Sketch3D, the path-based companion to ImportDWG. The caller resolves the plane from the
// work plane the user picked.
func ImportDWGFile(part *compdef.PartComponentDefinition, path string, plane sketch.Plane) (DWGImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DWGImportResult{}, fmt.Errorf("import dwg: read %q: %w", path, err)
	}
	return ImportDWG(part, data, plane)
}

// SketchImportResult summarises a drawing import (DWG/DXF): whether it landed in a 3D sketch
// (vs a 2D sketch on the chosen plane), how many entities were added, and any per-entity
// decode/convert warnings.
type SketchImportResult struct {
	Is3D        bool
	EntityCount int
	Warnings    []string
}

// DWGImportResult is the original name of SketchImportResult, kept so existing callers (the
// session, the router) compile unchanged.
type DWGImportResult = SketchImportResult

// ImportDWG decodes DWG bytes and adds their geometry to part. A planar drawing becomes a
// 2D Sketch on plane (the world origin mapping to the plane origin, per the caller's chosen
// work plane); a non-planar drawing becomes a Sketch3D and plane is ignored. The decode and
// the sketch conversion are shared with every drawing format (see sketch_from_drawing.go);
// only the dwg.Decode call is DWG-specific.
//
// Example:
//
//	res, err := exchange.ImportDWG(part, data, workPlane.Plane())
func ImportDWG(part *compdef.PartComponentDefinition, data []byte, plane sketch.Plane) (DWGImportResult, error) {
	dr, warns, err := dwg.Decode(data)
	if err != nil {
		return DWGImportResult{}, fmt.Errorf("import dwg: %w", err)
	}
	imp := importDrawing(part, dr, plane)
	return DWGImportResult{Is3D: imp.is3D, EntityCount: imp.entityCount, Warnings: append(warns, imp.warnings...)}, nil
}
