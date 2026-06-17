// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Import reads path as format into the part as imported-body features — routing mesh formats
// (STL/OBJ/3MF, one welded body) and STEP (every B-rep solid in the file) through the shared
// feature.ImportBodies reader. Each body becomes an ImportedBodyFeature recording its source
// (path, format, index) so a reopened .obk re-imports it. The part is recomputed so the bodies
// land in SurfaceBodies; a watertight body is a solid (reported via ImportResult.Solid).
//
// Example:
//
//	res, err := exchange.Import(part, "bracket.step", types.FormatSTEP)
func Import(part *compdef.PartComponentDefinition, path string, format types.ExchangeFormat) (ImportResult, error) {
	if format.IsSketch() {
		return ImportResult{}, fmt.Errorf("import %q: %s is a sketch format and imports into a sketch on a chosen work plane; use ImportDWGFile/ImportDXFFile", path, format)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ImportResult{}, fmt.Errorf("import: read %q: %w", path, err)
	}
	bodies, warns, err := feature.ImportBodiesFromData(format, data)
	if err != nil {
		return ImportResult{}, fmt.Errorf("import %q: %w", path, err)
	}
	if len(bodies) == 0 {
		return ImportResult{}, fmt.Errorf("import %q: no bodies found", path)
	}
	// Embed the source bytes in the document once and cite that resource from every body, so a
	// reopened .obk re-derives the bodies from itself — no external file path (ADR-0031).
	id := part.AddResource(resourceFor(format, path, data))
	imp := feature.NewImportedBodies(part.Features())
	for i, b := range bodies {
		imp.AddAt(b, id, string(format), i)
	}
	part.Recompute()
	return ImportResult{BodyCount: len(bodies), Solid: bodies[0].IsSolid(), Warnings: warns}, nil
}

// Export writes the part's bodies to path in format — a mesh format tessellates at the given
// resolution; STEP writes the exact B-rep (resolution ignored). An empty part errors rather than
// writing an empty file.
//
// Example:
//
//	res, err := exchange.Export(part, "p.step", types.FormatSTEP, "")
func Export(part *compdef.PartComponentDefinition, path string, format types.ExchangeFormat, res types.MeshResolution) (ExportResult, error) {
	bodies := part.SurfaceBodies().All()
	if len(bodies) == 0 {
		return ExportResult{}, fmt.Errorf("export %q: part has no bodies", path)
	}
	var (
		data  []byte
		tris  int
		warns []string
		err   error
	)
	switch {
	case format.IsMesh():
		data, tris, err = meshio.ExportBodies(format, bodies, res)
	case format == types.FormatSTEP:
		data, warns, err = step.Writer{}.ExportSolids(bodies, exchange.TranslationOptions{TargetUnitMM: 1})
	default:
		return ExportResult{}, fmt.Errorf("export: unsupported format %q (want stl|obj|3mf|step)", format)
	}
	if err != nil {
		return ExportResult{}, fmt.Errorf("export %q: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("export: write %q: %w", path, err)
	}
	return ExportResult{TriangleCount: tris, Warnings: warns}, nil
}

// FormatFromPath infers the exchange format from a file's extension (case-insensitive), so the
// File ▸ Import/Export menu can route by what the user typed. The bool is false for an unknown
// extension.
func FormatFromPath(path string) (types.ExchangeFormat, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".stl":
		return types.FormatSTL, true
	case ".obj":
		return types.FormatOBJ, true
	case ".3mf":
		return types.Format3MF, true
	case ".step", ".stp":
		return types.FormatSTEP, true
	case ".dwg":
		return types.FormatDWG, true
	case ".dxf":
		return types.FormatDXF, true
	default:
		return "", false
	}
}
