// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"

	"oblikovati.org/api/types"
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
// work plane); a non-planar drawing becomes a Sketch3D and plane is ignored. The whole path
// is shared with every drawing format: the registered [DrawingDecoder] decodes, and
// importSketchFormat places the result (#1631, audit I8).
//
// Example:
//
//	res, err := exchange.ImportDWG(part, data, workPlane.Plane())
func ImportDWG(part *compdef.PartComponentDefinition, data []byte, plane sketch.Plane) (DWGImportResult, error) {
	return importSketchFormat(part, types.FormatDWG, data, plane)
}
