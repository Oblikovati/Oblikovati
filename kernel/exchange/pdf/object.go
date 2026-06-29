// SPDX-License-Identifier: GPL-2.0-only

package pdf

// objectValue is one parsed PDF object. PDF objects form a small closed sum type
// (number, name, string, boolean, null, array, dictionary, indirect reference, stream);
// callers discriminate with a type switch. The isObject marker keeps the set closed.
type objectValue interface{ isObject() }

type (
	numberObj float64                // a numeric object
	nameObj   string                 // a /Name object (without the slash)
	stringObj []byte                 // a (literal) or <hex> string object
	boolObj   bool                   // true / false
	nullObj   struct{}               // null
	arrayObj  []objectValue          // an [ … ] array
	dictObj   map[string]objectValue // a << … >> dictionary, keyed by name
	refObj    struct{ num, gen int } // an "n g R" indirect reference
	streamObj struct {               // a dictionary followed by a stream body
		dict dictObj
		raw  []byte
	}
)

func (numberObj) isObject() {}
func (nameObj) isObject()   {}
func (stringObj) isObject() {}
func (boolObj) isObject()   {}
func (nullObj) isObject()   {}
func (arrayObj) isObject()  {}
func (dictObj) isObject()   {}
func (refObj) isObject()    {}
func (streamObj) isObject() {}

// parser turns the lexer's token stream into objectValues, with a pushback stack so it can
// recognise the three-token "n g R" indirect-reference form — which needs two tokens of
// look-back when the trailing keyword is not "R".
type parser struct {
	lex  *lexer
	back []token
}

func newParser(l *lexer) *parser { return &parser{lex: l} }

// read returns the next token, drawing from the pushback stack (LIFO) first.
func (p *parser) read() token {
	if n := len(p.back); n > 0 {
		t := p.back[n-1]
		p.back = p.back[:n-1]
		return t
	}
	return p.lex.next()
}

// unread pushes a token back so a later read returns it (LIFO: the last unread is read
// first, so parseNumberOrRef can unread its two-token look-ahead in reverse).
func (p *parser) unread(t token) { p.back = append(p.back, t) }

// parseValue parses a single object value. A bare keyword that is not a literal
// (true/false/null) is returned to the caller via pushback and signalled as nil — the
// caller (array/dict loop, or indirect-object reader) decides whether that ends its run.
func (p *parser) parseValue() objectValue {
	t := p.read()
	switch t.kind {
	case tokNumber:
		return p.parseNumberOrRef(t)
	case tokName:
		return nameObj(t.text)
	case tokString:
		return stringObj(t.str)
	case tokDictOpen:
		return p.parseDict()
	case tokArrayOpen:
		return p.parseArray()
	case tokKeyword:
		return literalKeyword(t)
	default:
		p.unread(t)
		return nil
	}
}

// parseNumberOrRef disambiguates a plain number from an "n g R" reference using two-token
// lookahead.
func (p *parser) parseNumberOrRef(first token) objectValue {
	t2 := p.read()
	if t2.kind == tokNumber {
		t3 := p.read()
		if t3.kind == tokKeyword && t3.text == "R" {
			return refObj{num: int(first.num), gen: int(t2.num)}
		}
		p.unread(t3)
	}
	p.unread(t2)
	return numberObj(first.num)
}

// literalKeyword maps the three literal keywords (true/false/null) to objects; any other
// keyword ends the current value (caller receives nil).
func literalKeyword(t token) objectValue {
	switch t.text {
	case "true":
		return boolObj(true)
	case "false":
		return boolObj(false)
	case "null":
		return nullObj{}
	}
	return nil
}

// parseArray reads array elements until the closing bracket.
func (p *parser) parseArray() arrayObj {
	var arr arrayObj
	for {
		t := p.read()
		if t.kind == tokArrayClose || t.kind == tokEOF {
			return arr
		}
		p.unread(t)
		v := p.parseValue()
		if v == nil { // a stray keyword inside an array — skip it to stay aligned.
			p.read()
			continue
		}
		arr = append(arr, v)
	}
}

// parseDict reads name/value pairs until the closing >>.
func (p *parser) parseDict() dictObj {
	d := dictObj{}
	for {
		t := p.read()
		if t.kind == tokDictClose || t.kind == tokEOF {
			return d
		}
		if t.kind != tokName { // resync on malformed key
			continue
		}
		v := p.parseValue()
		if v == nil {
			continue
		}
		d[t.text] = v
	}
}
