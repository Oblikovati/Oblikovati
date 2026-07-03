// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
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
	if format.IsPointCloud() {
		return ImportResult{}, fmt.Errorf("import %q: %s is a scan format and attaches a point cloud, not bodies; use ImportPointCloud (pointClouds.attach)", path, format)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ImportResult{}, fmt.Errorf("import: read %q: %w", path, err)
	}
	bodies, warns, err := feature.ImportBodiesFromData(format, data, workingUnitMM(part))
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
	// Registry lookup instead of a format switch: the export capability registers with the
	// format's extensions in format_routes.go, so menu recognition and dispatch cannot
	// drift (#1631, audit I8).
	route, ok := formatRoutes.byFormat[format]
	if !ok || route.exportBodies == nil {
		return ExportResult{}, fmt.Errorf("export: unsupported format %q (want stl|obj|3mf|step)", format)
	}
	data, tris, warns, err := route.exportBodies(bodies, res, exportUnits(part))
	if err != nil {
		return ExportResult{}, fmt.Errorf("export %q: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("export: write %q: %w", path, err)
	}
	return ExportResult{TriangleCount: tris, Warnings: warns}, nil
}

// exportUnits builds the translation options for an export: the kernel works in
// centimetres (TargetUnitMM = exchange.DBUnitMM), and the file is written in the
// document's preferred length unit so the exported numbers match what the user
// sees (Oblikovati/Oblikovati#146).
func exportUnits(part *compdef.PartComponentDefinition) exchange.TranslationOptions {
	return exchange.TranslationOptions{
		TargetUnitMM: workingUnitMM(part),
		FileUnit:     part.Units().PreferredName(param.Length),
	}
}

// workingUnitMM is the millimetre size of one of the part's stored (working) length units —
// the database-unit size the translators scale against (ADR-0042 Phase 2). The computation
// lives on the part (WorkingUnitMM) so the point-cloud attach and persistence paths share it
// (#1636); a cm document (working scale 1) yields the historical 10 mm and is unchanged.
func workingUnitMM(part *compdef.PartComponentDefinition) float64 {
	return part.WorkingUnitMM()
}

// FormatFromPath infers the exchange format from a file's extension (case-insensitive), so the
// File ▸ Import/Export menu can route by what the user typed. The bool is false for an unknown
// extension. It is a lookup over the same registry the dispatchers use (format_routes.go), so
// an extension this recognizes always has a routed format behind it (#1631, audit I8). The
// point-cloud scan formats resolve here too (#1646): a .ply ALWAYS resolves to FormatPLY — a
// point-cloud format, never a mesh (the documented rule, see api/types FormatPLY) — and the
// ASCII scan family (.xyz/.pts/.asc/.txt) stays unrouted (dispatched by pointcloud.IsScanFile).
func FormatFromPath(path string) (types.ExchangeFormat, bool) {
	return formatForExtension(filepath.Ext(path))
}
