// SPDX-License-Identifier: GPL-2.0-only

package textbuf

// Selection is a caret with an anchor. When Anchor == Caret it is a bare caret (no
// highlighted range); otherwise it spans [Anchor, Caret) in either direction. The editor holds
// one Selection; shift-movement moves Caret and leaves Anchor, plain movement collapses both.
type Selection struct {
	Anchor Position
	Caret  Position
}

// Empty reports whether nothing is selected (caret only).
func (s Selection) Empty() bool { return s.Anchor == s.Caret }

// Ordered returns the selection bounds in document order (start <= end), regardless of which
// end the caret is on — the form edit/delete primitives expect.
func (s Selection) Ordered() (start, end Position) {
	if s.Caret.Before(s.Anchor) {
		return s.Caret, s.Anchor
	}
	return s.Anchor, s.Caret
}

// Left returns the position one rune left of p, wrapping to the end of the previous line at a
// line start. At the document start it returns p unchanged.
func (b *Buffer) Left(p Position) Position {
	if p.Col > 0 {
		return Position{p.Line, p.Col - 1}
	}
	if p.Line > 0 {
		return Position{p.Line - 1, b.LineLen(p.Line - 1)}
	}
	return p
}

// Right returns the position one rune right of p, wrapping to the start of the next line at a
// line end. At the document end it returns p unchanged.
func (b *Buffer) Right(p Position) Position {
	if p.Col < b.LineLen(p.Line) {
		return Position{p.Line, p.Col + 1}
	}
	if p.Line < b.LineCount()-1 {
		return Position{p.Line + 1, 0}
	}
	return p
}

// Up moves the caret one line up, keeping goalCol (the column the caret is "aiming" for during
// a run of vertical moves) clamped to the target line's length. At the top line it snaps to
// column 0.
func (b *Buffer) Up(p Position, goalCol int) Position {
	if p.Line == 0 {
		return Position{0, 0}
	}
	return Position{p.Line - 1, clampInt(goalCol, 0, b.LineLen(p.Line-1))}
}

// Down moves the caret one line down, keeping goalCol clamped to the target line. At the
// bottom line it snaps to end-of-line.
func (b *Buffer) Down(p Position, goalCol int) Position {
	if p.Line >= b.LineCount()-1 {
		last := b.LineCount() - 1
		return Position{last, b.LineLen(last)}
	}
	return Position{p.Line + 1, clampInt(goalCol, 0, b.LineLen(p.Line+1))}
}

// LineEnd returns the end-of-line caret for p's line.
func (b *Buffer) LineEnd(p Position) Position { return Position{p.Line, b.LineLen(p.Line)} }

// LineHome returns the "smart home" target: the first non-blank column of p's line, or column
// 0 when the caret is already at (or before) that column — the toggle every code editor uses.
func (b *Buffer) LineHome(p Position) Position {
	indent := b.firstNonBlank(p.Line)
	if p.Col > indent {
		return Position{p.Line, indent}
	}
	return Position{p.Line, 0}
}

// DocStart and DocEnd return the first and last caret positions of the buffer.
func (b *Buffer) DocStart() Position { return Position{0, 0} }
func (b *Buffer) DocEnd() Position {
	last := b.LineCount() - 1
	return Position{last, b.LineLen(last)}
}

// firstNonBlank returns the column of the first non-space/tab rune on line i, or the line
// length when the line is all blank.
func (b *Buffer) firstNonBlank(i int) int {
	line := b.lines[i]
	for col, r := range line {
		if r != ' ' && r != '\t' {
			return col
		}
	}
	return len(line)
}

// WordLeft returns the start of the word before p: it steps left over trailing spaces, then
// over the run of like-classed runes (word vs. punctuation) to that run's start. At a line
// start it steps to the previous line's end. Movement reads the rune immediately left of the
// caret (the one Backspace would remove), so it stops between `foo` and ` bar`, not mid-word.
func (b *Buffer) WordLeft(p Position) Position {
	if p.Col == 0 {
		return b.Left(p)
	}
	q := b.skipLeftWhile(p, isSpace)
	if q.Col == 0 {
		return q
	}
	cls := runeClass(b.runeAt(b.Left(q)))
	for q.Col > 0 && runeClass(b.runeAt(b.Left(q))) == cls {
		q = Position{q.Line, q.Col - 1}
	}
	return q
}

// skipLeftWhile retreats within p's line while the rune immediately left of the caret matches
// pred, stopping at column 0.
func (b *Buffer) skipLeftWhile(p Position, pred func(rune) bool) Position {
	for p.Col > 0 && pred(b.runeAt(b.Left(p))) {
		p = Position{p.Line, p.Col - 1}
	}
	return p
}

// WordRight returns the start of the next word after p: it steps right over the current run,
// then over spaces, wrapping across line ends.
func (b *Buffer) WordRight(p Position) Position {
	q := p
	cls := b.classAt(q)
	for q.Col < b.LineLen(q.Line) && sameClass(b.runeAt(q), cls) {
		q = Position{q.Line, q.Col + 1}
	}
	if q.Col == b.LineLen(q.Line) {
		return b.Right(q)
	}
	return b.skipRightWhile(q, isSpace)
}

// skipRightWhile advances within p's line while the rune at the caret matches pred, stopping
// at line end.
func (b *Buffer) skipRightWhile(p Position, pred func(rune) bool) Position {
	for p.Col < b.LineLen(p.Line) && pred(b.runeAt(p)) {
		p = Position{p.Line, p.Col + 1}
	}
	return p
}

// WordStartAt returns the start column of the identifier word covering p, or p itself when p is
// not on a word rune — the left edge of a double-click selection.
func (b *Buffer) WordStartAt(p Position) Position {
	if !b.onWord(p) {
		return p
	}
	col := p.Col
	for col > 0 && isWord(b.lines[p.Line][col-1]) {
		col--
	}
	return Position{p.Line, col}
}

// WordEndAt returns the end column of the identifier word covering p, or p itself when p is not
// on a word rune — the right edge of a double-click selection.
func (b *Buffer) WordEndAt(p Position) Position {
	if !b.onWord(p) {
		return p
	}
	col := p.Col
	for col < b.LineLen(p.Line) && isWord(b.lines[p.Line][col]) {
		col++
	}
	return Position{p.Line, col}
}

// onWord reports whether p sits on an identifier rune (so a word actually exists there).
func (b *Buffer) onWord(p Position) bool {
	return p.Col < b.LineLen(p.Line) && isWord(b.runeAt(p))
}

// runeAt returns the rune at p (p.Col must be < line length).
func (b *Buffer) runeAt(p Position) rune { return b.lines[p.Line][p.Col] }
