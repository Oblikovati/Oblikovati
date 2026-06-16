// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// activeSheetMetalPart returns the active part, erroring (named by op) when there is none or
// it is not in the sheet-metal environment. Shared by every sheetMetal* operation so the
// "is this a sheet-metal part?" guard lives in one place.
func activeSheetMetalPart(s *app.Session, op string) (*compdef.PartComponentDefinition, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	if !part.IsSheetMetal() {
		return nil, fmt.Errorf("%s: the active part is not a sheet-metal part", op)
	}
	return part, nil
}
