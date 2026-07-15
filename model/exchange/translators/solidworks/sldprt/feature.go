// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import "bytes"

// Extrusion is a decoded extrude feature. Depth is the blind extent in metres. This is the first,
// minimal feature decode: a single blind boss extrude of the part's profile sketch.
type Extrusion struct {
	Depth float64
}

var (
	extrusionClass   = []byte("moExtrusion_c")
	lengthParamClass = []byte("moLengthParameter_c")
)

// Extrusions decodes the part's extrude features from the resolved-feature graph. Each moExtrusion_c
// is one extrude; its blind depth is the length parameter that follows it (the same 2.0-factor value
// encoding as a sketch dimension). Only the depth is read so far — direction, end condition and
// boss/cut are later work.
func (d *Document) Extrusions() []Extrusion {
	stream, ok := d.sketchStream()
	if !ok {
		return nil
	}
	var out []Extrusion
	for i := 0; ; {
		e := bytes.Index(stream[i:], extrusionClass)
		if e < 0 {
			break
		}
		at := i + e
		i = at + len(extrusionClass)
		if depth, ok := extrusionDepth(stream, at); ok {
			out = append(out, Extrusion{Depth: depth})
		}
	}
	return out
}

// extrusionDepth reads the blind depth (metres) of the extrude beginning at `at`: the value of the
// first length parameter that follows the feature in the graph.
func extrusionDepth(stream []byte, at int) (float64, bool) {
	lp := bytes.Index(stream[at:], lengthParamClass)
	if lp < 0 {
		return 0, false
	}
	return dimValueAfter(stream, at+lp+len(lengthParamClass))
}
