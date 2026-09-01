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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestZonedBorderAddsDivisionEntities: a zoned border exports its interior grid lines, so a 4×3-zone
// sheet has exactly five more entities (3 vertical + 2 horizontal divisions) than a plain one (#1989).
func TestZonedBorderAddsDivisionEntities(t *testing.T) {
	t.Parallel()
	c := drawingWithBaseView(t)
	_, plain, err := ExportDrawingDXF(c.Sheets().Active(), DefaultDrawingExportLayers(), types.DXFR2018)
	if err != nil {
		t.Fatalf("plain export: %v", err)
	}
	if err := c.Sheets().Active().SetZonedBorder(4, 3, types.NumericBorderLabel, types.AlphabeticalBorderLabel); err != nil {
		t.Fatalf("SetZonedBorder: %v", err)
	}
	_, zoned, err := ExportDrawingDXF(c.Sheets().Active(), DefaultDrawingExportLayers(), types.DXFR2018)
	if err != nil {
		t.Fatalf("zoned export: %v", err)
	}
	if zoned-plain != 5 {
		t.Errorf("zoned added %d entities, want 5 (3 vertical + 2 horizontal divisions)", zoned-plain)
	}
}

// TestTitleBlockCorner: the title-block origin lands in each requested corner within the margins (#1989).
func TestTitleBlockCorner(t *testing.T) {
	t.Parallel()
	const w, h, margin, tbW, blockH = 400.0, 300.0, 10.0, 80.0, 48.0
	cases := map[types.TitleBlockLocation][2]float64{
		types.BottomRightTitleBlock: {w - margin - tbW, margin},
		types.BottomLeftTitleBlock:  {margin, margin},
		types.TopLeftTitleBlock:     {margin, h - margin - blockH},
		types.TopRightTitleBlock:    {w - margin - tbW, h - margin - blockH},
	}
	for loc, want := range cases {
		x0, y0 := titleBlockCorner(loc, w, h, margin, margin, margin, margin, tbW, blockH)
		if x0 != want[0] || y0 != want[1] {
			t.Errorf("titleBlockCorner(%v) = (%g,%g), want (%g,%g)", loc, x0, y0, want[0], want[1])
		}
	}
}
