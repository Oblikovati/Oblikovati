// SPDX-License-Identifier: GPL-2.0-only

package pdf

import (
	"bytes"
	"fmt"

	"oblikovati.org/kernel/exchange/drawing"
)

// pdfPoint is a point already mapped to device space (PDF points) by the CTM in force when
// the path operator ran — so later cm changes never move already-placed geometry.
type pdfPoint struct{ x, y float64 }

// segKind tags a path segment as a straight line or a cubic Bézier.
type segKind int

const (
	segLine segKind = iota
	segCurve
)

// segment is one element of a subpath: a line (one end point) or a cubic Bézier (its two
// control points and its end point).
type segment struct {
	kind segKind
	pts  []pdfPoint
}

// subpath is a connected run of segments from a start point, optionally closed back to it.
type subpath struct {
	start  pdfPoint
	segs   []segment
	closed bool
}

// maxFormDepth bounds Form-XObject recursion against a malformed self-referential page.
const maxFormDepth = 16

// interp executes a content stream's path/painting/state operators against a current
// transformation matrix, accumulating the resulting curve geometry. Text, colour, and
// raster operators are ignored — only vector paths become geometry.
type interp struct {
	doc       *document
	resources dictObj
	ctm       matrix
	ctmStack  []matrix
	operands  []objectValue
	path      []subpath
	cur       pdfPoint
	open      bool // a subpath is in progress (an m without a following painting op)
	out       []drawing.Entity
	warns     []string
	seen      map[string]bool
	depth     int
}

// run interprets a content-stream byte slice, dispatching each operator and pushing each
// operand onto the stack until end of input.
func (in *interp) run(content []byte) {
	lex := newLexer(content)
	p := newParser(lex)
	for {
		t := p.read()
		if t.kind == tokEOF {
			return
		}
		if t.kind == tokKeyword {
			in.dispatchKeyword(t, lex)
			continue
		}
		p.unread(t)
		if v := p.parseValue(); v != nil {
			in.operands = append(in.operands, v)
		} else {
			p.read() // consume a stray token to make progress
		}
	}
}

// dispatchKeyword handles a keyword token: the three literals push operands, BI skips an
// inline image, and any other keyword is an operator that consumes the operand stack.
func (in *interp) dispatchKeyword(t token, lex *lexer) {
	switch t.text {
	case "true":
		in.operands = append(in.operands, boolObj(true))
	case "false":
		in.operands = append(in.operands, boolObj(false))
	case "null":
		in.operands = append(in.operands, nullObj{})
	case "BI":
		skipInlineImage(lex)
		in.operands = in.operands[:0]
	default:
		in.execute(t.text)
		in.operands = in.operands[:0]
	}
}

// execute dispatches one operator. Unhandled operators (colour, text, line params,
// marked-content) are intentional no-ops; their operands are cleared by the caller.
func (in *interp) execute(op string) {
	switch op {
	case "m", "l", "c", "v", "y", "re", "h":
		in.pathOp(op)
	case "S", "s", "f", "F", "f*", "B", "B*", "b", "b*", "n":
		in.paint(op)
	case "q":
		in.ctmStack = append(in.ctmStack, in.ctm)
	case "Q":
		in.popCTM()
	case "cm":
		in.ctm = concat(in.matrixOperand(), in.ctm)
	case "Do":
		in.doXObject()
	}
}

// popCTM restores the previous CTM, ignoring an unbalanced Q.
func (in *interp) popCTM() {
	if n := len(in.ctmStack); n > 0 {
		in.ctm = in.ctmStack[n-1]
		in.ctmStack = in.ctmStack[:n-1]
	}
}

// matrixOperand reads the six operands of a cm operator into a matrix.
func (in *interp) matrixOperand() matrix {
	return matrix{in.f(0), in.f(1), in.f(2), in.f(3), in.f(4), in.f(5)}
}

// f returns the i-th operand as a float (0 when missing or non-numeric).
func (in *interp) f(i int) float64 {
	if i < len(in.operands) {
		if n, ok := in.operands[i].(numberObj); ok {
			return float64(n)
		}
	}
	return 0
}

// device maps a user-space (x, y) through the current CTM to device space.
func (in *interp) device(x, y float64) pdfPoint {
	dx, dy := in.ctm.apply(x, y)
	return pdfPoint{dx, dy}
}

// warnOnce records a warning the first time a given message occurs, so a page that draws
// many images or hits one unsupported operator repeatedly yields a single note.
func (in *interp) warnOnce(msg string) {
	if in.seen == nil {
		in.seen = map[string]bool{}
	}
	if in.seen[msg] {
		return
	}
	in.seen[msg] = true
	in.warns = append(in.warns, msg)
}

// doXObject paints a named XObject: a Form is interpreted in-line under its matrix; an
// Image (raster) is skipped with a one-time warning.
func (in *interp) doXObject() {
	name, ok := in.firstName()
	if !ok {
		return
	}
	xdict, ok := in.doc.dictOf(in.resources["XObject"])
	if !ok {
		return
	}
	form, ok := in.doc.resolve(xdict[name]).(streamObj)
	if !ok {
		return
	}
	if sub, _ := in.doc.nameOf(form.dict["Subtype"]); sub == "Image" {
		in.warnOnce("import: skipped raster image XObject (only vector paths are imported)")
		return
	}
	in.runForm(form)
}

// runForm interprets a Form XObject's content with its own matrix and resources, isolating
// its graphics state (CTM, path) from the caller's.
func (in *interp) runForm(form streamObj) {
	if in.depth >= maxFormDepth {
		in.warnOnce("import: Form XObject nesting too deep; skipped")
		return
	}
	content, err := decodeStream(in.doc, form)
	if err != nil {
		in.warnOnce(fmt.Sprintf("import: undecodable Form XObject (%v)", err))
		return
	}
	saved := in.enterForm(form)
	in.run(content)
	in.leaveForm(saved)
}

// firstName returns the first name operand (the XObject name for a Do).
func (in *interp) firstName() (string, bool) {
	for _, o := range in.operands {
		if n, ok := o.(nameObj); ok {
			return string(n), true
		}
	}
	return "", false
}

// skipInlineImage advances the lexer past a BI…ID…EI inline image's binary payload, which
// must not be lexed as operators. Best-effort: it stops at the first EI delimited by
// whitespace (CAD plot-to-PDF content streams place raster data in XObjects, not inline).
func skipInlineImage(lex *lexer) {
	rest := lex.data[lex.pos:]
	if i := bytes.Index(rest, []byte("EI")); i >= 0 {
		lex.pos += i + 2
		return
	}
	lex.pos = len(lex.data)
}
