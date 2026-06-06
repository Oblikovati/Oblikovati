// SPDX-License-Identifier: GPL-2.0-only

package part21

import (
	"fmt"
	"strings"
)

// skipTrivia consumes whitespace and /* … */ comments. An unterminated comment is
// a lex error (it would otherwise silently swallow the rest of the file).
func (lx *lexer) skipTrivia() error {
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			lx.advance()
			continue
		}
		if c == '/' && lx.peek(1) == '*' {
			if err := lx.skipComment(); err != nil {
				return err
			}
			continue
		}
		return nil
	}
	return nil
}

// skipComment consumes a /* … */ block (already positioned on '/').
func (lx *lexer) skipComment() error {
	startLine, startCol := lx.line, lx.column
	lx.advance() // '/'
	lx.advance() // '*'
	for lx.pos < len(lx.src) {
		if lx.src[lx.pos] == '*' && lx.peek(1) == '/' {
			lx.advance() // '*'
			lx.advance() // '/'
			return nil
		}
		lx.advance()
	}
	return fmt.Errorf("part21: unterminated comment opened at %d:%d", startLine, startCol)
}

// lexKeyword scans a bare identifier/keyword. A '-' is absorbed only when it joins
// two identifier runs (the boilerplate markers ISO-10303-21 / END-ISO-10303-21);
// it is never a leading or trailing character, so the grammar's other uses of '-'
// (negative numbers) are unaffected.
func (lx *lexer) lexKeyword() (Token, error) {
	tok := lx.token(TokKeyword, "")
	start := lx.pos
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		if isKeywordPart(c) {
			lx.advance()
			continue
		}
		if c == '-' && isKeywordPart(lx.peek(1)) {
			lx.advance()
			continue
		}
		break
	}
	tok.Text = string(lx.src[start:lx.pos])
	return tok, nil
}

// lexRef scans an entity reference #123.
func (lx *lexer) lexRef() (Token, error) {
	tok := lx.token(TokRef, "")
	start := lx.pos
	lx.advance() // '#'
	for lx.pos < len(lx.src) && lx.src[lx.pos] >= '0' && lx.src[lx.pos] <= '9' {
		lx.advance()
	}
	if lx.pos == start+1 {
		return Token{}, fmt.Errorf("part21: '#' with no digits at %d:%d", tok.Line, tok.Column)
	}
	tok.Text = string(lx.src[start:lx.pos])
	return tok, nil
}

// lexEnum scans a dotted enumeration .T. .F. .STEEL.; a lone '.' is invalid here
// because numbers are dispatched separately.
func (lx *lexer) lexEnum() (Token, error) {
	tok := lx.token(TokEnum, "")
	start := lx.pos
	lx.advance() // leading '.'
	for lx.pos < len(lx.src) && lx.src[lx.pos] != '.' {
		lx.advance()
	}
	if lx.pos >= len(lx.src) {
		return Token{}, fmt.Errorf("part21: unterminated enumeration opened at %d:%d", tok.Line, tok.Column)
	}
	lx.advance() // trailing '.'
	tok.Text = string(lx.src[start:lx.pos])
	if len(tok.Text) < 3 {
		return Token{}, fmt.Errorf("part21: empty enumeration %q at %d:%d", tok.Text, tok.Line, tok.Column)
	}
	return tok, nil
}

// lexString scans a single-quoted string, decoding ” (escaped quote) and the
// Part 21 \X\ / \X2\ control/unicode escapes are passed through verbatim except ”.
func (lx *lexer) lexString() (Token, error) {
	tok := lx.token(TokString, "")
	lx.advance() // opening quote
	var b strings.Builder
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		if c == '\'' {
			if lx.peek(1) == '\'' { // doubled quote ⇒ literal quote
				b.WriteByte('\'')
				lx.advance()
				lx.advance()
				continue
			}
			lx.advance() // closing quote
			tok.Text = b.String()
			return tok, nil
		}
		b.WriteByte(c)
		lx.advance()
	}
	return Token{}, fmt.Errorf("part21: unterminated string opened at %d:%d", tok.Line, tok.Column)
}
