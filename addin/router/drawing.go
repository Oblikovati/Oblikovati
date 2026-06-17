// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/drawing"
)

// The drawing-document sheet surface (M14-F01, #384): list/add/remove sheets, pick the
// active sheet, point the drawing at the model it documents, and read a title block's
// fields resolved against that model's iProperties.

// registerDrawingHandlers wires the drawing.* methods.
func (r *Router) registerDrawingHandlers() {
	r.handlers[wire.MethodDrawingListSheets] = drawingListSheets
	r.handlers[wire.MethodDrawingAddSheet] = drawingAddSheet
	r.handlers[wire.MethodDrawingRemoveSheet] = drawingRemoveSheet
	r.handlers[wire.MethodDrawingSetActiveSheet] = drawingSetActiveSheet
	r.handlers[wire.MethodDrawingSetModelReference] = drawingSetModelReference
	r.handlers[wire.MethodDrawingTitleBlockFields] = drawingTitleBlockFields
}

// activeDrawing returns the active document's drawing content, erroring if no drawing is
// active. It (re)wires the title-block resolver to the workspace so fields resolve
// against the current referenced model.
func activeDrawing(s *app.Session) (*drawing.Content, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, fmt.Errorf("drawing: no active document")
	}
	c, ok := d.Content().(*drawing.Content)
	if !ok {
		return nil, fmt.Errorf("drawing: active document %q is not a drawing", d.FullDocumentName())
	}
	c.SetModelProperties(referencedModelProperties{ws: s.Workspace(), ref: c.ModelReference})
	return c, nil
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

// referencedModelProperties resolves a drawing's referenced model iProperties by looking
// the document up by name in the workspace on each read (so it tracks the model being
// opened or edited after the drawing references it). ref re-reads the drawing's current
// reference so a later setModelReference is honoured.
type referencedModelProperties struct {
	ws  *doc.Workspace
	ref func() string
}

func (p referencedModelProperties) Property(set, name string) (string, bool) {
	d, ok := p.ws.ByName(p.ref())
	if !ok || d.Content() == nil {
		return "", false
	}
	props, ok := d.Content().(interface{ Properties() *attr.PropertySets })
	if !ok {
		return "", false
	}
	ps, ok := props.Properties().Lookup(set)
	if !ok {
		return "", false
	}
	prop, ok := ps.Property(name)
	if !ok {
		return "", false
	}
	return propertyText(prop.Value()), true
}

// propertyText renders an iProperty value as the plain text a title block shows.
func propertyText(v attr.Value) string {
	if s, ok := v.Str(); ok {
		return s
	}
	if i, ok := v.Int(); ok {
		return strconv.FormatInt(i, 10)
	}
	if f, ok := v.Float(); ok {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	if b, ok := v.Bool(); ok {
		return strconv.FormatBool(b)
	}
	return ""
}
