// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
)

// Input routing — the methods a viewport (or a test) calls to drive the session.
// They implement Inventor's mouse/keyboard behavior at the logic level; the actual
// device events come from the window in production and from tests headlessly.

// SetPicker installs the hit-test used to resolve clicks to selectables. When the picker
// honours a priority ranking (RayPicker), the live SelectionFilterState ordering is pushed so
// reordering the Selection Filter window changes ambiguous-pick resolution (#1222).
func (s *Session) SetPicker(p Picker) {
	s.picker = p
	if pr, ok := p.(interface {
		SetPriorityRank(func(SelectionKind) int)
	}); ok {
		pr.SetPriorityRank(s.selectionFilterState.Rank)
	}
}

// PartFeatureTool is what a tool that commits a part feature IS: a Tool that can
// always build the draft of the feature it would commit, so the sick-config commit
// gate (commitBlockedReason, #1594) has a draft to inspect BY CONSTRUCTION and can
// never be skipped by omission (#1626, audit I3 — the #1521 bypass shape). Mirrors
// router.MutatingMethod (#1426): the capability is a property of the type, not of a
// hand-maintained list.
type PartFeatureTool interface {
	Tool
	DraftPreviewable
}

// StartFeatureTool activates a part-feature tool. Every activation site of a tool
// that commits a part feature must use this entry point — the compiler then
// guarantees the commit gate has a draft to inspect (#1626). Sketch, work-feature,
// assembly and navigation tools keep the plain StartTool, deliberately outside the
// gate; the activation-seam guard test pins which constructors may do so.
func (s *Session) StartFeatureTool(t PartFeatureTool) { s.StartTool(t) }

// StartTool activates an interactive tool, cancelling any tool already running.
func (s *Session) StartTool(t Tool) {
	if s.tool != nil {
		s.tool.tool.Cancel(s)
		s.Graphics().ClearInteraction() // drop the previous tool's preview/overlay graphics
		s.dropCommandMiniToolbars()     // and its mini-toolbars (M05-F07)
		s.dropCommandGizmos()           // and its triad/manipulators (M05-F13)
	}
	s.notice = ""
	s.tool = &ToolInstance{tool: t}
	t.Start(s)
	s.installToolFilter() // derive the filter from the tool's declared AcceptedKinds (engine)
}

// ActiveTool returns the running tool instance, or nil.
func (s *Session) ActiveTool() *ToolInstance { return s.tool }

// ActiveToolConsumesClicks reports whether the active tool takes raw viewport clicks as input (a
// [PlaneClickTool] — the sketch geometry tools and the point-cloud Crop Box tool). When it does, the
// viewport must route a press to the tool, not start a box-select that would swallow it (#645).
func (s *Session) ActiveToolConsumesClicks() bool {
	if s.tool == nil {
		return false
	}
	_, ok := s.tool.tool.(PlaneClickTool)
	return ok
}

// PickAt hit-tests the pixel through the installed picker without changing selection —
// the viewport uses it for hover feedback (which plane/face is under the cursor).
func (s *Session) PickAt(x, y float64, filter *SelectionFilter) (Selectable, bool) {
	if s.picker == nil {
		return nil, false
	}
	return s.picker.Pick(x, y, filter)
}

// OK finishes the active tool if it has enough input (Inventor's OK), clearing it on
// success. With no active tool it is a no-op error.
func (s *Session) OK() error {
	if s.tool == nil {
		return errors.New("app: no active tool to commit")
	}
	if !s.tool.tool.CanCommit() {
		return errors.New("app: active tool is not ready to commit")
	}
	// A configuration that would recompute SICK must never enter the design: evaluate the pending
	// feature speculatively and refuse the commit if it is invalid, so the tree never gains a sick
	// node (the OK button is likewise disabled while this holds). See commitBlockedReason.
	if reason := s.commitBlockedReason(); reason != "" {
		s.notice = reason
		return errors.New(reason)
	}
	toolName := s.tool.tool.Name()
	if err := s.tool.tool.Commit(s); err != nil {
		s.notice = err.Error() // surface why (the status bar shows it); keep the tool open
		return err
	}
	if fp, ok := s.tool.tool.(featureProducer); ok {
		s.EmitFeatureLifecycle(FeatureAdded, fp.AddedFeature()) // featureAdded for UI-driven creation (#1085)
	}
	// In-sketch tool commits (geometry creation, constraints, dimensions, 3D includes) each become
	// their own undo step, so Ctrl+Z reverts the last sketch operation while editing (#1270). The
	// recipe no-op guard makes a non-mutating commit record nothing.
	if s.InSketch() || s.InSketch3D() {
		s.RecordActiveEdit(toolName)
	}
	s.finishToolCommit()
	return nil
}

// finishToolCommit tears down the active tool after a successful commit: clear the notice, drop the
// tool, and retire its transient UI (selection filter, preview graphics, mini-toolbars, gizmos).
func (s *Session) finishToolCommit() {
	s.notice = ""
	s.tool = nil
	s.restoreSelectionFilter()      // hand selection back to the ambient filter (engine)
	s.Graphics().ClearInteraction() // a committed command's transient preview vanishes
	s.dropCommandMiniToolbars()     // command-bound mini-toolbars die with the tool (M05-F07)
	s.dropCommandGizmos()           // and the command-bound triad/manipulators (M05-F13)
}

// chainingTool is a tool whose Escape ENDS an in-progress chain — keeping the geometry already
// placed — rather than abandoning it. The continuous line tool is the case (#2024).
type chainingTool interface {
	FinishesOnCancel() bool
}

// activeChainingTool returns the active tool when it ends its chain on Escape.
func (s *Session) activeChainingTool() (chainingTool, bool) {
	if s.tool == nil {
		return nil, false
	}
	ct, ok := s.tool.tool.(chainingTool)
	return ct, ok
}

// CancelTool abandons the active tool (Inventor's Escape / Cancel), except for a chaining tool
// mid-chain, which finishes instead. That path goes through OK so the commit is recorded as an
// undo step and the tool is torn down exactly as a normal finish would be.
func (s *Session) CancelTool() {
	s.notice = ""
	if ct, ok := s.activeChainingTool(); ok && ct.FinishesOnCancel() && s.tool.tool.CanCommit() {
		_ = s.OK()
		return
	}
	if s.tool != nil {
		s.tool.tool.Cancel(s)
		s.tool = nil
		s.restoreSelectionFilter()      // hand selection back to the ambient filter (engine)
		s.Graphics().ClearInteraction() // a cancelled command's transient preview vanishes
		s.dropCommandMiniToolbars()     // command-bound mini-toolbars die with the tool (M05-F07)
		s.dropCommandGizmos()           // and the command-bound triad/manipulators (M05-F13)
	}
}

// autoCommitter is a Tool that should finish as soon as a pick makes it ready, rather
// than waiting for a separate OK — e.g. Create 2D Sketch enters the sketch the moment a
// plane is clicked.
type autoCommitter interface {
	AutoCommitOnPick() bool
}

// feedPick hands a picked selectable to the active tool and, when the tool opts into
// auto-commit and is now ready, finishes it immediately (Inventor's click-to-proceed).
func (s *Session) feedPick(sel Selectable) {
	s.tool.tool.Pick(s, sel)
	s.autoCommitAfterPick()
	if s.tool != nil {
		s.installToolFilter() // re-derive the filter for the tool's next step
	}
}

// modifierPicker is a Tool whose pick behavior depends on held modifiers — e.g. Extrude
// adds a region to its set on Ctrl+click rather than replacing the single selection.
type modifierPicker interface {
	PickWithMods(*Session, Selectable, Modifier)
}

// feedPickMods is feedPick for graphics clicks that carry modifiers: a modifier-aware
// tool sees the held keys (Ctrl to extend a multi-selection); others get the plain pick.
func (s *Session) feedPickMods(sel Selectable, mods Modifier) {
	if mp, ok := s.tool.tool.(modifierPicker); ok {
		mp.PickWithMods(s, sel, mods)
	} else {
		s.tool.tool.Pick(s, sel)
	}
	s.autoCommitAfterPick()
	if s.tool != nil {
		s.installToolFilter() // re-derive the filter for the tool's next step
	}
}

// autoCommitAfterPick commits the active tool when it opts into auto-commit and is now
// ready (used after both 3D picks and snap-aware sketch-entity picks).
func (s *Session) autoCommitAfterPick() {
	if ac, ok := s.tool.tool.(autoCommitter); ok && ac.AutoCommitOnPick() && s.tool.tool.CanCommit() {
		_ = s.OK()
	}
}

// Select replaces the selection with a single selectable and emits SelectionChanged
// (the entry point for browser-node and graphics selection outside a tool).
func (s *Session) Select(sel Selectable) {
	s.selection.Clear()
	if sel != nil && s.selection.Add(sel) {
		event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
	}
}

// SelectBrowserNode acts on a clicked browser node: while a tool is active the node's
// entity is fed to it as a pick (so clicking "XY Plane" in the tree picks it for the
// Create Sketch tool); otherwise it becomes the selection (e.g. to pre-select a plane).
func (s *Session) SelectBrowserNode(n BrowserNode) {
	if n.Select == nil {
		return
	}
	if s.tool != nil {
		s.feedPick(n.Select)
		return
	}
	s.Select(n.Select)
}

// Click is a left-button click at a viewport coordinate (the common case). It picks
// the front-most selectable honoring the active filter: while a tool is active the
// pick is fed to the tool; otherwise it joins the selection set.
func (s *Session) Click(x, y float64) {
	s.Pointer(PointerEvent{X: x, Y: y, Button: LeftButton})
}

// Pointer routes a pointer event per Inventor mouse behavior. Left selects/feeds the
// tool; right and middle are reserved for the marking menu and orbit/pan (handled by
// the viewport, no model effect here yet).
func (s *Session) Pointer(e PointerEvent) {
	if e.Button != LeftButton {
		return
	}
	// A sketch tool consumes plane-point clicks directly (not entity picks).
	if s.sketchClick(e.X, e.Y) {
		return
	}
	// In the sketch environment clicks normally pick sketch entities: fed to an active
	// constraint/dimension tool, or (with no tool) added to the selection. The exception is
	// a tool that projects 3D model references (Project Geometry) — its edges/vertices/datums
	// live in the 3D hit-test, so its clicks must run the RayPicker even in-sketch (#1496).
	if s.InSketch() {
		if s.toolPicksModelReferences() {
			s.pickModelReferenceInSketch(e)
			return
		}
		s.sketchEntityPointer(e)
		return
	}
	if s.picker == nil {
		return
	}
	sel, ok := s.picker.Pick(e.X, e.Y, s.pickFilter())
	if !ok {
		s.clearSelectionOnEmptyClick(e.Mods)
		return
	}
	if s.tool != nil {
		s.feedPickMods(sel, e.Mods)
		return
	}
	s.applyPickToSelection(sel, e.Mods)
}

// modelReferencePicker is an in-sketch tool that picks 3D MODEL references — B-rep edges and
// vertices, plus datum geometry — through the viewport rather than the active sketch's own 2D
// entities. Project Geometry is the case: the references it projects into the sketch live in the
// 3D hit-test, so the input router must run the RayPicker while a sketch is being edited (#1496).
type modelReferencePicker interface {
	PicksModelReferences() bool
}

// toolPicksModelReferences reports whether the active tool wants 3D model-reference picks while
// in-sketch, so Pointer routes its clicks to the RayPicker instead of the 2D sketch picker.
func (s *Session) toolPicksModelReferences() bool {
	if s.tool == nil {
		return false
	}
	mr, ok := s.tool.tool.(modelReferencePicker)
	return ok && mr.PicksModelReferences()
}

// pickModelReferenceInSketch runs the 3D hit-test for an in-sketch reference-picking tool and
// feeds any hit (a B-rep edge/vertex or datum) to the tool, so model geometry becomes projectable
// from the viewport while editing a sketch (#1496). A miss is ignored — there is no ambient sketch
// selection to clear, and the tool keeps waiting for a reference.
func (s *Session) pickModelReferenceInSketch(e PointerEvent) {
	if s.picker == nil {
		return
	}
	if sel, ok := s.picker.Pick(e.X, e.Y, s.pickFilter()); ok {
		s.feedPickMods(sel, e.Mods)
	}
}

// clearSelectionOnEmptyClick clears the selection when the user clicks empty space with
// no active tool — Inventor (GUID-B8F6E805): "click in an empty area to deselect". A
// modifier-held empty click is a no-op (it neither clears nor extends), and an active tool
// owns its own miss handling, so both are left untouched.
func (s *Session) clearSelectionOnEmptyClick(mods Modifier) {
	if s.tool != nil || mods.Has(ShiftMod) || mods.Has(CtrlMod) || s.selection.Count() == 0 {
		return
	}
	s.selection.Clear()
	event.Emit(s.bus, event.After, SelectionChanged{Count: 0})
}

// applyPickToSelection updates the selection set for a viewport pick with no active tool,
// mirroring Inventor (GUID-B8F6E805): a plain click replaces the selection; Shift/Ctrl+click
// toggles the clicked object's membership (add if new, remove if already selected).
func (s *Session) applyPickToSelection(sel Selectable, mods Modifier) {
	var changed bool
	if mods.Has(ShiftMod) || mods.Has(CtrlMod) {
		changed = s.selection.Toggle(sel)
	} else {
		s.selection.Clear()
		changed = s.selection.Add(sel)
	}
	if changed {
		event.Emit(s.bus, event.After, SelectionChanged{Count: s.selection.Count()})
	}
}

// keyEventToChord converts a device key event into a canonical [types.KeyChord],
// normalizing key synonyms (Return → Enter) so a binding matches however the platform
// spells the key.
func keyEventToChord(e KeyEvent) types.KeyChord {
	return types.KeyChord{
		Key:   normalizeKey(e.Key),
		Ctrl:  e.Mods.Has(CtrlMod),
		Alt:   e.Mods.Has(AltMod),
		Shift: e.Mods.Has(ShiftMod),
	}
}

// normalizeKey maps platform key synonyms onto the canonical token the binding table
// uses, then canonicalizes case so "z" and "Z" are one chord.
func normalizeKey(key string) string {
	if key == "Return" {
		return "Enter"
	}
	return types.CanonicalKey(key)
}

// PressKey routes a key press through the binding engine (M05-F17, #831). While the legacy
// command-alias input box is open the keystroke edits its buffer; otherwise the chord the
// event forms is resolved to an action and dispatched. With no matching binding it is a
// no-op. The built-in guards (e.g. undo is suppressed mid-tool) live in the dispatch.
//
// M26 F05: a modifier chord (Ctrl/Alt, e.g. Ctrl+S, Ctrl+Z) runs through the Command Window
// — its canonical word is echoed and then it dispatches — so a chord reads like a typed
// command ("Ctrl+S" shows "SAVE" and saves).
//
// #1751 S2: a bare alphanumeric key with no text field focused hands focus to the Command Window
// seeded with that character (see commandTypingSeed) — pressing a letter means "type a command",
// not fire a shortcut. Bare special keys (F1–F12, Delete, …) still resolve and dispatch directly.
func (s *Session) PressKey(e KeyEvent) error {
	if s.CommandInputActive() {
		return s.routeKeyToCommandInput(e)
	}
	chord := keyEventToChord(e)
	// A bare alphanumeric key with no text field focused means the user wants to type a command:
	// hand focus to the Command Window and seed the character, so the keystroke begins a command
	// line instead of firing a (deliberately nonexistent) bare-letter shortcut (#1751 S2). The
	// reserved-key policy lives in isReservedBareChord — special keys (F1–F12, Delete, …) fall
	// through to the binding engine below and may be bound bare.
	if seed, ok := commandTypingSeed(chord); ok {
		s.BeginCommandTyping(seed)
		return nil
	}
	actionID, ok := s.Bindings().ResolveChord(chord)
	if !ok {
		return nil
	}
	if chord.Ctrl || chord.Alt {
		return s.CommandLine().RunChord(s, actionID)
	}
	return s.Bindings().Dispatch(actionID, s)
}

// commandTypingSeed reports whether a chord should begin typing into the Command Window rather
// than resolve as a shortcut — a bare alphanumeric key (see isReservedBareChord) — and returns
// the character to seed the input with, lower-cased for a natural typed feel (the command
// vocabulary matches case-insensitively, so "l" and "L" resolve alike).
func commandTypingSeed(c types.KeyChord) (string, bool) {
	if !isReservedBareChord(c) {
		return "", false
	}
	return strings.ToLower(c.Key), true
}
