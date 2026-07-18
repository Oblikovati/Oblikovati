// SPDX-License-Identifier: GPL-2.0-only

package ipt

// Geometry-first sketch resolution. A line's curve node caches the exact infinite line as
// (midpoint, unit-direction) inline, so a pure-line profile can be reconstructed WITHOUT resolving
// any point reference — the safety net for clusters resolveByRefs cannot resolve (its reference and
// point counts disagree). See ipt-sketch-entities memory + the resolver-ceiling breakthrough.

import (
	"encoding/binary"
	"math"
	"sort"
)

// readInlineLine fills a line's inline geometry (midpoint, unit-direction) and construction flag
// from its curve node. The four f64 sit at C+24+4*w1 — after the w1 endpoint refs and one leading
// double — verified on w=2 and w=4 lines (k_base mid(1.5,0)dir(1,0); FlangeReelMotor's w=4 profile
// lines mid(0,±6.5)/(±6.5,0)). The construction bit is 0x00080000 of the flag word at C-8, guarded
// by the nullRef at C-16 that marks a real node header.
func readInlineLine(seg []byte, C int, lr *lineRefs) {
	w1 := int(binary.LittleEndian.Uint32(seg[C+4:]))
	base := C + 24 + 4*w1
	if base+32 <= len(seg) {
		lr.mid = Point2D{f64(seg, base), f64(seg, base+8)}
		lr.dir = Point2D{f64(seg, base+16), f64(seg, base+24)}
		lr.unit = absf(math.Hypot(lr.dir.X, lr.dir.Y)-1) < 1e-6
	}
	if C >= 16 && binary.LittleEndian.Uint32(seg[C-16:]) == nullRef {
		lr.constr = binary.LittleEndian.Uint32(seg[C-8:])&constructionFlag != 0
	}
}

// reconstructInlineLoop rebuilds a pure-line profile from each line's inline (midpoint, direction),
// the exact infinite line cached in the curve node — needing NO point-reference resolution. It is a
// safety net for clusters resolveByRefs cannot resolve (its distinct-reference count disagrees with
// the collected-point count, e.g. from garbage or construction points). Because it never uses ref
// counts, it is immune to the count-parity entanglement that blocks dropping construction/garbage
// points elsewhere: construction and non-unit (garbage) lines are simply discarded here.
//
// The kept lines are ordered around their centroid and consecutive infinite lines intersected into
// corners. The loop is emitted ONLY when it is a clean convex polygon whose EVERY corner coincides
// with a collected point — the file's own coordinates, an independent oracle — so a wrong ordering
// can never invent geometry. Scoped to line-only clusters (arcs/circles → ok=false, fall through).
func reconstructInlineLoop(items []sketchItem) (Sketch, bool) {
	var lines []lineRefs
	var pts []Point2D
	for _, it := range items {
		switch {
		case it.arc != nil || it.circle != nil || it.ellipse != nil:
			return Sketch{}, false
		case it.line != nil:
			if it.line.unit && !it.line.constr {
				lines = append(lines, *it.line)
			}
		case it.pt != nil:
			pts = append(pts, it.pt.p)
		}
	}
	if len(lines) < 3 {
		return Sketch{}, false
	}
	orderByCentroidAngle(lines)
	corners := make([]Point2D, 0, len(lines))
	for i := range lines {
		p, ok := intersectLines(lines[i], lines[(i+1)%len(lines)])
		if !ok || !pointInSet(p, pts) {
			return Sketch{}, false
		}
		corners = append(corners, p)
	}
	if !distinctCorners(corners) {
		return Sketch{}, false
	}
	out := Sketch{Points: corners, Resolved: true}
	for i := range corners {
		out.Lines = append(out.Lines, Line{A: corners[i], B: corners[(i+1)%len(corners)]})
	}
	return out, true
}

// orderByCentroidAngle sorts lines by the angle of their midpoint about the collective centroid, so
// consecutive lines bound consecutive edges of a convex loop.
func orderByCentroidAngle(lines []lineRefs) {
	var cx, cy float64
	for _, l := range lines {
		cx, cy = cx+l.mid.X, cy+l.mid.Y
	}
	cx, cy = cx/float64(len(lines)), cy/float64(len(lines))
	sort.Slice(lines, func(i, j int) bool {
		return math.Atan2(lines[i].mid.Y-cy, lines[i].mid.X-cx) < math.Atan2(lines[j].mid.Y-cy, lines[j].mid.X-cx)
	})
}

// intersectLines intersects two infinite lines (point + direction); ok=false when near-parallel.
func intersectLines(a, b lineRefs) (Point2D, bool) {
	den := a.dir.X*b.dir.Y - a.dir.Y*b.dir.X
	if absf(den) < 1e-9 {
		return Point2D{}, false
	}
	t := ((b.mid.X-a.mid.X)*b.dir.Y - (b.mid.Y-a.mid.Y)*b.dir.X) / den
	return Point2D{a.mid.X + t*a.dir.X, a.mid.Y + t*a.dir.Y}, true
}

// pointInSet reports whether p coincides (to sketch precision) with a collected point.
func pointInSet(p Point2D, pts []Point2D) bool {
	for _, q := range pts {
		if math.Hypot(q.X-p.X, q.Y-p.Y) < 1e-3 {
			return true
		}
	}
	return false
}

// distinctCorners reports whether every reconstructed corner is distinct — a degenerate loop
// (two coincident corners from duplicate/collinear edges) is rejected.
func distinctCorners(cs []Point2D) bool {
	for i := range cs {
		for j := i + 1; j < len(cs); j++ {
			if math.Hypot(cs[i].X-cs[j].X, cs[i].Y-cs[j].Y) < 1e-3 {
				return false
			}
		}
	}
	return true
}
