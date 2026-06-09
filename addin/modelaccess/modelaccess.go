// SPDX-License-Identifier: GPL-2.0-only

// Package modelaccess holds the small shared helpers the router and the operation
// registry both use to reach the active model from a session, so neither duplicates
// the "is there an active part?" logic.
package modelaccess

import (
	"errors"
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// ErrNoActiveDocument is returned when an operation needs a document but none is open.
var ErrNoActiveDocument = errors.New("modelaccess: no active document")

// ActivePart returns the active document's part component definition, or an error if
// there is no active document or it is not a part. This is the realized compdef
// (set via SetContent), not the doc-package placeholder ws.Add installs.
func ActivePart(s *app.Session) (*compdef.PartComponentDefinition, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, ErrNoActiveDocument
	}
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return nil, fmt.Errorf("modelaccess: active document %q is not a part", d.DisplayName())
	}
	return part, nil
}
