// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app/markingmenu"
)

// The session half of marking-menu persistence: wires a Store so customizations
// and the classic/radial style toggle survive across sessions.

// UseMarkingMenuStore installs persistence for the marking-menu customization,
// loading the stored menus over the session's defaults and the stored style flag.
func (s *Session) UseMarkingMenuStore(store markingmenu.Store) error {
	loaded, err := store.Load()
	if err != nil {
		return err
	}
	s.markingMenuStore = store
	if len(loaded.Menus) > 0 {
		menus, _ := markingmenu.ApplyToMenus(loaded)
		for env, menu := range menus {
			s.markingMenus[env] = menu
		}
	}
	s.classicContextMenu = loaded.Classic
	return nil
}

// saveMarkingMenuCustomization persists the current marking-menu state when a store
// is wired. Called by SetMarkingMenu, SetClassicContextMenu, and ToggleContextMenuStyle.
func (s *Session) saveMarkingMenuCustomization() {
	if s.markingMenuStore == nil {
		return
	}
	_ = s.markingMenuStore.Save(markingmenu.ToCustomization(s.markingMenus, s.classicContextMenu))
}

// OpenMarkingMenuEditor / CloseMarkingMenuEditor / MarkingMenuEditorOpen drive the
// Tools ▸ Customize Marking Menu panel's visibility.
func (s *Session) OpenMarkingMenuEditor()      { s.markingMenuEditorOpen = true }
func (s *Session) CloseMarkingMenuEditor()     { s.markingMenuEditorOpen = false }
func (s *Session) MarkingMenuEditorOpen() bool { return s.markingMenuEditorOpen }

// ResetMarkingMenu resets one environment's radial menu to the enriched defaults and
// persists the change.
func (s *Session) ResetMarkingMenu(env Environment) {
	defaults := defaultMarkingMenus()
	if menu, ok := defaults[env]; ok {
		s.markingMenus[env] = menu
	} else {
		s.markingMenus[env] = wire.MarkingMenuView{Environment: env}
	}
	s.saveMarkingMenuCustomization()
}
