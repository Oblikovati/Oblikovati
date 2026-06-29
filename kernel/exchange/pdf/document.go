// SPDX-License-Identifier: GPL-2.0-only

package pdf

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
)

// objHeader matches an indirect-object header "N G obj". We resolve objects by scanning
// these headers rather than trusting the cross-reference table: it is robust against the
// byte-offset drift that incremental-update PDFs introduce, and it sidesteps xref-stream
// parsing entirely (the later occurrence of a given object number wins, which is exactly
// the incremental-update override rule).
var objHeader = regexp.MustCompile(`(?m)(\d+)\s+(\d+)\s+obj\b`)

// document is a parsed PDF: a map from object number to the file offset of its header,
// plus a small resolved-object cache. It deliberately models only what page-geometry
// extraction needs.
type document struct {
	data    []byte
	offsets map[int]int
	cache   map[int]objectValue
}

// newDocument builds the object-offset index. It errors only when the file carries no
// classic indirect objects at all — the signature of an xref-stream / object-stream PDF,
// which this scoped reader does not support (it would otherwise silently import nothing).
func newDocument(data []byte) (*document, error) {
	offsets := map[int]int{}
	for _, m := range objHeader.FindAllSubmatchIndex(data, -1) {
		num, err := strconv.Atoi(string(data[m[2]:m[3]]))
		if err != nil {
			continue
		}
		offsets[num] = m[0] // later occurrence wins (incremental-update override)
	}
	if len(offsets) == 0 {
		return nil, fmt.Errorf("pdf: no classic indirect objects found (xref-stream / object-stream PDFs are unsupported); %d bytes", len(data))
	}
	return &document{data: data, offsets: offsets, cache: map[int]objectValue{}}, nil
}

// object returns the parsed object with the given number, or nullObj when it is missing
// or unparseable. Results are cached.
func (d *document) object(num int) objectValue {
	if v, ok := d.cache[num]; ok {
		return v
	}
	off, ok := d.offsets[num]
	if !ok {
		return nullObj{}
	}
	d.cache[num] = nullObj{} // guard against reference cycles during parse
	v := d.parseObjectAt(off)
	d.cache[num] = v
	return v
}

// parseObjectAt parses the indirect object whose header starts at off, attaching the
// stream body when the value is a dictionary followed by the stream keyword.
func (d *document) parseObjectAt(off int) objectValue {
	lex := &lexer{data: d.data, pos: off}
	p := newParser(lex)
	p.read() // object number
	p.read() // generation
	p.read() // "obj"
	v := p.parseValue()
	dict, ok := v.(dictObj)
	if !ok {
		return v
	}
	if t := p.read(); t.kind == tokKeyword && t.text == "stream" {
		return streamObj{dict: dict, raw: d.extractStream(lex, dict)}
	}
	return dict
}

// extractStream slices a stream body starting just after the stream keyword. It prefers a
// direct /Length when that lands exactly on an endstream marker, and otherwise searches
// for endstream — robust to a missing or indirect /Length and to FlateDecode payloads.
func (d *document) extractStream(lex *lexer, dict dictObj) []byte {
	start := skipStreamEOL(d.data, lex.pos)
	if n, ok := dict["Length"].(numberObj); ok {
		end := start + int(n)
		if end <= len(d.data) && endsAtEndstream(d.data, end) {
			return d.data[start:end]
		}
	}
	if rel := bytes.Index(d.data[start:], []byte("endstream")); rel >= 0 {
		return d.data[start : start+trimTrailingEOL(d.data[start:start+rel])]
	}
	return nil
}

// skipStreamEOL advances past the single EOL (CRLF or LF, tolerating a lone CR) that
// must follow the stream keyword before the body begins.
func skipStreamEOL(data []byte, pos int) int {
	if pos < len(data) && data[pos] == '\r' {
		pos++
	}
	if pos < len(data) && data[pos] == '\n' {
		pos++
	}
	return pos
}

// endsAtEndstream reports whether endstream follows position end (allowing one EOL of gap).
func endsAtEndstream(data []byte, end int) bool {
	rest := data[end:]
	rest = bytes.TrimLeft(rest, "\r\n")
	return bytes.HasPrefix(rest, []byte("endstream"))
}

// trimTrailingEOL returns the length of body with a single trailing EOL removed (the EOL
// that precedes the endstream keyword is not part of the stream data).
func trimTrailingEOL(body []byte) int {
	n := len(body)
	if n > 0 && body[n-1] == '\n' {
		n--
	}
	if n > 0 && body[n-1] == '\r' {
		n--
	}
	return n
}

// resolve dereferences an indirect reference (following a single hop), returning other
// values unchanged. A nil input resolves to nullObj so callers can type-switch safely.
func (d *document) resolve(v objectValue) objectValue {
	if v == nil {
		return nullObj{}
	}
	if r, ok := v.(refObj); ok {
		return d.object(r.num)
	}
	return v
}

// dictOf resolves v and returns it as a dictionary (a stream's dict counts), if it is one.
func (d *document) dictOf(v objectValue) (dictObj, bool) {
	switch g := d.resolve(v).(type) {
	case dictObj:
		return g, true
	case streamObj:
		return g.dict, true
	default:
		return nil, false
	}
}

// arrayOf resolves v and returns it as an array, if it is one.
func (d *document) arrayOf(v objectValue) (arrayObj, bool) {
	a, ok := d.resolve(v).(arrayObj)
	return a, ok
}

// nameOf resolves v and returns it as a name string, if it is one.
func (d *document) nameOf(v objectValue) (string, bool) {
	n, ok := d.resolve(v).(nameObj)
	return string(n), ok
}
