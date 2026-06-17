// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	kdraw "oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	dmodel "oblikovati.org/model/drawing"
)

// fakeViewBody is a named fake (CLAUDE.md: no inline stubs) resolving one body for any
// reference — the model a drawing's views project.
type fakeViewBody struct{ body *topo.Body }

func (f fakeViewBody) Body(string) (*topo.Body, bool) { return f.body, f.body != nil }

// drawingWithBaseView builds a drawing referencing a 2×3×4 box with one iso base view (general
// position, so the visible/hidden split is stable).
func drawingWithBaseView(t *testing.T) *dmodel.Content {
	t.Helper()
	c := dmodel.NewContent()
	c.SetBodyResolver(fakeViewBody{body: subd.ToBody(subd.Box(2, 3, 4), "box")})
	c.SetModelReference("box.opd")
	if _, err := c.Sheets().Active().Views().AddBase(dmodel.BaseViewSpec{
		Orientation: types.BaseViewIso, Scale: 2, CenterX: 120, CenterY: 100,
	}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	return c
}

// TestExportDrawingDXF checks a sheet exports to DXF with its view edges on the Visible/Hidden
// layers, the border rectangle, and the title-block text — the manufacturing/handoff artifact.
func TestExportDrawingDXF(t *testing.T) {
	c := drawingWithBaseView(t)
	data, n, err := ExportDrawingDXF(c.Sheets().Active(), DefaultDrawingExportLayers(), types.DXFR2018)
	if err != nil {
		t.Fatalf("ExportDrawingDXF: %v", err)
	}
	if n == 0 {
		t.Fatal("exported no entities")
	}
	dxf := string(data)
	for _, want := range []string{"LINE", "TEXT", "Visible", "Hidden", "Border", "TitleBlock", "Part Number"} {
		if !strings.Contains(dxf, want) {
			t.Errorf("DXF missing %q", want)
		}
	}
}

// TestExportDrawingDXFFile checks the file path writes a non-empty DXF.
func TestExportDrawingDXFFile(t *testing.T) {
	c := drawingWithBaseView(t)
	path := t.TempDir() + "/sheet.dxf"
	n, err := ExportDrawingDXFFile(c.Sheets().Active(), path, DrawingExportLayers{}, types.DXFR2000)
	if err != nil || n == 0 {
		t.Fatalf("ExportDrawingDXFFile = (%d, %v), want entities written", n, err)
	}
}

// TestSheetToDrawingClassifiesLayers checks visible vs hidden view curves land on distinct
// layers (the core value of the export for a drafter).
func TestSheetToDrawingClassifiesLayers(t *testing.T) {
	c := drawingWithBaseView(t)
	ents := SheetToDrawing(c.Sheets().Active(), DefaultDrawingExportLayers())
	var visible, hidden int
	for _, e := range ents {
		line, ok := e.(*kdraw.Line)
		if !ok {
			continue
		}
		switch line.Layer {
		case "Visible":
			visible++
		case "Hidden":
			hidden++
		}
	}
	if visible == 0 || hidden == 0 {
		t.Errorf("layers = %d visible / %d hidden, want both populated", visible, hidden)
	}
}
