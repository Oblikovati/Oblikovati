// SPDX-License-Identifier: GPL-2.0-only

// Package pdf is a clean-room, scoped reader for vector PDFs whose page content was
// generated from a CAD drawing (e.g. an AutoCAD plot-to-PDF). It decodes the page
// content streams into the format-neutral drawing model (kernel/exchange/drawing) so
// the shared Sketch importer can place them like any other drawing format.
//
// Scope (deliberately narrow): classic cross-reference tables (no xref streams / object
// streams), FlateDecode content streams, and the path-construction/painting subset of
// the content-stream operator set under the current transformation matrix. Text and
// raster images are skipped; only vector paths become geometry. Anything outside this
// subset is reported as a warning rather than failing the whole import.
package pdf

// tokenKind classifies one lexical token of PDF syntax (object syntax and the
// content-stream operator syntax share this lexer).
type tokenKind int

const (
	tokEOF        tokenKind = iota
	tokNumber               // an integer or real number (value in token.num)
	tokName                 // a /Name (token.text without the leading slash)
	tokString               // a (literal) or <hex> string (decoded bytes in token.str)
	tokKeyword              // a bare word: obj, R, true, stream, or a content operator (m, l, S, …)
	tokDictOpen             // <<
	tokDictClose            // >>
	tokArrayOpen            // [
	tokArrayClose           // ]
)

// token is one lexed unit. Only the field matching kind is meaningful.
type token struct {
	kind tokenKind
	num  float64
	text string
	str  []byte
}

// lexer scans PDF bytes one token at a time over a cursor, so a caller (the object
// parser or the content interpreter) can also reach into the raw bytes directly — needed
// to grab a stream's body and to skip an inline image's binary payload.
type lexer struct {
	data []byte
	pos  int
}

func newLexer(data []byte) *lexer { return &lexer{data: data} }

// isWhitespace reports the PDF whitespace bytes (incl. NUL and form feed).
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == 0
}

// isDelimiter reports the PDF delimiter bytes that bound a token without whitespace.
func isDelimiter(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}
	return false
}

// skipSpace advances over whitespace and % comments (a comment runs to end of line).
func (l *lexer) skipSpace() {
	for l.pos < len(l.data) {
		c := l.data[l.pos]
		if c == '%' {
			for l.pos < len(l.data) && l.data[l.pos] != '\n' && l.data[l.pos] != '\r' {
				l.pos++
			}
			continue
		}
		if !isWhitespace(c) {
			return
		}
		l.pos++
	}
}

// next returns the next token, or a tokEOF token at end of input.
func (l *lexer) next() token {
	l.skipSpace()
	if l.pos >= len(l.data) {
		return token{kind: tokEOF}
	}
	if t, ok := l.lexDelimiter(); ok {
		return t
	}
	if c := l.data[l.pos]; c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9') {
		return l.lexNumber()
	}
	return l.lexKeyword()
}

// lexDelimiter lexes a token that begins with a PDF delimiter (dict/array brackets, name,
// or string), reporting false when the cursor is not on such a delimiter.
func (l *lexer) lexDelimiter() (token, bool) {
	switch l.data[l.pos] {
	case '<':
		if l.peek(1) == '<' {
			l.pos += 2
			return token{kind: tokDictOpen}, true
		}
		return l.lexHexString(), true
	case '>':
		l.pos += 2 // ">>"; a lone '>' is malformed PDF, so consume the pair defensively
		return token{kind: tokDictClose}, true
	case '[':
		l.pos++
		return token{kind: tokArrayOpen}, true
	case ']':
		l.pos++
		return token{kind: tokArrayClose}, true
	case '/':
		return l.lexName(), true
	case '(':
		return l.lexLiteralString(), true
	}
	return token{}, false
}

// peek returns the byte n positions ahead of the cursor, or 0 past end.
func (l *lexer) peek(n int) byte {
	if l.pos+n < len(l.data) {
		return l.data[l.pos+n]
	}
	return 0
}
