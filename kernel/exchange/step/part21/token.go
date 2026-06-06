// SPDX-License-Identifier: GPL-2.0-only

// Package part21 implements the ISO 10303-21 ("Part 21") clear-text encoding: a
// tokenizer + recursive-descent parser that builds an EntityGraph (id→RawEntity),
// plus an emitter. It knows the SYNTAX only — entity semantics (which keyword is a
// PLANE) live in the geommap/topomap layers. This keeps the grammar self-contained
// and under the file-size budget. See the M17 STEP plan §2.1.
package part21

import "fmt"

// TokenKind classifies a Part 21 lexeme.
type TokenKind uint8

const (
	// TokKeyword is a bare upper-case identifier (ISO, HEADER, CARTESIAN_POINT, FUNCTION).
	TokKeyword TokenKind = iota
	// TokRef is an entity reference #123.
	TokRef
	// TokString is a single-quoted string (escapes decoded).
	TokString
	// TokInt is an integer literal.
	TokInt
	// TokReal is a real literal (has a '.' and/or exponent).
	TokReal
	// TokEnum is a dotted enumeration like .T. .F. .STEEL.
	TokEnum
	// TokLParen TokRParen delimit parameter/list groups.
	TokLParen
	TokRParen
	// TokComma separates list/parameter items.
	TokComma
	// TokEquals binds #id = ENTITY.
	TokEquals
	// TokSemicolon terminates a statement.
	TokSemicolon
	// TokStar is the inherited/derived placeholder '*'.
	TokStar
	// TokDollar is the null/omitted-value '$'.
	TokDollar
	// TokEOF marks end of input.
	TokEOF
)

// String renders the kind for diagnostics.
func (k TokenKind) String() string {
	switch k {
	case TokKeyword:
		return "keyword"
	case TokRef:
		return "ref"
	case TokString:
		return "string"
	case TokInt:
		return "int"
	case TokReal:
		return "real"
	case TokEnum:
		return "enum"
	case TokLParen:
		return "("
	case TokRParen:
		return ")"
	case TokComma:
		return ","
	case TokEquals:
		return "="
	case TokSemicolon:
		return ";"
	case TokStar:
		return "*"
	case TokDollar:
		return "$"
	case TokEOF:
		return "eof"
	default:
		return fmt.Sprintf("kind(%d)", uint8(k))
	}
}

// Token is one lexeme with its source position (1-based line/column) for errors.
type Token struct {
	Kind   TokenKind
	Text   string // decoded value for strings; raw text otherwise
	Line   int
	Column int
}

// String renders a token for error messages, e.g. `string "foo" at 3:12`.
func (t Token) String() string {
	return fmt.Sprintf("%s %q at %d:%d", t.Kind, t.Text, t.Line, t.Column)
}
