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
// and so wrongly closes an open profile) whenever the graph fully self-validates. It returns the
// segments and a parallel construction flag per segment. The result is accepted only when it fully
// self-validates (see assembleLines); any mismatch returns ok=false so the caller keeps
// loopFromVertices, so an incompletely-referenced sketch is never silently mis-reconstructed.
func reconstructLines(region []byte) (lines []Line, construction []bool, ok bool) {
	pts := placedPoints(region)
	if len(pts) == 0 {
		return nil, nil, false
	}
	ends := map[uint16]map[Point]bool{}
	seen := map[Point]bool{}
	var distinctOrder []Point // distinct coordinates in stream (serialization) order
	hasOrigin := false
	for i, pp := range pts {
		if !seen[pp.p] {
			seen[pp.p] = true
			distinctOrder = append(distinctOrder, pp.p)
		}
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
				set[origin] = true // an endpoint at the sketch origin omits its reference
			}
		}
	}
	return assembleLines(region, ends, distinctOrder)
}

// assembleLines turns the entity→endpoints map into ordered line segments with per-segment
// construction flags, but only if every guard holds; otherwise ok is false. Referenced entities are
// the normal (non-construction) lines. A construction line drops its endpoint references, so its
// endpoints are the cached points left uncovered; the construction-flag table (entityConstruction)
// says how many construction lines to expect. Each entity serialises its two endpoints together, so
// the leftover points stay adjacent per line in stream order (verified on generated parts with two
// construction lines, including interleaved draw order) — they are paired consecutively. Guards: each
// referenced entity has two endpoints, the referenced count equals the non-construction count, the
// leftover count is exactly two per construction line with no degenerate pair, and every cached point
// is covered.
func assembleLines(region []byte, ends map[uint16]map[Point]bool, distinctOrder []Point) ([]Line, []bool, bool) {
	flags := entityConstruction(region)
	nConstr := 0
	for _, c := range flags {
		if c {
			nConstr++
		}
	}
	lines, covered, ok := referencedLines(ends)
	if !ok || len(lines) != len(flags)-nConstr {
		return nil, nil, false
	}
	construction := make([]bool, len(lines))
	leftover := uncoveredInOrder(distinctOrder, covered)
	if len(leftover) != 2*nConstr {
		return nil, nil, false
	}
	for i := 0; i < len(leftover); i += 2 {
		if leftover[i] == leftover[i+1] {
			return nil, nil, false // a construction line with a zero-length (degenerate) span
		}
		lines = append(lines, Line{A: leftover[i], B: leftover[i+1]})
		construction = append(construction, true)
		covered[leftover[i]], covered[leftover[i+1]] = true, true
	}
	if len(covered) != len(distinctOrder) {
		return nil, nil, false
	}
	return lines, construction, true
}

// referencedLines builds one line per referenced entity (in entity-index order), reporting the set of
// covered points. ok is false if any entity does not have exactly two endpoints.
func referencedLines(ends map[uint16]map[Point]bool) ([]Line, map[Point]bool, bool) {
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
			return nil, nil, false
		}
		ab := make([]Point, 0, 2)
		for p := range set {
			ab = append(ab, p)
			covered[p] = true
		}
		lines = append(lines, Line{A: ab[0], B: ab[1]})
	}
	return lines, covered, true
}

// uncoveredInOrder returns the distinct points not in covered, preserving stream (serialization)
// order so a construction line's two endpoints — serialised together — stay adjacent for pairing.
func uncoveredInOrder(distinctOrder []Point, covered map[Point]bool) []Point {
	var out []Point
	for _, p := range distinctOrder {
		if !covered[p] {
			out = append(out, p)
		}
	}
	return out
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
