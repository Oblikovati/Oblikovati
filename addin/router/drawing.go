// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
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
	r.readOnly(wire.MethodDrawingListSheets, drawingListSheets)
	r.readOnly(wire.MethodDrawingAddSheet, drawingAddSheet)
	r.readOnly(wire.MethodDrawingRemoveSheet, drawingRemoveSheet)
	r.readOnly(wire.MethodDrawingSetActiveSheet, drawingSetActiveSheet)
	r.readOnly(wire.MethodDrawingSetModelReference, drawingSetModelReference)
	r.readOnly(wire.MethodDrawingTitleBlockFields, drawingTitleBlockFields)
	r.readOnly(wire.MethodDrawingExportDXF, drawingExportDXF)
}

// drawingExportDXF writes the active sheet (its views' visible/hidden edges, border and title
// block) to a DXF file. Views are re-projected first so the export reflects the current model.
func drawingExportDXF(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	var in wire.ExportDrawingDXFArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if in.Path == "" {
		return nil, fmt.Errorf("drawing: export path is required")
	}
	c.RecomputeViews()
	sheet := c.Sheets().Active()
	if sheet == nil {
		return nil, fmt.Errorf("drawing: no active sheet to export")
	}
	n, err := exchange.ExportDrawingDXFFile(sheet, in.Path, exchange.DefaultDrawingExportLayers(), types.DXFVersion(in.Version).Normalized())
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.ExportDrawingDXFResult{Path: in.Path, Entities: n})
}

// activeDrawing returns the active document's drawing content with its title-block
// resolver wired to the workspace (app.ActiveDrawing is shared with the Drawing-tab tools
// and the head sheet canvas, so resolution behaves identically on every path).
func activeDrawing(s *app.Session) (*drawing.Content, error) {
	return app.ActiveDrawing(s)
}

func drawingListSheets(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(listSheetsResult(c))
}

func drawingAddSheet(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddSheetArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	spec, err := sheetSpecOf(in)
	if err != nil {
		return nil, err
	}
	sh, err := c.Sheets().Add(spec)
	if err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.SheetResult{Sheet: sheetInfo(c, sh)})
}

func drawingRemoveSheet(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	var in wire.RemoveSheetArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := c.Sheets().Remove(in.Name); err != nil {
		return nil, err
	}
	s.ActiveDocument().MarkDirty()
	return json.Marshal(listSheetsResult(c))
}

func drawingSetActiveSheet(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetActiveSheetArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := c.Sheets().SetActive(in.Name); err != nil {
		return nil, err
	}
	return json.Marshal(wire.SheetResult{Sheet: sheetInfo(c, c.Sheets().Active())})
}

func drawingSetModelReference(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetModelReferenceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	c.SetModelReference(in.FullDocumentName)
	s.ActiveDocument().MarkDirty()
	return json.Marshal(wire.SetModelReferenceResult{ModelReference: c.ModelReference()})
}

func drawingTitleBlockFields(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	c, err := activeDrawing(s)
	if err != nil {
		return nil, err
	}
	var in wire.TitleBlockFieldsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sh, err := sheetByNameOrActive(c, in.Sheet)
	if err != nil {
		return nil, err
	}
	return json.Marshal(titleBlockFieldsResult(sh))
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
