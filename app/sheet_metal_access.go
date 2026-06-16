// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/compdef"
)

// Sheet-metal UI access (M13). The Sheet Metal ribbon tab and its tools act on the active
// part only when it is in the sheet-metal environment (created with the sheet-metal subtype,
// which seeds the rule). These mirror [hasActivePart]/[activePart] for that gate.

// hasActiveSheetMetalPart reports whether the active document is a sheet-metal part — the
// enable predicate for the Sheet Metal tab's commands.
func hasActiveSheetMetalPart(s *Session) bool {
	p, err := activePart(s)
	return err == nil && p.IsSheetMetal()
}

// activeSheetMetalPart returns the active sheet-metal part, erroring when the active document
// is not a part or is an ordinary (non-sheet-metal) part.
func activeSheetMetalPart(s *Session) (*compdef.PartComponentDefinition, error) {
	p, err := activePart(s)
	if err != nil {
		return nil, err
	}
	if !p.IsSheetMetal() {
		return nil, errors.New("the active part is not a sheet-metal part")
	}
	return p, nil
}
