// SPDX-License-Identifier: GPL-2.0-only

package app

// colorStylesCommand opens the Color Styles panel — assign a document color style to the
// selected body so it renders in that style's color (M16-F02 #403/#408).
func colorStylesCommand() *CommandDefinition {
	return NewCommand("View.ColorStyles", "Color Styles…", "Appearance", func(s *Session) error {
		s.OpenColorStylesPanel()
		return nil
	}).WithTab("View").WithEnable(hasActivePart).WithIcon("decal").WithButtonStyle(SmallIconButton).
		WithActive(func(s *Session) bool { return s.ColorStylesPanelOpen() }).
		WithTooltip("Color Styles — assign a color style to the selected body.")
}
