// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Oblikovati/oblikovati/event"
	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/material"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/renderer"
	"github.com/Oblikovati/oblikovati/scene"
	"github.com/Oblikovati/oblikovati/theme"
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
	pendingDim      *sketch.DimensionConstraint
	featureEdit     *featureEditState
	overlays        []renderer.DrawItem
	addins          *AddInManager
	grid            *GridSettings
	themes          *theme.Library
	themeStore      *theme.Store
	materials       *material.Library
	materialStore   *material.Store
	notice          string               // last user-facing notice (e.g. a failed-commit reason)
	visualStyle     renderer.VisualStyle // how the scene is drawn (View tab's Visual Style)
}

// Notice returns the last user-facing notice (a failed commit's reason), or "" — shown in
// the status bar so a failed OK is not silent.
func (s *Session) Notice() string { return s.notice }

// VisualStyle returns the active scene visual style (default Shaded with Edges).
func (s *Session) VisualStyle() renderer.VisualStyle { return s.visualStyle }

// SetVisualStyle sets how the scene is drawn (the View tab's Visual Style).
func (s *Session) SetVisualStyle(v renderer.VisualStyle) { s.visualStyle = v }

// NewSession creates an empty in-memory session with no persistence store. Its
// documents live only in memory — Save/Open return "no store configured" — which
// is what the unit tests and headless synthetic sessions want. The windowed head
// and the CLI inject a real store via [NewSessionWithStore].
func NewSession() *Session { return newSession(nil) }

// NewSessionWithStore creates a session whose workspace persists through store, so
// File ▸ Open/Save and the CLI fixture commands read and write real .obk packages
// on disk. Pass a persistence.PackageStore from the binary (the app package depends
// only on the doc.Store interface, never on persistence — the DI rule).
func NewSessionWithStore(store doc.Store) *Session { return newSession(store) }

func newSession(store doc.Store) *Session {
	return &Session{
		workspace:   doc.NewWorkspace(store),
		commands:    NewCommandManager(),
		bus:         event.NewBus(),
		selection:   NewSelection(),
		camera:      scene.NewCamera(800, 600),
		addins:      NewAddInManager(),
		visualStyle: renderer.ShadedWithEdges,
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
		s.pendingDim = nil // no dangling edit box after leaving the sketch
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

// ErrNoActiveDoc is returned by the save verbs when there is no active document to
// act on (an empty session).
var ErrNoActiveDoc = errors.New("app: no active document to save")

// ErrNeedsPath is returned by [Session.SaveActiveDocument] when the active document
// has never been saved — its name is not yet an .obk path (a freshly created
// "PartN"). File ▸ Save catches this and falls back to Save As to prompt for a
// destination.
var ErrNeedsPath = errors.New("app: document has no file path yet; use Save As")

// OpenDocument loads the package at path into the workspace and makes it the active,
// visible document. It is the core of File ▸ Open and the CLI open command.
//
//	d, err := session.OpenDocument("/models/bracket.obk")
func (s *Session) OpenDocument(path string) (*doc.Document, error) {
	return s.workspace.Open(path, true)
}

// SaveActiveDocument writes the active document back to its existing .obk path. A
// document that was never saved has no path yet — we detect this by the absence of
// the [doc.PackageExtension] suffix, since new documents are minted with bare names
// like "Part1" — and return [ErrNeedsPath] so the UI can prompt via Save As.
func (s *Session) SaveActiveDocument() error {
	d := s.workspace.ActiveDocument()
	if d == nil {
		return ErrNoActiveDoc
	}
	if !strings.HasSuffix(d.FullFileName(), doc.PackageExtension) {
		return ErrNeedsPath
	}
	return s.workspace.Save(d)
}

// SaveActiveDocumentAs writes the active document to path, which becomes its new
// identity. It is the core of File ▸ Save As and the CLI save-as command.
func (s *Session) SaveActiveDocumentAs(path string) error {
	d := s.workspace.ActiveDocument()
	if d == nil {
		return ErrNoActiveDoc
	}
	return s.workspace.SaveAs(d, path)
}
