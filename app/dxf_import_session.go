// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/exchange"
	"oblikovati.org/model/sketch"
)

// ImportDXFFile imports a .dxf into the active part: a planar drawing becomes a 2D sketch on
// plane (the user's chosen work plane), a non-planar drawing a Sketch3D. The import is an
// undoable edit. It is the DXF counterpart of ImportDWGFile and shares the plane choices
// (see DWGPlaneChoices).
//
// Example:
//
//	choices, _ := s.DWGPlaneChoices()
//	res, err := s.ImportDXFFile("floor.dxf", choices[0].Plane) // onto XY
func (s *Session) ImportDXFFile(path string, plane sketch.Plane) (exchange.SketchImportResult, error) {
	part, err := activePart(s)
	if err != nil {
		return exchange.SketchImportResult{}, err
	}
	// One undoable step for the whole import (not per added entity); idempotent baseline. See
	// ImportDWGFile for the rationale.
	s.EnsureActiveEditBaseline()
	res, err := exchange.ImportDXFFile(part, path, plane)
	if err != nil {
		return res, err
	}
	s.recordEdit(part, fmt.Sprintf("Import DXF (%d entities)", res.EntityCount))
	if res.EntityCount > 0 { // #1645: frame the imported drawing
		s.RequestFitView()
	}
	return res, nil
}

// ImportDXFOnPlane imports a .dxf onto the named work plane (case-insensitive; e.g. "XY
// Plane"), defaulting to the first origin plane when planeName is empty or unknown. It is the
// name-keyed entry the RPC/CLI layer calls; the GUI uses ImportDXFFile with a plane chosen
// from DWGPlaneChoices.
func (s *Session) ImportDXFOnPlane(path, planeName string) (exchange.SketchImportResult, error) {
	choices, err := s.DWGPlaneChoices()
	if err != nil {
		return exchange.SketchImportResult{}, err
	}
	if len(choices) == 0 {
		return exchange.SketchImportResult{}, fmt.Errorf("import dxf: active part has no work planes")
	}
	plane := choices[0].Plane
	if planeName != "" {
		match, ok := planeByName(choices, planeName)
		if !ok {
			return exchange.SketchImportResult{}, fmt.Errorf("import dxf: no work plane named %q (have %s)", planeName, planeNames(choices))
		}
		plane = match
	}
	return s.ImportDXFFile(path, plane)
}

// ExportActiveSketchDXF writes the active 2D sketch to a .dxf file of the given version,
// returning the number of curves written. It errors when no 2D sketch is active.
func (s *Session) ExportActiveSketchDXF(path string, version types.DXFVersion) (int, error) {
	sk := s.ActiveSketch()
	if sk == nil {
		return 0, fmt.Errorf("export dxf: no active 2D sketch to export")
	}
	return exchange.ExportDXFFile(sk, path, version, s.DocumentUnits())
}
