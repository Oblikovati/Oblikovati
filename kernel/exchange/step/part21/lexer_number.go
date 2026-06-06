// SPDX-License-Identifier: GPL-2.0-only

package part21

import "fmt"

// lexNumber scans an integer or real literal. A real has a fractional '.' and/or an
// exponent ('E'/'e'); otherwise it is an integer. Part 21 reals always carry a '.'.
func (lx *lexer) lexNumber() (Token, error) {
	line, col := lx.line, lx.column
	start := lx.pos
	lx.consumeSign()
	if !lx.consumeDigits() {
		return Token{}, fmt.Errorf("part21: number with no digits at %d:%d", line, col)
	}
	isReal := lx.consumeFraction()
	if lx.consumeExponent() {
		isReal = true
	}
	text := string(lx.src[start:lx.pos])
	kind := TokInt
	if isReal {
		kind = TokReal
	}
	return Token{Kind: kind, Text: text, Line: line, Column: col}, nil
}

// consumeSign consumes an optional leading +/-.
func (lx *lexer) consumeSign() {
	if lx.pos < len(lx.src) && (lx.src[lx.pos] == '+' || lx.src[lx.pos] == '-') {
		lx.advance()
	}
}

// consumeDigits consumes a run of decimal digits, reporting whether any were seen.
func (lx *lexer) consumeDigits() bool {
	any := false
	for lx.pos < len(lx.src) && lx.src[lx.pos] >= '0' && lx.src[lx.pos] <= '9' {
		lx.advance()
		any = true
	}
	return any
}

// consumeFraction consumes a '.' followed by optional digits, reporting whether a
// fractional part was present (which makes the literal a real).
func (lx *lexer) consumeFraction() bool {
	if lx.pos >= len(lx.src) || lx.src[lx.pos] != '.' {
		return false
	}
	lx.advance() // '.'
	lx.consumeDigits()
	return true
}

// consumeExponent consumes an E/e exponent with optional sign, reporting presence.
func (lx *lexer) consumeExponent() bool {
	if lx.pos >= len(lx.src) || (lx.src[lx.pos] != 'E' && lx.src[lx.pos] != 'e') {
		return false
	}
	lx.advance() // 'E'
	lx.consumeSign()
	lx.consumeDigits()
	return true
}
