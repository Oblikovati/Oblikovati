// SPDX-License-Identifier: GPL-2.0-only

package cmdline

import "strings"

// Vocabulary maps AutoCAD command words — the full name and its short aliases — to an
// Oblikovati action id (a command id like "Sketch.Line", or a reserved built-in id like
// "edit.undo"). AutoCAD allows many aliases per command (L and LINE both draw a line), so
// the table is many→one; lookups are case-insensitive. This is the built-in default layer
// only: the binding engine's user aliases override it (M26). The action ids here MUST be
// real registered command ids or built-in action ids — a test in the app package asserts
// every one resolves (CLAUDE.md: never invent ids).
type Vocabulary struct {
	toAction  map[string]string // UPPER(word) → action id
	canonical map[string]string // action id → its canonical word (the full name, listed first)
}

// vocabEntry is one command's row: the target action id and every word that invokes it.
type vocabEntry struct {
	action string
	words  []string
}

// builtinVocabulary is the AutoCAD→Oblikovati command map. It is the relevant sketch and
// part-modelling subset; F07 expands it to full coverage from the scraped AutoCAD corpus.
// Each word appears exactly once across the table (a word resolves to a single action); the
// first word of each entry is the action's canonical name (used to echo keyboard chords).
func builtinVocabulary() []vocabEntry {
	v := append(sketchVocabulary(), partVocabulary()...)
	return append(v, appVocabulary()...)
}

// appVocabulary maps the application-wide editing/file actions (built-in action ids).
func appVocabulary() []vocabEntry {
	return []vocabEntry{
		{"file.save", []string{"SAVE", "QSAVE"}},
		{"edit.undo", []string{"UNDO", "U"}},
		{"edit.redo", []string{"REDO", "MREDO"}},
	}
}

// sketchVocabulary maps AutoCAD 2D drawing/modify commands to Oblikovati sketch tools.
func sketchVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Sketch.Line", []string{"LINE", "L"}},
		{"Sketch.Circle", []string{"CIRCLE", "C"}},
		{"Sketch.Arc", []string{"ARC", "A"}},
		{"Sketch.Rectangle", []string{"RECTANG", "RECTANGLE", "REC"}},
		{"Sketch.Point", []string{"POINT", "PO"}},
		{"Sketch.Polygon", []string{"POLYGON", "POL"}},
		{"Sketch.Ellipse", []string{"ELLIPSE", "EL"}},
		{"Sketch.Spline", []string{"SPLINE", "SPL"}},
		{"Sketch.Fillet", []string{"FILLET", "F"}},
		{"Sketch.Trim", []string{"TRIM", "TR"}},
		{"Sketch.Extend", []string{"EXTEND", "EX"}},
		{"Sketch.Offset", []string{"OFFSET", "O"}},
		{"Sketch.Mirror", []string{"MIRROR", "MI"}},
	}
}

// partVocabulary maps AutoCAD 3D/solid commands to Oblikovati part features.
func partVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Create.Extrude", []string{"EXTRUDE", "EXT", "E"}},
		{"Create.Revolve", []string{"REVOLVE", "REV"}},
		{"Create.Sweep", []string{"SWEEP", "SW"}},
		{"Create.Loft", []string{"LOFT"}},
		{"Create.Coil", []string{"HELIX", "COIL"}},
		{"Modify.Hole", []string{"HOLE"}},
		{"Modify.Chamfer", []string{"CHAMFER", "CHA"}},
		{"Modify.Shell", []string{"SHELL"}},
	}
}

// DefaultVocabulary builds the built-in AutoCAD command vocabulary, panicking on a
// duplicate word (a programming error in the table above, caught by tests). The first word
// of each entry is recorded as that action's canonical name.
func DefaultVocabulary() *Vocabulary {
	v := &Vocabulary{toAction: map[string]string{}, canonical: map[string]string{}}
	for _, e := range builtinVocabulary() {
		if len(e.words) > 0 {
			v.canonical[e.action] = strings.ToUpper(e.words[0])
		}
		for _, w := range e.words {
			key := strings.ToUpper(w)
			if existing, dup := v.toAction[key]; dup {
				panic("cmdline: duplicate vocabulary word " + key + " maps to both " + existing + " and " + e.action)
			}
			v.toAction[key] = e.action
		}
	}
	return v
}

// CanonicalWord returns an action's canonical command word (the full name listed first in
// its table entry), or false when the action is not in the vocabulary.
func (v *Vocabulary) CanonicalWord(action string) (string, bool) {
	w, ok := v.canonical[action]
	return w, ok
}

// Resolve maps a typed command word (case-insensitive) to its action id, returning false
// when the word is not in the vocabulary.
func (v *Vocabulary) Resolve(word string) (string, bool) {
	a, ok := v.toAction[strings.ToUpper(strings.TrimSpace(word))]
	return a, ok
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
