// SPDX-License-Identifier: GPL-2.0-only

// Package exchange is the model-layer orchestration for foreign mesh-format import and
// export (M17-F04): it reads a file off disk, translates it through the kernel mesh
// engine (kernel/exchange/meshio), and injects the result into a part as an imported-
// body feature (import), or tessellates the part's bodies and writes a file (export).
// It satisfies the api/contract.MeshTranslator capability query (compile-time asserted
// below) so the host can route a request to it.
package exchange

import (
	"fmt"
	"os"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// MeshExchange is the model-layer mesh translator. The zero value is ready to use.
type MeshExchange struct{}

// compile-time assertion that MeshExchange satisfies the public capability contract.
var _ contract.MeshTranslator = MeshExchange{}

// Formats lists the mesh formats this translator handles (import and export).
func (MeshExchange) Formats() []types.ExchangeFormat {
	return []types.ExchangeFormat{types.FormatSTL, types.FormatOBJ, types.Format3MF}
}

// CanImport reports whether the format is a supported mesh format.
func (MeshExchange) CanImport(f types.ExchangeFormat) bool { return f.IsMesh() }

// CanExport reports whether the format is a supported mesh format.
func (MeshExchange) CanExport(f types.ExchangeFormat) bool { return f.IsMesh() }

// ImportResult reports what an import produced: the body count, whether the first body
// is a watertight solid, and any non-fatal warnings.
type ImportResult struct {
	BodyCount int
	Solid     bool
	Warnings  []string
}

// ImportInto reads path, translates it as format into a B-rep body, and adds it to part
// as an imported-body feature (re-imported on reopen via the recorded source). It
// recomputes the part so the body is in SurfaceBodies. A watertight mesh becomes a solid;
// an open mesh a surface body (reported via ImportResult.Solid, not an error).
//
// Example:
//
//	res, err := exchange.MeshExchange{}.ImportInto(part, "bolt.stl", types.FormatSTL)
func (MeshExchange) ImportInto(part *compdef.PartComponentDefinition, path string, format types.ExchangeFormat) (ImportResult, error) {
	if !format.IsMesh() {
		return ImportResult{}, fmt.Errorf("import: %q is not a mesh format (want stl|obj|3mf)", format)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ImportResult{}, fmt.Errorf("import: read %q: %w", path, err)
	}
	feat := fmt.Sprintf("import:%s#0", format)
	body, warns, err := meshio.ImportBody(format, data, feat, 0, exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	if err != nil {
		return ImportResult{}, fmt.Errorf("import %q: %w", path, err)
	}
	// Embed the source bytes and cite them by UUID so the document is self-contained (ADR-0031).
	id := part.AddResource(resourceFor(format, path, data))
	feature.NewImportedBodies(part.Features()).Add(body, id, string(format))
	part.Recompute()
	return ImportResult{BodyCount: 1, Solid: body.IsSolid(), Warnings: warns}, nil
}

// ExportResult reports what an export wrote: the total triangle count and any warnings.
type ExportResult struct {
	TriangleCount int
	Warnings      []string
}

// ExportFrom tessellates the part's bodies at the given resolution and writes them to
// path in format. Multiple bodies are merged into one mesh file (the formats carry a
// single mesh first cut). An empty part errors rather than writing an empty file.
//
// Example:
//
//	res, err := exchange.MeshExchange{}.ExportFrom(part, "p.stl", types.FormatSTL, types.ResolutionHigh)
func (MeshExchange) ExportFrom(part *compdef.PartComponentDefinition, path string, format types.ExchangeFormat, res types.MeshResolution) (ExportResult, error) {
	if !format.IsMesh() {
		return ExportResult{}, fmt.Errorf("export: %q is not a mesh format (want stl|obj|3mf)", format)
	}
	bodies := part.SurfaceBodies().All()
	if len(bodies) == 0 {
		return ExportResult{}, fmt.Errorf("export %q: part has no bodies", path)
	}
	data, tris, err := meshio.ExportBodies(format, bodies, res, exportUnits(part))
	if err != nil {
		return ExportResult{}, fmt.Errorf("export %q: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return ExportResult{}, fmt.Errorf("export: write %q: %w", path, err)
	}
	return ExportResult{TriangleCount: tris}, nil
}
