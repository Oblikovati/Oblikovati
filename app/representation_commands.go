// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/assembly"
)

// The Representations panel (M12-F04, Oblikovati/Oblikovati#361/#367) on the Assemble tab:
// capture a representation of each family (snapshotting the current scene) and create a model
// state from the active representations. Activation is by double-click / context menu on the
// browser's Representations and Model States folders.

// representationCommands returns the Representations-panel commands.
func representationCommands() []*CommandDefinition {
	return []*CommandDefinition{
		repCommand("Assembly.CaptureView", "Capture View", "rep-view",
			"Capture a design-view representation — the current visibility, appearance, sections, and camera.", (*Session).CaptureDesignView),
		repCommand("Assembly.CapturePosition", "Capture Position", "rep-position",
			"Capture a positional representation — the current constraint/joint values.", (*Session).CapturePositional),
		repCommand("Assembly.CaptureLOD", "Capture LOD", "rep-lod",
			"Capture a level-of-detail representation — the current component suppression.", (*Session).CaptureLOD),
		repCommand("Assembly.NewModelState", "Model State", "model-state",
			"Create a model state from the active representations (switches all three families together).", (*Session).NewModelState),
	}
}

// repCommand builds a Representations-panel command that runs run on an active assembly.
func repCommand(id, name, icon, tooltip string, run func(*Session) error) *CommandDefinition {
	return NewCommand(id, name, "Representations", run).
		WithTab("Assemble").WithRibbons(AssemblyRibbon).WithEnable(hasActiveAssembly).
		WithIcon(icon).WithButtonStyle(CompactIconButton).WithTooltip(tooltip)
}

// CaptureDesignView captures the current visibility/appearance/section state and camera into a
// new design-view representation.
func (s *Session) CaptureDesignView() error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	asm.Representations().CaptureDesignView("", s.capturedCamera())
	s.recordEdit(asm, "Capture View Representation")
	return nil
}

// CapturePositional captures the current constraint values into a new positional representation.
func (s *Session) CapturePositional() error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	asm.Representations().CapturePositional("")
	s.recordEdit(asm, "Capture Positional Representation")
	return nil
}

// CaptureLOD captures the current suppression state into a new level-of-detail representation.
func (s *Session) CaptureLOD() error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	asm.Representations().CaptureLOD("")
	s.recordEdit(asm, "Capture LOD Representation")
	return nil
}

// NewModelState creates a model state selecting the currently-active representation of each
// family — the single switch the user flips to recall that combination.
func (s *Session) NewModelState() error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	d, p, l := asm.Representations().ActiveSelection()
	asm.Representations().CreateModelState("", d, p, l)
	s.recordEdit(asm, "New Model State")
	return nil
}

// ActivateRepresentation applies a representation selected in the browser, dispatching to its
// family's collection (M12-F04).
func (s *Session) ActivateRepresentation(h RepresentationHandle) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	reps := asm.Representations()
	switch h.Family {
	case types.RepresentationDesignView:
		_, err = reps.ActivateDesignView(h.ID)
	case types.RepresentationPositional:
		_, err = reps.ActivatePositional(h.ID)
	case types.RepresentationLevelOfDetail:
		_, err = reps.ActivateLOD(h.ID)
	}
	if err != nil {
		return err
	}
	s.recordEdit(asm, "Activate Representation")
	return nil
}

// ActivateModelState switches the assembly to a model state selected in the browser, activating
// its three representations together.
func (s *Session) ActivateModelState(h ModelStateHandle) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	if _, err := asm.Representations().ActivateModelState(h.ID); err != nil {
		return err
	}
	s.recordEdit(asm, "Activate Model State")
	return nil
}

// DeleteRepresentation removes a representation selected in the browser, dispatching to its
// family's collection.
func (s *Session) DeleteRepresentation(h RepresentationHandle) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	reps := asm.Representations()
	switch h.Family {
	case types.RepresentationDesignView:
		reps.DeleteDesignView(h.ID)
	case types.RepresentationPositional:
		reps.DeletePositional(h.ID)
	case types.RepresentationLevelOfDetail:
		reps.DeleteLOD(h.ID)
	}
	s.recordEdit(asm, "Delete Representation")
	return nil
}

// DeleteModelState removes a model state selected in the browser.
func (s *Session) DeleteModelState(h ModelStateHandle) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	asm.Representations().DeleteModelState(h.ID)
	s.recordEdit(asm, "Delete Model State")
	return nil
}

// capturedCamera snapshots the session's current camera for a design-view capture.
func (s *Session) capturedCamera() *assembly.CapturedCamera {
	c := s.Camera()
	return &assembly.CapturedCamera{Eye: c.Eye, Target: c.Target, Up: c.Up, FOV: c.FOV, Orthographic: c.Orthographic}
}
