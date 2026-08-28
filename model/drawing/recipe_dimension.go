// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"encoding/hex"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// Drawing-recipe — the DIMENSIONS section (M48 #2226 split of recipe.go). The YAML shape of one
// drawing dimension (its attached vertex/edge keys as hex; the glyph and measured value re-derive on
// open) and the snapshot/restore of that section.

// dimensionRecipe is the YAML shape of one drawing dimension. The glyph and measured value are
// not stored — they re-derive on open from the attached vertices (KeyA/KeyB, hex of each vertex's
// reference key), which re-bind to the current model, so a dimension always reflects it.
type dimensionRecipe struct {
	Name     string  `yaml:"name"`
	Type     string  `yaml:"type,omitempty"`
	ViewName string  `yaml:"viewName"`
	KeyA     string  `yaml:"keyA,omitempty"`     // linear: first attached vertex
	KeyB     string  `yaml:"keyB,omitempty"`     // linear: second attached vertex
	EdgeKey  string  `yaml:"edgeKey,omitempty"`  // radial: circular edge; angular: first straight edge
	EdgeKeyB string  `yaml:"edgeKeyB,omitempty"` // angular: second straight edge
	Offset   float64 `yaml:"offsetMm,omitempty"`
	TextDX   float64 `yaml:"textDxMm,omitempty"` // user text nudge (drag-the-text)
	TextDY   float64 `yaml:"textDyMm,omitempty"`
	AxisHorz bool    `yaml:"axisHorizontal,omitempty"` // ordinate: measure the view-X offset, else view-Y
	// Text metadata (#1990/#1992/#1993/#1996) — the decorated text re-derives from these on open.
	Prefix       string                     `yaml:"prefix,omitempty"`
	Suffix       string                     `yaml:"suffix,omitempty"`
	OverrideText string                     `yaml:"overrideText,omitempty"`
	HideValue    bool                       `yaml:"hideValue,omitempty"`
	DualUnit     bool                       `yaml:"dualUnit,omitempty"`
	Tolerance    *types.DimensionTolerance  `yaml:"tolerance,omitempty"`
	Inspection   *types.InspectionDimension `yaml:"inspection,omitempty"`
	// Retrieved model dimension (#1991): the source parameter name and the 3D world endpoints it
	// spans (re-fetched from the model on open; the endpoints are the fallback when it is gone).
	RetrievedFrom string     `yaml:"retrievedFrom,omitempty"`
	WorldA        [3]float64 `yaml:"worldA,omitempty"`
	WorldB        [3]float64 `yaml:"worldB,omitempty"`
}

// point3Cells / point3FromCells convert a 3D point to/from its persisted [x,y,z] cells (#1991).
func point3Cells(p gmath.Point3) [3]float64 {
	return [3]float64{float64(p.X), float64(p.Y), float64(p.Z)}
}

func point3FromCells(c [3]float64) gmath.Point3 {
	return gmath.P3(gmath.Scalar(c[0]), gmath.Scalar(c[1]), gmath.Scalar(c[2]))
}

// dimensionRecipesOf snapshots a sheet's dimensions for persistence (the attached vertex keys as
// hex; the glyph and value re-derive on open).
func dimensionRecipesOf(sh *Sheet) []dimensionRecipe {
	if sh.dimensions == nil {
		return nil
	}
	out := make([]dimensionRecipe, 0, len(sh.dimensions.items))
	for _, d := range sh.dimensions.items {
		out = append(out, dimensionRecipe{
			Name: d.name, Type: d.dimType.String(), ViewName: d.viewName,
			KeyA: hex.EncodeToString(d.keyA), KeyB: hex.EncodeToString(d.keyB),
			EdgeKey: hex.EncodeToString(d.edgeKey), EdgeKeyB: hex.EncodeToString(d.edgeKeyB),
			Offset: d.offset, TextDX: d.textDX, TextDY: d.textDY, AxisHorz: d.axisHorizontal,
			Prefix: d.prefix, Suffix: d.suffix, OverrideText: d.overrideText,
			HideValue: d.hideValue, DualUnit: d.dualUnit,
			Tolerance: nonZeroTolerance(d.tolerance), Inspection: nonZeroInspection(d.inspection),
			RetrievedFrom: d.retrievedFrom, WorldA: point3Cells(d.worldA), WorldB: point3Cells(d.worldB),
		})
	}
	return out
}

// nonZeroTolerance / nonZeroInspection return a pointer to persist only when the metadata is set,
// so a plain dimension writes no tolerance/inspection block (#1990/#1996).
func nonZeroTolerance(t types.DimensionTolerance) *types.DimensionTolerance {
	if t.Type == types.NoTolerance {
		return nil
	}
	return &t
}

func nonZeroInspection(i types.InspectionDimension) *types.InspectionDimension {
	if i.Shape == types.NoInspectionBorder {
		return nil
	}
	return &i
}

// restoreDimensions rebuilds a sheet's dimensions from its recipe; each re-binds its attached
// vertices and re-measures on the next RecomputeViews (once the referenced model resolves).
func restoreDimensions(sh *Sheet, recs []dimensionRecipe) {
	if len(recs) == 0 {
		return
	}
	ds := sh.Dimensions()
	for _, dr := range recs {
		dimType, _ := types.ParseDrawingDimensionType(dr.Type)
		keyA, _ := hex.DecodeString(dr.KeyA)
		keyB, _ := hex.DecodeString(dr.KeyB)
		edgeKey, _ := hex.DecodeString(dr.EdgeKey)
		edgeKeyB, _ := hex.DecodeString(dr.EdgeKeyB)
		d := &DrawingDimension{
			name: dr.Name, dimType: dimType, viewName: dr.ViewName,
			keyA: keyA, keyB: keyB, edgeKey: edgeKey, edgeKeyB: edgeKeyB,
			offset: dr.Offset, textDX: dr.TextDX, textDY: dr.TextDY, axisHorizontal: dr.AxisHorz,
			prefix: dr.Prefix, suffix: dr.Suffix, overrideText: dr.OverrideText,
			hideValue: dr.HideValue, dualUnit: dr.DualUnit,
			retrievedFrom: dr.RetrievedFrom, worldA: point3FromCells(dr.WorldA), worldB: point3FromCells(dr.WorldB),
		}
		if dr.Tolerance != nil {
			d.tolerance = *dr.Tolerance
		}
		if dr.Inspection != nil {
			d.inspection = *dr.Inspection
		}
		ds.recompute(d)
		ds.items = append(ds.items, d)
	}
}
