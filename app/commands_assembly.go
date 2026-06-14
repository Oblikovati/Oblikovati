// SPDX-License-Identifier: GPL-2.0-only

package app

import "fmt"

// The Assemble ribbon tab (M11), shown for an assembly document (the AssemblyRibbon key,
// switched in by ribbonKeyForDocument). This file establishes the tab's panel structure and
// the active-assembly enable gate; each command's behavior lands in its own follow-up issue
// (Pattern/Mirror/Copy #765, BOM #768), which replaces the stub run below with a real tool. The
// assembly model + wire surface already exist — these are head/app wiring. Place (#763) is now
// real: its command starts the Place Component tool.

// assemblyTabCommands returns the Assemble tab commands in ribbon order, grouped into panels
// by their category (Component / Pattern / BOM).
func assemblyTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		placeComponentCommand(),
		assemblyToolCommand("Assembly.RectangularPattern", "Rectangular Pattern", "rectangular-pattern",
			"Rectangular Pattern — replicate the selected component on a grid (counts and spacing).",
			func() Tool { return NewAssemblyRectPatternTool() }),
		assemblyToolCommand("Assembly.CircularPattern", "Circular Pattern", "circular-pattern",
			"Circular Pattern — replicate the selected component around the Z axis (count and angle).",
			func() Tool { return NewAssemblyCircPatternTool() }),
		assemblyToolCommand("Assembly.Mirror", "Mirror", "mirror",
			"Mirror — add a mirror of the selected components across a plane.",
			func() Tool { return NewAssemblyMirrorTool() }),
		NewCommand("Assembly.Copy", "Copy", "Pattern", func(s *Session) error { return s.CopyComponents() }).
			WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
			WithIcon("copy").WithButtonStyle(SmallIconButton).
			WithTooltip("Copy — add an independent copy of each selected component."),
		assemblyStubCommand("Assembly.BOM", "Bill of Materials", "BOM", 768),
	}
}

// assemblyToolCommand builds an Assemble-tab command that starts a replication tool on the active
// assembly. The tool seeds its sources from the current selection and reads the rest from the
// generic tool-param dialog (#765).
func assemblyToolCommand(id, name, icon, tooltip string, newTool func() Tool) *CommandDefinition {
	return NewCommand(id, name, "Pattern", func(s *Session) error {
		if _, err := activeAssembly(s); err != nil {
			return err
		}
		s.StartTool(newTool())
		return nil
	}).WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
		WithIcon(icon).WithButtonStyle(SmallIconButton).WithTooltip(tooltip)
}

// placeComponentCommand builds the Place command: it starts the modal Place Component tool on
// the active assembly. The component file is chosen by the head's file dialog, which feeds the
// running tool through SetPlaceComponentDocument; each ground-plane click then drops an
// instance (#763).
func placeComponentCommand() *CommandDefinition {
	return NewCommand("Assembly.Place", "Place", "Component", runPlaceComponent).
		WithTab("Assemble").
		WithRibbons(AssemblyRibbon).
		WithEnable(hasActiveAssembly).
		WithTooltip("Place a component into the assembly.")
}

// runPlaceComponent starts the Place Component tool on the active assembly. The tool then waits
// for a chosen component (the head's file dialog) and for ground-plane clicks to drop instances.
func runPlaceComponent(s *Session) error {
	if _, err := activeAssembly(s); err != nil {
		return err
	}
	s.StartTool(NewPlaceComponentTool())
	return nil
}

// assemblyStubCommand builds one Assemble-tab command on the AssemblyRibbon, enabled when an
// assembly is active. Its run is a placeholder until the named follow-up issue wires the
// real tool — so the tab renders with the full panel structure now.
func assemblyStubCommand(id, name, panel string, issue int) *CommandDefinition {
	return NewCommand(id, name, panel, assemblyStubRun(name, issue)).
		WithTab("Assemble").
		WithRibbons(AssemblyRibbon).
		WithEnable(hasActiveAssembly).
		WithTooltip(fmt.Sprintf("%s — assembly command (implementation in #%d).", name, issue))
}

// assemblyStubRun reports that the command's behavior is not yet wired, naming its issue.
func assemblyStubRun(name string, issue int) func(*Session) error {
	return func(*Session) error {
		return fmt.Errorf("%s is not yet implemented (#%d)", name, issue)
	}
}
