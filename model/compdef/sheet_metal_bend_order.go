// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"
	"sort"

	"oblikovati.org/model/sheetmetal"
)

// Bend-order annotation (M13-F06, #809). The part's bends carry a press-brake sequence: the
// order they are folded in. The order is an overlay of bend feature names on the bend lineage
// (compdef.Bends), editable and persisted. An empty overlay is the natural creation order;
// bends the overlay omits keep their natural order after the listed ones.

// OrderedBends returns the part's bends in their press-brake sequence (the bend-order overlay
// applied to the natural lineage). The caller numbers them 1..N by position.
func (d *PartComponentDefinition) OrderedBends() []sheetmetal.Bend {
	bends := d.Bends()
	if len(d.bendOrder) == 0 {
		return bends
	}
	rank := make(map[string]int, len(d.bendOrder))
	for i, name := range d.bendOrder {
		rank[name] = i
	}
	sort.SliceStable(bends, func(i, j int) bool {
		ri, iok := rank[bends[i].Feature]
		rj, jok := rank[bends[j].Feature]
		if iok && jok {
			return ri < rj
		}
		return iok && !jok // a listed bend precedes an unlisted one; otherwise keep natural order
	})
	return bends
}

// SetBendOrder sets the press-brake sequence from a list of bend feature names, erroring on a
// name that is not a current bend (so a stale order can't silently mis-sequence the brake). An
// empty list resets to the natural order.
func (d *PartComponentDefinition) SetBendOrder(order []string) error {
	known := make(map[string]bool)
	for _, b := range d.Bends() {
		known[b.Feature] = true
	}
	for _, name := range order {
		if !known[name] {
			return fmt.Errorf("bend order: %q is not a bend of this part", name)
		}
	}
	d.bendOrder = append([]string(nil), order...)
	return nil
}
