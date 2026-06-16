// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/api/types"

// orientSpec maps a View-tab orientation command to its standard orientation.
type orientSpec struct {
	id, label string
	to        types.ViewOrientationTypeEnum
}

var orientSpecs = []orientSpec{
	{"View.Orient.Front", "Front", types.FrontViewOrientation},
	{"View.Orient.Back", "Back", types.BackViewOrientation},
	{"View.Orient.Top", "Top", types.TopViewOrientation},
	{"View.Orient.Bottom", "Bottom", types.BottomViewOrientation},
	{"View.Orient.Left", "Left", types.LeftViewOrientation},
	{"View.Orient.Right", "Right", types.RightViewOrientation},
	{"View.Orient.Iso", "Home (Isometric)", types.IsoTopRightViewOrientation},
}

// orientCommand builds one orientation command that jumps the active view to a standard
// orientation, fitting the model.
func orientCommand(sp orientSpec) *CommandDefinition {
	return NewCommand(sp.id, sp.label, "Navigate", func(s *Session) error {
		return s.SetViewOrientation(sp.to, true)
	}).WithTab("View").WithEnable(hasActivePart).
		WithTooltip(sp.label + " — orient the view to this standard direction, framed to fit.")
}

// orientViewCommands are the View tab's Navigate "Orient" split button: a primary that jumps
// to the standard isometric home, with a dropdown of the six face views (M16-F03 #404/#409).
// The ViewCube drives the same orientations interactively; this is the discoverable ribbon
// command equivalent.
func orientViewCommands() []*CommandDefinition {
	variants := make([]*CommandDefinition, 0, len(orientSpecs))
	for _, sp := range orientSpecs {
		variants = append(variants, orientCommand(sp))
	}
	primary := NewCommand("View.Orient", "Orient", "Navigate", func(s *Session) error {
		return s.SetViewOrientation(types.IsoTopRightViewOrientation, true)
	}).WithTab("View").WithEnable(hasActivePart).WithIcon("view-cube").WithButtonStyle(LargeIconButton).
		WithTooltip("Orient — jump to a standard view (front/top/iso…); the dropdown picks the direction.").
		WithVariants(variants...)
	// Only the primary is returned; RegisterStandardCommands registers the variants for id
	// dispatch by walking primary.Variants() (returning them here would double-register).
	return []*CommandDefinition{primary, namedViewsCommand()}
}

// namedViewsCommand opens the Named Views panel — save the current camera under a name and
// restore saved views (M16-F03 #404).
func namedViewsCommand() *CommandDefinition {
	return NewCommand("View.NamedViews", "Named Views…", "Navigate", func(s *Session) error {
		s.OpenNamedViewsPanel()
		return nil
	}).WithTab("View").WithEnable(hasActivePart).WithIcon("home").WithButtonStyle(SmallIconButton).
		WithActive(func(s *Session) bool { return s.NamedViewsPanelOpen() }).
		WithTooltip("Named Views — save the current view under a name and restore it later.")
}
