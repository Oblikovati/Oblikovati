// SPDX-License-Identifier: GPL-2.0-only

package app

// displayViewCommands are the View tab's display-settings toggles (M16-F07 #643): the ground
// plane visibility (the colored shadow-catching floor). The edge-color and the rest of the
// document display settings are reachable through the API today; this is the discoverable
// ribbon toggle for the most visible one.
func displayViewCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("View.GroundPlane", "Ground Plane", "Navigate", func(s *Session) error {
			s.SetGroundPlaneVisible(!s.GroundPlaneVisible())
			return nil
		}).WithTab("View").WithEnable(hasActivePart).WithIcon("home").WithButtonStyle(SmallIconButton).
			WithActive(func(s *Session) bool { return s.GroundPlaneVisible() }).
			WithTooltip("Ground Plane — show or hide the shadow-catching ground (View ▸ display settings)."),
	}
}
