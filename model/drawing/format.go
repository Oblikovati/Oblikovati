// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
)

// SheetSpec describes a sheet to create: a standard Size (or types.SheetSizeCustom
// with explicit WidthMM/HeightMM), an Orientation, and an optional Name ("" ⇒ the
// collection auto-assigns the next "Sheet:N").
type SheetSpec struct {
	Name        string
	Size        types.SheetSize
	Orientation types.SheetOrientation
	WidthMM     float64
	HeightMM    float64
}

// laidOutDimsMM resolves a spec's width and height in millimetres with orientation
// applied. A standard size reads its portrait dimensions from types.SheetDimensionsMM;
// a custom size uses the spec's explicit dimensions (which must be positive). Landscape
// swaps width and height.
func laidOutDimsMM(size types.SheetSize, orient types.SheetOrientation, customW, customH float64) (float64, float64, error) {
	w, h, ok := types.SheetDimensionsMM(size)
	if !ok {
		// A custom size's explicit dimensions are final; orientation is not applied
		// (the caller already gave the width and height it wants).
		if customW <= 0 || customH <= 0 {
			return 0, 0, fmt.Errorf("drawing: custom sheet needs positive width and height, got %g×%g mm", customW, customH)
		}
		return customW, customH, nil
	}
	if orient == types.SheetLandscape {
		w, h = h, w
	}
	return w, h, nil
}
