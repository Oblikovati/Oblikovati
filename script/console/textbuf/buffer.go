// SPDX-License-Identifier: GPL-2.0-only

// Package textbuf is the pure-Go text model behind the Script Console code editor: a
// rune-grid line buffer addressed by (line, column) Positions, plus caret/selection
// movement (cursor.go). It carries no UI or cgo dependency, so the editor's whole edit
// surface is unit-tested headlessly; the cgo widget (head/ui/code_editor.go) only forwards
// input events into these operations and reads back the buffer (lua-scripting-plan, ADR-0028).
package textbuf

import (
	"strings"

	"oblikovati.org/math"
)

// Position addresses a caret location: Line is a 0-based line index and Col is a 0-based
// rune offset within that line (Col == len(line runes) is the valid end-of-line caret).
// Columns count runes, not bytes, so multi-byte source (identifiers, string literals) maps
// to visual columns directly.
type Position struct {
	Line int
	Col  int
}

// Before reports whether p sorts strictly before q in document order.
func (p Position) Before(q Position) bool {
	if p.Line != q.Line {
		return p.Line < q.Line
	}
	return p.Col < q.Col
}

// Buffer is a mutable document stored as one rune slice per line. There is always at least
// one line (an empty document is a single empty line), so LineCount() >= 1 always holds and
// every Position with Line in [0, LineCount) is addressable.
type Buffer struct {
	lines [][]rune
}

// New builds a buffer from s, splitting on "\n" and stripping a trailing "\r" per line so
// CRLF and LF sources both load cleanly. An empty string yields a single empty line.
func New(s string) *Buffer {
	raw := strings.Split(s, "\n")
	lines := make([][]rune, len(raw))
	for i, line := range raw {
		lines[i] = []rune(strings.TrimSuffix(line, "\r"))
	}
	return &Buffer{lines: lines}
}

// LineCount returns the number of lines (always >= 1).
func (b *Buffer) LineCount() int { return len(b.lines) }

// Line returns line i as a string. It panics on an out-of-range index, naming the offending
// value, because every caller derives i from a clamped Position or a [0, LineCount) loop —
// an out-of-range index is a logic error, not user input.
func (b *Buffer) Line(i int) string {
	if i < 0 || i >= len(b.lines) {
		panic(lineRangeError(i, len(b.lines)))
	}
	return string(b.lines[i])
}

// LineLen returns the rune length of line i (the valid end-of-line column).
func (b *Buffer) LineLen(i int) int {
	if i < 0 || i >= len(b.lines) {
		panic(lineRangeError(i, len(b.lines)))
	}
	return len(b.lines[i])
}

// String renders the whole buffer back to a single "\n"-joined string (the form Run sends to
// the Lua engine and the editor persists).
func (b *Buffer) String() string {
	parts := make([]string, len(b.lines))
	for i, line := range b.lines {
		parts[i] = string(line)
	}
	return strings.Join(parts, "\n")
}

// Clamp returns p constrained to a real location in the buffer: Line into [0, LineCount) and
// Col into [0, LineLen]. Callers clamp externally-supplied Positions (e.g. a mouse hit) before
// editing so the edit primitives can assume valid input.
func (b *Buffer) Clamp(p Position) Position {
	if p.Line < 0 {
		return Position{0, 0}
	}
	if p.Line >= len(b.lines) {
		last := len(b.lines) - 1
		return Position{last, len(b.lines[last])}
	}
	return Position{p.Line, math.Clamp(p.Col, 0, len(b.lines[p.Line]))}
}

// Insert inserts text at p (assumed already valid) and returns the caret Position just past
// the inserted text. Embedded "\n" split the current line, so pasting multi-line text works
// through the same path as typing one rune.
func (b *Buffer) Insert(p Position, text string) Position {
	if text == "" {
		return p
	}
	segs := strings.Split(text, "\n")
	line := b.lines[p.Line]
	tail := append([]rune(nil), line[p.Col:]...)
	b.lines[p.Line] = append(line[:p.Col], []rune(segs[0])...)
	if len(segs) == 1 {
		end := Position{p.Line, p.Col + len([]rune(segs[0]))}
		b.lines[p.Line] = append(b.lines[p.Line], tail...)
		return end
	}
	return b.insertMultiline(p, segs, tail)
}

// insertMultiline completes a multi-line Insert: segs[0] is already appended to the split
// line; this stitches the middle lines and rejoins tail after the final segment, returning the
// caret at the end of that final segment.
func (b *Buffer) insertMultiline(p Position, segs []string, tail []rune) Position {
	mid := make([][]rune, len(segs)-1)
	for i, seg := range segs[1:] {
		mid[i] = []rune(seg)
	}
	lastIdx := len(mid) - 1
	endCol := len(mid[lastIdx])
	mid[lastIdx] = append(mid[lastIdx], tail...)
	b.lines = insertLines(b.lines, p.Line+1, mid)
	return Position{p.Line + len(segs) - 1, endCol}
}

// DeleteRange removes the half-open span [a, b) (a must not be after b; both assumed valid)
// and returns the removed text, so an undo record can restore it. The caret lands at a.
func (b *Buffer) DeleteRange(a, c Position) string {
	if a == c {
		return ""
	}
	removed := b.textInRange(a, c)
	head := append([]rune(nil), b.lines[a.Line][:a.Col]...)
	tail := append([]rune(nil), b.lines[c.Line][c.Col:]...)
	b.lines[a.Line] = append(head, tail...)
	if c.Line > a.Line {
		b.lines = append(b.lines[:a.Line+1], b.lines[c.Line+1:]...)
	}
	return removed
}

// ReplaceRange deletes [a, c) and inserts text at a, returning the caret past the insertion —
// the single primitive the editor and undo/redo route every mutation through.
func (b *Buffer) ReplaceRange(a, c Position, text string) Position {
	b.DeleteRange(a, c)
	return b.Insert(a, text)
}

// TextInRange returns the text of the half-open span [a, c) without mutating the buffer,
// ordering the endpoints first. It backs Copy/Cut and selection inspection.
func (b *Buffer) TextInRange(a, c Position) string {
	if c.Before(a) {
		a, c = c, a
	}
	return b.textInRange(a, c)
}

// textInRange returns the text of the half-open span [a, c) (assumed ordered and valid),
// joining intermediate lines with "\n".
func (b *Buffer) textInRange(a, c Position) string {
	if a.Line == c.Line {
		return string(b.lines[a.Line][a.Col:c.Col])
	}
	var sb strings.Builder
	sb.WriteString(string(b.lines[a.Line][a.Col:]))
	for ln := a.Line + 1; ln < c.Line; ln++ {
		sb.WriteByte('\n')
		sb.WriteString(string(b.lines[ln]))
	}
	sb.WriteByte('\n')
	sb.WriteString(string(b.lines[c.Line][:c.Col]))
	return sb.String()
}
