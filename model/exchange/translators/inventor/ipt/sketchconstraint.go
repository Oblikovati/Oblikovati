// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"sort"
)

// Sketch geometric-constraint decode (Inventor 2027). The type tags InventorLoader keys on
// (Horizontal2D=0x90874D98, Parallel2D=0x90874D95, …) are ABSENT in the 2027 node graph; a
// constraint is instead a dc node whose +16 word is the constraint List6 header 0x30000006,
// followed by two entity references (+36/+40) and a discriminator word at +44 (t44). The t44
// families were pinned by a single-variable corpus (one constraint per part, byte-diffed):
//
//	0x0000003e  coincidence  (line ref, endpoint ref)     — reproduced structurally by shared points
//	0x01030000  axis-align   (line ref, 0x00003b00)       — Horizontal if the line is horizontal, else Vertical
//	0x00400000  line-relate  (line ref, line ref)         — Parallel if the two lines are parallel, else Perpendicular
//	0x00000000  radius/dia   (0x10, circle ref, param@+60) — a dimension (value in the linked dN parameter)
//
// Horizontal vs Vertical, and Parallel vs Perpendicular, are byte-IDENTICAL in the node — the
// distinction is carried by the referenced geometry (which the diff proved). So these constraints
// are classified from the resolved lines' orientation, and because the geometry already satisfies
// them, applying them never moves a point (correctness-first): they only remove degrees of freedom.

const (
	constraintHdr = 0x30000006 // a constraint/dimension node's +16 word
	conRef1Off    = 36         // first entity reference
	conRef2Off    = 40         // second entity reference
	conDiscOff    = 44         // the t44 discriminator

	coincidenceDisc = 0x0000003e // line↔endpoint coincidence
	axisAlignDisc   = 0x01030000 // horizontal/vertical (one line)
	lineRelateDisc  = 0x00400000 // parallel/perpendicular (two lines)
	axisRadiusDisc  = 0x01150000 // a revolve radius dimension (distance from the x=0 centreline)
	midpointDisc    = 0x00000000 // midpoint (line + a point at its midpoint); shared with radius dims
	groundDisc      = 0x00010300 // ground: fully fix one entity at its current position
	emptyListMark   = 0x00000010 // the +32 empty-extra-list marker of a plain two-ref constraint node
)

// rawCon is a decoded constraint/dimension node before its references are resolved.
type rawCon struct {
	off    int
	r1, r2 uint32
	disc   uint32
}

// collectRawCons reads every constraint/dimension node (a dc node whose +16 word is the constraint
// List6 header), recording its two entity references and the t44 discriminator.
func collectRawCons(seg []byte) []rawCon {
	var out []rawCon
	for i := 0; i+conDiscOff+4 <= len(seg); i++ {
		if binary.LittleEndian.Uint32(seg[i:]) != dcNodeTag ||
			binary.LittleEndian.Uint32(seg[i+8:]) != nullRef ||
			binary.LittleEndian.Uint32(seg[i+16:]) != constraintHdr {
			continue
		}
		out = append(out, rawCon{
			off:  i,
			r1:   binary.LittleEndian.Uint32(seg[i+conRef1Off:]),
			r2:   binary.LittleEndian.Uint32(seg[i+conRef2Off:]),
			disc: binary.LittleEndian.Uint32(seg[i+conDiscOff:]),
		})
	}
	return out
}

// circleByEntityID maps each circle's ENTITY id (its centre-point reference + 1, the id a
// constraint uses to name the circle) to the circle with its centre resolved.
func circleByEntityID(seg []byte, vc map[uint32]Point2D) map[uint32]circleEnt {
	m := map[uint32]circleEnt{}
	for _, it := range collectItems(seg) {
		if it.circle == nil || !it.circle.hasRef {
			continue
		}
		ce := *it.circle
		if p, ok := vc[ce.ref]; ok {
			ce.inline = p
		}
		m[ce.ref+1] = ce
	}
	return m
}

func r4(v float64) int64 { return int64(math.Round(v * 1e4)) }

// geoEps bounds the direction test for classifying axis-alignment and parallel/perpendicular.
const geoEps = 1e-6

func isHorizontal(l [2]Point2D) bool { return math.Abs(l[0].Y-l[1].Y) < geoEps }
func isVertical(l [2]Point2D) bool   { return math.Abs(l[0].X-l[1].X) < geoEps }

// linesParallel reports whether the two segments are parallel (their direction cross product ≈ 0).
func linesParallel(a, b [2]Point2D) bool {
	ax, ay := a[1].X-a[0].X, a[1].Y-a[0].Y
	bx, by := b[1].X-b[0].X, b[1].Y-b[0].Y
	return math.Abs(ax*by-ay*bx) < geoEps*maxLen(ax, ay, bx, by)
}

// linesPerpendicular reports whether the two segments are perpendicular (direction dot ≈ 0).
func linesPerpendicular(a, b [2]Point2D) bool {
	ax, ay := a[1].X-a[0].X, a[1].Y-a[0].Y
	bx, by := b[1].X-b[0].X, b[1].Y-b[0].Y
	return math.Abs(ax*bx+ay*by) < geoEps*maxLen(ax, ay, bx, by)
}

// linesCollinear reports whether the two segments lie on the same infinite line: parallel AND
// one segment's start lies on the other's line (its offset from that line is ≈ 0).
func linesCollinear(a, b [2]Point2D) bool {
	if !linesParallel(a, b) {
		return false
	}
	ax, ay := a[1].X-a[0].X, a[1].Y-a[0].Y
	bx, by := b[0].X-a[0].X, b[0].Y-a[0].Y
	return math.Abs(ax*by-ay*bx) < geoEps*maxLen(ax, ay, bx, by)
}

// lengthsEqual reports whether two segments have the same length (to sketch precision).
func lengthsEqual(a, b [2]Point2D) bool {
	la := math.Hypot(a[1].X-a[0].X, a[1].Y-a[0].Y)
	lb := math.Hypot(b[1].X-b[0].X, b[1].Y-b[0].Y)
	return math.Abs(la-lb) < 1e-4
}

// midpointOf returns the midpoint of a segment.
func midpointOf(l [2]Point2D) Point2D {
	return Point2D{X: (l[0].X + l[1].X) / 2, Y: (l[0].Y + l[1].Y) / 2}
}

func maxLen(ax, ay, bx, by float64) float64 {
	return math.Max(math.Hypot(ax, ay), math.Hypot(bx, by))
}

// vertexCoords maps each geometry vertex reference (a curve endpoint id) to its coordinate. It
// resolves PER CLUSTER — each sketch's entities sit in one byte-offset run (clusterItems), and
// within a run the sorted distinct vertex references rank-align to the cluster's points in creation
// order (the correspondence resolveByRefs uses). Per-cluster alignment is what lets a multi-sketch
// part (a shaft's axis sketch + profile + work geometry) resolve — a global alignment fails on the
// count mismatch. A cluster whose counts disagree is skipped, so its constraints aren't bound to
// mis-aligned geometry.
func vertexCoords(seg []byte) map[uint32]Point2D {
	m := map[uint32]Point2D{}
	for _, cl := range clusterItems(seg) {
		refs := clusterVertexRefs(cl)
		pts := clusterPoints(cl)
		// A dimension inserts an extra TEXT point into the cluster (its label position), created
		// after the geometry, so a cluster can hold more points than curve-referenced vertices. The
		// geometry points come first (lower node id / byte offset), so the sorted references still
		// rank-align to the leading points; the trailing text points are simply left unmapped. Only
		// a cluster with FEWER points than references is genuinely inconsistent and skipped.
		if len(refs) == 0 || len(refs) > len(pts) {
			continue
		}
		for i, r := range refs {
			m[r] = pts[i]
		}
	}
	return m
}

// clusterVertexRefs returns the distinct point references in one cluster, ascending — every
// reference a curve makes to a point node: a line's two endpoints, an arc's endpoints AND centre,
// and a circle's centre. A centre matters even though it is not a "vertex": every centre has its
// own point node in clusterPoints, so omitting it leaves the reference set short of the point set
// and misaligns the rank correspondence (a circle-bearing cluster would resolve nothing; an arc's
// centre would rank-align to the wrong point). Including all centres keeps refs and points in step.
func clusterVertexRefs(cl []sketchItem) []uint32 {
	seen := map[uint32]bool{}
	for _, it := range cl {
		switch {
		case it.line != nil && it.line.paired:
			seen[it.line.a], seen[it.line.b] = true, true
		case it.arc != nil:
			seen[it.arc.start], seen[it.arc.end], seen[it.arc.center] = true, true, true
		case it.circle != nil && it.circle.hasRef:
			seen[it.circle.ref] = true
		}
	}
	refs := make([]uint32, 0, len(seen))
	for r := range seen {
		refs = append(refs, r)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i] < refs[j] })
	return refs
}

// clusterPoints returns the cluster's point coordinates in creation (byte-offset) order.
func clusterPoints(cl []sketchItem) []Point2D {
	var pts []Point2D
	for _, it := range cl {
		if it.pt != nil {
			pts = append(pts, it.pt.p)
		}
	}
	return pts
}

// coincidenceGroups maps each line reference to every point pinned to it by a 0x3e coincidence
// node — its two endpoints, plus any interior point-on-line vertex.
func coincidenceGroups(seg []byte, vc map[uint32]Point2D) map[uint32][]Point2D {
	grp := map[uint32][]Point2D{}
	for _, c := range collectRawCons(seg) {
		if c.disc != coincidenceDisc || c.r1&refBit == 0 || c.r2&refBit == 0 {
			continue
		}
		line, vert := c.r1&^refBit, c.r2&^refBit
		p, ok := vc[vert]
		if !ok {
			line, vert = c.r2&^refBit, c.r1&^refBit
			if p, ok = vc[vert]; !ok {
				continue
			}
		}
		grp[line] = append(grp[line], p)
	}
	return grp
}

// lineEndpoints maps each line reference to its two endpoint coordinates (a line with exactly two
// coincidence points). A line carrying an extra interior point-on-line vertex has >2 points and is
// intentionally left out here — point-on-line resolves its own endpoints via extremePair in
// decodePointOnLine, so this core resolver stays unchanged (recovering such lines here showed no
// benefit on the corpus and would broaden the resolver's behaviour without cause).
func lineEndpoints(seg []byte, vc map[uint32]Point2D) map[uint32][2]Point2D {
	out := map[uint32][2]Point2D{}
	for line, ps := range coincidenceGroups(seg, vc) {
		if len(ps) == 2 {
			out[line] = [2]Point2D{ps[0], ps[1]}
		}
	}
	return out
}
