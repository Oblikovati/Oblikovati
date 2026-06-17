// SPDX-License-Identifier: GPL-2.0-only

package cmdline

import (
	"sort"
	"strings"
)

// Vocabulary maps AutoCAD command words onto Oblikovati action ids (a command id like
// "Sketch.Line", or a reserved built-in id like "edit.undo"). AutoCAD allows many aliases
// per command (RECTANG and RECTANGLE both draw a rectangle), so the table is many→one;
// lookups are case-insensitive. This is the built-in default layer only: the binding
// engine's user aliases override it (M26).
//
// Two invariants make the table the source of truth for a generated command manual:
//   - every word is MULTI-letter — single-letter activation is reserved for the keybinding
//     editor (a personalised Shift/Control chord), never the static list;
//   - every entry carries a one-line summary and a usage example, so Manual()/RenderManual()
//     emit a complete, machine-parseable manual.
//
// The action ids here MUST be real registered command ids or built-in action ids — a test
// in the app package asserts every one resolves (CLAUDE.md: never invent ids).
type Vocabulary struct {
	toAction  map[string]string // UPPER(word) → action id
	canonical map[string]string // action id → its canonical word (the full name, listed first)
	commands  []Command         // one row per entry, build order, for the manual
}

// vocabEntry is one command's row: its action id, the multi-letter words that invoke it
// (canonical full name first), a one-line manual summary, and a usage example.
type vocabEntry struct {
	action  string
	words   []string
	summary string
	example string
}

// Command is one row of the generated command manual: the action it runs, the words that
// invoke it (canonical first), a one-line description and a usage example. It is the public,
// parseable shape the manual is built from.
type Command struct {
	Action    string
	Canonical string
	Words     []string
	Summary   string
	Example   string
}

// builtinVocabulary is the AutoCAD→Oblikovati command map: the sketch, part, surface,
// work-plane, assembly, sheet-metal, drawing, 3D-sketch, manage and view subsets. Each word
// appears exactly once across the whole table (a word resolves to a single action); the first
// word of each entry is the action's canonical name (used to echo keyboard chords).
func builtinVocabulary() []vocabEntry {
	var v []vocabEntry
	v = append(v, sketchVocabulary()...)
	v = append(v, partVocabulary()...)
	v = append(v, surfaceVocabulary()...)
	v = append(v, workplaneVocabulary()...)
	v = append(v, assemblyVocabulary()...)
	v = append(v, sheetMetalVocabulary()...)
	v = append(v, drawingVocabulary()...)
	v = append(v, sketch3DVocabulary()...)
	v = append(v, manageVocabulary()...)
	v = append(v, viewVocabulary()...)
	v = append(v, moldFreeformMeshVocabulary()...)
	v = append(v, appVocabulary()...)
	return v
}

// DefaultVocabulary builds the built-in AutoCAD command vocabulary, panicking on a table
// error caught by tests: a duplicate word, a single-letter word (single letters are the
// keybinding editor's domain), or an entry missing its manual summary/example. The first
// word of each entry is recorded as that action's canonical name.
func DefaultVocabulary() *Vocabulary {
	v := &Vocabulary{toAction: map[string]string{}, canonical: map[string]string{}}
	for _, e := range builtinVocabulary() {
		v.addEntry(e)
	}
	return v
}

// addEntry registers one table row, validating the table invariants.
func (v *Vocabulary) addEntry(e vocabEntry) {
	if len(e.words) == 0 {
		panic("cmdline: vocabulary entry for " + e.action + " has no words")
	}
	if e.summary == "" || e.example == "" {
		panic("cmdline: vocabulary entry " + e.words[0] + " is missing its manual summary/example")
	}
	v.canonical[e.action] = strings.ToUpper(e.words[0])
	for _, w := range e.words {
		key := strings.ToUpper(w)
		if len([]rune(key)) < 2 {
			panic("cmdline: single-letter vocabulary word " + key + " (" + e.action +
				") — single letters belong to the keybinding editor, not the static list")
		}
		if existing, dup := v.toAction[key]; dup {
			panic("cmdline: duplicate vocabulary word " + key + " maps to both " + existing + " and " + e.action)
		}
		v.toAction[key] = e.action
	}
	v.commands = append(v.commands, Command{
		Action: e.action, Canonical: strings.ToUpper(e.words[0]),
		Words: upperWords(e.words), Summary: e.summary, Example: e.example,
	})
}

// upperWords returns the words upper-cased (the canonical on-screen form), preserving order.
func upperWords(words []string) []string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = strings.ToUpper(w)
	}
	return out
}

// Manual returns every command as a parseable manual row, sorted by canonical name — the
// single source of truth for the generated command manual.
func (v *Vocabulary) Manual() []Command {
	out := make([]Command, len(v.commands))
	copy(out, v.commands)
	sort.Slice(out, func(i, j int) bool { return out[i].Canonical < out[j].Canonical })
	return out
}

// CanonicalWord returns an action's canonical command word (the word listed first in its
// table entry), or false when the action is not in the vocabulary.
func (v *Vocabulary) CanonicalWord(action string) (string, bool) {
	w, ok := v.canonical[action]
	return w, ok
}

// Matches returns the autocomplete suggestions for a typed prefix: the canonical name of
// every command any of whose words begins with the prefix (case-insensitive), de-duplicated
// and sorted. So "RE" → [REALISTIC, RECTANG, REDO, REVOLVE]. An empty prefix returns nil.
func (v *Vocabulary) Matches(prefix string) []string {
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	if prefix == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for word, action := range v.toAction {
		if !strings.HasPrefix(word, prefix) {
			continue
		}
		if canon := v.canonical[action]; !seen[canon] {
			seen[canon] = true
			out = append(out, canon)
		}
	}
	sort.Strings(out)
	return out
}

// Resolve maps a typed command word (case-insensitive) to its action id, returning false
// when the word is not in the vocabulary.
func (v *Vocabulary) Resolve(word string) (string, bool) {
	a, ok := v.toAction[strings.ToUpper(strings.TrimSpace(word))]
	return a, ok
}

// Words returns every command word in the vocabulary (for validation/tests), in no
// particular order.
func (v *Vocabulary) Words() []string {
	out := make([]string, 0, len(v.toAction))
	for w := range v.toAction {
		out = append(out, w)
	}
	return out
}

// Actions returns the distinct action ids the vocabulary targets (for validation/tests),
// in no particular order.
func (v *Vocabulary) Actions() []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range v.toAction {
		if !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	return out
}
