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

// SheetFormat is a reusable sheet template (#1989): a size/orientation, an optional zoned border
// (HZones×VZones with the two label modes; 0 zones ⇒ the plain default border) and the title-block
// corner. A drawing defines named formats once and stamps new sheets from them.
type SheetFormat struct {
	Name               string
	Size               types.SheetSize
	Orientation        types.SheetOrientation
	WidthMM, HeightMM  float64
	HZones, VZones     int
	HLabelMode         types.BorderLabelMode
	VLabelMode         types.BorderLabelMode
	TitleBlockLocation types.TitleBlockLocation
}

// DefineFormat registers a reusable sheet format under its name (replacing any prior format of that
// name). It errors on an empty name (#1989).
func (s *Sheets) DefineFormat(f SheetFormat) error {
	if f.Name == "" {
		return fmt.Errorf("drawing: a sheet format needs a name")
	}
	if s.formats == nil {
		s.formats = map[string]SheetFormat{}
	}
	s.formats[f.Name] = f
	return nil
}

// AddUsingFormat adds a sheet named sheetName stamped from the registered format formatName: its
// size/orientation, zoned border and title-block corner (#1989). It errors on an unknown format.
func (s *Sheets) AddUsingFormat(sheetName, formatName string) (*Sheet, error) {
	f, ok := s.formats[formatName]
	if !ok {
		return nil, fmt.Errorf("drawing: no sheet format named %q", formatName)
	}
	sh, err := s.Add(SheetSpec{Name: sheetName, Size: f.Size, Orientation: f.Orientation, WidthMM: f.WidthMM, HeightMM: f.HeightMM})
	if err != nil {
		return nil, err
	}
	if f.HZones > 0 && f.VZones > 0 {
		if err := sh.SetZonedBorder(f.HZones, f.VZones, f.HLabelMode, f.VLabelMode); err != nil {
			return nil, err
		}
	}
	sh.SetTitleBlockLocation(f.TitleBlockLocation)
	return sh, nil
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
