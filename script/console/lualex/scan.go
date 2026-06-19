// SPDX-License-Identifier: GPL-2.0-only

package lualex

// scanComment consumes a `--` comment. A `--[[`/`--[==[` opener begins a long comment that may
// span lines (handled like a long string); otherwise it runs to end-of-line.
func (s *scanner) scanComment() {
	start := s.i
	level := longBracketLevel(s.src, s.i+2)
	if level < 0 {
		s.emit(KindComment, start, len(s.src))
		s.i = len(s.src)
		return
	}
	s.consumeLong(start, s.i+2, level, KindComment, true)
}

// scanLongString consumes a `[[`/`[==[` long string literal, possibly spanning lines.
func (s *scanner) scanLongString() {
	start := s.i
	level := longBracketLevel(s.src, s.i)
	s.consumeLong(start, s.i, level, KindString, false)
}

// consumeLong scans the body of a long bracket that opened at tokenStart, with the opener
// ending at openEnd and the given level. It emits one token of kind k from tokenStart to the
// closer (or end-of-line) and, when the line ends first, records the open State so the next
// line resumes via resumeLong. isComment tags the carried State.
func (s *scanner) consumeLong(tokenStart, openEnd, level int, k Kind, isComment bool) {
	end, closed := scanLongTail(s.src, openEnd, level)
	s.emit(k, tokenStart, end)
	if !closed {
		s.carry = State{InLong: true, Level: level, IsComment: isComment}
		s.i = len(s.src)
		return
	}
	s.i = end
}

// scanQuoted consumes a single- or double-quoted string starting at the quote q, honouring
// backslash escapes and stopping at the matching quote or end-of-line (an unterminated string
// simply ends at the line, which the diagnostics pass will flag separately).
func (s *scanner) scanQuoted(q rune) {
	start := s.i
	j := s.i + 1
	for j < len(s.src) {
		switch s.src[j] {
		case '\\':
			j += 2
			continue
		case q:
			j++
			s.emit(KindString, start, j)
			s.i = j
			return
		}
		j++
	}
	s.emit(KindString, start, len(s.src))
	s.i = len(s.src)
}

// scanNumber consumes a numeric literal: hex (0x… with optional hex fraction/exponent) or
// decimal with optional fraction and exponent. It is liberal — exact validity is the parser's
// job; the lexer only needs the span to colour.
func (s *scanner) scanNumber() {
	start := s.i
	if s.src[s.i] == '0' && (s.peekNext() == 'x' || s.peekNext() == 'X') {
		s.i += 2
		s.consumeWhile(func(r rune) bool { return isHexDigit(r) || r == '.' })
		s.consumeExponent('p', 'P')
	} else {
		s.consumeWhile(func(r rune) bool { return isDigit(r) || r == '.' })
		s.consumeExponent('e', 'E')
	}
	s.emit(KindNumber, start, s.i)
}

// consumeExponent advances over an exponent suffix (lo/hi marker then optional sign and digits)
// when present at the cursor.
func (s *scanner) consumeExponent(lo, hi rune) {
	if s.i >= len(s.src) || (s.src[s.i] != lo && s.src[s.i] != hi) {
		return
	}
	s.i++
	if s.i < len(s.src) && (s.src[s.i] == '+' || s.src[s.i] == '-') {
		s.i++
	}
	s.consumeWhile(isDigit)
}

// scanIdent consumes an identifier and classifies it as a keyword, a known builtin, or a plain
// identifier.
func (s *scanner) scanIdent() {
	start := s.i
	s.i++
	s.consumeWhile(isIdentPart)
	s.emit(classifyWord(string(s.src[start:s.i])), start, s.i)
}

// scanOperator emits a single-rune operator/punctuation token (multi-rune operators are still
// coloured the same, so one rune per token is sufficient for highlighting).
func (s *scanner) scanOperator() {
	s.emit(KindOperator, s.i, s.i+1)
	s.i++
}

// consumeWhile advances the cursor over runes satisfying pred.
func (s *scanner) consumeWhile(pred func(rune) bool) {
	for s.i < len(s.src) && pred(s.src[s.i]) {
		s.i++
	}
}

// longBracketLevel returns the `=` count of a long-bracket opener at position i (`[` `=`*n `[`),
// or -1 when there is no opener there. Level 0 is `[[`, level 2 is `[==[`.
func longBracketLevel(src []rune, i int) int {
	if i >= len(src) || src[i] != '[' {
		return -1
	}
	j := i + 1
	for j < len(src) && src[j] == '=' {
		j++
	}
	if j < len(src) && src[j] == '[' {
		return j - (i + 1)
	}
	return -1
}

// scanLongTail scans from `from` for a long-bracket closer `]` `=`*level `]`, returning the
// index just past the closer and true, or len(src) and false when the line ends first.
func scanLongTail(src []rune, from, level int) (end int, closed bool) {
	for j := from; j < len(src); j++ {
		if src[j] != ']' {
			continue
		}
		k := j + 1
		for k < len(src) && src[k] == '=' {
			k++
		}
		if k-(j+1) == level && k < len(src) && src[k] == ']' {
			return k + 1, true
		}
	}
	return len(src), false
}

// isDigit / isHexDigit / isSpaceRune / isIdentStart / isIdentPart classify single runes for
// the scanners.
func isDigit(r rune) bool     { return r >= '0' && r <= '9' }
func isSpaceRune(r rune) bool { return r == ' ' || r == '\t' }
func isHexDigit(r rune) bool {
	return isDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}
func isIdentStart(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
func isIdentPart(r rune) bool { return isIdentStart(r) || isDigit(r) }
