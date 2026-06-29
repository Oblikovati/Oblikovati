// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/exchange"
	"oblikovati.org/model/sketch"
)

// ImportPDFFile imports a vector .pdf (a CAD drawing plotted to PDF) into the active part:
// each page's curve geometry becomes a 2D sketch on plane (the user's chosen work plane).
// The whole import is one undoable edit. It reuses DWGPlaneChoices for the plane list — the
// drawing-import target planes are the same regardless of source format.
//
// Example:
//
//	choices, _ := s.DWGPlaneChoices()
//	res, err := s.ImportPDFFile("plan.pdf", choices[0].Plane) // onto XY
func (s *Session) ImportPDFFile(path string, plane sketch.Plane) (exchange.PDFImportResult, error) {
	part, err := activePart(s)
	if err != nil {
		return exchange.PDFImportResult{}, err
	}
	// Open the undo baseline at the pre-import state so the (often many thousands of)
	// per-entity sketch additions collapse into ONE undo step. Idempotent.
	s.EnsureActiveEditBaseline()
	res, err := exchange.ImportPDFFile(part, path, plane)
	if err != nil {
		return res, err
	}
	s.recordEdit(part, fmt.Sprintf("Import PDF (%d entities)", res.EntityCount))
	return res, nil
}

// ImportPDFOnPlane imports a .pdf onto the named work plane (case-insensitive; e.g.
// "XY Plane"), defaulting to the first origin plane when planeName is empty or unknown. It
// is the name-keyed entry the RPC/CLI layer calls; the GUI uses ImportPDFFile with a plane
// chosen from DWGPlaneChoices.
func (s *Session) ImportPDFOnPlane(path, planeName string) (exchange.PDFImportResult, error) {
	choices, err := s.DWGPlaneChoices()
	if err != nil {
		return exchange.PDFImportResult{}, err
	}
	if len(choices) == 0 {
		return exchange.PDFImportResult{}, fmt.Errorf("import pdf: active part has no work planes")
	}
	plane := choices[0].Plane
	if planeName != "" {
		match, ok := planeByName(choices, planeName)
		if !ok {
			return exchange.PDFImportResult{}, fmt.Errorf("import pdf: no work plane named %q (have %s)", planeName, planeNames(choices))
		}
		plane = match
	}
	return s.ImportPDFFile(path, plane)
}
