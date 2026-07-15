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
				set[origin] = true // an endpoint at the sketch origin omits its reference
			}
		}
	}
	return assembleLines(region, ends, distinct)
}

// assembleLines turns the entity→endpoints map into ordered line segments with per-segment
// construction flags, but only if every guard holds; otherwise ok is false. Referenced entities are
// the normal (non-construction) lines. A construction line drops its endpoint references, so its
// endpoints are the cached points left uncovered; the construction-flag table (entityConstruction)
// says how many construction lines to expect. The guards: each referenced entity has exactly two
// endpoints, the referenced count equals the number of non-construction entities, the leftover points
// pair exactly into the expected construction lines, and every cached point is covered. Only a single
// construction line is reconstructed — with more, pairing the leftover points is ambiguous, so it
// falls back.
func assembleLines(region []byte, ends map[uint16]map[Point]bool, distinct map[Point]bool) ([]Line, []bool, bool) {
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
	leftover := uncovered(distinct, covered)
	switch nConstr {
	case 0:
	case 1:
		if len(leftover) != 2 {
			return nil, nil, false
		}
		lines = append(lines, Line{A: leftover[0], B: leftover[1]})
		construction = append(construction, true)
		covered[leftover[0]], covered[leftover[1]] = true, true
	default:
		return nil, nil, false // ambiguous pairing of >1 construction line's endpoints
	}
	if len(covered) != len(distinct) {
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

// uncovered returns the cached distinct points not in covered, in sorted order for a stable pairing.
func uncovered(distinct, covered map[Point]bool) []Point {
	var out []Point
	for p := range distinct {
		if !covered[p] {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].X < out[j].X || (out[i].X == out[j].X && out[i].Y < out[j].Y)
	})
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
