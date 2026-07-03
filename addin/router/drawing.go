// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/drawing"
	"oblikovati.org/model/exchange"
)

// The drawing-document sheet surface (M14-F01, #384): list/add/remove sheets, pick the
// active sheet, point the drawing at the model it documents, and read a title block's
// fields resolved against that model's iProperties.

// registerDrawingHandlers wires the drawing.* methods.
func (r *Router) registerDrawingHandlers() {
	r.readOnly(wire.MethodDrawingListSheets, ctxQuery(activeDrawing, drawingListSheets))
	r.mutating(wire.MethodDrawingAddSheet, "Add Sheet", typedCtx(activeDrawing, drawingAddSheet))
	r.mutating(wire.MethodDrawingRemoveSheet, "Delete Sheet", typedCtx(activeDrawing, drawingRemoveSheet))
	r.readOnly(wire.MethodDrawingSetActiveSheet, typedCtx(activeDrawing, drawingSetActiveSheet))
	r.mutating(wire.MethodDrawingSetModelReference, "Set Model Reference", typedCtx(activeDrawing, drawingSetModelReference))
	r.readOnly(wire.MethodDrawingTitleBlockFields, typedCtx(activeDrawing, drawingTitleBlockFields))
	r.readOnly(wire.MethodDrawingExportDXF, typedCtx(activeDrawing, drawingExportDXF))
}

// drawingExportDXF writes the active sheet (its views' visible/hidden edges, border and title
// block) to a DXF file. Views are re-projected first so the export reflects the current model.
func drawingExportDXF(_ *app.Session, c *drawing.Content, in wire.ExportDrawingDXFArgs) (wire.ExportDrawingDXFResult, error) {
	if in.Path == "" {
		return wire.ExportDrawingDXFResult{}, fmt.Errorf("drawing: export path is required")
	}
	c.RecomputeViews()
	sheet := c.Sheets().Active()
	if sheet == nil {
		return wire.ExportDrawingDXFResult{}, fmt.Errorf("drawing: no active sheet to export")
	}
	n, err := exchange.ExportDrawingDXFFile(sheet, in.Path, exchange.DefaultDrawingExportLayers(), types.DXFVersion(in.Version).Normalized())
	if err != nil {
		return wire.ExportDrawingDXFResult{}, err
	}
	return wire.ExportDrawingDXFResult{Path: in.Path, Entities: n}, nil
}

// activeDrawing returns the active document's drawing content with its title-block
// resolver wired to the workspace (app.ActiveDrawing is shared with the Drawing-tab tools
// and the head sheet canvas, so resolution behaves identically on every path).
func activeDrawing(s *app.Session) (*drawing.Content, error) {
	return app.ActiveDrawing(s)
}

func drawingListSheets(_ *app.Session, c *drawing.Content) (wire.ListSheetsResult, error) {
	return listSheetsResult(c), nil
}

func drawingAddSheet(s *app.Session, c *drawing.Content, in wire.AddSheetArgs) (wire.SheetResult, error) {
	spec, err := sheetSpecOf(in)
	if err != nil {
		return wire.SheetResult{}, err
	}
	sh, err := c.Sheets().Add(spec)
	if err != nil {
		return wire.SheetResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.SheetResult{Sheet: sheetInfo(c, sh)}, nil
}

func drawingRemoveSheet(s *app.Session, c *drawing.Content, in wire.RemoveSheetArgs) (wire.ListSheetsResult, error) {
	if err := c.Sheets().Remove(in.Name); err != nil {
		return wire.ListSheetsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listSheetsResult(c), nil
}

func drawingSetActiveSheet(_ *app.Session, c *drawing.Content, in wire.SetActiveSheetArgs) (wire.SheetResult, error) {
	if err := c.Sheets().SetActive(in.Name); err != nil {
		return wire.SheetResult{}, err
	}
	return wire.SheetResult{Sheet: sheetInfo(c, c.Sheets().Active())}, nil
}

func drawingSetModelReference(s *app.Session, c *drawing.Content, in wire.SetModelReferenceArgs) (wire.SetModelReferenceResult, error) {
	c.SetModelReference(in.FullDocumentName)
	s.ActiveDocument().MarkDirty()
	return wire.SetModelReferenceResult{ModelReference: c.ModelReference()}, nil
}

func drawingTitleBlockFields(_ *app.Session, c *drawing.Content, in wire.TitleBlockFieldsArgs) (wire.TitleBlockFieldsResult, error) {
	sh, err := sheetByNameOrActive(c, in.Sheet)
	if err != nil {
		return wire.TitleBlockFieldsResult{}, err
	}
	return titleBlockFieldsResult(sh), nil
}

// sheetByNameOrActive returns the named sheet, or the active sheet when name is empty.
func sheetByNameOrActive(c *drawing.Content, name string) (*drawing.Sheet, error) {
	if name == "" {
		sh := c.Sheets().Active()
		if sh == nil {
			return nil, fmt.Errorf("drawing: no active sheet")
		}
		return sh, nil
	}
	sh, ok := c.Sheets().ByName(name)
	if !ok {
		return nil, fmt.Errorf("drawing: no sheet named %q", name)
	}
	return sh, nil
}

// sheetSpecOf converts the wire request into a model SheetSpec, parsing the size and
// orientation spellings (empty size ⇒ custom).
func sheetSpecOf(in wire.AddSheetArgs) (drawing.SheetSpec, error) {
	size := types.SheetSizeCustom
	if in.Size != "" {
		parsed, ok := types.ParseSheetSize(in.Size)
		if !ok {
			return drawing.SheetSpec{}, fmt.Errorf("drawing: unknown sheet size %q", in.Size)
		}
		size = parsed
	}
	orient := types.SheetPortrait
	if in.Orientation != "" {
		parsed, ok := types.ParseSheetOrientation(in.Orientation)
		if !ok {
			return drawing.SheetSpec{}, fmt.Errorf("drawing: unknown sheet orientation %q", in.Orientation)
		}
		orient = parsed
	}
	return drawing.SheetSpec{Name: in.Name, Size: size, Orientation: orient, WidthMM: in.WidthMM, HeightMM: in.HeightMM}, nil
}

func listSheetsResult(c *drawing.Content) wire.ListSheetsResult {
	out := wire.ListSheetsResult{ModelReference: c.ModelReference()}
	for i := 0; i < c.Sheets().Count(); i++ {
		out.Sheets = append(out.Sheets, sheetInfo(c, c.Sheets().Item(i)))
	}
	return out
}

func sheetInfo(c *drawing.Content, sh *drawing.Sheet) wire.SheetInfo {
	return wire.SheetInfo{
		Name:          sh.Name(),
		Size:          sh.Size().String(),
		Orientation:   sh.Orientation().String(),
		WidthMM:       sh.WidthMM(),
		HeightMM:      sh.HeightMM(),
		Active:        c.Sheets().Active() == sh,
		HasBorder:     sh.Border() != nil,
		HasTitleBlock: sh.TitleBlock() != nil,
	}
}

func titleBlockFieldsResult(sh *drawing.Sheet) wire.TitleBlockFieldsResult {
	tb, ok := sh.TitleBlock().(*drawing.TitleBlock)
	if !ok {
		return wire.TitleBlockFieldsResult{}
	}
	out := wire.TitleBlockFieldsResult{DefinitionName: tb.DefinitionName()}
	for _, f := range tb.Fields() {
		out.Fields = append(out.Fields, wire.TitleBlockField{Name: f.Name, Value: f.Value, Source: f.Source})
	}
	return out
}
