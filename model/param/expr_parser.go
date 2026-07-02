// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	"slices"
	"strconv"
)

// parser is a recursive-descent parser over a token slice, implementing the
// grammar:
//
//	expr    := term (('+' | '-') term)*
//	term    := factor (('*' | '/') factor)*
//	factor  := '-' factor | primary
//	primary := number [unit] | ident '(' args ')' | ident | '(' expr ')'
type parser struct {
	tokens []token
	pos    int
}

// parse tokenizes and parses src into an AST root.
func parse(src string) (node, error) {
	tokens, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	root, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokEOF {
		return nil, fmt.Errorf("param: unexpected %q at position %d", p.peek().text, p.peek().pos)
	}
	return root, nil
}

func (p *parser) peek() token { return p.tokens[p.pos] }

func (p *parser) advance() token {
	t := p.tokens[p.pos]
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}

// parseExpr handles + and - at the lowest precedence.
func (p *parser) parseExpr() (node, error) {
	return p.parseBinary(p.parseTerm, tokPlus, tokMinus)
}

// parseTerm handles * and /.
func (p *parser) parseTerm() (node, error) {
	return p.parseBinary(p.parseFactor, tokStar, tokSlash)
}

// parseBinary parses a left-associative run of the given operators above the
// next sub-parser, factoring the shared structure of parseExpr and parseTerm.
func (p *parser) parseBinary(sub func() (node, error), ops ...tokenKind) (node, error) {
	lhs, err := sub()
	if err != nil {
		return nil, err
	}
	for slices.Contains(ops, p.peek().kind) {
		op := p.advance()
		rhs, err := sub()
		if err != nil {
			return nil, err
		}
		lhs = binaryNode{op: op.text[0], lhs: lhs, rhs: rhs}
	}
	return lhs, nil
}

// parseFactor handles unary minus, then a primary.
func (p *parser) parseFactor() (node, error) {
	if p.peek().kind == tokMinus {
		p.advance()
		operand, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return unaryNode{operand: operand}, nil
	}
	return p.parsePrimary()
}

// parsePrimary parses a number (with optional unit), a function call, a
// reference, or a parenthesized expression.
func (p *parser) parsePrimary() (node, error) {
	tok := p.peek()
	switch tok.kind {
	case tokNumber:
		return p.parseNumber()
	case tokIdent:
		return p.parseIdent()
	case tokLParen:
		return p.parseParen()
	default:
		return nil, fmt.Errorf("param: expected a value at position %d, found %q", tok.pos, tok.text)
	}
}

// parseNumber parses a numeric literal and an optional trailing unit name.
func (p *parser) parseNumber() (node, error) {
	tok := p.advance()
	value, err := strconv.ParseFloat(tok.text, 64)
	if err != nil {
		return nil, fmt.Errorf("param: invalid number %q at position %d", tok.text, tok.pos)
	}
	unit := Unitless
	if next := p.peek(); next.kind == tokIdent {
		if def, ok := lookupUnit(next.text); ok {
			p.advance()
			value, unit = value*def.factor, def.category
		}
	}
	return numberNode{q: Quantity{value, unit}}, nil
}

// parseIdent parses either a function call (ident followed by '(') or a
// parameter reference.
func (p *parser) parseIdent() (node, error) {
	tok := p.advance()
	if p.peek().kind != tokLParen {
		return &refNode{name: tok.text}, nil
	}
	args, err := p.parseArgs()
	if err != nil {
		return nil, err
	}
	return callNode{fn: tok.text, args: args}, nil
}

// parseArgs parses a parenthesized, comma-separated argument list.
func (p *parser) parseArgs() ([]node, error) {
	p.advance() // consume '('
	var args []node
	if p.peek().kind == tokRParen {
		p.advance()
		return args, nil
	}
	for {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peek().kind != tokComma {
			break
		}
		p.advance()
	}
	if p.peek().kind != tokRParen {
		return nil, fmt.Errorf("param: expected ')' at position %d", p.peek().pos)
	}
	p.advance()
	return args, nil
}

// parseParen parses '(' expr ')'.
func (p *parser) parseParen() (node, error) {
	p.advance() // consume '('
	inner, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokRParen {
		return nil, fmt.Errorf("param: expected ')' at position %d", p.peek().pos)
	}
	p.advance()
	return inner, nil
}
