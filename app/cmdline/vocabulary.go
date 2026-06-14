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
	var v []vocabEntry
	v = append(v, sketchVocabulary()...)
	v = append(v, partVocabulary()...)
	v = append(v, surfaceVocabulary()...)
	v = append(v, workplaneVocabulary()...)
	v = append(v, appVocabulary()...)
	return v
}

// appVocabulary maps the application-wide editing/file actions (built-in action ids).
func appVocabulary() []vocabEntry {
	return []vocabEntry{
		{"file.save", []string{"SAVE", "QSAVE"}},
		{"edit.undo", []string{"UNDO", "U"}},
		{"edit.redo", []string{"REDO", "MREDO"}},
		{"tool.cancel", []string{"CANCEL"}},
	}
}

// sketchVocabulary maps AutoCAD 2D drawing/modify commands (and the sketch lifecycle) to
// Oblikovati sketch tools. The single-letter aliases follow AutoCAD's acad.pgp where they
// fit Oblikovati's verbs (e.g. L/LINE, C/CIRCLE, A/ARC, F/FILLET, O/OFFSET).
func sketchVocabulary() []vocabEntry {
	return append(sketchDrawVocabulary(), sketchModifyVocabulary()...)
}

// sketchDrawVocabulary maps the 2D geometry-creation commands.
func sketchDrawVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Sketch.Line", []string{"LINE", "L"}},
		{"Sketch.Circle", []string{"CIRCLE", "C"}},
		{"Sketch.Arc", []string{"ARC", "A"}},
		{"Sketch.Rectangle", []string{"RECTANG", "RECTANGLE", "REC"}},
		{"Sketch.Point", []string{"POINT", "PO"}},
		{"Sketch.Polygon", []string{"POLYGON", "POL"}},
		{"Sketch.Ellipse", []string{"ELLIPSE", "EL"}},
		{"Sketch.Spline", []string{"SPLINE", "SPL"}},
		{"Sketch.Slot", []string{"SLOT"}},
		{"Sketch.Text", []string{"TEXT", "MTEXT", "T"}},
		{"Sketch.Dimension", []string{"DIMENSION", "DIM"}},
		{"Sketch.AutoDimension", []string{"AUTODIMENSION", "AUTODIM"}},
		{"Sketch.Project", []string{"PROJECT", "PROJECTGEOMETRY"}},
		{"Sketch.Create2D", []string{"SKETCH2D", "NEWSKETCH"}},
		{"Sketch.Create3D", []string{"SKETCH3D"}},
		{"Sketch.Finish", []string{"FINISHSKETCH", "FINISH"}},
	}
}

// sketchModifyVocabulary maps the 2D editing commands.
func sketchModifyVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Sketch.Fillet", []string{"FILLET", "F"}},
		{"Sketch.Chamfer", []string{"CHAMFER", "CHA"}},
		{"Sketch.Trim", []string{"TRIM", "TR"}},
		{"Sketch.Extend", []string{"EXTEND", "EX"}},
		{"Sketch.Offset", []string{"OFFSET", "O"}},
		{"Sketch.Mirror", []string{"MIRROR", "MI"}},
		{"Sketch.Move", []string{"MOVE", "M"}},
		{"Sketch.Copy", []string{"COPY", "CO", "CP"}},
		{"Sketch.Rotate", []string{"ROTATE", "RO"}},
		{"Sketch.Scale", []string{"SCALE", "SC"}},
		{"Sketch.Stretch", []string{"STRETCH", "S"}},
		{"Sketch.Split", []string{"BREAK", "BR"}},
		{"Sketch.RectangularPattern", []string{"ARRAYRECT"}},
		{"Sketch.CircularPattern", []string{"ARRAYPOLAR"}},
	}
}

// partVocabulary maps AutoCAD 3D/solid commands to Oblikovati part features. The 3D edge
// blends use AutoCAD's solid-editing names (FILLETEDGE/CHAMFEREDGE) so they stay distinct
// from the 2D sketch FILLET/CHAMFER.
func partVocabulary() []vocabEntry {
	return append(partCreateVocabulary(), partModifyVocabulary()...)
}

// partCreateVocabulary maps the solid-creation features.
func partCreateVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Create.Extrude", []string{"EXTRUDE", "EXT", "E"}},
		{"Create.Revolve", []string{"REVOLVE", "REV"}},
		{"Create.Sweep", []string{"SWEEP", "SW"}},
		{"Create.Loft", []string{"LOFT"}},
		{"Create.Coil", []string{"HELIX", "COIL"}},
		{"Create.Rib", []string{"RIB"}},
		{"Create.Emboss", []string{"EMBOSS"}},
		{"Create.Decal", []string{"DECAL"}},
	}
}

// partModifyVocabulary maps the solid-editing features.
func partModifyVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Modify.Hole", []string{"HOLE"}},
		{"Modify.Boss", []string{"BOSS"}},
		{"Modify.Fillet", []string{"FILLETEDGE"}},
		{"Modify.Chamfer", []string{"CHAMFEREDGE"}},
		{"Modify.Shell", []string{"SHELL"}},
		{"Modify.Thread", []string{"THREAD"}},
		{"Modify.Thicken", []string{"THICKEN"}},
		{"Modify.Draft", []string{"DRAFT", "TAPER"}},
		{"Modify.FaceOffset", []string{"OFFSETFACE"}},
		{"Modify.DeleteFace", []string{"DELETEFACE"}},
		{"Modify.ReplaceFace", []string{"REPLACEFACE"}},
	}
}

// surfaceVocabulary maps AutoCAD surface commands (SURF*) to Oblikovati surface features.
func surfaceVocabulary() []vocabEntry {
	return []vocabEntry{
		{"Surface.Patch", []string{"SURFPATCH", "PATCH"}},
		{"Surface.Trim", []string{"SURFTRIM"}},
		{"Surface.Extend", []string{"SURFEXTEND"}},
		{"Surface.Offset", []string{"SURFOFFSET"}},
		{"Surface.Ruled", []string{"RULESURF", "RULED"}},
		{"Surface.Stitch", []string{"SURFSTITCH", "STITCH"}},
		{"Surface.Sculpt", []string{"SCULPT"}},
		{"Surface.MidSurface", []string{"MIDSURFACE"}},
	}
}

// workplaneVocabulary maps the work-plane (datum) constructors. AutoCAD has no direct
// equivalent, so these use Oblikovati-natural words.
func workplaneVocabulary() []vocabEntry {
	return []vocabEntry{
		{"WorkPlane.Offset", []string{"WORKPLANE", "PLANE"}},
		{"WorkPlane.Midplane", []string{"MIDPLANE"}},
		{"WorkPlane.ThreePoints", []string{"PLANE3P"}},
		{"WorkPlane.Tangent", []string{"TANPLANE"}},
		{"WorkPlane.NormalToAxis", []string{"NORMALPLANE"}},
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
