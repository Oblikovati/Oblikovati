// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/bom"
	"oblikovati.org/model/bomapi"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// The assembly bill-of-materials surface (M11-F05, #730): read a structured (nested) or
// parts-only (flat) BOM view of the active assembly, and export a view to CSV with optional
// custom property columns. The BOM derives live from the current occurrence tree.

// registerAssemblyBOMHandlers wires the assembly.bom* methods.
func (r *Router) registerAssemblyBOMHandlers() {
	r.readOnly(wire.MethodAssemblyBOMView, typedAssembly(assemblyBOMView))
	r.readOnly(wire.MethodAssemblyBOMExport, typedAssembly(assemblyBOMExport))
	r.mutating(wire.MethodAssemblySetBOMStructure, "Set BOM Structure", typedAssembly(assemblySetBOMStructure))
}

// assemblySetBOMStructure sets an occurrence's per-occurrence BOM-structure override (#1978).
func assemblySetBOMStructure(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetBOMStructureArgs) (wire.SetBOMStructureResult, error) {
	o, err := occurrenceByID(asm, in.Occurrence, wire.MethodAssemblySetBOMStructure)
	if err != nil {
		return wire.SetBOMStructureResult{}, err
	}
	spelling := string(in.Structure)
	if spelling == "" {
		spelling = "default"
	}
	st, ok := bom.ParseStructure(spelling)
	if !ok || st == bom.Varies {
		return wire.SetBOMStructureResult{}, fmt.Errorf("%s: cannot set BOM structure %q (want default/normal/phantom/reference/purchased/inseparable)", wire.MethodAssemblySetBOMStructure, in.Structure)
	}
	o.SetBOMStructureOverride(spelling)
	s.ActiveDocument().MarkDirty()
	return wire.SetBOMStructureResult{Occurrence: o.ID(), Structure: currentBOMStructure(o)}, nil
}

// currentBOMStructure reports an occurrence's BOM-structure override, or "default" when it inherits.
func currentBOMStructure(o *occurrence.Occurrence) types.BOMStructure {
	if ov := o.BOMStructureOverride(); ov != "" {
		return types.BOMStructure(ov)
	}
	return types.BOMDefault
}

// assemblyBOMView returns the requested BOM view of the active assembly.
func assemblyBOMView(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.BOMViewArgs) (wire.BOMViewResult, error) {
	view, err := bomViewByKind(asm, in.View)
	if err != nil {
		return wire.BOMViewResult{}, err
	}
	return wire.BOMViewResult{View: in.View, Rows: bomRowInfos(view.Rows)}, nil
}

// assemblyBOMExport renders a BOM view of the active assembly to CSV with the standard
// columns plus one per requested component property.
func assemblyBOMExport(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.BOMExportArgs) (wire.BOMExportResult, error) {
	view, err := bomViewByKind(asm, in.View)
	if err != nil {
		return wire.BOMExportResult{}, err
	}
	csv, err := bom.ExportCSV(view, exportColumns(in.Columns))
	if err != nil {
		return wire.BOMExportResult{}, fmt.Errorf("%s: %w", wire.MethodAssemblyBOMExport, err)
	}
	return wire.BOMExportResult{CSV: csv}, nil
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
