// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/app/keymap"

// The session half of M05-F17 (#831): the binding engine lives in [Bindings]; this file
// wires its persistence, mirroring UseOptionsStore. The head installs a FileStore at
// startup so rebinds and custom aliases survive across runs.

// UseKeymapStore installs persistence for the keyboard customization and loads the
// stored overlay into the binding engine.
func (s *Session) UseKeymapStore(store keymap.Store) error {
	loaded, err := store.Load()
	if err != nil {
		return err
	}
	b := s.Bindings()
	b.store = store
	b.custom = loaded
	return nil
}
