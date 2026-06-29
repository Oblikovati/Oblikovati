// SPDX-License-Identifier: GPL-2.0-only

package pdf

import "strconv"

// lexNumber reads an integer or real number token. PDF numbers carry no exponent; a
// malformed run parses as 0 (a defensive fallback — the surrounding operator simply
// receives a zero operand rather than the whole stream aborting).
func (l *lexer) lexNumber() token {
	start := l.pos
	for l.pos < len(l.data) {
		c := l.data[l.pos]
		if c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9') {
			l.pos++
			continue
		}
		break
	}
	v, _ := strconv.ParseFloat(string(l.data[start:l.pos]), 64)
	return token{kind: tokNumber, num: v}
}

// lexKeyword reads a bare word: an object keyword (obj/R/true/false/null/stream) or a
// content-stream operator (m, l, c, re, cm, S, …). It runs to the next whitespace or
// delimiter.
func (l *lexer) lexKeyword() token {
	start := l.pos
	for l.pos < len(l.data) && !isWhitespace(l.data[l.pos]) && !isDelimiter(l.data[l.pos]) {
		l.pos++
	}
	if l.pos == start { // a stray delimiter we don't special-case (e.g. ')','{'); consume one byte.
		l.pos++
		return token{kind: tokKeyword, text: string(l.data[start:l.pos])}
	}
	return token{kind: tokKeyword, text: string(l.data[start:l.pos])}
}

// lexName reads a /Name token, decoding #xx hex escapes (PDF 1.2+). The returned text
// omits the leading slash.
func (l *lexer) lexName() token {
	l.pos++ // consume '/'
	var b []byte
	for l.pos < len(l.data) {
		c := l.data[l.pos]
		if isWhitespace(c) || isDelimiter(c) {
			break
		}
		if c == '#' && l.pos+2 < len(l.data) {
			if h, ok := hexByte(l.data[l.pos+1], l.data[l.pos+2]); ok {
				b = append(b, h)
				l.pos += 3
				continue
			}
		}
		b = append(b, c)
		l.pos++
	}
	return token{kind: tokName, text: string(b)}
}

// lexLiteralString reads a (…) string, honouring balanced parentheses, backslash escapes,
// and line-continuations. The bytes are returned decoded; we never interpret them as
// geometry (text is skipped) but must consume them correctly so operators stay aligned.
func (l *lexer) lexLiteralString() token {
	l.pos++ // consume '('
	var b []byte
	depth := 1
	for l.pos < len(l.data) {
		c := l.data[l.pos]
		l.pos++
		switch c {
		case '\\':
			b = l.appendEscaped(b)
		case '(':
			depth++
			b = append(b, c)
		case ')':
			depth--
			if depth == 0 {
				return token{kind: tokString, str: b}
			}
			b = append(b, c)
		default:
			b = append(b, c)
		}
	}
	return token{kind: tokString, str: b}
}

// appendEscaped consumes and appends a backslash escape's resolved byte (no-op at EOF).
func (l *lexer) appendEscaped(b []byte) []byte {
	if l.pos < len(l.data) {
		b = append(b, l.escape(l.data[l.pos]))
		l.pos++
	}
	return b
}

// escape resolves a backslash escape's payload byte (the common control escapes; an
// unknown escape yields the literal byte, per the spec).
func (l *lexer) escape(c byte) byte {
	switch c {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	default:
		return c
	}
}

// lexHexString reads a <…> hex string. Whitespace between digits is ignored; an odd final
// digit is padded with 0 (spec §7.3.4.3).
func (l *lexer) lexHexString() token {
	l.pos++ // consume '<'
	var b []byte
	hi := -1
	for l.pos < len(l.data) {
		c := l.data[l.pos]
		l.pos++
		if c == '>' {
			break
		}
		if isWhitespace(c) {
			continue
		}
		v := hexDigit(c)
		if v < 0 {
			continue
		}
		if hi < 0 {
			hi = v
		} else {
			b = append(b, byte(hi<<4|v))
			hi = -1
		}
	}
	if hi >= 0 {
		b = append(b, byte(hi<<4))
	}
	return token{kind: tokString, str: b}
}

// hexDigit returns the value of a hex digit, or -1 if c is not one.
func hexDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// hexByte combines two hex digits into a byte (for #xx name escapes).
func hexByte(a, b byte) (byte, bool) {
	hi, lo := hexDigit(a), hexDigit(b)
	if hi < 0 || lo < 0 {
		return 0, false
	}
	return byte(hi<<4 | lo), true
}
