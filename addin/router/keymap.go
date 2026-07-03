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
	r.readOnly(wire.MethodKeymapList, listKeymap)
	r.readOnly(wire.MethodKeymapSetChord, typed(setKeymapChord))
	r.readOnly(wire.MethodKeymapSetAlias, typed(setKeymapAlias))
	r.readOnly(wire.MethodKeymapReset, typed(resetKeymapBinding))
	r.readOnly(wire.MethodKeymapResetAll, resetKeymapAll)
	r.readOnly(wire.MethodKeymapExport, exportKeymap)
	r.readOnly(wire.MethodKeymapImport, typed(importKeymap))
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
func setKeymapChord(s *app.Session, in wire.SetChordArgs) (wire.OKResult, error) {
	chord, err := types.ParseChord(in.Chord)
	if err != nil {
		return wire.OKResult{}, err
	}
	if err := s.Bindings().SetChord(in.ActionID, chord); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// setKeymapAlias sets one action's typed alias (wire keymap.setAlias).
func setKeymapAlias(s *app.Session, in wire.SetAliasArgs) (wire.OKResult, error) {
	if err := s.Bindings().SetAlias(in.ActionID, in.Alias); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// resetKeymapBinding restores one action to its defaults (wire keymap.reset).
func resetKeymapBinding(s *app.Session, in wire.ResetBindingArgs) (wire.OKResult, error) {
	if err := s.Bindings().Reset(in.ActionID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
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
func importKeymap(s *app.Session, in wire.KeymapExport) (wire.OKResult, error) {
	if err := s.Bindings().Import(keymap.Customization{Chords: in.Chords, Aliases: in.Aliases}); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}
