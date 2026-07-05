//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Tools ▸ Customize Keyboard panel (M05-F17, #831): the predefined catalog of every
// bindable action with its editable shortcut and alias, per-row reset, and reset-all. The
// binding engine owns resolution, conflict checks and persistence; this file is the ImGui
// surface that reads the catalog each frame and calls the edit verbs on Enter.

const keymapBufLen = 48

const keymapFilterBufLen = 64

// keymapUI is the panel's cross-frame state: per-action edit buffers (seeded lazily, then
// dropped after an apply so the canonical value re-seeds), the command-name filter buffer,
// and the last apply error.
var keymapUI = struct {
	chord   map[string][]byte
	alias   map[string][]byte
	filter  []byte
	lastErr string
}{}

// drawKeymapEditor renders the customization panel while it is open. The title-bar X closes
// it (issue #1232: the window had no close button), alongside the Done button.
func drawKeymapEditor(s *app.Session) {
	if !s.KeymapEditorOpen() {
		return
	}
	if keymapUI.chord == nil {
		dropKeymapBuffers()
	}
	dialogSizeOnce(720, 560)
	visible, open := native.BeginClosable("Customize Keyboard")
	if visible {
		drawKeymapBody(s)
	}
	native.End()
	if !open {
		s.CloseKeymapEditor()
	}
}

// drawKeymapBody renders the panel contents: the help line, the filter box, the editable
// table, and the footer buttons.
func drawKeymapBody(s *app.Session) {
	native.Text("Type a shortcut (e.g. Ctrl+Shift+E) or an alias, then Enter. Empty clears it.")
	if keymapUI.lastErr != "" {
		native.Text("! " + keymapUI.lastErr)
	}
	native.Separator()
	drawKeymapFilter()
	drawKeymapTable(s)
	native.Separator()
	if native.Button("Reset All") {
		recordKeymapResult(s.Bindings().ResetAll())
		dropKeymapBuffers()
	}
	native.SameLine()
	if native.Button("Done") {
		s.CloseKeymapEditor()
	}
}

// drawKeymapFilter renders the command-name filter box (issue #1232: there was no way to
// find a command by name in the unsorted list).
func drawKeymapFilter() {
	native.Text("Filter")
	native.SameLine()
	native.SetNextItemWidth(-1)
	native.InputText("##keymap-filter", keymapUI.filter)
}

// drawKeymapTable renders the catalog as an editable four-column table, alphabetically
// sorted and narrowed by the filter box (issue #1232).
func drawKeymapTable(s *app.Session) {
	if !native.BeginTable("##keymap", 4, 0, 0) {
		return
	}
	native.TableSetupColumn("Command")
	native.TableSetupColumn("Shortcut")
	native.TableSetupColumn("Alias")
	native.TableSetupColumn("")
	native.TableSetupScrollFreeze(0, 1)
	native.TableHeadersRow()
	for _, b := range s.Bindings().CatalogFiltered(bufString(keymapUI.filter)) {
		drawKeymapRow(s, b)
	}
	native.EndTable()
}

// drawKeymapRow renders one action: its name, editable shortcut/alias fields, and a reset.
func drawKeymapRow(s *app.Session, b app.Binding) {
	native.TableNextRow()
	native.TableNextColumn()
	native.Text(b.DisplayName)

	native.TableNextColumn()
	if editKeymapField(b.ActionID, keymapUI.chord, b.Chord.String(), "##chord-") {
		applyChord(s, b.ActionID, bufString(keymapUI.chord[b.ActionID]))
	}

	native.TableNextColumn()
	if editKeymapField(b.ActionID, keymapUI.alias, b.Alias, "##alias-") {
		recordKeymapResult(s.Bindings().SetAlias(b.ActionID, bufString(keymapUI.alias[b.ActionID])))
		delete(keymapUI.alias, b.ActionID)
	}

	native.TableNextColumn()
	if b.Customized && native.Button("Reset##"+b.ActionID) {
		recordKeymapResult(s.Bindings().Reset(b.ActionID))
		delete(keymapUI.chord, b.ActionID)
		delete(keymapUI.alias, b.ActionID)
	}
}

// editKeymapField draws a per-action InputText seeded lazily from current, returning true
// on Enter.
func editKeymapField(id string, buffers map[string][]byte, current, idPrefix string) bool {
	buf, ok := buffers[id]
	if !ok {
		buf = make([]byte, keymapBufLen)
		copyText(buf, current)
		buffers[id] = buf
	}
	native.SetNextItemWidth(-1)
	return native.InputTextSubmit(idPrefix+id, buf)
}

// applyChord parses a typed chord and rebinds the action, recording any error.
func applyChord(s *app.Session, actionID, text string) {
	chord, err := types.ParseChord(text)
	if err == nil {
		err = s.Bindings().SetChord(actionID, chord)
	}
	recordKeymapResult(err)
	delete(keymapUI.chord, actionID)
}

// recordKeymapResult clears the panel error on success, else shows the reason.
func recordKeymapResult(err error) {
	if err != nil {
		keymapUI.lastErr = err.Error()
		return
	}
	keymapUI.lastErr = ""
}

// dropKeymapBuffers forces every field to re-seed from the model (after a reset-all). It
// keeps the filter buffer allocated so the active filter survives a Reset All.
func dropKeymapBuffers() {
	keymapUI.chord = map[string][]byte{}
	keymapUI.alias = map[string][]byte{}
	if keymapUI.filter == nil {
		keymapUI.filter = make([]byte, keymapFilterBufLen)
	}
}
