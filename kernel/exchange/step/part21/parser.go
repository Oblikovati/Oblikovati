// SPDX-License-Identifier: GPL-2.0-only

package part21

import "fmt"

// File is a parsed Part 21 exchange file: its HEADER and its resolved DATA graph.
type File struct {
	Header Header
	Graph  *EntityGraph
}

// parser is a recursive-descent parser over a one-token-lookahead lexer.
type parser struct {
	lx   *lexer
	cur  Token
	peek Token
}

// Parse parses a complete Part 21 file (HEADER + DATA). Malformed input yields an
// error naming the offending token and its line:column.
//
// Example:
//
//	f, err := part21.Parse(stepBytes)
//	pt := f.Graph.EntitiesOfType("CARTESIAN_POINT")
func Parse(src []byte) (*File, error) {
	p, err := newParser(src)
	if err != nil {
		return nil, err
	}
	if err := p.expectKeyword("ISO-10303-21"); err != nil {
		return nil, err
	}
	if err := p.expect(TokSemicolon); err != nil {
		return nil, err
	}
	header, err := p.parseHeader()
	if err != nil {
		return nil, err
	}
	graph, err := p.parseData()
	if err != nil {
		return nil, err
	}
	return &File{Header: header, Graph: graph}, p.expectKeyword("END-ISO-10303-21")
}

// newParser primes the two-token window.
func newParser(src []byte) (*parser, error) {
	p := &parser{lx: newLexer(src)}
	if err := p.fill(&p.cur); err != nil {
		return nil, err
	}
	return p, p.fill(&p.peek)
}

// fill loads one token from the lexer into dst.
func (p *parser) fill(dst *Token) error {
	tok, err := p.lx.next()
	if err != nil {
		return err
	}
	*dst = tok
	return nil
}

// advance consumes the current token, sliding the window forward.
func (p *parser) advance() error {
	p.cur = p.peek
	return p.fill(&p.peek)
}

// expect consumes the current token when it matches kind, else errors.
func (p *parser) expect(kind TokenKind) error {
	if p.cur.Kind != kind {
		return fmt.Errorf("part21: expected %s, got %s", kind, p.cur)
	}
	return p.advance()
}

// expectKeyword consumes a specific bare keyword (HEADER, DATA, ENDSEC, …).
func (p *parser) expectKeyword(name string) error {
	if p.cur.Kind != TokKeyword || p.cur.Text != name {
		return fmt.Errorf("part21: expected keyword %q, got %s", name, p.cur)
	}
	return p.advance()
}

// atKeyword reports whether the current token is a specific keyword (no consume).
func (p *parser) atKeyword(name string) bool {
	return p.cur.Kind == TokKeyword && p.cur.Text == name
}
