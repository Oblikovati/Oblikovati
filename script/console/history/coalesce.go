// SPDX-License-Identifier: GPL-2.0-only

package history

import "oblikovati.org/script/console/textbuf"

// coalesce tries to fold the just-made change c into the previous change prev in place,
// returning true when it did. A run of typed identifier runes, of backspaces, or of forward
// deletes collapses to one undo step; a space, tab or newline always starts a fresh step, so
// undo lands on word boundaries the way a programmer expects.
func coalesce(prev *Change, c Change) bool {
	return mergeInsert(prev, c) || mergeBackspace(prev, c) || mergeForwardDelete(prev, c)
}

// mergeInsert folds a single non-blank typed rune into a contiguous insertion run.
func mergeInsert(prev *Change, c Change) bool {
	if !isInsert(*prev) || !isInsert(c) || !singleRune(c.Inserted) || isBreakRune(c.Inserted) {
		return false
	}
	// A run that ended on a space/tab/newline is closed: the next rune starts a new step, so
	// "foo bar" undoes as bar / space / foo rather than folding the word onto the space.
	if endsOnBreak(prev.Inserted) {
		return false
	}
	if c.At != textbuf.Advance(prev.At, prev.Inserted) {
		return false
	}
	prev.Inserted += c.Inserted
	return true
}

// mergeBackspace folds a single-rune deletion that abuts the previous deletion on its left
// (the Backspace direction) into one growing-leftward removal.
func mergeBackspace(prev *Change, c Change) bool {
	if !isDelete(*prev) || !isDelete(c) || !singleRune(c.Removed) || c.Removed == "\n" {
		return false
	}
	if textbuf.Advance(c.At, c.Removed) != prev.At {
		return false
	}
	prev.At = c.At
	prev.Removed = c.Removed + prev.Removed
	return true
}

// mergeForwardDelete folds a single-rune deletion at the same anchor (the Delete-key
// direction, where following text shifts back into the caret) into the previous removal.
func mergeForwardDelete(prev *Change, c Change) bool {
	if !isDelete(*prev) || !isDelete(c) || !singleRune(c.Removed) || c.Removed == "\n" {
		return false
	}
	if c.At != prev.At {
		return false
	}
	prev.Removed += c.Removed
	return true
}

// isInsert / isDelete classify a Change as a pure insertion or pure deletion (a replacement
// that both removes and inserts is neither, so it is never coalesced).
func isInsert(c Change) bool { return c.Removed == "" && c.Inserted != "" }
func isDelete(c Change) bool { return c.Inserted == "" && c.Removed != "" }

// singleRune reports whether s is exactly one rune.
func singleRune(s string) bool { return len([]rune(s)) == 1 }

// isBreakRune reports whether the single-rune string s ends a coalescing run (whitespace or
// newline), so the next keystroke begins a new undo step.
func isBreakRune(s string) bool {
	switch s {
	case " ", "\t", "\n":
		return true
	default:
		return false
	}
}

// endsOnBreak reports whether s's last rune is a break rune (the run is already closed).
func endsOnBreak(s string) bool {
	r := []rune(s)
	return len(r) > 0 && isBreakRune(string(r[len(r)-1]))
}
