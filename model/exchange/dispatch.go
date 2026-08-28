// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		return ExportResult{}, fmt.Errorf("export: unsupported format %q (want stl|obj|3mf|gltf|step)", format)
	}
	// glTF v1 is GLB-only: the encoder emits a self-contained GLB container,
	// and a JSON .gltf destination is a typed error naming the supported
	// extension (R1-2, change-review CHG-2). The CLI guard alone was not
	// enough — the direct model API must enforce it too.
	if format == types.FormatGLTF && strings.ToLower(filepath.Ext(path)) != ".glb" {
		return ExportResult{}, fmt.Errorf("export %q: glTF requires a .glb destination (JSON .gltf output is not supported in this version)", path)
	}
	data, tris, warns, err := route.exportBodies(bodies, res, exportOptions(part, path))
	if err != nil {
		return ExportResult{}, fmt.Errorf("export %q: %w", path, err)
	}
	if err := writeExportFile(path, data); err != nil {
		return ExportResult{}, err
	}
	return ExportResult{TriangleCount: tris, Warnings: warns}, nil
}

// writeExportFile writes data to path atomically: the bytes go to a temp file
// in the destination's directory, then the temp file is renamed over the
// destination. A failure at any step removes the temp file and leaves a
// pre-existing destination untouched — no truncate-then-write window
// (R4-9, change-review CHG-6). The temp file's mode is set BEFORE the rename:
// a pre-existing destination keeps its mode (stat + Chmod), a new destination
// gets 0o644 — os.CreateTemp creates 0600, which would otherwise replace a
// 0644 destination with a 0600 file (CHG3-3).
func writeExportFile(path string, data []byte) error {
	tmpName, err := writeExportTemp(path, data)
	if err != nil {
		return err
	}
	if err := setExportTempMode(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("export: rename over %q: %w", path, err)
	}
	return nil
}

// writeExportTemp writes data to a temp file beside path and closes it; on any
// failure the temp is removed and the destination untouched.
func writeExportTemp(path string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp*")
	if err != nil {
		return "", fmt.Errorf("export: create temp for %q: %w", path, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("export: write %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("export: close %q: %w", path, err)
	}
	return tmpName, nil
}

// setExportTempMode gives the temp file the destination's mode before the rename:
// a pre-existing destination keeps its mode (stat + Chmod), a new destination gets
// 0o644 (os.CreateTemp creates 0600, which would downgrade a 0644 file — CHG3-3).
func setExportTempMode(tmpName, path string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("export: chmod temp for %q: %w", path, err)
	}
	return nil
}

// exportOptions is exportUnits plus the product name a format that carries one writes. The name
// comes from the destination file's base name — the part definition carries none, and a reader's
// model tree showing "plate" beats a constant placeholder (#2055).
func exportOptions(part *compdef.PartComponentDefinition, path string) exchange.TranslationOptions {
	opts := exportUnits(part)
	opts.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return opts
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
