// SPDX-License-Identifier: GPL-2.0-only

package bomapi_test

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/bomapi"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// TestBOMAdapterPartsOnlyTotalsQuantity checks the contract adapter surfaces the model's
// parts-only view: two placements of one definition total to a single row of quantity 2,
// reported with the default ("normal") structure and the parts-only kind.
func TestBOMAdapterPartsOnlyTotalsQuantity(t *testing.T) {
	part := compdef.NewPartComponentDefinition()
	occs := occurrence.NewOccurrences()
	occs.AddByComponentDefinition("p:1", part, math.Identity4())
	occs.AddByComponentDefinition("p:2", part, math.Translation4(math.V3(2, 0, 0)))

	view := bomapi.New(occs).PartsOnly()
	if view.Kind() != types.BOMPartsOnly {
		t.Errorf("view kind = %q, want partsOnly", view.Kind())
	}
	rows := view.Rows()
	if len(rows) != 1 || rows[0].Quantity() != 2 {
		t.Fatalf("rows = %d, want one row of quantity 2", len(rows))
	}
	if rows[0].Structure() != types.BOMNormal || rows[0].ItemNumber() != 1 {
		t.Errorf("row = {item %d, %q}, want {1, normal}", rows[0].ItemNumber(), rows[0].Structure())
	}
}
