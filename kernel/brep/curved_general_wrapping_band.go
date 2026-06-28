// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// OUTSIDE-keep wrapping-band emission for the general curved∩curved cut/join (Oblikovati#1476). A boolean
// that keeps the part of a ruled side OUTSIDE the other solid produces a band that WRAPS the whole azimuth —
// a tube with no contractible outer loop. The half-space (u,v) emission assumes a contractible outer
// (loops[0]) with the rest as holes, so it mis-files one of the tube's two ends (a rim) as a "hole" and the
// mesher carves the wrong region. The two correct shapes are reproduced here (these are the shapes the
// removed per-pair bespoke handlers used to build; the general pipeline is now the only path):
//
//   - a HOLED tube (the fat wall punched by the rod): two constant-v rims plus contractible imprint holes.
//     Bridged by a seam ruling into one keyhole outer loop (rim→seam→rim→seam), holes inside — the
//     seam-cut form periodicBandGrid meshes (the holed-wall shape).
//   - a TWO-RIM band (a rod stub poking out): one cap rim + one imprint rim, both full-wrap, no holes —
//     emitted as a two-closed-loop face the ruled saddle loft (twoClosedRimBandMesh) lofts rim-to-rim.
//
// Disjoint bands (the two stubs a rod leaves either side of the fat) are split first by connected component,
// so each emits its own face. Gated to ruledUV solidMode+wrapping, so every half-space/torus/intersect path
// is untouched.

// wrappingSolidFaces emits the kept region of a ruled solid-membership side as one curvedFace per connected
// band (#1476). ok=false unless this is the solid-membership wrapping case (the cut/join OUTSIDE/tunnel wall),
// so the half-space and the non-wrapping intersect paths fall through to the ordinary (u,v) emission.
func (c ruledUV) wrappingSolidFaces(kept []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) ([]curvedFace, bool) {
	if !c.solidMode || !c.wrapsAllU() {
		return nil, false
	}
	var faces []curvedFace
	for _, comp := range keptComponents(kept, c.vPeriodic()) {
		face, ok := c.bandFace(comp, segs, surface, f)
		if !ok {
			return nil, false
		}
		faces = append(faces, face)
	}
	return faces, len(faces) > 0
}

// bandFace builds one curvedFace for a single connected wrapping band: the loops are classified into the
// band's two full-wrap ends and its contractible holes, then assembled as a keyhole (holed tube) or a
// two-closed-loop band (a clean stub) — the two shapes the curved mesher renders correctly (#1476).
func (c ruledUV) bandFace(comp []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) (curvedFace, bool) {
	loops := dropArtificialLoops(&c, chainLoops(keptBoundaryEdges(comp, c.vPeriodic())), segs)
	emitted, ok := emitKeptLoops(&c, loops, segs)
	if !ok {
		return curvedFace{}, false
	}
	var ends, holes []emittedLoop
	for _, e := range emitted {
		if c.loopWrapsU(e) {
			ends = append(ends, e)
		} else {
			holes = append(holes, e)
		}
	}
	if len(ends) != 2 {
		return curvedFace{}, false // a wrapping band has exactly two full-wrap ends; anything else defers
	}
	if len(holes) > 0 && allRimEdges(ends[0].face) && allRimEdges(ends[1].face) {
		return c.keyholeTubeFace(holes, surface, f), true // a holed tube (the fat wall): bridge the rims
	}
	return curvedFace{
		surface: surface, reversed: f.reversed, lineage: f.lineage,
		loops: c.orientWrappingBand(emitted),
	}, true
}

// keyholeTubeFace assembles a holed tube (the fat wall punched by the rod) as a single contractible face: its
// two constant-v rims are bridged by a seam ruling into one outer loop (rim→seam→rim→seam), with the imprint
// holes inside. This is the seam-cut form holedConicWallMesh unrolls — the natural two-rim tube has no
// contractible outer, so the unroller (which needs the outer to span the v-extent) cannot chart it. The rims
// are the ORIGINAL band circles, so the outer's rim edges weld to the planar caps that share them (#1476).
func (c ruledUV) keyholeTubeFace(holes []emittedLoop, surface geom.Surface, f curvedFace) curvedFace {
	loops := make([]curvedLoop, 0, len(holes)+1)
	loops = append(loops, curvedLoop{edges: c.keyholeOuter()})
	for _, h := range holes {
		loops = append(loops, curvedLoop{edges: h.face})
	}
	return curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage, loops: loops}
}

// keyholeOuter bridges the band's two rims into one outer loop at their NATIVE seam (the rim circles' own
// angle-zero ruling, where both rims are anchored to the same azimuth, so the bridge is a v-ruling on the
// surface). The loop runs up the bridge, around the top rim (reversed when the source side traverses its top
// rim opposite its cap), back down the bridge, around the bottom rim — the holed-wall keyhole shape, but
// reusing the original rim circles so the outer's rims weld to the caps without re-seaming (#1476).
func (c ruledUV) keyholeOuter() []loopEdge {
	seamBot := c.band.bottomCirc.PointAt(0)
	seamTop := c.band.topCirc.PointAt(0)
	ruling := geom.NewLineSegment(seamBot, seamTop)
	top := loopEdge{curve: c.band.topCirc, t0: 0, t1: 1}
	bot := loopEdge{curve: c.band.bottomCirc, t0: 0, t1: 1}
	if c.band.topRimReversed {
		top.t0, top.t1 = 1, 0 // top rim opposite its cap; bottom forward (the band convention)
	} else {
		bot.t0, bot.t1 = 1, 0
	}
	return []loopEdge{{curve: ruling, t0: 0, t1: 1}, top, {curve: ruling, t0: 1, t1: 0}, bot}
}

// orientWrappingBand applies the ruled band's rim convention to a wrapping band's loops: the rim at the TOP
// (vMax) end is reversed when the source side traverses its top rim reversed, so it keeps the sense opposite
// the cap that shares it; the bottom (vMin) rim and the imprint loops keep their arrangement winding (the
// imprint sense is already set by emitImprintRun, #1477). Unlike orientLoops this keys the reversal on each
// rim's v-LEVEL, not its position in the loop list — a stub's single cap rim can be the vMin OR vMax end (the
// rod's two ends), and a tube carries both — so the position-0 rule of the half-space path does not fit (#1476).
func (c ruledUV) orientWrappingBand(emitted []emittedLoop) []curvedLoop {
	out := make([]curvedLoop, 0, len(emitted))
	for _, e := range emitted {
		edges := e.face
		if allRimEdges(e.face) && c.isTopRim(e.mv) && c.band.topRimReversed {
			edges = reverseEdgeChain(e.face)
		}
		out = append(out, curvedLoop{edges: edges})
	}
	return out
}

// isTopRim reports whether a loop's mean axial level is nearer the band's vMax (top) rim than its vMin
// (bottom) rim — the rim the topRimReversed convention flips (#1476).
func (c ruledUV) isTopRim(mv float64) bool {
	return stdmath.Abs(mv-c.band.vMax) < stdmath.Abs(mv-c.band.vMin)
}

// loopWrapsU reports whether an emitted boundary loop spans the WHOLE azimuth — a band end (a rim or a
// full-wrap imprint loop) rather than a contractible hole. A pure rim run is always full-wrap; an imprint
// loop is sampled and its (seam-relative) u-extent measured, so the lens that wraps the rod (a stub's inner
// rim) is told apart from the small lens that punches the fat wall (a hole) (#1476).
func (c ruledUV) loopWrapsU(e emittedLoop) bool {
	if allRimEdges(e.face) {
		return true
	}
	uMin, uMax := stdmath.Inf(1), stdmath.Inf(-1)
	for _, le := range e.face {
		for k := 0; k <= 16; k++ {
			u := float64(c.paramOf(le.curve.PointAt(le.t0 + (le.t1-le.t0)*float64(k)/16)).X)
			uMin, uMax = stdmath.Min(uMin, u), stdmath.Max(uMax, u)
		}
	}
	return uMax-uMin >= 2*stdmath.Pi-0.5
}

// keptComponents groups the kept arrangement cells into connected bands by shared (seam-folded) edges, so the
// two disjoint stubs a rod leaves either side of the fat become separate faces (#1476). Two cells are in the
// same band iff they share an edge; the seam fold makes cells on either side of the azimuth seam adjacent, so
// a band wrapping the seam stays one component.
func keptComponents(kept []Face2D, vPeriodic bool) [][]Face2D {
	w := newSeamWelder(vPeriodic)
	uf := newBandUF(len(kept))
	edgeCell := map[[2]int]int{}
	for ci, cell := range kept {
		eachCellEdge(cell, func(a, b math.Point2) {
			key := canonEdgeKey(w.add(a), w.add(b))
			if prev, ok := edgeCell[key]; ok {
				uf.union(prev, ci)
			} else {
				edgeCell[key] = ci
			}
		})
	}
	groups := map[int][]Face2D{}
	order := []int{}
	for ci := range kept {
		r := uf.find(ci)
		if _, seen := groups[r]; !seen {
			order = append(order, r)
		}
		groups[r] = append(groups[r], kept[ci])
	}
	out := make([][]Face2D, 0, len(order))
	for _, r := range order {
		out = append(out, groups[r])
	}
	return out
}

// eachCellEdge calls fn for every boundary edge of a cell (its outer ring and each hole ring).
func eachCellEdge(cell Face2D, fn func(a, b math.Point2)) {
	ring := func(poly []math.Point2) {
		for i, n := 0, len(poly); i < n; i++ {
			a, b := poly[i], poly[(i+1)%n]
			if a.DistanceTo(b) > arrTol {
				fn(a, b)
			}
		}
	}
	ring(cell.Outer)
	for _, h := range cell.Holes {
		ring(h)
	}
}

// canonEdgeKey returns an undirected edge key (endpoints sorted) so the two cells sharing an edge map it to
// the same key regardless of traversal direction.
func canonEdgeKey(u, v int) [2]int {
	if u > v {
		return [2]int{v, u}
	}
	return [2]int{u, v}
}

// bandUF is a tiny union-find over kept-cell indices for the connected-band grouping.
type bandUF struct{ parent []int }

func newBandUF(n int) *bandUF {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &bandUF{parent: p}
}

func (u *bandUF) find(i int) int {
	for u.parent[i] != i {
		u.parent[i] = u.parent[u.parent[i]]
		i = u.parent[i]
	}
	return i
}

func (u *bandUF) union(a, b int) { u.parent[u.find(a)] = u.find(b) }
