// SPDX-License-Identifier: GPL-2.0-only

package part21

import (
	"fmt"
	"strconv"
	"strings"
)

// parseParamList parses a parenthesized, comma-separated value list: ( v , v , … ).
// An empty list () yields a zero-length, non-nil slice.
func (p *parser) parseParamList() ([]Value, error) {
	if err := p.expect(TokLParen); err != nil {
		return nil, err
	}
	values := []Value{}
	if p.cur.Kind == TokRParen {
		return values, p.advance()
	}
	for {
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values = append(values, v)
		if p.cur.Kind != TokComma {
			break
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	return values, p.expect(TokRParen)
}

// parseValue parses one parameter value, dispatching on the current token.
func (p *parser) parseValue() (Value, error) {
	tok := p.cur
	switch tok.Kind {
	case TokRef:
		return p.scalarValue(Value{Kind: ValRef, Ref: refID(tok), position: tok})
	case TokInt:
		return p.intValue(tok)
	case TokReal:
		return p.realValue(tok)
	case TokString:
		return p.scalarValue(Value{Kind: ValString, Str: tok.Text, position: tok})
	case TokEnum:
		return p.scalarValue(Value{Kind: ValEnum, Enum: strings.Trim(tok.Text, "."), position: tok})
	case TokDollar:
		return p.scalarValue(Value{Kind: ValNull, position: tok})
	case TokStar:
		return p.scalarValue(Value{Kind: ValDerived, position: tok})
	case TokLParen:
		return p.listValue(tok)
	case TokKeyword:
		return p.typedValue(tok)
	default:
		return Value{}, fmt.Errorf("part21: unexpected %s where a value was expected", tok)
	}
}

// scalarValue consumes the current scalar token and returns the prepared value.
func (p *parser) scalarValue(v Value) (Value, error) {
	return v, p.advance()
}

// intValue parses an integer literal.
func (p *parser) intValue(tok Token) (Value, error) {
	n, err := strconv.ParseInt(tok.Text, 10, 64)
	if err != nil {
		return Value{}, fmt.Errorf("part21: bad integer %q at %d:%d", tok.Text, tok.Line, tok.Column)
	}
	return p.scalarValue(Value{Kind: ValInt, Int: n, position: tok})
}

// realValue parses a real literal.
func (p *parser) realValue(tok Token) (Value, error) {
	f, err := strconv.ParseFloat(tok.Text, 64)
	if err != nil {
		return Value{}, fmt.Errorf("part21: bad real %q at %d:%d", tok.Text, tok.Line, tok.Column)
	}
	return p.scalarValue(Value{Kind: ValReal, Real: f, position: tok})
}

// listValue parses a nested ( … ) list as a value.
func (p *parser) listValue(tok Token) (Value, error) {
	items, err := p.parseParamList()
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: ValList, List: items, position: tok}, nil
}

// typedValue parses a KEYWORD(args) value appearing inside a parameter list (a
// select/typed parameter, e.g. LENGTH_MEASURE(25.4)).
func (p *parser) typedValue(tok Token) (Value, error) {
	if err := p.advance(); err != nil { // consume keyword
		return Value{}, err
	}
	args, err := p.parseParamList()
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: ValTyped, Keyword: tok.Text, List: args, position: tok}, nil
}

// refID parses the numeric id from a #123 reference token (validated by the lexer).
func refID(tok Token) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(tok.Text, "#"))
	return n
}
