// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/bom"
	"oblikovati.org/model/bomapi"
	"oblikovati.org/model/compdef"
)

// The assembly bill-of-materials surface (M11-F05, #730): read a structured (nested) or
// parts-only (flat) BOM view of the active assembly, and export a view to CSV with optional
// custom property columns. The BOM derives live from the current occurrence tree.

// registerAssemblyBOMHandlers wires the assembly.bom* methods.
func (r *Router) registerAssemblyBOMHandlers() {
	r.handlers[wire.MethodAssemblyBOMView] = assemblyBOMView
	r.handlers[wire.MethodAssemblyBOMExport] = assemblyBOMExport
}

// assemblyBOMView returns the requested BOM view of the active assembly.
func assemblyBOMView(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.BOMViewArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	view, err := bomViewByKind(asm, in.View)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.BOMViewResult{View: in.View, Rows: bomRowInfos(view.Rows)})
}

// assemblyBOMExport renders a BOM view of the active assembly to CSV with the standard
// columns plus one per requested component property.
func assemblyBOMExport(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.BOMExportArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	view, err := bomViewByKind(asm, in.View)
	if err != nil {
		return nil, err
	}
	csv, err := bom.ExportCSV(view, exportColumns(in.Columns))
	if err != nil {
		return nil, fmt.Errorf(errCtxWrap, wire.MethodAssemblyBOMExport, err)
	}
	return json.Marshal(wire.BOMExportResult{CSV: csv})
}

// bomViewByKind builds the active assembly's BOM and selects the requested view.
func bomViewByKind(asm *compdef.AssemblyComponentDefinition, kind types.BOMViewKind) (*bom.View, error) {
	b := bom.New(asm.Occurrences())
	switch kind {
	case types.BOMStructured:
		return b.Structured(), nil
	case types.BOMPartsOnly:
		return b.PartsOnly(), nil
	}
	return nil, fmt.Errorf("%s: unknown view %q (want structured/partsOnly)", wire.MethodAssemblyBOMView, kind)
}

// exportColumns is the standard column set plus one property column per requested name.
func exportColumns(extra []string) []bom.Column {
	cols := bom.StandardColumns()
	for _, name := range extra {
		cols = append(cols, bom.PropertyColumn(name))
	}
	return cols
}

// bomRowInfos renders a slice of BOM rows (recursing into structured children) as DTOs.
func bomRowInfos(rows []*bom.Row) []wire.BOMRowInfo {
	out := make([]wire.BOMRowInfo, len(rows))
	for i, r := range rows {
		out[i] = bomRowInfo(r)
	}
	return out
}

// bomRowInfo renders one BOM row (and its children) as its wire DTO.
func bomRowInfo(r *bom.Row) wire.BOMRowInfo {
	return wire.BOMRowInfo{
		ItemNumber:  r.ItemNumber,
		PartNumber:  r.PartNumber,
		Description: r.Description,
		Structure:   bomapi.BOMStructure(r.Structure),
		Quantity:    r.Quantity,
		Properties:  r.Properties,
		Children:    bomRowInfos(r.Children),
	}
}
