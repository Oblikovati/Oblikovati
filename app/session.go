// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/event"
	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/renderer"
	"github.com/Oblikovati/oblikovati/scene"
)

// Session is the running application state and the seam tests drive synthetically.
// It owns the open documents ([doc.Workspace]), the command registry, the selection,
// the active interactive tool, and the event bus. A test (or the ImGui shell) drives
// it through Execute / the input methods (Click, PressKey…) — there is no GPU or
// window involved, so "operating the UI" is fully unit-testable (ADR-0014/0004).
type Session struct {
	workspace       *doc.Workspace
	commands        *CommandManager
	bus             *event.Bus
	selection       *Selection
	tool            *ToolInstance
	picker          Picker
	camera          scene.Camera
	camTween        cameraTween
	sketchReturnCam scene.Camera
	activeSketch    *sketch.Sketch
	overlays        []renderer.DrawItem
	addins          *AddInManager
	grid            *GridSettings
}

// NewSession creates an empty in-memory session.
func NewSession() *Session {
	return &Session{
		workspace: doc.NewWorkspace(nil),
		commands:  NewCommandManager(),
		bus:       event.NewBus(),
		selection: NewSelection(),
		camera:    scene.NewCamera(800, 600),
		addins:    NewAddInManager(),
	}
}

// AddIns returns the add-in registry (ApplicationAddIns).
func (s *Session) AddIns() *AddInManager { return s.addins }

// Camera returns the viewport camera (used by picking and sketch tools); SetCamera
// updates it (the window feeds orbit/zoom in production) and keeps the picker's view in
// sync so a click hit-tests against the current camera.
func (s *Session) Camera() scene.Camera { return s.camera }

func (s *Session) SetCamera(c scene.Camera) {
	s.camera = c
	if ca, ok := s.picker.(interface{ SetCamera(scene.Camera) }); ok {
		ca.SetCamera(c)
	}
}

// EnterSketch activates a sketch for editing (the Sketch environment); ExitSketch
// leaves it. ActiveSketch returns the sketch being edited, or nil. Entering swings the
// camera to face the sketch plane head-on (remembering the prior view); exiting swings
// it back — the head ticks these transitions (TickCameraAnimation).
func (s *Session) EnterSketch(sk *sketch.Sketch) {
	s.activeSketch = sk
	sk.Edit()
	s.sketchReturnCam = s.camera
	s.animateCameraTo(s.camera.Facing(
		sk.Plane().Origin(), sk.Plane().Normal().AsVector(), sk.Plane().YAxis().AsVector(),
	),
		sketchViewTweenSeconds)
}

func (s *Session) ExitSketch() {
	if s.activeSketch != nil {
		s.activeSketch.ExitEdit()
		s.activeSketch = nil
		s.animateCameraTo(s.sketchReturnCam, sketchViewTweenSeconds)
	}
}
func (s *Session) ActiveSketch() *sketch.Sketch { return s.activeSketch }

// Workspace/Commands/Events/Selection expose the session's subsystems.
func (s *Session) Workspace() *doc.Workspace { return s.workspace }
func (s *Session) Commands() *CommandManager { return s.commands }
func (s *Session) Events() *event.Bus        { return s.bus }
func (s *Session) Selection() *Selection     { return s.selection }

// ActiveDocument returns the workspace's active document, or nil.
func (s *Session) ActiveDocument() *doc.Document { return s.workspace.ActiveDocument() }

// Execute runs the command with the given id, firing CommandStarted (Before) and
// CommandEnded (After) and refusing a disabled command.
func (s *Session) Execute(id string) error {
	c, ok := s.commands.ByID(id)
	if !ok {
		return fmt.Errorf("app: no command %q", id)
	}
	if !c.IsEnabled(s) {
		return fmt.Errorf("app: command %q is disabled", id)
	}
	event.Emit(s.bus, event.Before, CommandStarted{ID: id})
	err := c.run(s)
	event.Emit(s.bus, event.After, CommandEnded{ID: id, Failed: err != nil})
	return err
}

// Invoke is the alias-driven entry (Inventor command alias): typing a command alias
// runs the matching command.
func (s *Session) Invoke(alias string) error {
	c, ok := s.commands.ByAlias(alias)
	if !ok {
		return fmt.Errorf("app: no command for alias %q", alias)
	}
	return s.Execute(c.id)
}
