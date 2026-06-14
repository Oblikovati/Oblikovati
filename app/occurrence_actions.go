// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/occurrence"
)

// Session actions invoked from the assembly browser's right-click menu (#764): ground, suppress,
// and delete a placed component occurrence. Each mutates the active assembly through the model's
// vetoable operations (so an add-in can refuse one), recomputes when the change affects geometry,
// and records an undo step — the occurrence counterpart of the part browser actions in
// app/browser_actions.go. A vetoed change surfaces its reason as the returned error, leaving the
// occurrence unchanged.

// SuppressOccurrence sets o's suppression and recomputes so an unsuppressed feature drops the
// occurrence's body from its participants and the range box updates. A no-op toggle records
// nothing.
func (s *Session) SuppressOccurrence(o *occurrence.Occurrence, suppressed bool) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	if o == nil {
		return errors.New("app: SuppressOccurrence with nil occurrence")
	}
	if o.Suppressed() == suppressed {
		return nil
	}
	if err := asm.SetOccurrenceSuppressed(o, suppressed); err != nil {
		return err
	}
	asm.RecomputeFeatures()
	s.recordEdit(asm, "Suppress Component")
	return nil
}

// ToggleOccurrenceSuppressed flips o's suppression (the browser's Suppress/Unsuppress entry).
func (s *Session) ToggleOccurrenceSuppressed(o *occurrence.Occurrence) error {
	if o == nil {
		return errors.New("app: ToggleOccurrenceSuppressed with nil occurrence")
	}
	return s.SuppressOccurrence(o, !o.Suppressed())
}

// GroundOccurrence fixes (grounded=true) or releases o in assembly space. Grounding is a
// constraint flag with no geometry effect on its own (the solver consumes it in M12), so it
// records an undo step without recomputing. A no-op toggle records nothing.
func (s *Session) GroundOccurrence(o *occurrence.Occurrence, grounded bool) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	if o == nil {
		return errors.New("app: GroundOccurrence with nil occurrence")
	}
	if o.Grounded() == grounded {
		return nil
	}
	o.SetGrounded(grounded)
	s.recordEdit(asm, "Ground Component")
	return nil
}

// ToggleOccurrenceGrounded flips o's grounded flag (the browser's Ground/Unground entry).
func (s *Session) ToggleOccurrenceGrounded(o *occurrence.Occurrence) error {
	if o == nil {
		return errors.New("app: ToggleOccurrenceGrounded with nil occurrence")
	}
	return s.GroundOccurrence(o, !o.Grounded())
}

// DeleteOccurrence removes o from the active assembly, clears the selection (the deleted node no
// longer exists at its old identity), recomputes, and records an undo step. Undo restores the
// occurrence through the assembly recipe and rebinds its component reference.
func (s *Session) DeleteOccurrence(o *occurrence.Occurrence) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	if o == nil {
		return errors.New("app: DeleteOccurrence with nil occurrence")
	}
	if err := asm.DeleteOccurrence(o); err != nil {
		return err
	}
	s.selection.Clear()
	asm.RecomputeFeatures()
	s.recordEdit(asm, "Delete Component")
	return nil
}
