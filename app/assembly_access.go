// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/compdef"
)

// activeAssembly returns the active document's assembly content, erroring when there is no
// active document or it is not an assembly — the assembly counterpart of [activePart], used
// by the Assemble ribbon commands and assembly tools (M11).
func activeAssembly(s *Session) (*compdef.AssemblyComponentDefinition, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, errors.New("app: no active document")
	}
	asm, ok := d.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		return nil, errors.New("app: active document is not an assembly")
	}
	return asm, nil
}

// hasActiveAssembly reports whether the active document is an assembly — the enable
// predicate for the Assemble ribbon commands (mirrors [hasActivePart]).
func hasActiveAssembly(s *Session) bool {
	asm, _ := activeAssembly(s)
	return asm != nil
}
