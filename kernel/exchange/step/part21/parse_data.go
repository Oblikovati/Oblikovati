// SPDX-License-Identifier: GPL-2.0-only

package part21

import "fmt"

// parseData parses the DATA … ENDSEC; section into an EntityGraph.
func (p *parser) parseData() (*EntityGraph, error) {
	if err := p.expectKeyword("DATA"); err != nil {
		return nil, err
	}
	if err := p.expect(TokSemicolon); err != nil {
		return nil, err
	}
	graph := newEntityGraph()
	for !p.atKeyword("ENDSEC") {
		if p.cur.Kind == TokEOF {
			return nil, fmt.Errorf("part21: DATA section not terminated by ENDSEC")
		}
		ent, err := p.parseInstance()
		if err != nil {
			return nil, err
		}
		if err := graph.add(ent); err != nil {
			return nil, err
		}
	}
	return graph, p.endSection()
}

// endSection consumes `ENDSEC;`.
func (p *parser) endSection() error {
	if err := p.expectKeyword("ENDSEC"); err != nil {
		return err
	}
	return p.expect(TokSemicolon)
}

// parseInstance parses one `#id = …;` statement (simple or complex).
func (p *parser) parseInstance() (*RawEntity, error) {
	idTok := p.cur
	if idTok.Kind != TokRef {
		return nil, fmt.Errorf("part21: expected entity #id, got %s", idTok)
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	if err := p.expect(TokEquals); err != nil {
		return nil, err
	}
	ent := &RawEntity{ID: refID(idTok)}
	if err := p.parseInstanceBody(ent); err != nil {
		return nil, err
	}
	return ent, p.expect(TokSemicolon)
}

// parseInstanceBody fills a simple KEYWORD(args) or a complex (A(..)B(..)) body.
func (p *parser) parseInstanceBody(ent *RawEntity) error {
	if p.cur.Kind == TokLParen {
		return p.parseComplexComponents(ent)
	}
	if p.cur.Kind != TokKeyword {
		return fmt.Errorf("part21: expected entity keyword, got %s", p.cur)
	}
	ent.Keyword = p.cur.Text
	if err := p.advance(); err != nil {
		return err
	}
	params, err := p.parseParamList()
	if err != nil {
		return err
	}
	ent.Params = params
	return nil
}

// parseComplexComponents parses a complex instance ( A(..) B(..) … ).
func (p *parser) parseComplexComponents(ent *RawEntity) error {
	if err := p.expect(TokLParen); err != nil {
		return err
	}
	for p.cur.Kind == TokKeyword {
		keyword := p.cur.Text
		if err := p.advance(); err != nil {
			return err
		}
		args, err := p.parseParamList()
		if err != nil {
			return err
		}
		ent.Components = append(ent.Components, ComplexPart{Keyword: keyword, Params: args})
	}
	if len(ent.Components) == 0 {
		return fmt.Errorf("part21: empty complex instance #%d", ent.ID)
	}
	return p.expect(TokRParen)
}
