// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/kernel/exchange/dwg"
	"oblikovati.org/kernel/exchange/dxf"
	"oblikovati.org/kernel/exchange/pdf"
)

// DrawingDecoder is one drawing format's decode entry (#1631, audit I8): the format it
// serves, the file extensions it claims, and the decode from raw bytes to the format-neutral
// drawing model every importer shares (sketch_from_drawing.go). Format identity, extensions
// and decode travel together through one registration (format_routes.go), so the format
// dispatch and the extension mapping — formerly two independently maintained switches — can
// never drift apart (the #1416 shape: a format the menu recognized but the dispatch refused).
type DrawingDecoder interface {
	// Format names the exchange format this decoder serves; it must be a sketch
	// format (types.ExchangeFormat.IsSketch), enforced at registry construction.
	Format() types.ExchangeFormat
	// Extensions lists the file extensions (lowercase, dot-prefixed: ".dwg") the
	// format is recognized by.
	Extensions() []string
	// Decode parses src into format-neutral drawings: one for a single-model file
	// (DWG/DXF), one per page for a paged plot set (PDF). The warnings are
	// per-entity decode notes; err is a hard parse failure.
	Decode(src []byte) ([]*drawing.Drawing, []string, error)
}

// dwgDrawingDecoder wraps the clean-room DWG codec (kernel/exchange/dwg).
type dwgDrawingDecoder struct{}

func (dwgDrawingDecoder) Format() types.ExchangeFormat { return types.FormatDWG }
func (dwgDrawingDecoder) Extensions() []string         { return []string{".dwg"} }
func (dwgDrawingDecoder) Decode(src []byte) ([]*drawing.Drawing, []string, error) {
	dr, warns, err := dwg.Decode(src)
	if err != nil {
		return nil, nil, err
	}
	return []*drawing.Drawing{dr}, warns, nil
}

// dxfDrawingDecoder wraps the ASCII DXF codec (kernel/exchange/dxf).
type dxfDrawingDecoder struct{}

func (dxfDrawingDecoder) Format() types.ExchangeFormat { return types.FormatDXF }
func (dxfDrawingDecoder) Extensions() []string         { return []string{".dxf"} }
func (dxfDrawingDecoder) Decode(src []byte) ([]*drawing.Drawing, []string, error) {
	dr, warns, err := dxf.Decode(src)
	if err != nil {
		return nil, nil, err
	}
	return []*drawing.Drawing{dr}, warns, nil
}

// pdfDrawingDecoder wraps the vector-PDF page decoder (kernel/exchange/pdf). A PDF plot
// set decodes to one drawing per page, so its entry shape is the general one — the
// single-drawing formats are the one-element case.
type pdfDrawingDecoder struct{}

func (pdfDrawingDecoder) Format() types.ExchangeFormat { return types.FormatPDF }
func (pdfDrawingDecoder) Extensions() []string         { return []string{".pdf"} }
func (pdfDrawingDecoder) Decode(src []byte) ([]*drawing.Drawing, []string, error) {
	return pdf.DecodePages(src)
}

// Each drawing codec must satisfy the decode seam it is registered under.
var (
	_ DrawingDecoder = dwgDrawingDecoder{}
	_ DrawingDecoder = dxfDrawingDecoder{}
	_ DrawingDecoder = pdfDrawingDecoder{}
)
