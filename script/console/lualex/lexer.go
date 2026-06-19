// SPDX-License-Identifier: GPL-2.0-only

// Package lualex is a Lua 5.4 syntax tokenizer for the Script Console editor's highlighting.
// It works one line at a time and carries a small State between lines, so the editor can
// re-tokenize only the lines it draws while still colouring long strings and block comments
// that span many lines correctly. It is pure Go with no UI dependency, so token classification
// is unit-tested headlessly and the cgo renderer just maps a Token's Kind to a theme colour.
package lualex

// Kind is the syntactic class of a token, mapped by the renderer to a theme colour.
type Kind int

const (
	// KindIdent is a plain identifier (variable / field name) — the default text colour.
	KindIdent Kind = iota
	KindKeyword
	KindBuiltin // a standard-library global or library table (print, string, math, …)
	KindString
	KindNumber
	KindComment
	KindOperator
)

// Token is a coloured span on one line: [Start, End) are rune columns (matching textbuf's
// rune-based Positions), so multi-byte source highlights at the right visual columns.
type Token struct {
	Kind  Kind
	Start int
	End   int
}

// State carries cross-line context: when a long string or long comment opened on a previous
// line and has not yet closed, InLong is true, Level is the bracket's `=` count, and IsComment
// distinguishes `--[[ ]]` from `[[ ]]`. The zero State means "start of a fresh line, no
// pending construct" — the State to pass for the first line of a buffer.
type State struct {
	InLong    bool
	Level     int
	IsComment bool
}

// TokenizeLine tokenizes one line given the State left by the previous line and returns the
// line's tokens (in column order) plus the State to feed the next line. Whitespace produces no
// token; the renderer leaves those columns the default colour.
func TokenizeLine(line string, in State) ([]Token, State) {
	r := []rune(line)
	sc := &scanner{src: r}
	if in.InLong {
		if cont := sc.resumeLong(in); cont.InLong {
			return sc.toks, cont
		}
	}
	sc.scanTokens()
	return sc.toks, sc.carry
}

// scanner walks one line's runes, appending tokens. A pointer receiver keeps the growing token
// slice, cursor, and the carry State (set when a long bracket runs off the end of the line) in
// one place across the scan helpers (scan.go).
type scanner struct {
	src   []rune
	i     int
	toks  []Token
	carry State
}

// resumeLong consumes the tail of a long string/comment that opened on an earlier line. It
// returns the State for the next line: still-open (InLong true) when the closer is absent, or
// the zero State once closed, with the cursor advanced past the closer so scanTokens continues.
func (s *scanner) resumeLong(in State) State {
	end, closed := scanLongTail(s.src, 0, in.Level)
	kind := KindString
	if in.IsComment {
		kind = KindComment
	}
	s.emit(kind, 0, end)
	if !closed {
		return in
	}
	s.i = end
	return State{}
}

// scanTokens is the main per-line loop, dispatching on the rune at the cursor.
func (s *scanner) scanTokens() {
	for s.i < len(s.src) {
		switch r := s.src[s.i]; {
		case isSpaceRune(r):
			s.i++
		case r == '-' && s.peek(1) == '-':
			s.scanComment()
		case r == '"' || r == '\'':
			s.scanQuoted(r)
		case r == '[' && longBracketLevel(s.src, s.i) >= 0:
			s.scanLongString()
		case isDigit(r) || (r == '.' && isDigit(s.peek(1))):
			s.scanNumber()
		case isIdentStart(r):
			s.scanIdent()
		default:
			s.scanOperator()
		}
	}
}

// emit appends a token spanning [start, end); it drops empty spans so callers can emit
// unconditionally.
func (s *scanner) emit(k Kind, start, end int) {
	if end > start {
		s.toks = append(s.toks, Token{Kind: k, Start: start, End: end})
	}
}

// peek returns the rune n positions ahead of the cursor, or -1 past end-of-line.
func (s *scanner) peek(n int) rune {
	j := s.i + n
	if j < 0 || j >= len(s.src) {
		return -1
	}
	return s.src[j]
}
