// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/app/keymap"
)

// registerKeymapHandlers wires command alias & keyboard-shortcut customization (M05-F17,
// #831): list the catalog, rebind a shortcut, set an alias, reset one/all, import/export.
func (r *Router) registerKeymapHandlers() {
	r.handlers[wire.MethodKeymapList] = listKeymap
	r.handlers[wire.MethodKeymapSetChord] = setKeymapChord
	r.handlers[wire.MethodKeymapSetAlias] = setKeymapAlias
	r.handlers[wire.MethodKeymapReset] = resetKeymapBinding
	r.handlers[wire.MethodKeymapResetAll] = resetKeymapAll
	r.handlers[wire.MethodKeymapExport] = exportKeymap
	r.handlers[wire.MethodKeymapImport] = importKeymap
}

// listKeymap returns the full binding catalog (wire keymap.list).
func listKeymap(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	cat := s.Bindings().Catalog()
	out := make([]wire.BindingInfo, len(cat))
	for i, b := range cat {
		out[i] = wire.BindingInfo{
			ActionID: b.ActionID, DisplayName: b.DisplayName, Kind: b.Kind,
			Chord: b.Chord.String(), DefaultChord: b.DefaultChord.String(),
			Alias: b.Alias, Customized: b.Customized,
		}
	}
	return json.Marshal(wire.ListBindingsResult{Bindings: out})
}

// setKeymapChord rebinds one action's shortcut (wire keymap.setChord).
func setKeymapChord(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetChordArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	chord, err := types.ParseChord(req.Chord)
	if err != nil {
		return nil, err
	}
	if err := s.Bindings().SetChord(req.ActionID, chord); err != nil {
		return nil, err
	}
	return ok()
}

// setKeymapAlias sets one action's typed alias (wire keymap.setAlias).
func setKeymapAlias(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetAliasArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.Bindings().SetAlias(req.ActionID, req.Alias); err != nil {
		return nil, err
	}
	return ok()
}

// resetKeymapBinding restores one action to its defaults (wire keymap.reset).
func resetKeymapBinding(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ResetBindingArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.Bindings().Reset(req.ActionID); err != nil {
		return nil, err
	}
	return ok()
}

// resetKeymapAll discards every customization (wire keymap.resetAll).
func resetKeymapAll(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	if err := s.Bindings().ResetAll(); err != nil {
		return nil, err
	}
	return ok()
}

// exportKeymap returns the user's customization delta (wire keymap.export).
func exportKeymap(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	c := s.Bindings().Export()
	return json.Marshal(wire.KeymapExport{Chords: c.Chords, Aliases: c.Aliases})
}

// importKeymap replaces the customization with the imported delta (wire keymap.import).
func importKeymap(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.KeymapExport
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.Bindings().Import(keymap.Customization{Chords: req.Chords, Aliases: req.Aliases}); err != nil {
		return nil, err
	}
	return ok()
}
