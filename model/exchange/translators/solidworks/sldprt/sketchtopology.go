// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"encoding/binary"
	"sort"
)

// A point record binds its cached endpoint to the sketch entity that owns it: after the `1e 00 x y`
// coordinate comes an MFC object reference `<tag> <entityIdx:u16> ff ff ff ff`, where tag is 0x80ac
// or 0x80ae (both spellings occur — the 0x80ae form shows up in dimensioned sketches). Grouping the
// cached points by that entity index recovers each line's exact endpoints — open or closed — instead
// of guessing a loop from the vertex cloud. Verified on generated known parts: closed square/triangle
// (shared corners), disjoint open lines, and two lines meeting at the origin.
var entityRefSuffix = []byte{0xff, 0xff, 0xff, 0xff}

// origin is the sketch origin. An endpoint coincident with it omits its owning-entity reference (the
// origin is the sketch's anchor), so an entity left with a single endpoint is completed with the
// origin when the origin is a cached point — see reconstructLines.
var origin = Point{X: 0, Y: 0}

// placedPoint pairs a cached coordinate with the byte offset of its record, so each point's own
// owning-entity reference can be read from the bytes that immediately follow its coordinate.
type placedPoint struct {
	off int
	p   Point
}

// placedPoints returns every cached coordinate with its record offset, in stream order (duplicates
// kept: a corner coincident with the origin is cached twice and each instance is its own record).
func placedPoints(region []byte) []placedPoint {
	var out []placedPoint
	for i := 0; ; {
		m := bytes.Index(region[i:], pointMarker)
		if m < 0 {
			break
		}
		at := i + m
		if p, ok := readPoint(region, at+len(pointMarker)); ok {
			out = append(out, placedPoint{off: at, p: p})
		}
		i = at + 1
	}
	return out
}

// reconstructLines rebuilds a pure-line sketch's segments exactly from the point→entity reference
// graph. It replaces the geometry-first loopFromVertices (which orders vertices around their centroid
// and so wrongly closes an open profile) whenever the graph fully self-validates. The result is
// accepted only when: one entity per construction-table entry (entityConstruction), every entity has
// exactly two distinct endpoints (after origin completion), and the endpoint set equals the cached
// distinct points. Any mismatch returns ok=false so the caller keeps loopFromVertices — so a sketch
// whose references are incomplete (e.g. a construction line, whose endpoints carry no reference) is
// never silently mis-reconstructed.
func reconstructLines(region []byte) ([]Line, bool) {
	pts := placedPoints(region)
	if len(pts) == 0 {
		return nil, false
	}
	ends := map[uint16]map[Point]bool{}
	distinct := map[Point]bool{}
	hasOrigin := false
	for i, pp := range pts {
		distinct[pp.p] = true
		hasOrigin = hasOrigin || pp.p == origin
		lo := pp.off + len(pointMarker) + 16 // just past the x,y coordinate
		hi := lo + 40
		if i+1 < len(pts) && pts[i+1].off < hi {
			hi = pts[i+1].off
		}
		if hi > len(region) {
			hi = len(region)
		}
		for _, idx := range entityRefsIn(region[lo:hi]) {
			if ends[idx] == nil {
				ends[idx] = map[Point]bool{}
			}
			ends[idx][pp.p] = true
		}
	}
	if hasOrigin {
		for _, set := range ends {
			if len(set) == 1 {
				set[origin] = true
			}
		}
	}
	return validatedLines(region, ends, distinct)
}

// validatedLines turns the entity→endpoints map into ordered line segments, but only if every guard
// holds; otherwise ok is false. The entity count must match the construction-table entity count, each
// entity must have exactly two endpoints, and the endpoints must exactly cover the cached points.
func validatedLines(region []byte, ends map[uint16]map[Point]bool, distinct map[Point]bool) ([]Line, bool) {
	if len(ends) == 0 || len(ends) != len(entityConstruction(region)) {
		return nil, false
	}
	idxs := make([]int, 0, len(ends))
	for idx := range ends {
		idxs = append(idxs, int(idx))
	}
	sort.Ints(idxs)
	covered := map[Point]bool{}
	lines := make([]Line, 0, len(idxs))
	for _, idx := range idxs {
		set := ends[uint16(idx)]
		if len(set) != 2 {
			return nil, false
		}
		ab := make([]Point, 0, 2)
		for p := range set {
			ab = append(ab, p)
			covered[p] = true
		}
		lines = append(lines, Line{A: ab[0], B: ab[1]})
	}
	if len(covered) != len(distinct) {
		return nil, false
	}
	for p := range distinct {
		if !covered[p] {
			return nil, false
		}
	}
	return lines, true
}

// entityRefsIn returns the entity indices referenced by the owning-entity links in a point record's
// tail: each `<0x80ac|0x80ae> <idx:u16> ff ff ff ff`. The four-byte 0xff suffix rejects coincidental
// tag-byte matches, so only real references are collected.
func entityRefsIn(seg []byte) []uint16 {
	var out []uint16
	for i := 0; i+8 <= len(seg); i++ {
		if seg[i+1] != 0x80 || (seg[i] != 0xac && seg[i] != 0xae) {
			continue
		}
		if !bytes.Equal(seg[i+4:i+8], entityRefSuffix) {
			continue
		}
		out = append(out, binary.LittleEndian.Uint16(seg[i+2:]))
	}
	return out
}
