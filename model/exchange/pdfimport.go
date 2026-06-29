// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"os"

	"oblikovati.org/kernel/exchange/pdf"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// PDFImportResult is SketchImportResult under the PDF name, kept parallel to
// DWGImportResult so callers read symmetrically.
type PDFImportResult = SketchImportResult

// ImportPDFFile reads a vector .pdf file and imports it into part, each page's curve
// geometry becoming its own 2D sketch on the chosen plane — the path-based companion to
// ImportPDF.
func ImportPDFFile(part *compdef.PartComponentDefinition, path string, plane sketch.Plane) (PDFImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PDFImportResult{}, fmt.Errorf("import pdf: read %q: %w", path, err)
	}
	return ImportPDF(part, data, plane)
}

// ImportPDF decodes vector PDF bytes (a CAD drawing plotted to PDF) and adds each page's
// geometry to part as a 2D sketch on plane — one sketch per page, so a multi-page plot set
// imports as several sketches. The decode and the per-page sketch conversion are shared
// with every drawing format (see sketch_from_drawing.go); only pdf.DecodePages is
// PDF-specific.
//
// Example:
//
//	res, err := exchange.ImportPDF(part, data, workPlane.Plane())
func ImportPDF(part *compdef.PartComponentDefinition, data []byte, plane sketch.Plane) (PDFImportResult, error) {
	pages, warns, err := pdf.DecodePages(data)
	if err != nil {
		return PDFImportResult{}, fmt.Errorf("import pdf: %w", err)
	}
	res := PDFImportResult{Warnings: warns}
	for _, dr := range pages {
		imp := importDrawing(part, dr, plane)
		res.EntityCount += imp.entityCount
		res.Is3D = res.Is3D || imp.is3D
		res.Warnings = append(res.Warnings, imp.warnings...)
	}
	return res, nil
}
