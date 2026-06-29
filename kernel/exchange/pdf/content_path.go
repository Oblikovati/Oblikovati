// SPDX-License-Identifier: GPL-2.0-only

package pdf

// pathOp dispatches the path-construction operators. Coordinates are transformed to device
// space immediately so a later cm never moves already-placed points.
func (in *interp) pathOp(op string) {
	switch op {
	case "m":
		in.moveTo(in.f(0), in.f(1))
	case "l":
		in.lineTo(in.f(0), in.f(1))
	case "c":
		in.curveTo(in.device(in.f(0), in.f(1)), in.device(in.f(2), in.f(3)), in.device(in.f(4), in.f(5)))
	case "v":
		in.curveTo(in.cur, in.device(in.f(0), in.f(1)), in.device(in.f(2), in.f(3)))
	case "y":
		end := in.device(in.f(2), in.f(3))
		in.curveTo(in.device(in.f(0), in.f(1)), end, end)
	case "re":
		in.rect(in.f(0), in.f(1), in.f(2), in.f(3))
	case "h":
		in.closeSubpath()
	}
}

// moveTo begins a new subpath at (x, y).
func (in *interp) moveTo(x, y float64) {
	in.cur = in.device(x, y)
	in.path = append(in.path, subpath{start: in.cur})
	in.open = true
}

// lineTo adds a straight segment to (x, y), starting an implicit subpath if none is open.
func (in *interp) lineTo(x, y float64) {
	p := in.device(x, y)
	in.appendSeg(segment{kind: segLine, pts: []pdfPoint{p}})
	in.cur = p
}

// curveTo adds a cubic Bézier with the two given control points and end point (already in
// device space).
func (in *interp) curveTo(c1, c2, end pdfPoint) {
	in.appendSeg(segment{kind: segCurve, pts: []pdfPoint{c1, c2, end}})
	in.cur = end
}

// appendSeg appends a segment to the current subpath, opening one at the current point when
// a path operator arrives without a preceding moveto.
func (in *interp) appendSeg(s segment) {
	if !in.open || len(in.path) == 0 {
		in.path = append(in.path, subpath{start: in.cur})
		in.open = true
	}
	last := len(in.path) - 1
	in.path[last].segs = append(in.path[last].segs, s)
}

// rect adds a closed rectangular subpath (the re operator); each corner is transformed by
// the CTM so a rotated/sheared rectangle stays a parallelogram.
func (in *interp) rect(x, y, w, h float64) {
	c0 := in.device(x, y)
	in.path = append(in.path, subpath{
		start: c0,
		segs: []segment{
			{kind: segLine, pts: []pdfPoint{in.device(x+w, y)}},
			{kind: segLine, pts: []pdfPoint{in.device(x+w, y+h)}},
			{kind: segLine, pts: []pdfPoint{in.device(x, y+h)}},
		},
		closed: true,
	})
	in.cur = c0
	in.open = true
}

// closeSubpath marks the current subpath closed (the h operator).
func (in *interp) closeSubpath() {
	if n := len(in.path); n > 0 {
		in.path[n-1].closed = true
		in.cur = in.path[n-1].start
	}
}

// paint flushes the current path to geometry per the painting operator, then clears it. The
// n operator (clip / no-paint) discards the path without emitting — that is how page-border
// clip rectangles are kept out of the sketch. The close-then-paint operators (s, b, b*)
// close every subpath first, as do the fill operators (PDF closes subpaths implicitly when
// filling).
func (in *interp) paint(op string) {
	switch op {
	case "n":
		// clip-only or no-op paint: emit nothing.
	case "S":
		in.flush(false)
	default: // s, f, F, f*, B, B*, b, b*
		in.flush(true)
	}
	in.path = nil
	in.open = false
}

// flush converts the accumulated subpaths to drawing entities; closeAll forces every
// subpath closed (for fill/close-and-paint operators).
func (in *interp) flush(closeAll bool) {
	for i := range in.path {
		if closeAll {
			in.path[i].closed = true
		}
		in.out = append(in.out, subpathEntities(in.path[i])...)
	}
}

// formState captures the graphics state runForm must restore after a Form XObject.
type formState struct {
	ctm       matrix
	ctmDepth  int
	resources dictObj
	path      []subpath
	cur       pdfPoint
	open      bool
}

// enterForm isolates the caller's state, applies the form's /Matrix, and switches to the
// form's /Resources (falling back to the inherited ones).
func (in *interp) enterForm(form streamObj) formState {
	saved := formState{ctm: in.ctm, ctmDepth: len(in.ctmStack), resources: in.resources, path: in.path, cur: in.cur, open: in.open}
	if m, ok := in.formMatrix(form.dict["Matrix"]); ok {
		in.ctm = concat(m, in.ctm)
	}
	if r, ok := in.doc.dictOf(form.dict["Resources"]); ok {
		in.resources = r
	}
	in.path = nil
	in.open = false
	in.depth++
	return saved
}

// leaveForm restores the state captured by enterForm.
func (in *interp) leaveForm(saved formState) {
	in.depth--
	in.ctm = saved.ctm
	in.ctmStack = in.ctmStack[:saved.ctmDepth]
	in.resources = saved.resources
	in.path = saved.path
	in.cur = saved.cur
	in.open = saved.open
}

// formMatrix reads a Form XObject's /Matrix array into a matrix, if present and well-formed.
func (in *interp) formMatrix(v objectValue) (matrix, bool) {
	arr, ok := in.doc.arrayOf(v)
	if !ok || len(arr) != 6 {
		return matrix{}, false
	}
	var m matrix
	for i := 0; i < 6; i++ {
		n, ok := in.doc.resolve(arr[i]).(numberObj)
		if !ok {
			return matrix{}, false
		}
		m[i] = float64(n)
	}
	return m, true
}
