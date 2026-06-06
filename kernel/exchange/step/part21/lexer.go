// SPDX-License-Identifier: GPL-2.0-only

package part21

import "fmt"

// lexer scans a Part 21 byte stream into tokens. It tracks 1-based line/column so
// every parse error names the offending position. Comments (/* … */) and runs of
// whitespace are skipped between tokens.
type lexer struct {
	src    []byte
	pos    int
	line   int
	column int
}

// newLexer starts a lexer at the first byte (line 1, column 1).
func newLexer(src []byte) *lexer {
	return &lexer{src: src, pos: 0, line: 1, column: 1}
}

// next returns the next token, or an error citing the position of malformed input.
func (lx *lexer) next() (Token, error) {
	if err := lx.skipTrivia(); err != nil {
		return Token{}, err
	}
	if lx.pos >= len(lx.src) {
		return lx.token(TokEOF, ""), nil
	}
	c := lx.src[lx.pos]
	switch {
	case c == '#':
		return lx.lexRef()
	case c == '\'':
		return lx.lexString()
	case c == '.':
		return lx.lexEnum()
	case c == '+' || c == '-' || (c >= '0' && c <= '9'):
		return lx.lexNumber()
	case isKeywordStart(c):
		return lx.lexKeyword()
	default:
		return lx.lexPunct()
	}
}

// lexPunct lexes the single-character grammar punctuation.
func (lx *lexer) lexPunct() (Token, error) {
	c := lx.src[lx.pos]
	kind, ok := punctKind(c)
	if !ok {
		return Token{}, fmt.Errorf("part21: unexpected byte %q at %d:%d", string(c), lx.line, lx.column)
	}
	tok := lx.token(kind, string(c))
	lx.advance()
	return tok, nil
}

// punctKind maps a punctuation byte to its token kind.
func punctKind(c byte) (TokenKind, bool) {
	switch c {
	case '(':
		return TokLParen, true
	case ')':
		return TokRParen, true
	case ',':
		return TokComma, true
	case '=':
		return TokEquals, true
	case ';':
		return TokSemicolon, true
	case '*':
		return TokStar, true
	case '$':
		return TokDollar, true
	default:
		return 0, false
	}
}

// token builds a token stamped with the lexer's current position.
func (lx *lexer) token(kind TokenKind, text string) Token {
	return Token{Kind: kind, Text: text, Line: lx.line, Column: lx.column}
}

// advance consumes one byte, maintaining line/column.
func (lx *lexer) advance() {
	if lx.src[lx.pos] == '\n' {
		lx.line++
		lx.column = 1
	} else {
		lx.column++
	}
	lx.pos++
}

// peek returns the next byte ahead of the cursor, or 0 past the end.
func (lx *lexer) peek() byte {
	i := lx.pos + 1
	if i >= len(lx.src) {
		return 0
	}
	return lx.src[i]
}

// isKeywordStart reports whether c can begin a keyword (letter or underscore).
func isKeywordStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// isKeywordPart reports whether c can continue a keyword (alnum or underscore).
func isKeywordPart(c byte) bool {
	return isKeywordStart(c) || (c >= '0' && c <= '9')
}
