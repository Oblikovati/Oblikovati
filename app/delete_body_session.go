// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/topo"
)

// Deleting a solid body from the browser (#2046).

// DeleteBody appends a Delete Body feature removing b from the active part — the browser body
// node's Delete. It goes through the recipe rather than dropping the body in place, so the
// deletion is undoable, suppressible and reorderable (#2046).
//
//	if err := s.DeleteBody(handle.Body); err != nil { ... }
func (s *Session) DeleteBody(b *topo.Body) error {
	if b == nil {
		return errors.New("app: DeleteBody: nil body")
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	pf := part.Features().AddDeleteBody(b.ReferenceKey())
	part.Recompute()
	s.recordEdit(part, "Delete Body")
	if !pf.Health().OK() {
		return errors.New("delete body: " + pf.Health().Reason)
	}
	s.selection.Clear() // the deleted body's handle would dangle
	return nil
}
