// SPDX-License-Identifier: GPL-2.0-only

package editor

import "oblikovati.org/script/console/textbuf"

// EnclosingCall returns the dotted identifier chain of the call the caret is currently inside —
// the name just before the nearest unmatched '(' or '{' on the caret's line — for the editor's
// signature help. ok is false when the caret is not within a call. The chain is left-to-right,
// e.g. inside `oblikovati.sketch.rectangle{ w` it returns {"oblikovati","sketch","rectangle"}.
// Scanning is limited to the caret's line, which covers the console's typical one-line calls.
func (m *Model) EnclosingCall() (chain []string, ok bool) {
	line := []rune(m.buf.Line(m.sel.Caret.Line))
	open := enclosingOpen(line, m.sel.Caret.Col)
	if open < 0 {
		return nil, false
	}
	chain = dottedChainEndingAt(line, open)
	return chain, len(chain) > 0
}

// MethodChainAt returns the dotted identifier chain whose final segment covers position p — the
// word under the mouse plus everything dotted to its left — for hover docs. Hovering `create` in
// `oblikovati.documents.create` yields {"oblikovati","documents","create"}; hovering off any word
// yields ok=false.
func (m *Model) MethodChainAt(p textbuf.Position) (chain []string, ok bool) {
	line := []rune(m.buf.Line(p.Line))
	end := wordEndAt(line, p.Col)
	if end == 0 {
		return nil, false
	}
	chain = dottedChainEndingAt(line, end)
	return chain, len(chain) > 0
}

// wordEndAt returns the end index of the identifier word covering col, or 0 when col is not on a
// word. col may sit anywhere within the word or just past its last rune.
func wordEndAt(line []rune, col int) int {
	if col < 0 || col > len(line) {
		return 0
	}
	end := col
	for end < len(line) && isIdentRune(line[end]) {
		end++
	}
	if end == 0 || !isIdentRune(line[end-1]) {
		return 0
	}
	return end
}

// enclosingOpen scans left from col tracking bracket nesting and returns the index of the
// nearest '(' or '{' that encloses col (is not yet closed), or -1 when the caret is not inside a
// bracket on this line.
func enclosingOpen(line []rune, col int) int {
	depth := 0
	for i := min(col, len(line)) - 1; i >= 0; i-- {
		switch line[i] {
		case ')', '}':
			depth++
		case '(', '{':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// dottedChainEndingAt reads the identifier chain `ident (. ident)*` ending just before index end
// (the call's opening bracket), returning it left-to-right. An empty result means no name
// precedes the bracket (e.g. an anonymous `(` grouping).
func dottedChainEndingAt(line []rune, end int) []string {
	var chain []string
	i := end
	for {
		start := scanIdentLeft(line, i)
		if start == i {
			break // no identifier here
		}
		chain = append([]string{string(line[start:i])}, chain...)
		if start == 0 || line[start-1] != '.' {
			break
		}
		i = start - 1 // step over the '.'
	}
	return chain
}

// scanIdentLeft returns the start index of the identifier run ending just before col, or col
// when the preceding rune is not an identifier rune.
func scanIdentLeft(line []rune, col int) int {
	i := col
	for i > 0 && isIdentRune(line[i-1]) {
		i--
	}
	return i
}

// isIdentRune reports whether r can appear in a Lua identifier.
func isIdentRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	default:
		return r == '_'
	}
}
