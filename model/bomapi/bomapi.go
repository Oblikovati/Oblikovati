// SPDX-License-Identifier: GPL-2.0-only

// Package bomapi adapts the model's bill-of-materials (model/bom) to the public scalar
// contract surface (api/contract), the way bodyapi/geomapi adapt their kernels. The BOM
// rows/views are data structs; these thin adapters expose them as the contract's getter
// interfaces for an in-proc consumer (the head). The over-the-wire surface is served from
// addin/router (M11-F05, #730).
package bomapi

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/model/bom"
	"oblikovati.org/model/occurrence"
)

// New returns the contract BOM read surface over the occurrences' live bill of materials.
func New(root *occurrence.Occurrences) contract.BOM { return bomAdapter{bom.New(root)} }

type bomAdapter struct{ b *bom.BOM }

func (a bomAdapter) Structured() contract.BOMView { return viewAdapter{a.b.Structured()} }
func (a bomAdapter) PartsOnly() contract.BOMView  { return viewAdapter{a.b.PartsOnly()} }

type viewAdapter struct{ v *bom.View }

func (a viewAdapter) Kind() types.BOMViewKind { return viewKind(a.v.Kind) }
func (a viewAdapter) Rows() []contract.BOMRow { return wrapRows(a.v.Rows) }

type rowAdapter struct{ r *bom.Row }

func (a rowAdapter) ItemNumber() int               { return a.r.ItemNumber }
func (a rowAdapter) PartNumber() string            { return a.r.PartNumber }
func (a rowAdapter) Description() string           { return a.r.Description }
func (a rowAdapter) Structure() types.BOMStructure { return BOMStructure(a.r.Structure) }
func (a rowAdapter) Quantity() int                 { return a.r.Quantity }
func (a rowAdapter) Children() []contract.BOMRow   { return wrapRows(a.r.Children) }

// wrapRows adapts a slice of model rows to the contract row surface.
func wrapRows(rows []*bom.Row) []contract.BOMRow {
	out := make([]contract.BOMRow, len(rows))
	for i, r := range rows {
		out[i] = rowAdapter{r}
	}
	return out
}

// BOMStructure maps a model structure to its public wire spelling (the model's own
// String()), shared by the contract adapter and the router.
func BOMStructure(s bom.Structure) types.BOMStructure { return types.BOMStructure(s.String()) }

// viewKind maps a model view kind to its public spelling.
func viewKind(k bom.ViewKind) types.BOMViewKind {
	if k == bom.PartsOnlyView {
		return types.BOMPartsOnly
	}
	return types.BOMStructured
}

var (
	_ contract.BOM     = bomAdapter{}
	_ contract.BOMView = viewAdapter{}
	_ contract.BOMRow  = rowAdapter{}
)
