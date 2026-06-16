// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/sheetmetal"
)

// Flat-pattern cosmetic centerlines (M13-F06, #809). A cosmetic centerline is a manufacturing
// annotation line on the flat pattern (e.g. a centerline through a hole pattern). They are
// part state — added, listed, deleted, and persisted — not geometry features.

// AddCosmeticCenterline appends a cosmetic centerline (a line from start to end in flat 2D)
// and returns its index.
func (d *PartComponentDefinition) AddCosmeticCenterline(start, end gmath.Point2) int {
	d.centerlines = append(d.centerlines, sheetmetal.CosmeticCenterline{Start: start, End: end})
	return len(d.centerlines) - 1
}

// CosmeticCenterlines returns the flat pattern's cosmetic centerlines in creation order.
func (d *PartComponentDefinition) CosmeticCenterlines() []sheetmetal.CosmeticCenterline {
	return d.centerlines
}

// DeleteCosmeticCenterline removes the centerline at index, erroring when it is out of range.
func (d *PartComponentDefinition) DeleteCosmeticCenterline(index int) error {
	if index < 0 || index >= len(d.centerlines) {
		return fmt.Errorf("cosmetic centerline: index %d out of range (have %d)", index, len(d.centerlines))
	}
	d.centerlines = append(d.centerlines[:index], d.centerlines[index+1:]...)
	return nil
}
