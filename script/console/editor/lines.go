// SPDX-License-Identifier: GPL-2.0-only

package editor

import (
	"strings"

	"oblikovati.org/script/console/textbuf"
)

// lineCommentPrefix is what ToggleLineComment adds/removes; the trailing space keeps commented
// code readable and is tolerated (but not required) on removal.
const lineCommentPrefix = "-- "

// indentStep is one block-indent level (four spaces — Lua convention, no hard tabs).
const indentStep = "    "

// ToggleLineComment comments the lines spanning the selection (or the caret line) when any is
// uncommented, and uncomments them when all non-blank lines are already commented — the usual
// Ctrl-/ behaviour. The whole change is one undo step and the affected lines stay selected so
// the shortcut can be pressed repeatedly.
func (m *Model) ToggleLineComment() {
	first, last := m.selectedLineRange()
	lines := m.linesIn(first, last)
	if allCommented(lines) {
		m.replaceLines(first, last, mapLines(lines, uncommentLine))
		return
	}
	col := minIndent(lines)
	m.replaceLines(first, last, mapLines(lines, func(s string) string { return commentLineAt(s, col) }))
}

// IndentSelection prefixes one indent step to every line spanning the selection; OutdentSelection
// removes up to one step of leading whitespace from each. Both keep the block selected.
func (m *Model) IndentSelection() {
	first, last := m.selectedLineRange()
	lines := m.linesIn(first, last)
	m.replaceLines(first, last, mapLines(lines, func(s string) string { return indentStep + s }))
}

func (m *Model) OutdentSelection() {
	first, last := m.selectedLineRange()
	lines := m.linesIn(first, last)
	m.replaceLines(first, last, mapLines(lines, outdentLine))
}

// selectedLineRange returns the first and last line indices the selection touches. A selection
// that ends exactly at the start of a line does not pull that line in (the caret sits before it).
func (m *Model) selectedLineRange() (first, last int) {
	start, end := m.sel.Ordered()
	last = end.Line
	if end.Col == 0 && end.Line > start.Line {
		last--
	}
	return start.Line, last
}

// linesIn returns the buffer lines in [first, last].
func (m *Model) linesIn(first, last int) []string {
	out := make([]string, 0, last-first+1)
	for i := first; i <= last; i++ {
		out = append(out, m.buf.Line(i))
	}
	return out
}

// replaceLines swaps lines [first,last] for newLines as one journalled edit and re-selects the
// resulting block so repeated indent/comment keystrokes keep operating on it.
func (m *Model) replaceLines(first, last int, newLines []string) {
	a := textbuf.Position{Line: first, Col: 0}
	c := textbuf.Position{Line: last, Col: m.buf.LineLen(last)}
	m.record(a, c, strings.Join(newLines, "\n"))
	endLine := first + len(newLines) - 1
	m.sel = textbuf.Selection{
		Anchor: textbuf.Position{Line: first, Col: 0},
		Caret:  textbuf.Position{Line: endLine, Col: len([]rune(newLines[len(newLines)-1]))},
	}
	m.goalCol = m.sel.Caret.Col
}

// mapLines applies f to every line.
func mapLines(lines []string, f func(string) string) []string {
	out := make([]string, len(lines))
	for i, s := range lines {
		out[i] = f(s)
	}
	return out
}

// allCommented reports whether every non-blank line is already a line comment (so the toggle
// should uncomment rather than nest another comment).
func allCommented(lines []string) bool {
	any := false
	for _, s := range lines {
		t := strings.TrimLeft(s, " \t")
		if t == "" {
			continue
		}
		any = true
		if !strings.HasPrefix(t, "--") {
			return false
		}
	}
	return any
}

// commentLineAt inserts the comment prefix at column col (the block's shared indent), leaving a
// blank line untouched so toggling does not litter empty lines with comments.
func commentLineAt(s string, col int) string {
	if strings.TrimSpace(s) == "" {
		return s
	}
	r := []rune(s)
	if col > len(r) {
		col = len(r)
	}
	return string(r[:col]) + lineCommentPrefix + string(r[col:])
}

// uncommentLine removes the first "-- " (or "--") after the leading whitespace.
func uncommentLine(s string) string {
	indent := s[:len(s)-len(strings.TrimLeft(s, " \t"))]
	body := s[len(indent):]
	if !strings.HasPrefix(body, "--") {
		return s
	}
	body = strings.TrimPrefix(body[2:], " ")
	return indent + body
}

// outdentLine removes up to one indent step of leading spaces, or a single leading tab.
func outdentLine(s string) string {
	if strings.HasPrefix(s, "\t") {
		return s[1:]
	}
	n := 0
	for n < len(indentStep) && n < len(s) && s[n] == ' ' {
		n++
	}
	return s[n:]
}

// minIndent returns the smallest leading-whitespace width among the non-blank lines, so a block
// comment is inserted at a shared column.
func minIndent(lines []string) int {
	min := -1
	for _, s := range lines {
		if strings.TrimSpace(s) == "" {
			continue
		}
		w := len([]rune(s)) - len([]rune(strings.TrimLeft(s, " \t")))
		if min == -1 || w < min {
			min = w
		}
	}
	if min < 0 {
		return 0
	}
	return min
}
