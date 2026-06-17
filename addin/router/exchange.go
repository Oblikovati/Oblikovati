// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/exchange"
)

// registerExchangeHandlers wires the foreign mesh-format import/export methods
// (documents.import / documents.export, M17-F04).
func (r *Router) registerExchangeHandlers() {
	r.handlers[wire.MethodDocumentsImport] = importDocument
	r.handlers[wire.MethodDocumentsExport] = exportDocument
	r.handlers[wire.MethodImportDWG] = importDWG
	r.handlers[wire.MethodImportDXF] = importDXF
	r.handlers[wire.MethodExportDXF] = exportDXF
}

// importDWG imports a .dwg into the active part: a planar drawing onto the named
// work plane (default: first origin plane), a non-planar one into a 3D sketch.
func importDWG(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ImportDWGArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	res, err := s.ImportDWGOnPlane(in.Path, in.Plane)
	if err != nil {
		return nil, fmt.Errorf("import.dwg: %w", err)
	}
	return json.Marshal(wire.ImportDWGResult{Is3D: res.Is3D, EntityCount: res.EntityCount, Warnings: res.Warnings})
}

// importDXF imports a .dxf into the active part: a planar drawing onto the named work plane
// (default: first origin plane), a non-planar one into a 3D sketch.
func importDXF(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ImportDXFArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	res, err := s.ImportDXFOnPlane(in.Path, in.Plane)
	if err != nil {
		return nil, fmt.Errorf("import.dxf: %w", err)
	}
	return json.Marshal(wire.ImportDXFResult{Is3D: res.Is3D, EntityCount: res.EntityCount, Warnings: res.Warnings})
}

// exportDXF writes the active 2D sketch to a .dxf file at the requested version.
func exportDXF(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ExportDXFArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	n, err := s.ExportActiveSketchDXF(in.Path, types.DXFVersion(in.Version))
	if err != nil {
		return nil, fmt.Errorf("export.dxf: %w", err)
	}
	return json.Marshal(wire.ExportDXFResult{EntityCount: n})
}

// importDocument reads a foreign file (STL/OBJ/3MF mesh, or STEP B-rep) into the active part as
// imported-body
// feature — a watertight mesh becomes a solid downstream features can operate on.
func importDocument(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.ImportRequest
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	res, err := exchange.Import(part, in.Path, types.ExchangeFormat(in.Format))
	if err != nil {
		return nil, fmt.Errorf("documents.import: %w", err)
	}
	return json.Marshal(wire.ImportResponse{BodyCount: res.BodyCount, Solid: res.Solid, Warnings: res.Warnings})
}

// exportDocument writes the active part's bodies to a foreign file: a mesh format tessellates at
// the requested resolution (the tessellation-density knob); STEP writes the exact B-rep.
func exportDocument(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.ExportRequest
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	res, err := exchange.Export(part, in.Path, types.ExchangeFormat(in.Format), types.MeshResolution(in.Resolution))
	if err != nil {
		return nil, fmt.Errorf("documents.export: %w", err)
	}
	return json.Marshal(wire.ExportResponse{TriangleCount: res.TriangleCount, Warnings: res.Warnings})
}
