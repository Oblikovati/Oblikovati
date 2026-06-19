// SPDX-License-Identifier: GPL-2.0-only

package editor

import (
	"strings"

	"oblikovati.org/script/console/textbuf"
)

// Match is one found occurrence: the half-open span [Start, End) on a single line.
type Match struct {
	Start textbuf.Position
	End   textbuf.Position
}

// Find returns every literal, case-sensitive occurrence of query in the document, in document
// order. Matches do not span line breaks (a console search is line-oriented); an empty query
// matches nothing.
func (m *Model) Find(query string) []Match {
	if query == "" {
		return nil
	}
	q := []rune(query)
	var out []Match
	for line := 0; line < m.buf.LineCount(); line++ {
		out = appendLineMatches(out, line, []rune(m.buf.Line(line)), q)
	}
	return out
}

// appendLineMatches appends every occurrence of q within line's runes to out.
func appendLineMatches(out []Match, line int, hay, q []rune) []Match {
	for col := 0; col+len(q) <= len(hay); col++ {
		if runesEqual(hay[col:col+len(q)], q) {
			out = append(out, Match{
				Start: textbuf.Position{Line: line, Col: col},
				End:   textbuf.Position{Line: line, Col: col + len(q)},
			})
			col += len(q) - 1 // non-overlapping
		}
	}
	return out
}

// SelectMatch moves the caret to a match, selecting it so Find-Next visibly steps through hits.
func (m *Model) SelectMatch(match Match) {
	m.sel = textbuf.Selection{Anchor: match.Start, Caret: match.End}
	m.goalCol = match.End.Col
}

// ReplaceAll replaces every occurrence of query with repl as a single undo step and returns the
// number replaced. It rebuilds the whole document text once, so even many replacements coalesce
// into one journalled change. A no-op (zero matches) leaves history untouched.
func (m *Model) ReplaceAll(query, repl string) int {
	if query == "" {
		return 0
	}
	old := m.buf.String()
	n := strings.Count(old, query)
	if n == 0 {
		return 0
	}
	a := m.buf.DocStart()
	c := m.buf.DocEnd()
	m.collapseTo(m.record(a, c, strings.ReplaceAll(old, query, repl)))
	return n
}

// runesEqual reports whether two equal-length rune slices match.
func runesEqual(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
