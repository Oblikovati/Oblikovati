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
func reconstructLines(region []byte) (lines []Line, construction []bool, standalone []Point, ok bool) {
	pts := placedPoints(region)
	if len(pts) == 0 {
		return nil, nil, nil, false
	}
	ends := map[uint16]map[Point]bool{}
	seen := map[Point]bool{}
	var distinctOrder []Point // distinct coordinates in stream (serialization) order
	for i, pp := range pts {
		if !seen[pp.p] {
			seen[pp.p] = true
			distinctOrder = append(distinctOrder, pp.p)
		}
		lo := pp.off + len(pointMarker) + 16 // just past the x,y coordinate
		// A point's record — holding its owning-entity references — runs to the next point. Scan the
		// whole record so a reference displaced by a class registration (the first entity's point) is
		// still seen. Only the last point is capped, since the trailing entity-record table follows it.
		hi := lo + 40
		if i+1 < len(pts) {
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
	// Fill the endpoints whose references were dropped. Which points those are depends on whether the
	// sketch anchors at the origin: if the first cached point IS the origin, every short entity's
	// missing endpoint is the origin (lines meeting there); otherwise only the first cached point lost
	// its reference (to the first entity's class registration), completing the one entity left short.
	if pts[0].p == origin {
		for _, set := range ends {
			if len(set) == 1 {
				set[origin] = true
			}
		}
	} else {
		completeFirstPoint(ends, pts[0].p)
	}
	lines, construction, standalone, ok = assembleLines(region, ends, distinctOrder)
	return lines, construction, standalone, ok
}

// completeFirstPoint fills the first entity's start endpoint, whose reference the sketch's first class
// registration displaces (the class name is written where the reference would go). The first cached
// point is that start; when it is still uncovered and exactly one entity is left short, it completes
// that entity. Guarded to one short entity so an ambiguous case is left for the assembler to reject.
func completeFirstPoint(ends map[uint16]map[Point]bool, first Point) {
	if pointCovered(ends, first) {
		return
	}
	var short map[Point]bool
	for _, set := range ends {
		if len(set) == 1 {
			if short != nil {
				return // more than one short entity: which one owns the first point is ambiguous
			}
			short = set
		}
	}
	if short != nil {
		short[first] = true
	}
}

// pointCovered reports whether p is already an endpoint of any entity.
func pointCovered(ends map[uint16]map[Point]bool, p Point) bool {
	for _, set := range ends {
		if set[p] {
			return true
		}
	}
	return false
}

// indexedLine is a reconstructed line tagged with the entity index that referenced it, so its
// construction flag can be recovered from the draw-ordered construction-flag table.
type indexedLine struct {
	idx  int
	line Line
}

// assembleLines turns the entity→endpoints map into ordered line segments with per-segment
// construction flags, but only if every guard holds; otherwise ok is false. There are two ways a
// construction line is stored, handled as two cases against the construction-flag table
// (entityConstruction, one flag per entity in draw order):
//   - A centerline keeps its endpoint references, so every entity is referenced; its construction
//     flag is read from the table by draw position (the referenced indices, normalised to zero-based).
//   - A construction line toggled from a normal line drops its endpoint references, so it is not
//     referenced; its endpoints are the leftover cached points, paired consecutively in stream order.
//
// A cached point that is neither a referenced endpoint nor a construction endpoint is a standalone
// sketch point (the origin excepted — it is the sketch's own origin, not a free point). A mix of the
// two construction storages, or leftover that neither pairs nor forms standalone points, falls back.
func assembleLines(region []byte, ends map[uint16]map[Point]bool, distinctOrder []Point) ([]Line, []bool, []Point, bool) {
	flags := entityConstruction(region)
	nConstr := 0
	for _, c := range flags {
		if c {
			nConstr++
		}
	}
	refs, covered, ok := referencedLines(ends)
	if !ok {
		return nil, nil, nil, false
	}
	leftover := pointsExcept(uncoveredInOrder(distinctOrder, covered), origin)
	switch {
	case len(refs) == len(flags):
		return referencedWithFlags(refs, flags, nConstr, leftover)
	case len(refs) == len(flags)-nConstr:
		return leftoverConstruction(refs, nConstr, leftover)
	default:
		return nil, nil, nil, false
	}
}

// referencedWithFlags handles the all-referenced case (normal lines and centerlines): each line's
// construction flag comes from the flag table at its normalised draw position. Leftover points are
// standalone. The recovered construction count must match the table, else the mapping is untrusted.
func referencedWithFlags(refs []indexedLine, flags []bool, nConstr int, standalone []Point) ([]Line, []bool, []Point, bool) {
	minIdx := refs[0].idx // refs are sorted by index
	lines := make([]Line, len(refs))
	construction := make([]bool, len(refs))
	got := 0
	for i, r := range refs {
		lines[i] = r.line
		if pos := r.idx - minIdx; pos >= 0 && pos < len(flags) && flags[pos] {
			construction[i] = true
			got++
		}
	}
	if got != nConstr {
		return nil, nil, nil, false
	}
	return lines, construction, standalone, true
}

// leftoverConstruction handles the toggled-construction case: the referenced entities are the normal
// lines, and the leftover points are the construction lines' endpoints, paired consecutively.
func leftoverConstruction(refs []indexedLine, nConstr int, leftover []Point) ([]Line, []bool, []Point, bool) {
	lines := make([]Line, len(refs))
	for i, r := range refs {
		lines[i] = r.line
	}
	construction := make([]bool, len(refs))
	if len(leftover) != 2*nConstr {
		return nil, nil, nil, false // construction endpoints mixed with standalone points: ambiguous
	}
	for i := 0; i < len(leftover); i += 2 {
		if leftover[i] == leftover[i+1] {
			return nil, nil, nil, false // a construction line with a zero-length (degenerate) span
		}
		lines = append(lines, Line{A: leftover[i], B: leftover[i+1]})
		construction = append(construction, true)
	}
	return lines, construction, nil, true
}

// referencedLines builds one line per referenced entity, sorted by entity index, reporting the set of
// covered points. ok is false if any entity does not have exactly two endpoints.
func referencedLines(ends map[uint16]map[Point]bool) ([]indexedLine, map[Point]bool, bool) {
	idxs := make([]int, 0, len(ends))
	for idx := range ends {
		idxs = append(idxs, int(idx))
	}
	sort.Ints(idxs)
	covered := map[Point]bool{}
	lines := make([]indexedLine, 0, len(idxs))
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
		lines = append(lines, indexedLine{idx: idx, line: Line{A: ab[0], B: ab[1]}})
	}
	return lines, covered, true
}

// pointsExcept returns pts with any occurrence of skip removed, preserving order.
func pointsExcept(pts []Point, skip Point) []Point {
	out := pts[:0:0]
	for _, p := range pts {
		if p != skip {
			out = append(out, p)
		}
	}
	return out
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
