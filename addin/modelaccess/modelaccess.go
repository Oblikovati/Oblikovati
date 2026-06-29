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

// ActiveParameterHolder returns the active document's content as a parameter holder — a part
// OR an assembly — or an error if there is no active document or it holds no parameters (a
// drawing, say). The derived-parameter-table wire surface resolves its target through this so
// it is not part-only (M39-F02, #1558).
func ActiveParameterHolder(s *app.Session) (compdef.ParameterHolder, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, ErrNoActiveDocument
	}
	holder, ok := d.Content().(compdef.ParameterHolder)
	if !ok {
		return nil, fmt.Errorf("modelaccess: active document %q holds no parameters (not a part or assembly)", d.DisplayName())
	}
	return holder, nil
}

// ActiveAssembly returns the active document's assembly component definition, or an
// error if there is no active document or it is not an assembly. The assembly-feature
// and occurrence surfaces resolve their target through this.
func ActiveAssembly(s *app.Session) (*compdef.AssemblyComponentDefinition, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, ErrNoActiveDocument
	}
	asm, ok := d.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		return nil, fmt.Errorf("modelaccess: active document %q is not an assembly", d.DisplayName())
	}
	return asm, nil
}
