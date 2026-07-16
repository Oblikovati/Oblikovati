// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"encoding/binary"
	"math"
)

// The sketch entity graph names each entity's endpoints by POINT INDEX, so a sketch decodes exactly
// even when its entity-kind class string is absent (a sketch that re-uses a kind first registered by
// an earlier sketch writes the class by object tag, losing the sgLineHandle/sgArcHandle string). The
// index space is the RAW cached-point record sequence — every `1e 00 x y` record in stream order with
// DUPLICATES KEPT (a sketch anchored at the origin caches a duplicate origin that occupies an index).
//
// Each entity record is anchored on its construction-flag record `80 bf 00 00 <flag:u32>` (after the
// last cached point, one per entity in draw order) and carries:
//
//	u16 @ flag+31 = start point index, u16 @ flag+33 = end point index
//
// a == b marks a closed curve (circle); a != b an open one (line or arc). The KIND comes from the
// record size, read as the flag-to-flag gap: a line record is 92 bytes and an arc/circle record 112,
// and an arc's flag sits 12 bytes later inside its record, so each gap uniquely names BOTH kinds.
//
// Validated exact on box10 (square), tri, twoline_open (disjoint open lines), rrect (4 lines + 4
// fillet arcs incl. centres), cyl/circoff/twocirc/arc90 (radii), and perno_fmtb — the re-used-class
// oracle whose 8 sketches carry only one class string. This is format B (SW2026); format A serialises
// entities under an older schema and has no such record, so it keeps the class-string path.
const (
	lineRecordSize = 92  // bytes, flag-to-flag, line -> line
	arcRecordSize  = 112 // bytes, flag-to-flag, arc -> arc
	lineToArcGap   = 104 // line record + the arc's 12-byte later flag
	arcToLineGap   = 100 // arc record - the arc's 12-byte later flag
)

// entityIndexRecord is one entity's flag record: its stream offset, endpoint point indices, and
// whether it is construction geometry.
type entityIndexRecord struct {
	off          int
	a, b         int
	construction bool
}

// decodeEntityGraph rebuilds a sketch's entities exactly from the point-index graph. It returns the
// geometry with a per-line construction flag (index-aligned to lines). ok is false unless every guard
// holds — at least two entities (a single entity has no gap to size it), every gap a known record
// size, a consistent kind chain, every index in range, and every arc/circle centre equidistant from
// its endpoints — so a sketch it cannot form exactly is left to the class-string path rather than
// guessed. The gap check also rejects a sketch holding an ellipse or spline (their records are sized
// differently), so those are never mistaken for circles.
func decodeEntityGraph(region []byte) (lines []Line, arcs []Arc, circles []Circle, lineConstruction []bool, ok bool) {
	raw := rawPointSequence(region)
	recs, ok := entityIndexRecords(region)
	if !ok || len(recs) < 2 {
		return nil, nil, nil, nil, false
	}
	isArc, ok := kindsFromGaps(recs)
	if !ok {
		return nil, nil, nil, nil, false
	}
	for i, r := range recs {
		if r.a < 0 || r.a >= len(raw) || r.b < 0 || r.b >= len(raw) {
			return nil, nil, nil, nil, false // an index outside the cached points: the space is polluted
		}
		if !isArc[i] {
			if raw[r.a] == raw[r.b] {
				return nil, nil, nil, nil, false // a degenerate (zero-length) line
			}
			lines = append(lines, Line{A: raw[r.a], B: raw[r.b]})
			lineConstruction = append(lineConstruction, r.construction)
			continue
		}
		if r.a == r.b {
			c, ok := circleAt(raw, r.a)
			if !ok {
				return nil, nil, nil, nil, false
			}
			circles = append(circles, c)
			continue
		}
		a, ok := arcAt(raw, r.a, r.b)
		if !ok {
			return nil, nil, nil, nil, false
		}
		arcs = append(arcs, a)
	}
	return lines, arcs, circles, lineConstruction, true
}

// rawPointSequence returns every cached coordinate in record order, duplicates kept — the space the
// entity records index into.
func rawPointSequence(region []byte) []Point {
	placed := placedPoints(region)
	out := make([]Point, 0, len(placed))
	for _, p := range placed {
		out = append(out, p.p)
	}
	return out
}

// entityIndexRecords reads the per-entity flag records that follow the last cached point, each with
// its endpoint point indices. ok is false when no cached point anchors the scan.
func entityIndexRecords(region []byte) ([]entityIndexRecord, bool) {
	last := lastPointOffset(region)
	if last < 0 {
		return nil, false
	}
	var out []entityIndexRecord
	for i := last; ; {
		m := bytes.Index(region[i:], constructionAnchor)
		if m < 0 {
			break
		}
		at := i + m
		i = at + 1
		if at+35 > len(region) {
			break
		}
		flag := binary.LittleEndian.Uint32(region[at+len(constructionAnchor):])
		if flag != entityFlagNormal && flag != entityFlagConstruction {
			continue
		}
		out = append(out, entityIndexRecord{
			off:          at,
			a:            int(binary.LittleEndian.Uint16(region[at+31:])),
			b:            int(binary.LittleEndian.Uint16(region[at+33:])),
			construction: flag&constructionBit != 0,
		})
	}
	return out, true
}

// kindsFromGaps solves each entity's kind from the flag-to-flag record sizes. Each gap names the kind
// of both the entity it starts at and the next one, so the chain must agree at every overlap; any
// unknown gap or disagreement returns ok=false.
func kindsFromGaps(recs []entityIndexRecord) ([]bool, bool) {
	isArc := make([]bool, len(recs))
	for k := 0; k+1 < len(recs); k++ {
		first, second, ok := kindsOfGap(recs[k+1].off - recs[k].off)
		if !ok {
			return nil, false
		}
		if k > 0 && isArc[k] != first {
			return nil, false // the chain disagrees about entity k's kind
		}
		isArc[k], isArc[k+1] = first, second
	}
	return isArc, true
}

// kindsOfGap maps a flag-to-flag gap onto the kinds of the two entities it spans (true = arc/circle).
func kindsOfGap(gap int) (first, second, ok bool) {
	switch gap {
	case lineRecordSize:
		return false, false, true
	case lineToArcGap:
		return false, true, true
	case arcToLineGap:
		return true, false, true
	case arcRecordSize:
		return true, true, true
	default:
		return false, false, false
	}
}

// circleAt builds the full circle whose rim point is at index rim: SolidWorks caches a circle's centre
// immediately before its rim point. ok is false if there is no preceding point or the radius is zero.
func circleAt(raw []Point, rim int) (Circle, bool) {
	if rim < 1 {
		return Circle{}, false
	}
	c := raw[rim-1]
	r := dist(c, raw[rim])
	if r <= 0 {
		return Circle{}, false
	}
	return Circle{Center: c, Radius: r}, true
}

// arcAt builds the arc spanning the endpoints at indices a and b. Its centre is a cached point
// adjacent to them — before the start for a standalone arc, between them for an arc chained into a
// profile (a fillet) — so the candidates are checked and only an equidistant one is accepted, never
// assumed.
func arcAt(raw []Point, a, b int) (Arc, bool) {
	start, end := raw[a], raw[b]
	for _, i := range []int{a - 1, a + 1, b - 1, b + 1} {
		if i < 0 || i >= len(raw) {
			continue
		}
		c := raw[i]
		r := dist(c, start)
		if r > 1e-9 && math.Abs(dist(c, end)-r) <= 1e-7 {
			return Arc{Center: c, Start: start, End: end, Radius: r}, true
		}
	}
	return Arc{}, false
}
