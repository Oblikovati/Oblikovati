// SPDX-License-Identifier: GPL-2.0-only

package param

import "fmt"

// tokenKind enumerates the lexical token categories of an expression.
type tokenKind int

const (
	tokEOF tokenKind = iota
	tokNumber
	tokIdent
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokLParen
	tokRParen
	tokComma
)

// token is a lexed token with its source position (byte offset) for error
// reporting.
type token struct {
	kind tokenKind
	text string
	pos  int
}

// lexer turns expression source into a token slice. It reports the position of
// any illegal character.
type lexer struct {
	src string
	pos int
}

// lex returns all tokens up to and including EOF, or a positioned error.
func lex(src string) ([]token, error) {
	l := &lexer{src: src}
	var tokens []token
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.kind == tokEOF {
			return tokens, nil
		}
	}
}

// next scans the next token, skipping leading whitespace.
func (l *lexer) next() (token, error) {
	l.skipSpaces()
	if l.pos >= len(l.src) {
		return token{kind: tokEOF, pos: l.pos}, nil
	}
	start := l.pos
	c := l.src[l.pos]
	switch {
	case isDigit(c) || c == '.':
		return l.lexNumber(), nil
	case isIdentStart(c):
		return l.lexIdent(), nil
	default:
		return l.lexOperator(start, c)
	}
}

// lexOperator scans a single-character operator/punctuation token.
func (l *lexer) lexOperator(start int, c byte) (token, error) {
	kind, ok := operatorKinds[c]
	if !ok {
		return token{}, fmt.Errorf("param: unexpected character %q at position %d", string(c), start)
	}
	l.pos++
	return token{kind: kind, text: string(c), pos: start}, nil
}

// operatorKinds maps a punctuation byte to its token kind.
var operatorKinds = map[byte]tokenKind{
	'+': tokPlus, '-': tokMinus, '*': tokStar, '/': tokSlash,
	'(': tokLParen, ')': tokRParen, ',': tokComma,
}

// lexNumber scans a float literal (digits, optional fraction and exponent).
func (l *lexer) lexNumber() token {
	start := l.pos
	for l.pos < len(l.src) && isNumberByte(l.src[l.pos], l.pos-start, l.src[start:]) {
		l.pos++
	}
	return token{kind: tokNumber, text: l.src[start:l.pos], pos: start}
}

// lexIdent scans an identifier (parameter name, function name, or unit name).
func (l *lexer) lexIdent() token {
	start := l.pos
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		l.pos++
	}
	return token{kind: tokIdent, text: l.src[start:l.pos], pos: start}
}

func (l *lexer) skipSpaces() {
	for l.pos < len(l.src) && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t') {
		l.pos++
	}
}

func isDigit(c byte) bool      { return c >= '0' && c <= '9' }
func isIdentStart(c byte) bool { return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isIdentPart(c byte) bool  { return isIdentStart(c) || isDigit(c) }
