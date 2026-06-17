// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
)

// propertyLookup resolves a referenced model's iProperty for a title block (set, name
// → value). Title blocks hold this hook rather than a back-pointer to the document, so
// the package stays free of doc/workspace dependencies.
type propertyLookup func(set, name string) (string, bool)

// Sheet is one sheet of a drawing: its size/orientation (which fix its laid-out width
// and height in millimetres), its border, and its title block.
type Sheet struct {
	name        string
	size        types.SheetSize
	orientation types.SheetOrientation
	width       float64 // laid-out mm (orientation applied)
	height      float64
	border      *Border
	titleBlock  *TitleBlock
	views       *DrawingViews
}

// Views returns the sheet's drawing views (projections of the referenced model).
func (s *Sheet) Views() *DrawingViews { return s.views }

// compile-time: a sheet and its sub-objects satisfy the api/contract surface (ADR-0018).
var (
	_ contract.DrawingSheet      = (*Sheet)(nil)
	_ contract.DrawingBorder     = (*Border)(nil)
	_ contract.DrawingTitleBlock = (*TitleBlock)(nil)
)

// Name returns the sheet's display name (unique within the drawing).
func (s *Sheet) Name() string { return s.name }

// Size returns the standard sheet size, or types.SheetSizeCustom for an explicit
// width/height.
func (s *Sheet) Size() types.SheetSize { return s.size }

// Orientation returns how the standard dimensions are laid out (portrait/landscape).
func (s *Sheet) Orientation() types.SheetOrientation { return s.orientation }

// WidthMM and HeightMM return the laid-out dimensions in millimetres.
func (s *Sheet) WidthMM() float64  { return s.width }
func (s *Sheet) HeightMM() float64 { return s.height }

// Border returns the sheet's border as a contract.DrawingBorder, or nil if it has none.
func (s *Sheet) Border() contract.DrawingBorder {
	if s.border == nil {
		return nil
	}
	return s.border
}

// TitleBlock returns the sheet's title block as a contract.DrawingTitleBlock, or nil if
// it has none. Type-assert to *drawing.TitleBlock to enumerate its resolved fields.
func (s *Sheet) TitleBlock() contract.DrawingTitleBlock {
	if s.titleBlock == nil {
		return nil
	}
	return s.titleBlock
}

// Sheets is a drawing's ordered, named sheet collection, tracking which sheet is active.
type Sheets struct {
	items       []*Sheet
	active      int
	lookup      propertyLookup // handed to each new title block; set by the owning Content
	bodyResolve bodyLookup     // handed to each sheet's views; set by the owning Content
}

func newSheets() *Sheets { return &Sheets{} }

// addDefault seeds the default first sheet (A3 landscape, bordered, standard title
// block) — the state a freshly created drawing opens in.
func (s *Sheets) addDefault() {
	// A new drawing always starts with one valid sheet; the default spec cannot error.
	_, _ = s.Add(SheetSpec{Size: types.SheetSizeA3, Orientation: types.SheetLandscape})
}

// Add appends a sheet built from spec, gives it a border and the standard title block,
// and makes it the active sheet. A custom size needs positive WidthMM/HeightMM.
func (s *Sheets) Add(spec SheetSpec) (*Sheet, error) {
	w, h, err := laidOutDimsMM(spec.Size, spec.Orientation, spec.WidthMM, spec.HeightMM)
	if err != nil {
		return nil, err
	}
	name := spec.Name
	if name == "" {
		name = s.nextName()
	}
	if _, exists := s.ByName(name); exists {
		return nil, fmt.Errorf("drawing: sheet %q already exists", name)
	}
	sh := &Sheet{
		name: name, size: spec.Size, orientation: spec.Orientation, width: w, height: h,
		border:     newBorder(DefaultBorderDefinition()),
		titleBlock: newTitleBlock(DefaultTitleBlockDefinition(), s.lookup),
		views:      newDrawingViews(s.bodyResolve),
	}
	s.items = append(s.items, sh)
	s.active = len(s.items) - 1
	return sh, nil
}

// nextName returns the lowest unused "Sheet:N" name.
func (s *Sheets) nextName() string {
	for n := len(s.items) + 1; ; n++ {
		name := fmt.Sprintf("Sheet:%d", n)
		if _, exists := s.ByName(name); !exists {
			return name
		}
	}
}

// Count returns the number of sheets.
func (s *Sheets) Count() int { return len(s.items) }

// Item returns the sheet at index i, or nil if out of range.
func (s *Sheets) Item(i int) *Sheet {
	if i < 0 || i >= len(s.items) {
		return nil
	}
	return s.items[i]
}

// ByName returns the named sheet and whether it exists.
func (s *Sheets) ByName(name string) (*Sheet, bool) {
	for _, sh := range s.items {
		if sh.name == name {
			return sh, true
		}
	}
	return nil, false
}

// Active returns the active sheet, or nil if the drawing has no sheets.
func (s *Sheets) Active() *Sheet { return s.Item(s.active) }

// SetActive makes the named sheet active, erroring if it does not exist.
func (s *Sheets) SetActive(name string) error {
	for i, sh := range s.items {
		if sh.name == name {
			s.active = i
			return nil
		}
	}
	return fmt.Errorf("drawing: no sheet named %q", name)
}

// Remove deletes the named sheet. A drawing must keep at least one sheet, so removing
// the last one errors. The active sheet stays valid (it shifts to a neighbour).
func (s *Sheets) Remove(name string) error {
	if len(s.items) <= 1 {
		return fmt.Errorf("drawing: cannot remove %q — a drawing must keep at least one sheet", name)
	}
	for i, sh := range s.items {
		if sh.name == name {
			s.items = append(s.items[:i], s.items[i+1:]...)
			if s.active >= len(s.items) {
				s.active = len(s.items) - 1
			}
			return nil
		}
	}
	return fmt.Errorf("drawing: no sheet named %q", name)
}
