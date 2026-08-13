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
	r.mutating(wire.MethodDrawingAddDefaultBorder, "Add Border", typedCtx(activeDrawing, drawingAddDefaultBorder))
	r.mutating(wire.MethodDrawingSetTitleBlock, "Set Title Block", typedCtx(activeDrawing, drawingSetTitleBlock))
	r.mutating(wire.MethodDrawingSetSheetRevision, "Set Sheet Revision", typedCtx(activeDrawing, drawingSetSheetRevision))
	r.mutating(wire.MethodDrawingDefineSheetFormat, "Define Sheet Format", typedCtx(activeDrawing, drawingDefineSheetFormat))
	r.mutating(wire.MethodDrawingAddSheetUsingFormat, "Add Sheet", typedCtx(activeDrawing, drawingAddSheetUsingFormat))
}

// drawingAddDefaultBorder gives a sheet a zoned border (#1989).
func drawingAddDefaultBorder(s *app.Session, c *drawing.Content, in wire.AddDefaultBorderArgs) (wire.SheetResult, error) {
	sh, err := sheetByNameOrActive(c, in.Sheet)
	if err != nil {
		return wire.SheetResult{}, err
	}
	hMode, ok := types.ParseBorderLabelMode(in.HLabelMode)
	if !ok {
		return wire.SheetResult{}, fmt.Errorf("drawing: unknown border label mode %q", in.HLabelMode)
	}
	vMode, ok := types.ParseBorderLabelMode(in.VLabelMode)
	if !ok {
		return wire.SheetResult{}, fmt.Errorf("drawing: unknown border label mode %q", in.VLabelMode)
	}
	if err := sh.SetZonedBorder(in.HZones, in.VZones, hMode, vMode); err != nil {
		return wire.SheetResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.SheetResult{Sheet: sheetInfo(c, sh)}, nil
}

// drawingSetTitleBlock moves a sheet's title block to a corner (#1989).
func drawingSetTitleBlock(s *app.Session, c *drawing.Content, in wire.SetTitleBlockArgs) (wire.SheetResult, error) {
	sh, err := sheetByNameOrActive(c, in.Sheet)
	if err != nil {
		return wire.SheetResult{}, err
	}
	loc, ok := types.ParseTitleBlockLocation(in.Location)
	if !ok {
		return wire.SheetResult{}, fmt.Errorf("drawing: unknown title-block location %q", in.Location)
	}
	sh.SetTitleBlockLocation(loc)
	s.ActiveDocument().MarkDirty()
	return wire.SheetResult{Sheet: sheetInfo(c, sh)}, nil
}

// drawingSetSheetRevision sets a sheet's revision string (#1989).
func drawingSetSheetRevision(s *app.Session, c *drawing.Content, in wire.SetSheetRevisionArgs) (wire.SheetResult, error) {
	sh, err := sheetByNameOrActive(c, in.Sheet)
	if err != nil {
		return wire.SheetResult{}, err
	}
	sh.SetRevision(in.Revision)
	s.ActiveDocument().MarkDirty()
	return wire.SheetResult{Sheet: sheetInfo(c, sh)}, nil
}

// drawingDefineSheetFormat registers a reusable sheet format (#1989).
func drawingDefineSheetFormat(s *app.Session, c *drawing.Content, in wire.DefineSheetFormatArgs) (wire.ListSheetsResult, error) {
	f, err := sheetFormatOf(in)
	if err != nil {
		return wire.ListSheetsResult{}, err
	}
	if err := c.Sheets().DefineFormat(f); err != nil {
		return wire.ListSheetsResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return listSheetsResult(c), nil
}

// drawingAddSheetUsingFormat adds a sheet stamped from a registered format (#1989).
func drawingAddSheetUsingFormat(s *app.Session, c *drawing.Content, in wire.AddSheetUsingFormatArgs) (wire.SheetResult, error) {
	sh, err := c.Sheets().AddUsingFormat(in.Name, in.Format)
	if err != nil {
		return wire.SheetResult{}, err
	}
	s.ActiveDocument().MarkDirty()
	return wire.SheetResult{Sheet: sheetInfo(c, sh)}, nil
}

// sheetFormatOf converts a define-format request into a model SheetFormat, parsing the size,
// orientation, label modes and title-block location spellings.
func sheetFormatOf(in wire.DefineSheetFormatArgs) (drawing.SheetFormat, error) {
	if in.Name == "" {
		return drawing.SheetFormat{}, fmt.Errorf("drawing: a sheet format needs a name")
	}
	size := types.SheetSizeCustom
	if in.Size != "" {
		parsed, ok := types.ParseSheetSize(in.Size)
		if !ok {
			return drawing.SheetFormat{}, fmt.Errorf("drawing: unknown sheet size %q", in.Size)
		}
		size = parsed
	}
	orient, ok := types.ParseSheetOrientation(orientationOrPortrait(in.Orientation))
	if !ok {
		return drawing.SheetFormat{}, fmt.Errorf("drawing: unknown sheet orientation %q", in.Orientation)
	}
	hMode, vMode, loc, err := formatBorderAndTitle(in)
	if err != nil {
		return drawing.SheetFormat{}, err
	}
	return drawing.SheetFormat{
		Name: in.Name, Size: size, Orientation: orient, WidthMM: in.WidthMM, HeightMM: in.HeightMM,
		HZones: in.HZones, VZones: in.VZones, HLabelMode: hMode, VLabelMode: vMode, TitleBlockLocation: loc,
	}, nil
}

// formatBorderAndTitle parses a format request's two border-label modes and its title-block corner.
func formatBorderAndTitle(in wire.DefineSheetFormatArgs) (hMode, vMode types.BorderLabelMode, loc types.TitleBlockLocation, err error) {
	hMode, ok := types.ParseBorderLabelMode(in.HLabelMode)
	if !ok {
		return 0, 0, 0, fmt.Errorf("drawing: unknown border label mode %q", in.HLabelMode)
	}
	vMode, ok = types.ParseBorderLabelMode(in.VLabelMode)
	if !ok {
		return 0, 0, 0, fmt.Errorf("drawing: unknown border label mode %q", in.VLabelMode)
	}
	loc, ok = types.ParseTitleBlockLocation(in.TitleBlockLocation)
	if !ok {
		return 0, 0, 0, fmt.Errorf("drawing: unknown title-block location %q", in.TitleBlockLocation)
	}
	return hMode, vMode, loc, nil
}

// orientationOrPortrait defaults an empty orientation spelling to portrait.
func orientationOrPortrait(s string) string {
	if s == "" {
		return "portrait"
	}
	return s
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
	info := wire.SheetInfo{
		Name:          sh.Name(),
		Size:          sh.Size().String(),
		Orientation:   sh.Orientation().String(),
		WidthMM:       sh.WidthMM(),
		HeightMM:      sh.HeightMM(),
		Active:        c.Sheets().Active() == sh,
		HasBorder:     sh.Border() != nil,
		HasTitleBlock: sh.TitleBlock() != nil,
		Revision:      sh.Revision(),
	}
	if b := sh.Border(); b != nil {
		if bd, ok := b.(*drawing.Border); ok {
			info.BorderHZones, info.BorderVZones = bd.ZoneCounts()
		}
	}
	if tb := sh.TitleBlockRef(); tb != nil {
		info.TitleBlockLocation = tb.Location().String()
	}
	return info
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
