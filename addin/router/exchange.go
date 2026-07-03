// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange"
)

// registerExchangeHandlers wires the foreign mesh-format import/export methods
// (documents.import / documents.export, M17-F04).
func (r *Router) registerExchangeHandlers() {
	r.mutating(wire.MethodDocumentsImport, "Import", typedPart(importDocument))
	r.readOnly(wire.MethodDocumentsExport, typedPart(exportDocument))
	r.readOnly(wire.MethodImportDWG, typed(importDWG))
	r.readOnly(wire.MethodImportDXF, typed(importDXF))
	r.readOnly(wire.MethodImportPDF, typed(importPDF))
	r.readOnly(wire.MethodExportDXF, typed(exportDXF))
}

// importDWG imports a .dwg into the active part: a planar drawing onto the named
// work plane (default: first origin plane), a non-planar one into a 3D sketch.
func importDWG(s *app.Session, in wire.ImportDWGArgs) (wire.ImportDWGResult, error) {
	res, err := s.ImportDWGOnPlane(in.Path, in.Plane)
	if err != nil {
		return wire.ImportDWGResult{}, fmt.Errorf("import.dwg: %w", err)
	}
	return wire.ImportDWGResult{Is3D: res.Is3D, EntityCount: res.EntityCount, Warnings: res.Warnings}, nil
}

// importDXF imports a .dxf into the active part: a planar drawing onto the named work plane
// (default: first origin plane), a non-planar one into a 3D sketch.
func importDXF(s *app.Session, in wire.ImportDXFArgs) (wire.ImportDXFResult, error) {
	res, err := s.ImportDXFOnPlane(in.Path, in.Plane)
	if err != nil {
		return wire.ImportDXFResult{}, fmt.Errorf("import.dxf: %w", err)
	}
	return wire.ImportDXFResult{Is3D: res.Is3D, EntityCount: res.EntityCount, Warnings: res.Warnings}, nil
}

// importPDF imports a vector .pdf (a CAD drawing plotted to PDF) into the active part: each
// page's vector paths become a 2D sketch on the named work plane (default: first origin
// plane). Text and raster images in the page are skipped.
func importPDF(s *app.Session, in wire.ImportPDFArgs) (wire.ImportPDFResult, error) {
	res, err := s.ImportPDFOnPlane(in.Path, in.Plane)
	if err != nil {
		return wire.ImportPDFResult{}, fmt.Errorf("import.pdf: %w", err)
	}
	return wire.ImportPDFResult{Is3D: res.Is3D, EntityCount: res.EntityCount, Warnings: res.Warnings}, nil
}

// exportDXF writes the active 2D sketch to a .dxf file at the requested version.
func exportDXF(s *app.Session, in wire.ExportDXFArgs) (wire.ExportDXFResult, error) {
	n, err := s.ExportActiveSketchDXF(in.Path, types.DXFVersion(in.Version))
	if err != nil {
		return wire.ExportDXFResult{}, fmt.Errorf("export.dxf: %w", err)
	}
	return wire.ExportDXFResult{EntityCount: n}, nil
}

// importDocument reads a foreign file (STL/OBJ/3MF mesh, or STEP B-rep) into the active part as
// imported-body
// feature — a watertight mesh becomes a solid downstream features can operate on.
func importDocument(_ *app.Session, part *compdef.PartComponentDefinition, in wire.ImportRequest) (wire.ImportResponse, error) {
	res, err := exchange.Import(part, in.Path, types.ExchangeFormat(in.Format))
	if err != nil {
		return wire.ImportResponse{}, fmt.Errorf("documents.import: %w", err)
	}
	return wire.ImportResponse{BodyCount: res.BodyCount, Solid: res.Solid, Warnings: res.Warnings}, nil
}

// exportDocument writes the active part's bodies to a foreign file: a mesh format tessellates at
// the requested resolution (the tessellation-density knob); STEP writes the exact B-rep.
func exportDocument(_ *app.Session, part *compdef.PartComponentDefinition, in wire.ExportRequest) (wire.ExportResponse, error) {
	res, err := exchange.Export(part, in.Path, types.ExchangeFormat(in.Format), types.MeshResolution(in.Resolution))
	if err != nil {
		return wire.ExportResponse{}, fmt.Errorf("documents.export: %w", err)
	}
	return wire.ExportResponse{TriangleCount: res.TriangleCount, Warnings: res.Warnings}, nil
}
