// SPDX-License-Identifier: GPL-2.0-only

package textbuf

// Rune classes for word-wise caret movement: a word run (Lua identifier runes), a run of
// other non-space punctuation, and whitespace (handled separately as a skip). Splitting these
// makes WordLeft/WordRight stop at the boundaries a programmer expects — between `foo` and
// `.bar`, not in the middle of either.
const (
	classOther = iota
	classWord
	classSpace
)

// isSpace reports whether r is an in-line blank (space or tab); newlines never appear inside a
// buffer line.
func isSpace(r rune) bool { return r == ' ' || r == '\t' }

// isWord reports whether r is a Lua identifier rune ([A-Za-z0-9_]).
func isWord(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	default:
		return r == '_'
	}
}

// runeClass maps a rune to its movement class.
func runeClass(r rune) int {
	switch {
	case isSpace(r):
		return classSpace
	case isWord(r):
		return classWord
	default:
		return classOther
	}
}

// classAt returns the movement class of the rune at p, or classSpace when p is at (or past)
// end-of-line so callers treat the line boundary as a stop.
func (b *Buffer) classAt(p Position) int {
	if p.Col >= b.LineLen(p.Line) {
		return classSpace
	}
	return runeClass(b.runeAt(p))
}

// sameClass reports whether rune r belongs to class cls (and is not whitespace, which word
// movement always treats as a separate skip).
func sameClass(r rune, cls int) bool { return cls != classSpace && runeClass(r) == cls }
