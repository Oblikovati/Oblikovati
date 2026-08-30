// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"sort"
)

// Sketch decoding — the SHARED RECORD DECODING (M48 #2229 split of sketch.go). The byte-level scan of
// PmDCSegment into raw per-entity records: forEachNamelessNode walks the geometry-carrier nodes,
// collectItems reads every point/line/circle/arc/ellipse by its invariant curve signature, and the
// record types (idPoint/lineRefs/circleEnt/arcEnt/ellipseEnt/sketchItem) carry the decoded fields.
// Turning these records into a Sketch (reference resolution / convex fallback) lives in
// sketch_assemble.go; the public Sketch model and entry points in sketch.go.

const (
	geomTag     = 0x8000000C // 2D geometry payload tag on a standalone point
	curveMarker = 0x30000002 // first word of the curve signature
	refBit      = 0x80000000 // set on an entity-reference u32 (a line's endpoint refs)
	// nodeTagBase is the geometry-node type tag with its low byte masked off: a real tag is
	// 0x800000XX, so `tag & 0xFFFFFF00 == nodeTagBase` accepts only genuine node tags.
	nodeTagBase = 0x80000000
)

// curveHead is the first word (0x30000002, little-endian bytes) of a sketch curve's 16-byte
// header, which is: 0x30000002 | w1 | w2 | 0x10. Keying on it (rather than the preceding tag)
// finds curves in both standalone sketches and feature-consumed profiles, whose leading tag
// differs (0x8000000C vs 0x80000016). After the header: for a line, two endpoint entity-refs (at
// +16 and +20); for a circle, the centre (2 float64 from +16), an entity-ref, and the radius
// (float64 at +36).
//
// The two middle words w1,w2 are small per-endpoint counts — 2 at a plain vertex, but 3 (rarely
// 4) where extra geometry meets it. They must NOT be hardcoded: keying on the old fixed 0x02,0x02
// header silently dropped a shaft's axis edge and an interior segment (LeverShaft decoded 11 of 13
// lines), leaving an OPEN profile that resolved by count yet revolved into a wrong solid. Anchoring
// on the invariant 0x30000002 … 0x10 with w1,w2 in [curveWordMin,curveWordMax] recovers them while
// still rejecting a stray 0x30000002 whose following words are float mantissa (large) or 1.
var curveHead = []byte{0x02, 0, 0, 0x30}

const (
	curveTailMark = 0x10 // the invariant 4th word of a curve header
	curveWordMin  = 2    // w1,w2 bounds: a curve endpoint's count is ≥2; >4 is not a curve header
	curveWordMax  = 4
	// ellipseSentinel is the u32 at curve+72 that marks a sketch ellipse. An ellipse's node has the
	// same shape as a circle's (centre by reference, +16 not a ref) but adds an inline unit
	// major-axis (+36/+44) and two semi-axis lengths (majorR@+52, minorR@+60); a real circle instead
	// carries denormal garbage (~1e-308) in those slots and a node reference (0x800000xx) at +72, so
	// the sentinel plus normal-range radii cleanly tell the two apart.
	ellipseSentinel = 0x01170000

	// constructionFlag marks a curve entity as Inventor construction / reference geometry — a dashed
	// diagonal, centreline, or reference line that constrains the sketch but is NOT part of a
	// profile. It is bit 0x00080000 of the curve's dcNode FLAG word, which sits in the 24-byte node
	// header preceding the curve signature: tag@C-24, id@C-20, nullRef@C-16, marker@C-12, flag@C-8,
	// typetag@C-4, then the 0x30000002 curve header at C. Validated against live Inventor's
	// Construction property: bit-exact counts on FlangeReelMotor (2/2) and EncoderFrame (2/2), zero
	// on the shaft fixtures, and it never fires on real profile geometry.
	//
	// NOT YET ACTED ON — deliberately. A construction curve references the sketch's List6 constraint
	// nodes rather than geometry points, so today it decodes as a phantom line; but simply skipping
	// construction curves in collectItems regressed the library batch (SOLID 35→29). Removing the
	// curve leaves its now-unreferenced points still collected, which breaks the #refs==#points
	// parity resolveByRefs relies on — the same entanglement that blocks the denormal-garbage-point
	// fix. Construction geometry must be dropped together with its construction-only points, and that
	// requires the resolver rework (see the memory note ipt-sketch-entities). This constant is kept
	// so that work has the marker ready. To read it: flag@C-8, guarded by nullRef present at C-16.
	constructionFlag = 0x00080000
)

// idPoint is a decoded geometry point tagged with its entity id, needed to resolve which
// point a line's endpoint reference names.
type idPoint struct {
	id uint32
	p  Point2D
}

// lineRefs is a line's two endpoint entity references (paired == both were real refs), plus the
// line's inline geometry: a curve node caches the exact infinite line as (midpoint, unit-direction)
// f64 immediately after its endpoint refs (at C+24+4*w1). This is decoded WITHOUT reference
// resolution, so it recovers a line's position/orientation even when the point-ref rank-alignment
// fails — the basis for reconstructInlineLoop. unit is set when the stored direction is unit length
// (a genuine line); an uninitialised/garbage slot holds a huge non-unit value and clears it.
type lineRefs struct {
	a, b   uint32
	paired bool
	mid    Point2D
	dir    Point2D
	unit   bool
	constr bool // Inventor construction/reference geometry (not part of a profile)
}

// circleEnt is a decoded circle plus the entity reference to its centre point. Inventor
// stores a circle's centre BY REFERENCE (the inline centre bytes are zero); the inline
// value is correct only for an origin circle that has no separate centre-point node.
type circleEnt struct {
	radius float64
	inline Point2D
	ref    uint32
	hasRef bool
}

// arcEnt is a decoded arc: entity references to its two endpoints and its centre, plus the
// radius (stored inline). Endpoints and centre resolve via the shared point-reference map.
type arcEnt struct {
	start, end, center uint32
	radius             float64
}

// ellipseEnt is a decoded ellipse: the entity reference to its centre point (resolved via the
// shared point-reference map, like a circle's centre), plus the inline unit major-axis direction
// and the two semi-axis lengths.
type ellipseEnt struct {
	ref            uint32
	axisX, axisY   float64
	majorR, minorR float64
}

// sketchItem is one decoded sketch entity in file order (by byte offset): a point, a
// circle, a line, an arc, or an ellipse — exactly one is non-nil.
type sketchItem struct {
	off     int
	pt      *idPoint
	circle  *circleEnt
	line    *lineRefs
	arc     *arcEnt
	ellipse *ellipseEnt
}

// collectItems reads every sketch entity — points (nameless node whose geometry tag is
// not a curve, plus its id and coords) and curves (by their invariant signature, split
// into lines with endpoint refs vs circles) — and returns them sorted by byte offset.
func collectItems(seg []byte) []sketchItem {
	var items []sketchItem
	forEachNamelessNode(seg, func(j int) {
		// The geometry tag at +20 must be a real node-type tag of the form 0x800000XX (line
		// 0x8000000C, consumed 0x8000000E, point 0x80000017, …) — not merely a word with the high
		// bit set. Requiring the exact shape (matching the curve scan below) rejects phantom points:
		// a stray node whose +16 flag holds a float and whose +20 word is garbage-with-the-top-bit
		// (e.g. 0xFFFF9673) otherwise reads as a spurious (0,0) vertex, and — now that the point
		// gate accepts nonzero +16 flags — would land in a neighbouring cluster and break its loop.
		if binary.LittleEndian.Uint32(seg[j+20:])&0xFFFFFF00 != nodeTagBase ||
			binary.LittleEndian.Uint32(seg[j+24:]) == curveMarker {
			return
		}
		x, y := f64(seg, j+24), f64(seg, j+32)
		if !math.IsNaN(x) && !math.IsNaN(y) && absf(x) < 1e4 && absf(y) < 1e4 {
			ip := idPoint{id: binary.LittleEndian.Uint32(seg[j+4:]), p: Point2D{x, y}}
			items = append(items, sketchItem{off: j, pt: &ip})
		}
	})
	for c := 0; ; {
		k := indexOf(seg[c:], curveHead)
		if k < 0 {
			break
		}
		C := c + k
		c = C + 1
		if C < 4 || C+44 > len(seg) {
			continue
		}
		// A curve header is 0x30000002 | w1 | w2 | 0x10 with w1,w2 the small per-endpoint counts,
		// preceded by a geometry-node TYPE tag of the form 0x800000XX (line 0x80000019, consumed
		// 0x80000016, standalone 0x8000000C). Requiring the invariant tail word (0x10), the bounded
		// middle words, and the exact node-tag shape — not merely the top bit — rejects both a stray
		// 0x30000002 inside float data and the parallel dimension/constraint nodes (whose preceding
		// word is a float, not a node tag), either of which would otherwise inject a phantom curve.
		if w1 := binary.LittleEndian.Uint32(seg[C+4:]); w1 < curveWordMin || w1 > curveWordMax {
			continue
		}
		if w2 := binary.LittleEndian.Uint32(seg[C+8:]); w2 < curveWordMin || w2 > curveWordMax {
			continue
		}
		if binary.LittleEndian.Uint32(seg[C+12:]) != curveTailMark ||
			binary.LittleEndian.Uint32(seg[C-4:])&0xFFFFFF00 != nodeTagBase {
			continue
		}
		if w16 := binary.LittleEndian.Uint32(seg[C+16:]); w16&refBit != 0 {
			w20 := binary.LittleEndian.Uint32(seg[C+20:])
			// A curve with two endpoint refs AND a centre ref + positive radius is an arc; the
			// same shape without those is a straight line.
			if center := binary.LittleEndian.Uint32(seg[C+32:]); center&refBit != 0 && w20&refBit != 0 {
				if r := f64(seg, C+36); r > 1e-6 && r < 1e5 {
					items = append(items, sketchItem{off: C, arc: &arcEnt{
						start: w16 &^ refBit, end: w20 &^ refBit, center: center &^ refBit, radius: r,
					}})
					continue
				}
			}
			lr := lineRefs{a: w16 &^ refBit, b: w20 &^ refBit, paired: w20&refBit != 0}
			readInlineLine(seg, C, &lr)
			items = append(items, sketchItem{off: C, line: &lr})
		} else if cref := binary.LittleEndian.Uint32(seg[C+32:]); cref&refBit != 0 && isEllipseNode(seg, C) {
			// Same node shape as a referenced-centre circle, but the sentinel + normal-range radii
			// (a circle carries denormals and a node ref there) identify a genuine ellipse.
			items = append(items, sketchItem{off: C, ellipse: &ellipseEnt{
				ref: cref &^ refBit, axisX: f64(seg, C+36), axisY: f64(seg, C+44),
				majorR: f64(seg, C+52), minorR: f64(seg, C+60),
			}})
		} else {
			ce := circleEnt{radius: f64(seg, C+36), inline: Point2D{f64(seg, C+16), f64(seg, C+24)}}
			if cref&refBit != 0 {
				ce.ref, ce.hasRef = cref&^refBit, true
			}
			items = append(items, sketchItem{off: C, circle: &ce})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].off < items[j].off })
	return items
}

// isEllipseNode reports whether the referenced-centre curve at C is a sketch ellipse rather than
// a circle. It requires the ellipse sentinel at +72 AND both semi-axis lengths (majorR@+52,
// minorR@+60) in a normal positive range: a circle stores denormal garbage (~1e-308) in those
// two slots and a node reference (0x800000xx) at +72, so all three checks fail on a circle. The
// three independent signals make a false positive (a circle read as an ellipse) impossible.
func isEllipseNode(seg []byte, C int) bool {
	if C+80 > len(seg) || binary.LittleEndian.Uint32(seg[C+72:]) != ellipseSentinel {
		return false
	}
	majorR, minorR := f64(seg, C+52), f64(seg, C+60)
	return majorR > 1e-6 && majorR < 1e5 && minorR > 1e-6 && minorR < 1e5
}

// forEachNamelessNode calls fn(nodeStart) for each nameless graph node (a sketch entity
// carrier): dcNodeTag(+0), id(+4), nullRef(+8), a schema marker(+12), a flag word(+16), then
// a geometry tag(+20). The +12 marker's VALUE is not keyed on (it is 0x0C20 in API-authored
// corpus parts but varies in real parts — 0x0A96, 0x0B5C, …), but its HIGH BIT is: a
// geometry carrier's marker is a small enum (< 0x80000000), whereas a structural/typed node
// sets the high bit there (e.g. 0x80000018), so the high bit cleanly rejects non-geometry
// nodes. The +16 flag word is NOT constrained: it is 0 for interior vertices but 0x40 for the
// sketch origin and 0x8000 for some points, so the earlier w@16==0 gate silently dropped the
// origin — forcing a fragile off-by-one synthesis in resolveByRefs and failing outright on any
// profile that drops two or more flagged points. collectItems still validates the geometry tag
// and coordinate range, so this coarse structural filter (nullRef + small marker) suffices.
func forEachNamelessNode(seg []byte, fn func(j int)) {
	for i := 0; i+24 <= len(seg); i++ {
		if binary.LittleEndian.Uint32(seg[i:]) != dcNodeTag {
			continue
		}
		if binary.LittleEndian.Uint32(seg[i+8:]) != nullRef ||
			binary.LittleEndian.Uint32(seg[i+12:]) >= nodeTagBase {
			continue
		}
		fn(i)
	}
}
