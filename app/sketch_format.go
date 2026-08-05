// SPDX-License-Identifier: GPL-2.0-only

package app

// The Format panel (#2015). Its buttons follow one rule: with geometry selected they CONVERT the
// selection; with nothing selected they arm a CREATION MODE for what is drawn next — the
// behaviour lives in sketch_format_modes.go, this file only registers the commands.
//
// The panel appears on both the Sketch and 3D Sketch tabs, but not identically: a centerline is a
// revolve/mirror axis and a centre point is a hole-centre marker on a planar sketch, so neither
// has meaning in a 3D sketch and neither is registered there.

// formatButton describes one Format-panel toggle, so the 2D and 3D registrations are built from
// one list rather than written twice.
type formatButton struct {
	id      string
	name    string
	icon    string
	tooltip string
	run     func(*Session)
	// active reports the toggle's current state so the ribbon draws it highlighted. Without it
	// an armed creation mode looked identical to a disarmed one and nothing on screen said the
	// next line would be construction geometry (#2041).
	active func(*Session) bool
	in3D   bool
}

// formatButtons is the panel's toggles in ribbon order. The four that convert-or-arm come first,
// then the display toggle.
func formatButtons() []formatButton {
	return append(formatConvertButtons(), formatButton{
		id: "ShowFormat", name: "Show Format", icon: "show-format",
		tooltip: "Show Format — display the sketch with default attributes, hiding per-entity " +
			"line type, colour and thickness overrides.",
		run: func(s *Session) { s.ToggleShowFormat() }, active: (*Session).ShowFormat, in3D: true,
	})
}

// formatConvertButtons is the four toggles that convert a selection or arm a creation mode.
// Centerline and Center Point are 2D-only: a revolve/mirror axis and a hole-centre marker are
// planar-sketch concepts with no meaning in a 3D sketch.
func formatConvertButtons() []formatButton {
	return []formatButton{
		{
			id: "Construction", name: "Construction", icon: "construction",
			tooltip: "Construction — convert the selected geometry to construction, or with nothing " +
				"selected draw new geometry as construction (excluded from profiles).",
			run: func(s *Session) { s.ToggleConstruction() }, active: (*Session).ConstructionMode, in3D: true,
		},
		{
			id: "DrivenDimension", name: "Driven Dimension", icon: "driven-dimension",
			tooltip: "Driven Dimension — switch the selected dimension between driving the geometry " +
				"and being driven by it, or with nothing selected create new dimensions driven.",
			run: func(s *Session) { s.ToggleDrivenDimension() }, active: (*Session).DrivenDimensionMode, in3D: true,
		},
		{
			id: "Centerline", name: "Centerline", icon: "centerline",
			tooltip: "Centerline — convert the selected line(s) to a centerline axis (revolve/mirror), " +
				"or with nothing selected draw new lines as centerlines.",
			run: func(s *Session) { s.ToggleCenterline() }, active: (*Session).CenterlineMode,
		},
		{
			id: "CenterPoint", name: "Center Point", icon: "center-point",
			tooltip: "Center Point — convert the selected point(s) to hole-centre markers, or with " +
				"nothing selected place new points as centre points.",
			run: func(s *Session) { s.ToggleCenterPoint() }, active: (*Session).CenterPointMode,
		},
	}
}

// sketchFormatCommands registers the Format panel on the Sketch tab and, for the toggles that
// have meaning there, on the 3D Sketch tab.
func sketchFormatCommands() []*CommandDefinition {
	var cmds []*CommandDefinition
	for _, b := range formatButtons() {
		cmds = append(cmds, formatCommand(b, "Sketch."+b.id, "Sketch", SketchEnvironment, inSketch))
		if b.in3D {
			cmds = append(cmds, formatCommand(b, "Sketch3D."+b.id, tab3DSketch, Sketch3DEnvironment, inSketch3D))
		}
	}
	return append(cmds, sketchFormatListCommands()...)
}

// formatCommand builds one Format-panel button for a given tab and environment.
func formatCommand(b formatButton, id, tab string, env Environment, enable func(*Session) bool) *CommandDefinition {
	run := b.run
	return NewCommand(id, b.name, "Format", func(s *Session) error {
		run(s)
		return nil
	}).WithTab(tab).WithEnvironment(env).WithEnable(enable).WithActive(b.active).
		WithIcon(b.icon).WithButtonStyle(SmallIconButton).WithTooltip(b.tooltip)
}
