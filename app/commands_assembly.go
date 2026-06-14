// SPDX-License-Identifier: GPL-2.0-only

package app

// The Assemble ribbon tab (M11), shown for an assembly document (the AssemblyRibbon key,
// switched in by ribbonKeyForDocument). It carries the Component panel (Place, #763), the Pattern
// panel (Rectangular/Circular Pattern, Mirror, Copy, #765), and the BOM panel (Bill of Materials,
// #768). The assembly model + wire surface already exist — these commands are the head/app wiring
// that drives them.

// assemblyTabCommands returns the Assemble tab commands in ribbon order, grouped into panels
// by their category (Component / Pattern / BOM).
func assemblyTabCommands() []*CommandDefinition {
	return []*CommandDefinition{
		placeComponentCommand(),
		assemblyToolCommand("Assembly.RectangularPattern", "Rectangular Pattern", "Pattern", "rectangular-pattern",
			"Rectangular Pattern — replicate the selected component on a grid (counts and spacing).",
			func() Tool { return NewAssemblyRectPatternTool() }),
		assemblyToolCommand("Assembly.CircularPattern", "Circular Pattern", "Pattern", "circular-pattern",
			"Circular Pattern — replicate the selected component around the Z axis (count and angle).",
			func() Tool { return NewAssemblyCircPatternTool() }),
		assemblyToolCommand("Assembly.Mirror", "Mirror", "Pattern", "mirror",
			"Mirror — add a mirror of the selected components across a plane.",
			func() Tool { return NewAssemblyMirrorTool() }),
		assemblyToolCommand("Assembly.Chamfer", "Chamfer", "Modify", "chamfer",
			"Chamfer — bevel a picked component edge on every instance of that component.",
			func() Tool { return NewAssemblyChamferTool() }),
		assemblyToolCommand("Assembly.Fillet", "Fillet", "Modify", "fillet",
			"Fillet — round a picked component edge on every instance of that component.",
			func() Tool { return NewAssemblyFilletTool() }),
		NewCommand("Assembly.Copy", "Copy", "Pattern", func(s *Session) error { return s.CopyComponents() }).
			WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
			WithIcon("copy").WithButtonStyle(SmallIconButton).
			WithTooltip("Copy — add an independent copy of each selected component."),
		NewCommand("Assembly.BOM", "Bill of Materials", "BOM", func(s *Session) error {
			if _, err := activeAssembly(s); err != nil {
				return err
			}
			s.OpenBOM()
			return nil
		}).WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
			WithIcon("bom").WithButtonStyle(LargeIconButton).
			WithTooltip("Bill of Materials — list the assembly's components (structured or parts-only) and export to CSV."),
	}
}

// assemblyToolCommand builds an Assemble-tab command that starts a replication tool on the active
// assembly. The tool seeds its sources from the current selection and reads the rest from the
// generic tool-param dialog (#765).
func assemblyToolCommand(id, name, panel, icon, tooltip string, newTool func() Tool) *CommandDefinition {
	return NewCommand(id, name, panel, func(s *Session) error {
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
