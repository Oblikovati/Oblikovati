// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// addKeymapCommand registers a command so the catalog has a known command row.
func addKeymapCommand(t *testing.T, s *app.Session, id, alias string) {
	t.Helper()
	cmd := app.NewCommand(id, id, "Test", func(*app.Session) error { return nil })
	if alias != "" {
		cmd = cmd.WithAlias(alias)
	}
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("register %q: %v", id, err)
	}
}

func TestKeymapListServesCatalog(t *testing.T) {
	r, s := seededSession(t)
	addKeymapCommand(t, s, "Test.Probe", "G")

	var res wire.ListBindingsResult
	call(t, r, s, "keymap.list", "{}", &res)

	var extrude *wire.BindingInfo
	var undo *wire.BindingInfo
	for i := range res.Bindings {
		switch res.Bindings[i].ActionID {
		case "Test.Probe":
			extrude = &res.Bindings[i]
		case app.ActionUndo:
			undo = &res.Bindings[i]
		}
	}
	if extrude == nil || extrude.Chord != "G" || extrude.Kind != "command" {
		t.Fatalf("Extrude row = %+v, want chord G / kind command", extrude)
	}
	if undo == nil || undo.Chord != "Ctrl+Z" || undo.Kind != "builtin" {
		t.Fatalf("Undo row = %+v, want chord Ctrl+Z / kind builtin", undo)
	}
}

func TestKeymapSetChordOverWire(t *testing.T) {
	r, s := seededSession(t)
	addKeymapCommand(t, s, "Test.Probe", "G")

	call(t, r, s, "keymap.setChord", `{"actionId":"Test.Probe","chord":"Ctrl+Shift+G"}`, nil)
	if c, ok := s.Bindings().EffectiveChord("Test.Probe"); !ok || c.String() != "Ctrl+Shift+G" {
		t.Fatalf("effective chord = (%q, %v), want Ctrl+Shift+E", c.String(), ok)
	}
}

func TestKeymapSetChordConflictErrorsOverWire(t *testing.T) {
	r, s := seededSession(t)
	addKeymapCommand(t, s, "Test.Probe", "G")

	if _, err := r.Handle(s, "keymap.setChord", []byte(`{"actionId":"Test.Probe","chord":"Ctrl+Z"}`)); err == nil {
		t.Error("rebinding to Ctrl+Z (undo) should fail over the wire")
	}
}

func TestKeymapResetOverWire(t *testing.T) {
	r, s := seededSession(t)
	addKeymapCommand(t, s, "Test.Probe", "G")
	call(t, r, s, "keymap.setChord", `{"actionId":"Test.Probe","chord":"Ctrl+Shift+G"}`, nil)

	call(t, r, s, "keymap.reset", `{"actionId":"Test.Probe"}`, nil)
	if c, ok := s.Bindings().EffectiveChord("Test.Probe"); !ok || c.String() != "G" {
		t.Fatalf("after reset, chord = (%q, %v), want default G", c.String(), ok)
	}
}

func TestKeymapExportImportRoundTripOverWire(t *testing.T) {
	r, s := seededSession(t)
	addKeymapCommand(t, s, "Test.Probe", "G")
	call(t, r, s, "keymap.setAlias", `{"actionId":"Test.Probe","alias":"EXT"}`, nil)

	var exp wire.KeymapExport
	call(t, r, s, "keymap.export", "{}", &exp)
	if exp.Aliases["Test.Probe"] != "EXT" {
		t.Fatalf("export = %+v, want the EXT alias", exp)
	}

	call(t, r, s, "keymap.resetAll", "{}", nil)
	if s.Bindings().EffectiveAlias("Test.Probe") != "" {
		t.Fatal("resetAll should have cleared the alias")
	}

	call(t, r, s, "keymap.import", `{"aliases":{"Test.Probe":"EXT"}}`, nil)
	if s.Bindings().EffectiveAlias("Test.Probe") != "EXT" {
		t.Error("import should restore the alias")
	}
}
