// SPDX-License-Identifier: GPL-2.0-only

package pdf

import (
	"bytes"
	"fmt"
)

// page is one decoded page: its resource dictionary (for XObject lookups), its media box
// lower-left (subtracted so content sits near the origin), and its concatenated, decoded
// content-stream bytes.
type page struct {
	resources dictObj
	originX   float64
	originY   float64
	content   []byte
}

// pages walks the page tree and returns each leaf page in document order.
func (d *document) pages() ([]page, error) {
	cat, ok := d.catalog()
	if !ok {
		return nil, fmt.Errorf("pdf: no document catalog (/Type /Catalog) found")
	}
	root, ok := d.dictOf(cat["Pages"])
	if !ok {
		return nil, fmt.Errorf("pdf: catalog has no page tree (/Pages)")
	}
	var out []page
	d.walkPageTree(root, dictObj{}, [2]float64{}, 0, &out)
	if len(out) == 0 {
		return nil, fmt.Errorf("pdf: page tree contains no pages")
	}
	return out, nil
}

// walkPageTree recurses the /Pages tree, propagating the inheritable /Resources and
// /MediaBox attributes, and appends one page per leaf. Depth is bounded to stop a
// malformed self-referential tree.
func (d *document) walkPageTree(node, resources dictObj, origin [2]float64, depth int, out *[]page) {
	if depth > 64 {
		return
	}
	resources, origin = inherit(d, node, resources, origin)
	kids, hasKids := d.arrayOf(node["Kids"])
	if !hasKids {
		*out = append(*out, d.leafPage(node, resources, origin))
		return
	}
	for _, kid := range kids {
		child, ok := d.dictOf(kid)
		if !ok {
			continue
		}
		d.walkPageTree(child, resources, origin, depth+1, out)
	}
}

// inherit overrides the inherited resources/media-box origin with this node's own when present.
func inherit(d *document, node, resources dictObj, origin [2]float64) (dictObj, [2]float64) {
	if r, ok := d.dictOf(node["Resources"]); ok {
		resources = r
	}
	if mb, ok := d.mediaBoxOrigin(node["MediaBox"]); ok {
		origin = mb
	}
	return resources, origin
}

// leafPage assembles a leaf page from its inherited attributes and content streams.
func (d *document) leafPage(node, resources dictObj, origin [2]float64) page {
	return page{
		resources: resources,
		originX:   origin[0],
		originY:   origin[1],
		content:   d.pageContent(node["Contents"]),
	}
}

// pageContent decodes and concatenates a page's content stream(s); a /Contents value is
// either one stream or an array of streams (joined with a newline so the last operator of
// one and the first of the next stay separated).
func (d *document) pageContent(contents objectValue) []byte {
	if arr, ok := d.arrayOf(contents); ok {
		var buf [][]byte
		for _, e := range arr {
			buf = append(buf, d.streamBytes(e))
		}
		return bytes.Join(buf, []byte{'\n'})
	}
	return d.streamBytes(contents)
}

// streamBytes resolves v to a stream and returns its decoded bytes (empty on any failure —
// a page with one unreadable stream still imports the rest).
func (d *document) streamBytes(v objectValue) []byte {
	s, ok := d.resolve(v).(streamObj)
	if !ok {
		return nil
	}
	out, err := decodeStream(d, s)
	if err != nil {
		return nil
	}
	return out
}

// mediaBoxOrigin resolves a /MediaBox to its lower-left corner, if it is a 4-number array.
func (d *document) mediaBoxOrigin(v objectValue) ([2]float64, bool) {
	arr, ok := d.arrayOf(v)
	if !ok || len(arr) != 4 {
		return [2]float64{}, false
	}
	llx, okx := d.resolve(arr[0]).(numberObj)
	lly, oky := d.resolve(arr[1]).(numberObj)
	if !okx || !oky {
		return [2]float64{}, false
	}
	return [2]float64{float64(llx), float64(lly)}, true
}

// catalog locates the document catalog, preferring the trailer's /Root and falling back to
// scanning every object for /Type /Catalog (some files have an unparsed or absent trailer).
func (d *document) catalog() (dictObj, bool) {
	if root, ok := d.rootFromTrailer(); ok {
		return root, true
	}
	return d.scanForType("Catalog")
}

// rootFromTrailer parses the last trailer dictionary and resolves its /Root.
func (d *document) rootFromTrailer() (dictObj, bool) {
	idx := bytes.LastIndex(d.data, []byte("trailer"))
	if idx < 0 {
		return nil, false
	}
	lex := &lexer{data: d.data, pos: idx + len("trailer")}
	dict, ok := newParser(lex).parseValue().(dictObj)
	if !ok {
		return nil, false
	}
	return d.dictOf(dict["Root"])
}

// scanForType returns the first object that is a dictionary with the given /Type.
func (d *document) scanForType(typ string) (dictObj, bool) {
	for num := range d.offsets {
		dict, ok := d.dictOf(d.object(num))
		if !ok {
			continue
		}
		if name, ok := d.nameOf(dict["Type"]); ok && name == typ {
			return dict, true
		}
	}
	return nil, false
}
