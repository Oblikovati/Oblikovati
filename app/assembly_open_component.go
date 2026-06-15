// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/occurrence"
)

// OpenOccurrenceDocument brings a placed component's document forward into a visible, active
// tab — the Edit/Open path (browser double-click or context menu) for an occurrence (#764).
// Placement loads the component in the background with no tab ([OpenComponentForPlacement]);
// this is how the user opens it to edit. The document is already in memory, so this just flips
// it visible and active (or loads it from the store if the assembly was reopened). It errors
// when the occurrence was placed from a bare in-memory definition (no document name).
func (s *Session) OpenOccurrenceDocument(o *occurrence.Occurrence) error {
	name := o.ComponentName()
	if name == "" {
		return fmt.Errorf("occurrence %q has no document to open", o.Name())
	}
	d, err := s.workspace.Open(name, true) // visible + active ⇒ a tab the viewport switches to
	if err != nil {
		return err
	}
	s.documentHistory(d)
	s.loadViewState(d)
	return nil
}
