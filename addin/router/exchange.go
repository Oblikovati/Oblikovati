// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati/api/types"
	"oblikovati/api/wire"

	"oblikovati/addin/modelaccess"
	"oblikovati/app"
	"oblikovati/model/exchange"
)

// registerExchangeHandlers wires the foreign mesh-format import/export methods
// (documents.import / documents.export, M17-F04).
func (r *Router) registerExchangeHandlers() {
	r.handlers[wire.MethodDocumentsImport] = importDocument
	r.handlers[wire.MethodDocumentsExport] = exportDocument
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
