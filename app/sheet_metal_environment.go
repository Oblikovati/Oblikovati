// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/api/types"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// Entering the sheet-metal environment (M13). A part becomes sheet metal when its rule is
// seeded (compdef.EnableSheetMetal) and the document carries the sheet-metal flavor; the Sheet
// Metal ribbon tab and its tools gate on that (IsSheetMetal). These two actions are the UI
// entry points — convert the part you already have, or start a new one already in the
// environment — so a user is never stuck with the Sheet Metal tools greyed out and no way in.

// ConvertActiveToSheetMetal enters the sheet-metal environment on the active ordinary part: it
// stamps the sheet-metal document flavor and seeds the rule, so the Sheet Metal tab and tools
// light up. A no-op (nil) when the part is already sheet metal; an error when the active
// document is not a part.
func (s *Session) ConvertActiveToSheetMetal() error {
	d := s.ActiveDocument()
	if d == nil {
		return errors.New("convert to sheet metal: no active document")
	}
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return errors.New("convert to sheet metal: the active document is not a part")
	}
	if part.IsSheetMetal() {
		return nil
	}
	if err := s.StampDocumentSubType(d, types.SubTypeSheetMetalPart); err != nil {
		return err
	}
	if _, err := part.EnableSheetMetal(); err != nil {
		return err
	}
	s.recordEdit(part, "Convert to Sheet Metal")
	return nil
}

// NewSheetMetalPart creates a new part already in the sheet-metal environment — the launch
// action for starting a sheet-metal design from scratch (the counterpart of [Session.NewPart]).
func (s *Session) NewSheetMetalPart() (*doc.Document, error) {
	d, err := s.NewPart()
	if err != nil {
		return nil, err
	}
	if err := s.StampDocumentSubType(d, types.SubTypeSheetMetalPart); err != nil {
		return nil, err
	}
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return nil, errors.New("new sheet metal part: content is not a part")
	}
	if _, err := part.EnableSheetMetal(); err != nil {
		return nil, err
	}
	return d, nil
}

// canConvertToSheetMetal enables the Convert command: an active part that is not already sheet
// metal. The one entry point that stays lit on the Sheet Metal tab before conversion.
func canConvertToSheetMetal(s *Session) bool {
	p, err := activePart(s)
	return err == nil && !p.IsSheetMetal()
}
