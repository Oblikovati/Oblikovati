// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/model/colorscheme"
)

// colorSchemeCommands are the View tab's "Color Scheme" panel: every application color scheme
// (M16-F06 #642) as a mutually-exclusive option of one selection box (Inventor's combo
// control). Selecting one activates it — repainting the viewport background + selection colors;
// the active scheme drives the box's current selection. The option set is the built-in gallery
// (stable ids "View.ColorScheme.<name>").
func colorSchemeCommands() []*CommandDefinition {
	schemes := colorscheme.NewRegistry().Schemes()
	cmds := make([]*CommandDefinition, 0, len(schemes))
	for _, sc := range schemes {
		name := sc.Name
		cmds = append(cmds, NewCommand("View.ColorScheme."+name, name, "Color Scheme", func(s *Session) error {
			return s.SetColorScheme(name)
		}).WithTab("View").WithKind(ComboControl).
			WithTooltip(name+" — the viewport background and highlight/selection colors.").
			WithActive(func(s *Session) bool { return s.ActiveColorScheme().Name == name }))
	}
	return cmds
}
