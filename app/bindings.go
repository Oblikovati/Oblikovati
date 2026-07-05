// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/app/cmdline"
	"oblikovati.org/app/keymap"
)

// The binding engine (M05-F17, #831). Every keyboard shortcut and typed command alias
// resolves through here. A bindable target is identified by a flat action id: a
// registered command id verbatim, or one of the reserved built-in ids below (session
// actions — undo/redo/cancel/commit/visibility — that are not registry commands). The
// hardcoded shortcuts of session_input.go migrate into the built-in table, so they are
// rebindable like any command. Defaults are derived live from the command registry
// (a command's WithAlias letter is its default chord) plus the built-in table; only the
// user's deltas are stored (keymap.Customization).

// Reserved built-in action ids — session actions that are not registry commands.
const (
	ActionUndo             = "edit.undo"
	ActionRedo             = "edit.redo"
	ActionDelete           = "edit.delete"
	ActionSave             = "file.save"
	ActionCancel           = "tool.cancel"
	ActionCommit           = "tool.commit"
	ActionToggleVisibility = "workplane.toggleVisibility"
	ActionPreviousView     = "view.previous"
	ActionHomeView         = "view.home"
)

const (
	bindingKindCommand = "command"
	bindingKindBuiltin = "builtin"
)

// builtinAction is a bindable session action that is not a registered command. Its
// dispatch closes over no state; it receives the Session at call time, and its guard
// lives inside the dispatch so the pre-existing semantics are preserved exactly.
type builtinAction struct {
	id           string
	displayName  string
	defaultChord types.KeyChord
	extraChords  []types.KeyChord // extra default chords that also resolve here (redo's Ctrl+Shift+Z)
	dispatch     func(*Session) error
}

// builtinActions is the fixed table of session actions migrated out of PressKey.
func builtinActions() []builtinAction {
	return []builtinAction{
		{id: ActionUndo, displayName: "Undo", defaultChord: mustChord("Ctrl+Z"), dispatch: dispatchUndo},
		{id: ActionRedo, displayName: "Redo", defaultChord: mustChord("Ctrl+Y"),
			extraChords: []types.KeyChord{mustChord("Ctrl+Shift+Z")}, dispatch: dispatchRedo},
		{id: ActionDelete, displayName: "Delete", defaultChord: mustChord("Delete"), dispatch: dispatchDelete},
		{id: ActionSave, displayName: "Save", defaultChord: mustChord("Ctrl+S"), dispatch: dispatchSave},
		{id: ActionCancel, displayName: "Cancel / Deselect", defaultChord: mustChord("Escape"), dispatch: dispatchCancel},
		{id: ActionCommit, displayName: "Finish Command", defaultChord: mustChord("Enter"), dispatch: dispatchCommit},
		// Toggle Visibility ships unbound: it is a single-letter shortcut, so the user binds it
		// as a Shift/Control chord in the keybinding editor (M26), like any single-letter command.
		{id: ActionToggleVisibility, displayName: "Toggle Visibility", dispatch: dispatchToggleVisibility},
		{id: ActionPreviousView, displayName: "Previous View", defaultChord: mustChord("F5"), dispatch: dispatchPreviousView},
		{id: ActionHomeView, displayName: "Home View", defaultChord: mustChord("F6"), dispatch: dispatchHomeView},
	}
}

// dispatchPreviousView restores the last recorded view (F5 — Inventor's Previous View).
func dispatchPreviousView(s *Session) error {
	s.PreviousView()
	return nil
}

// dispatchHomeView swings to the model's Home / isometric view (F6).
func dispatchHomeView(s *Session) error {
	s.GoHome()
	return nil
}

// dispatchUndo runs undo unless a bounded transaction (an open recipe group) is mid-record —
// undoing then would corrupt the unit being written. A merely-armed interactive tool is NOT
// such a transaction: in continuous drawing each segment auto-commits, so between clicks no
// group is open and Ctrl+Z rewinds the last committed segment with the tool still running —
// Inventor's behavior. The old `s.tool != nil` guard was far broader than its own comment
// ("transaction in progress") and silently no-op'd undo for the entire sketch session (#1750).
func dispatchUndo(s *Session) error {
	if s.InTransaction() {
		return nil
	}
	return s.Undo()
}

func dispatchRedo(s *Session) error {
	if s.InTransaction() {
		return nil
	}
	return s.Redo()
}

// dispatchCancel is ESC, the universal cancel (M26): it stops any pending command-line
// question, closes any feature/tool creation or editing window (the active tool), and with
// nothing in progress clears the selection. It always asks the head to return keyboard focus
// to the command-window input so the next command can be typed immediately.
func dispatchCancel(s *Session) error {
	s.RequestCommandInputFocus()
	cancelled := false
	if _, ok := s.Prompts().Pending(); ok {
		s.Prompts().CancelPending()
		cancelled = true
	}
	if s.tool != nil {
		s.CancelTool() // closes the active feature/tool creation or editing window
		cancelled = true
	}
	if !cancelled {
		s.Select(nil)
	}
	return nil
}

// dispatchCommit finishes the active tool; a no-op with no tool.
func dispatchCommit(s *Session) error {
	if s.tool != nil {
		return s.OK()
	}
	return nil
}

func dispatchToggleVisibility(s *Session) error {
	s.ToggleSelectedWorkPlaneVisibility()
	return nil
}

// dispatchDelete is the Delete key: it removes the selected sketch entities while editing a
// sketch (issue #1232). With no sketch open / nothing selected it is a no-op, so it never
// destroys 3D geometry by accident — feature/body delete stays on the browser menu.
func dispatchDelete(s *Session) error { return s.DeleteSelectedSketchEntities() }

// dispatchSave saves the active document (Ctrl+S / the "SAVE" command word, M26 F05).
func dispatchSave(s *Session) error { return s.SaveActiveDocument() }

// isBareLetterChord reports whether c is a single printable character with no modifier. Such
// chords are reserved (M26): single-letter shortcuts must carry Shift or Control, so they are
// never a default and the keybinding editor rejects them. Multi-character keys (Escape, F5)
// and modified chords (Ctrl+S) are unaffected.
func isBareLetterChord(c types.KeyChord) bool {
	return !c.IsZero() && len([]rune(c.Key)) == 1 && !c.Ctrl && !c.Alt && !c.Shift
}

// mustChord parses a constant chord literal from the built-in table, panicking on a
// malformed one (a programming error, never reachable with the valid literals above).
func mustChord(s string) types.KeyChord {
	c, err := types.ParseChord(s)
	if err != nil {
		panic(fmt.Sprintf("app: invalid built-in chord literal %q: %v", s, err))
	}
	return c
}

// Bindings is the live resolver: the command registry, the built-in action table, the
// built-in AutoCAD command vocabulary, and the user's customization overlay, with the
// store to persist edits.
type Bindings struct {
	cmds     *CommandManager
	builtins []builtinAction
	vocab    *cmdline.Vocabulary // built-in AutoCAD name/alias → action id (M26)
	custom   keymap.Customization
	store    keymap.Store // nil ⇒ in-session only
}

// newBindings builds the engine over a command registry, with no customization yet.
func newBindings(cmds *CommandManager) *Bindings {
	return &Bindings{
		cmds: cmds, builtins: builtinActions(),
		vocab: cmdline.DefaultVocabulary(), custom: keymap.Defaults(),
	}
}

// Bindings returns the session's binding engine, constructing it on first use over the
// command registry. Lazy so the constructor stays small and commands registered after
// NewSession are still seen (the engine reads the registry live).
func (s *Session) Bindings() *Bindings {
	if s.bindings == nil {
		s.bindings = newBindings(s.commands)
	}
	return s.bindings
}

// bindable is the merged view of one action (command or built-in) the engine iterates.
type bindable struct {
	id           string
	displayName  string
	kind         string
	defaultChord types.KeyChord
	extraChords  []types.KeyChord
}

// bindables returns every bindable action: commands first (registration order), then
// the built-ins. A command's default chord is its WithAlias letter (empty ⇒ unbound).
func (b *Bindings) bindables() []bindable {
	out := make([]bindable, 0, len(b.cmds.defs)+len(b.builtins))
	for _, c := range b.cmds.All() {
		out = append(out, bindable{id: c.ID(), displayName: c.DisplayName(), kind: bindingKindCommand, defaultChord: commandDefaultChord(c)})
	}
	for _, ba := range b.builtins {
		out = append(out, bindable{
			id: ba.id, displayName: ba.displayName, kind: bindingKindBuiltin,
			defaultChord: ba.defaultChord, extraChords: ba.extraChords,
		})
	}
	return out
}

// commandDefaultChord derives a command's default shortcut: its predefined WithDefaultChord if
// set, else its single-letter alias — but a bare single letter is dropped, since single-letter
// shortcuts are personalised as Shift/Control chords in the keybinding editor (M26).
func commandDefaultChord(c *CommandDefinition) types.KeyChord {
	if c.DefaultChord() != "" {
		dc, _ := types.ParseChord(c.DefaultChord())
		return dc
	}
	dc, _ := types.ParseChord(c.Alias())
	if isBareLetterChord(dc) {
		return types.KeyChord{}
	}
	return dc
}

// findBindable locates one action by id.
func (b *Bindings) findBindable(actionID string) (bindable, bool) {
	for _, bd := range b.bindables() {
		if bd.id == actionID {
			return bd, true
		}
	}
	return bindable{}, false
}

// EffectiveChord returns an action's current shortcut: the user override if present
// (an empty override means explicitly unbound), else the derived default.
func (b *Bindings) EffectiveChord(actionID string) (types.KeyChord, bool) {
	if raw, ok := b.custom.Chords[actionID]; ok {
		c, err := types.ParseChord(raw)
		if err != nil || c.IsZero() {
			return types.KeyChord{}, false
		}
		return c, true
	}
	bd, ok := b.findBindable(actionID)
	if !ok || bd.defaultChord.IsZero() {
		return types.KeyChord{}, false
	}
	return bd.defaultChord, true
}

// EffectiveAlias returns an action's typed command alias ("" if none). Aliases are
// purely user-defined; there is no predefined alias.
func (b *Bindings) EffectiveAlias(actionID string) string { return b.custom.Aliases[actionID] }

// ResolveChord maps a pressed chord to the action it triggers. Effective bindings win;
// an action's extra default chords resolve only while it keeps its default (not
// overridden), preserving e.g. Ctrl+Shift+Z → redo.
func (b *Bindings) ResolveChord(c types.KeyChord) (string, bool) {
	if c.IsZero() {
		return "", false
	}
	target := c.String()
	for _, bd := range b.bindables() {
		if ec, ok := b.EffectiveChord(bd.id); ok && ec.String() == target {
			return bd.id, true
		}
	}
	for _, bd := range b.bindables() {
		if _, overridden := b.custom.Chords[bd.id]; overridden {
			continue
		}
		for _, extra := range bd.extraChords {
			if extra.String() == target {
				return bd.id, true
			}
		}
	}
	return "", false
}

// ResolveAlias maps a typed alias (case-insensitive) to its action. A user-defined alias
// wins; failing that, the built-in AutoCAD vocabulary resolves the word — but only to an
// action that actually exists in this session, so a stale table entry can never dispatch
// to a missing command (M26).
func (b *Bindings) ResolveAlias(alias string) (string, bool) {
	if alias == "" {
		return "", false
	}
	if id, ok := b.resolveUserAlias(alias); ok {
		return id, true
	}
	if id, ok := b.vocab.Resolve(alias); ok {
		if _, exists := b.findBindable(id); exists {
			return id, true
		}
	}
	return "", false
}

// resolveUserAlias maps a typed alias to its action using ONLY the user's overrides (not
// the built-in vocabulary). The SetAlias conflict guard uses this so a user may freely
// rebind a word the vocabulary already covers (their alias overrides the default).
func (b *Bindings) resolveUserAlias(alias string) (string, bool) {
	for id, a := range b.custom.Aliases {
		if strings.EqualFold(a, alias) {
			return id, true
		}
	}
	return "", false
}

// Dispatch runs an action by id: a built-in via its guarded session handler, else the
// registered command via Session.Execute.
func (b *Bindings) Dispatch(actionID string, s *Session) error {
	for _, ba := range b.builtins {
		if ba.id == actionID {
			return ba.dispatch(s)
		}
	}
	return s.Execute(actionID)
}

// SetChord rebinds an action's shortcut, rejecting a chord already bound elsewhere. A
// zero chord clears the binding. Persists when a store is wired.
func (b *Bindings) SetChord(actionID string, c types.KeyChord) error {
	if _, ok := b.findBindable(actionID); !ok {
		return fmt.Errorf("app: unknown action %q for SetChord; expected a command or built-in action id", actionID)
	}
	if isBareLetterChord(c) {
		return fmt.Errorf("app: single-letter shortcut %q must use Shift or Control (M26)", c.String())
	}
	if !c.IsZero() {
		if owner, ok := b.ResolveChord(c); ok && owner != actionID {
			return fmt.Errorf("app: chord %q already bound to %q (%s)", c.String(), owner, b.displayName(owner))
		}
	}
	if b.custom.Chords == nil {
		b.custom.Chords = map[string]string{}
	}
	b.custom.Chords[actionID] = c.String()
	return b.persist()
}

// SetAlias sets an action's typed alias, rejecting one already bound elsewhere. An
// empty alias clears it.
func (b *Bindings) SetAlias(actionID, alias string) error {
	if _, ok := b.findBindable(actionID); !ok {
		return fmt.Errorf("app: unknown action %q for SetAlias; expected a command or built-in action id", actionID)
	}
	if len([]rune(alias)) == 1 {
		return fmt.Errorf("app: single-letter alias %q is not allowed — bind it as a Shift/Control chord instead (M26)", alias)
	}
	if alias != "" {
		if owner, ok := b.resolveUserAlias(alias); ok && owner != actionID {
			return fmt.Errorf("app: alias %q already bound to %q (%s)", alias, owner, b.displayName(owner))
		}
	}
	if alias == "" {
		delete(b.custom.Aliases, actionID)
		return b.persist()
	}
	if b.custom.Aliases == nil {
		b.custom.Aliases = map[string]string{}
	}
	b.custom.Aliases[actionID] = alias
	return b.persist()
}

// Reset restores one action's shortcut and alias to their defaults.
func (b *Bindings) Reset(actionID string) error {
	delete(b.custom.Chords, actionID)
	delete(b.custom.Aliases, actionID)
	return b.persist()
}

// ResetAll discards every customization.
func (b *Bindings) ResetAll() error {
	b.custom = keymap.Defaults()
	return b.persist()
}

// Export returns a copy of the user's customization delta (for keymap.export).
func (b *Bindings) Export() keymap.Customization { return b.custom.Clone() }

// Import replaces the customization with the given delta and persists it.
func (b *Bindings) Import(c keymap.Customization) error {
	b.custom = c.Clone()
	return b.persist()
}

// Binding is one catalog row: an action with its effective and default shortcut, its
// alias, and whether the user has customized it.
type Binding struct {
	ActionID     string
	DisplayName  string
	Kind         string
	Chord        types.KeyChord
	DefaultChord types.KeyChord
	Alias        string
	Customized   bool
}

// Catalog returns every bindable action with its effective + default binding, in a
// stable display order (commands first, then built-ins).
func (b *Bindings) Catalog() []Binding {
	bindables := b.bindables()
	out := make([]Binding, 0, len(bindables))
	for _, bd := range bindables {
		eff, _ := b.EffectiveChord(bd.id)
		_, chordCustom := b.custom.Chords[bd.id]
		_, aliasCustom := b.custom.Aliases[bd.id]
		out = append(out, Binding{
			ActionID: bd.id, DisplayName: bd.displayName, Kind: bd.kind,
			Chord: eff, DefaultChord: bd.defaultChord, Alias: b.EffectiveAlias(bd.id),
			Customized: chordCustom || aliasCustom,
		})
	}
	return out
}

// CheckDefaults verifies the out-of-the-box bindings are self-consistent: no command id
// reuses a reserved built-in id, and no two actions share a default chord. It is a
// developer-error guard — call it after registering the standard commands.
func (b *Bindings) CheckDefaults() error {
	builtinID := map[string]bool{}
	for _, ba := range b.builtins {
		builtinID[ba.id] = true
	}
	for _, c := range b.cmds.All() {
		if builtinID[c.ID()] {
			return fmt.Errorf("app: command id %q collides with a reserved built-in action id", c.ID())
		}
	}
	seen := map[string]string{}
	for _, bd := range b.bindables() {
		if bd.defaultChord.IsZero() {
			continue
		}
		key := bd.defaultChord.String()
		if other, dup := seen[key]; dup {
			return fmt.Errorf("app: default chord %q bound to both %q and %q", key, other, bd.id)
		}
		seen[key] = bd.id
	}
	return nil
}

// Completions returns the built-in AutoCAD vocabulary's autocomplete suggestions for a typed
// command prefix (M26) — the canonical names of matching commands.
func (b *Bindings) Completions(prefix string) []string { return b.vocab.Matches(prefix) }

// CanonicalWord returns the command word to echo when an action is triggered by a keyboard
// chord (M26 F05): its AutoCAD vocabulary name ("SAVE", "UNDO", "EXTRUDE") when it has one,
// else its display name upper-cased — so a chord reads on the command line like a typed
// command.
func (b *Bindings) CanonicalWord(actionID string) string {
	if w, ok := b.vocab.CanonicalWord(actionID); ok {
		return w
	}
	return strings.ToUpper(b.displayName(actionID))
}

// displayName resolves an action id to its label for error messages.
func (b *Bindings) displayName(actionID string) string {
	if bd, ok := b.findBindable(actionID); ok {
		return bd.displayName
	}
	return actionID
}

// persist saves the customization when a store is wired (in-session only otherwise).
func (b *Bindings) persist() error {
	if b.store == nil {
		return nil
	}
	return b.store.Save(b.custom)
}
