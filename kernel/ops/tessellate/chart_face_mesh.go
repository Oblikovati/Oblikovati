// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The chart-driven curved-face mesher (ADR-0061): REGION from the face's chart, POINTS from its
// shared edges.
//
// It is the general answer for a trimmed face on a periodic surface whose boundary makes a shape no
// wrapping mesher recognises — the band wrapping a ring's azimuth, the torus carrying two drill
// windows, the sphere and cylinder of the folded-window family. Each of those used to end at the
// surface's WHOLE parametric domain, which covers material the face does not carry and omits the
// face's own boundary.
//
// It works in the COVERING space, not in a cut branch, and that is the whole of why it is general:
// the chart's contours already close there (a wrapping region runs rim → seam → rim → seam), so no
// seam has to be cut, no keyhole assembled, and no boundary point invented. The construction is the
// one the periodic B-spline cover already uses (#1510, periodic_nurbs_cover.go): replicate the
// boundary and the interior grid one period either way, triangulate the lot ONCE with the boundary
// segments as constraints, then keep each triangle whose centroid lies in the chart's own window and
// on its material side. Period-shifted copies of a boundary point are the SAME 3D point, so welding
// the kept triangles closes every seam; a sphere pole's whole row welds to one vertex and the
// degenerate triangles around it drop out.

// chartFaceMesh meshes a trimmed curved face from the parametric trim it carries. ok=false in two
// different situations, and the log is what tells them apart: the face was never this mesher's (it
// carries no chart, or its surface wraps in neither direction), which is the ordinary route onto the
// generic (u,v) trim path; or the mesher OWNED the face and gave it up, which is a degradation and is
// recorded into log unconditionally, HERE rather than at any caller, so a caller holding the router's
// log can neither forget the report nor condition it (#3520, chart_decline.go). That the log a caller
// holds IS the router's is a convention archguard enforces, not the compiler — see
// TestOnlyTheCurvedFaceRouterOwnsAChartDeclineLog.
//
// Example: if m, ok := chartFaceMesh(f, f.Geometry(), q, log); ok { return m }
func chartFaceMesh(f *topo.Face, s geom.Surface, q Quality, log *chartDeclineLog) (*Mesh, bool) {
	r, ok := newChartRegion(f, s)
	if !ok {
		return nil, false // never this mesher's face: no chart recorded, or an aperiodic surface
	}
	m, why := chartRegionMesh(f, s, r, q)
	if why != "" {
		log.declined(why)
		return nil, false
	}
	return m, true
}

// chartRegionMesh builds the covering mesh for a region this mesher owns, or names WHY it gave up — the
// reason the router reports, so a reader can act on it rather than being told only that something fell
// back.
func chartRegionMesh(f *topo.Face, s geom.Surface, r chartRegion, q Quality) (*Mesh, string) {
	chains := chartBoundaryChains(f, s, r, q)
	b := newChartCover(s, r, q)
	loops := b.addChains(chains)
	b.addInterior(chains)
	kept, why := b.meshOrRefusal(b.keptWithoutRimEars(loops))
	if why != "" {
		return nil, why
	}
	return weldAndCertifyChartMesh(b, kept, chains)
}

// weldAndCertifyChartMesh does the three steps that turn kept covering triangles into a shippable
// face: it WELDS them (period-shifted replicas of a boundary point are one 3D point, which is what
// closes the seam), REPAIRS folds, and then GATES the result. The gate is the part with the receipt
// below; the weld and the fold repair are here because the gate is only meaningful on the welded mesh.
//
// The gate accepts the mesh only when its unpaired edges are EXACTLY the boundary segments it was
// given — the same SET, not merely the same count.
//
// The set, because a count cancels. It was a count, and on the merged cocylindrical wall at
// PropertyQuality it read 578 against a rim of 578 while FIVE of those free edges were no rim segment
// at all and five rim segments carried the wrong number of triangles: the face passed a gate it should
// have failed, and the body it belongs to shipped ten unpaired edges. Neighbouring a wrong edge with a
// missing one is exactly the shape a chart that disagrees with its own boundary produces, so the
// cancelling pair is the case worth catching, not an unlikely coincidence.
//
// Both directions still matter. Extra free edges mean the mesh tore (a seam that did not close, a region
// the triangulation lost). MISSING ones mean it closed over its own boundary — a covering of the whole
// surface has no free edges at all, and that is precisely the full-domain degradation this mesher exists
// to remove. Either way the face is DECLINED and the router's defect reporter speaks, rather than the
// wrong mesh shipping quietly.
func weldAndCertifyChartMesh(b *chartCover, kept [][3]int, chains []chartChain) (*Mesh, string) {
	pos, nrm, idx := weldCoverTriangles(b.pos, b.nrm, kept)
	m := patchMeshFrom(pos, nrm, idx)
	validate.RepairFolds(m, 8)
	// The weld can take every triangle away even though the covering kept some — a sphere pole's whole
	// row welds to ONE vertex and the triangles around it become degenerate. chartRimMismatch answers
	// (-1, -1) for a mesh there is nothing to compare, and -1 unpaired edges is not an offending value
	// a reader can act on, so the empty case is named rather than formatted (#3520 review M2).
	if m == nil || m.TriangleCount() == 0 {
		return nil, fmt.Sprintf("welding its %d kept covering triangle(s) left no triangle at all", len(kept))
	}
	extra, missing := chartRimMismatch(m, chains, b.weld)
	if extra != 0 || missing != 0 {
		return nil, fmt.Sprintf("the mesh it built is not bounded by its own rim: %d unpaired edge(s) "+
			"are no rim segment and %d rim segment(s) it does not bound", extra, missing)
	}
	return m, ""
}

// chartRimMismatch is the gate's own comparison, in the two numbers it decides on: how many of the
// mesh's unpaired edges are no rim segment, and how many rim segments the mesh does not bound. (0, 0) is
// a patch bounded by exactly its rim.
//
// It is the ONE place a rim is keyed, and it is handed the covering's OWN weld resolution rather than
// deriving one: the rule that binds triangles to the rim (chart_face_rim_side.go) keys the same set,
// and two derivations that agree only while the covering and the welded mesh share a bounding box are
// an agreement by luck (#3518 review M9). The tests read the gate through it rather than counting
// segments their own way: a count taken on a different weld grid, or per chain instead of per face, is
// a different question, and a corpus row that asks a different question from the gate is not asserting
// the gate. (−1, −1) for a mesh there is nothing to compare.
func chartRimMismatch(m *Mesh, chains []chartChain, grid float64) (extra, missing int) {
	if m == nil || m.TriangleCount() == 0 {
		return -1, -1
	}
	rim := chainSegmentKeys(chains, grid)
	for _, e := range weldedFreeEdgeKeys(m, grid) {
		if !rim[e] {
			extra++
			continue
		}
		delete(rim, e)
	}
	return extra, len(rim)
}

// weldedFreeEdgeKeys is the mesh's unpaired edges, keyed the way a boundary segment is — so the two can
// be compared as SETS and not merely counted.
func weldedFreeEdgeKeys(m *Mesh, grid float64) [][2][3]int64 {
	torn := tornMeshEdges(m)
	out := make([][2][3]int64, 0, len(torn))
	for _, t := range torn {
		out = append(out, orderedSegmentKey(m.Positions[t.lo], m.Positions[t.hi], grid))
	}
	return out
}

// chartCover is the shared covering accumulator (covering_vertices.go) with the chart's own
// bookkeeping: the region it reads material from, the grid's parameter lines, and the pad past the
// branch window within which a replica is worth carrying.
type chartCover struct {
	coverVertices
	s        geom.Surface
	r        chartRegion
	us, vs   []float64 // the interior grid's parameter lines
	padU     float64   // how far past the branch window a replica is kept
	padV     float64
	rim      int            // vertices [0, rim) are the boundary chains', laid before any interior node
	rimChain map[[2]int]int // each boundary segment's own chain, directed (chart_face_rim_side.go)
	// node is the interior grid's lattice: which covering vertex each (shift, station pair) was laid
	// at. It is what lets the structured interior be emitted as a quad mesh instead of triangulated
	// (chart_structured_interior.go).
	node   *gridNodeIndex
	chains int // how many boundary chains the face has
	// weld is the ONE resolution this covering welds and keys a rim at, so the rule that binds
	// triangles to the rim and the gate that judges the result cannot key it differently (#3518
	// review M9).
	weld float64
	// rimSideConflict is a chain whose own segments named two material sides — a refusal, not a mesh.
	// It is deliberately STICKY across the ear-splitting rounds: a contradiction is a statement about
	// the CHART, and splitting an ear adds interior points without changing what the chart says, so a
	// later round that happened to read unanimous would be luck rather than a repair. Refusing the
	// face is the conservative direction and the router reports it (#3518 review N3).
	rimSideConflict string
}

// newChartCover sizes the covering: the trim-local (u,v) metric, the interior grid's parameter lines
// (the SAME adaptive breakpoints the full-domain grid uses, so a charted face is faceted at the density
// the quality asks for and not at one of this mesher's own), and the replication pad.
func newChartCover(s geom.Surface, r chartRegion, q Quality) *chartCover {
	b := &chartCover{s: s, r: r, us: chartStations(s, r, q, true), vs: chartStations(s, r, q, false)}
	b.su, b.sv = trimMetricScale(s, r.contours[0])
	b.us, b.vs = balancedCoverGrid(b.us, b.vs, b.su, b.sv)
	cell := coarsestCoverCell(b.us, b.vs, b.su, b.sv)
	b.padU = chartPad(cell, b.su, r.uHi-r.uLo)
	b.padV = chartPad(cell, b.sv, r.vHi-r.vLo)
	b.normalAt = func(u, v float64) math.Vector3 { return s.NormalAt(r.fold(u, v)) }
	b.carry = b.inPad
	return b
}

// chartCoverPadStations is how many of the covering's COARSEST cells of replicated covering are kept
// either side of the branch window. Within that band the point set is exactly periodic, so a triangle
// whose circumcircle fits inside it is built identically on both sides of the window and exactly one of
// its replicas is kept.
//
// The measure is the coarsest cell of the WHOLE covering, not the axis's own station gap, and that
// distinction is the whole of the pad's correctness. A STRAIGHT axis gets no chord subdivision, so a
// cylinder wall's covering is three ROWS tall against 256 columns and its triangles reach the whole
// height; three column gaps is 0.22 mm on the #1738 corner junction against circumcircles of several
// millimetres. Measured there at PropertyQuality, the two ends of the window triangulated the same rim
// differently and the canonical window kept one triangle from each: 871 unpaired edges against a rim of
// 868, three rim segments carrying two triangles each, and the body cracked with 6 free edges.
//
// Replicating the whole period instead is correct but costs what the pad exists to avoid: measured on
// the figure-eight torus band at PropertyQuality, 23.16 s against 1.69 s for the same 97460 triangles,
// because a doubly-periodic covering has nine shifts.
const chartCoverPadStations = 3

// balancedCoverGrid gives a STRAIGHT axis the cell size the other axis's chord asks for.
//
// The breakpoints each axis brings are the CHORD's, and a straight axis — a cylinder or cone's height —
// has no chord to resolve, so it arrives with nothing but the package's minimum-cell floor. Its cells
// are then as long as the whole face while the wrapping axis's are a chord apart, and the covering's
// triangles reach right across it: measured on the #1738 corner junction, the wall's facets cut 0.16 mm
// INTO a solid of radius 3, and 2617 interior points of a 60³ audit read outside the body they are
// inside. The mesh is watertight and its area is right to 0.07 %, which is why only a membership oracle
// sees it. Square cells are also what the replication pad's own premise assumes.
//
// Only a floored axis is refined. An axis the chord already subdivided carries the density the quality
// asked for, and second-guessing it moves every curved face's faceting (measured: re-balancing a
// torus's tube against its ring drove the figure-eight band from 110.947 mm² to the whole torus).
func balancedCoverGrid(us, vs []float64, su, sv float64) ([]float64, []float64) {
	uCell, vCell := widestStationGap(us)*su, widestStationGap(vs)*sv
	return balancedAxis(us, su, vCell), balancedAxis(vs, sv, uCell)
}

// balancedAxis subdivides each cell of a FLOORED axis into equal parts until none is longer than target
// (a 3D length), never exceeding maxInteriorCells cells in all. An axis the chord subdivided, a target
// of zero or a degenerate scale leaves it alone.
func balancedAxis(ps []float64, scale, target float64) []float64 {
	if len(ps) > minInteriorCells+1 {
		return ps // the chord already chose this axis's density
	}
	steps := balancedSubdivision(ps, scale, target)
	if steps < 2 {
		return ps
	}
	out := make([]float64, 0, (len(ps)-1)*steps+1)
	for i := 0; i+1 < len(ps); i++ {
		for k := range steps {
			out = append(out, ps[i]+(ps[i+1]-ps[i])*float64(k)/float64(steps))
		}
	}
	return append(out, ps[len(ps)-1])
}

// balancedSubdivision is how many equal parts each of an axis's cells is cut into: enough that its
// widest is no longer than target, and few enough that the axis stays under maxInteriorCells cells.
func balancedSubdivision(ps []float64, scale, target float64) int {
	if len(ps) < 2 || target <= 0 || scale <= 0 {
		return 1
	}
	widest := widestStationGap(ps) * scale
	steps := int(stdmath.Ceil(widest / target))
	return min(steps, maxInteriorCells/(len(ps)-1))
}

// coarsestCoverCell is the diagonal, as a 3D length, of the largest cell the interior grid leaves — the
// scale of the largest empty circle the triangulation can have, and so of the largest circumcircle the
// pad has to contain.
func coarsestCoverCell(us, vs []float64, su, sv float64) float64 {
	return stdmath.Hypot(widestStationGap(us)*su, widestStationGap(vs)*sv)
}

// widestStationGap is the largest gap between consecutive stations of one axis (0 for fewer than two).
func widestStationGap(ps []float64) float64 {
	gap := 0.0
	for i := 1; i < len(ps); i++ {
		gap = stdmath.Max(gap, ps[i]-ps[i-1])
	}
	return gap
}

// chartPad is the replication pad on one axis, in that axis's own parameter: chartCoverPadStations of
// the covering's coarsest cell, measured as a 3D length and carried back through the axis's metric. A
// pad wider than the window itself is pointless — the whole period is already replicated — so it is
// capped there.
func chartPad(cell, scale, span float64) float64 {
	if scale <= 0 {
		return span
	}
	return stdmath.Min(chartCoverPadStations*cell/scale, span)
}

// inPad reports whether a replicated (u,v) is close enough to the branch window to be worth carrying.
func (b *chartCover) inPad(u, v float64) bool {
	if b.r.uPer && (u < b.r.uLo-b.padU || u > b.r.uHi+b.padU) {
		return false
	}
	return !b.r.vPer || (v >= b.r.vLo-b.padV && v <= b.r.vHi+b.padV)
}

// chartStations are one axis's grid parameter lines over the branch window. The closing station of a
// wrapping axis is dropped: it is the same surface line as the opening one, and its replica a period
// along supplies it.
func chartStations(s geom.Surface, r chartRegion, q Quality, alongU bool) []float64 {
	lo, hi := r.uLo, r.uHi
	if !alongU {
		lo, hi = r.vLo, r.vHi
	}
	ps := atLeastMinimumCells(unionIsoparmParams(s, r.uLo, r.uHi, r.vLo, r.vHi, alongU, q), lo, hi)
	if wraps := alongU && r.uPer || !alongU && r.vPer; wraps && len(ps) > 1 {
		return ps[:len(ps)-1]
	}
	return ps
}

// atLeastMinimumCells floors an axis's station count at the package's own minimum cell count.
//
// A STRAIGHT axis — a cylinder or cone's height — needs no chord subdivision, so the adaptive
// breakpoints are just its two ends and the covering gets no interior row at all. The triangulation
// then has nothing but the boundary to work with and reaches right across the face for its diagonals
// (measured on a windowed rod wall: triangles whose plane passed within 0.1 of the axis, and a volume
// integral an eighth of the truth). adaptiveStep already floors every step at minInteriorCells cells
// for the same reason; this applies that floor to the breakpoints.
func atLeastMinimumCells(ps []float64, lo, hi float64) []float64 {
	if len(ps) > minInteriorCells || hi <= lo {
		return ps
	}
	out := make([]float64, 0, minInteriorCells+1)
	for i := 0; i <= minInteriorCells; i++ {
		out = append(out, lo+(hi-lo)*float64(i)/minInteriorCells)
	}
	return out
}

// addChains lays every boundary chain into the covering at each period shift, returning the constraint
// pairs the triangulation is aligned to.
func (b *chartCover) addChains(chains []chartChain) [][]int {
	var loops [][]int
	var segs []rimSegment
	// One covering LOCATION is one vertex, and the resolution that decides it is the boundary's own.
	// A face whose boundary touches itself hands the same (u,v) to addChain more than once, and a CDT
	// cannot recover a constraint incident to a vertex another vertex sits on (#3551,
	// coverVertices.MergeCoincidentLocations). The grid is taken from the chains rather than from
	// b.pos, because b.pos is empty here and the replicas about to be laid carry no 3D point the
	// chains do not already have — so it is the same number weldGrid gives afterwards.
	b.weld = weldGrid(chainPoints(chains))
	b.MergeCoincidentLocations(b.weld)
	for si, sh := range b.r.shifts() {
		for ci, c := range chains {
			pairs := b.addChain(c.p3, c.uv, sh[0], sh[1], si)
			for _, p := range pairs {
				segs = append(segs, rimSegment{a: p[0], b: p[1], chain: ci})
			}
			loops = append(loops, pairs...)
		}
	}
	b.rim, b.chains = len(b.pos), len(chains)
	b.rimChain = b.directedRimSegments(segs, chains, b.weld)
	return loops
}

// chainPoints is every chain's 3D points, grouped, for the one weld resolution this covering uses.
func chainPoints(chains []chartChain) [][]math.Point3 {
	out := make([][]math.Point3, 0, len(chains))
	for _, c := range chains {
		out = append(out, c.p3)
	}
	return out
}

// addInterior lays the covering's grid nodes: every station pair the chart covers and which stands
// clear of the boundary, replicated at each period shift. One 3D point per node, shared by its
// replicas, so the seam welds exactly.
func (b *chartCover) addInterior(chains []chartChain) {
	b.node = newGridNodeIndex(len(b.r.shifts()), len(b.us), len(b.vs))
	margin := b.nodeMargin()
	for i, u := range b.us {
		for j, v := range b.vs {
			if !b.stationIsMaterial(i, j) {
				continue
			}
			clear, keep := b.nodeClearance(chains, i, j, u, v, margin)
			if !keep {
				continue
			}
			fu, fv := b.r.fold(u, v)
			b.node.record(i, j, b.addReplicas(b.s.PointAt(fu, fv), u, v), clear)
		}
	}
}

// addReplicas adds one interior node at every period shift that lands inside the pad, returning the
// covering vertex it laid at each shift (-1 where the pad declined it) so the grid node index can hold
// the lattice the structured interior is emitted from (chart_structured_interior.go).
func (b *chartCover) addReplicas(p math.Point3, u, v float64) []int {
	at := make([]int, len(b.r.shifts()))
	for si, sh := range b.r.shifts() {
		at[si] = -1
		if b.inPad(u+sh[0], v+sh[1]) {
			at[si] = b.add(p, u+sh[0], v+sh[1], si)
		}
	}
	return at
}

// stationIsMaterial reports whether the grid node at station (i, j) is material.
//
// The query is pulled half a grid gap INWARD at a bounded axis's end, where the node sits exactly on
// the chart's border. An even-odd count on a border answers by which side the ray was cast from, not by
// the geometry — measured, a sphere's v = +π/2 pole row read OUTSIDE while v = −π/2 read inside, and the
// cap around the north pole came back missing (32 free edges on the rod ∪ ball sphere). Half a grid gap
// is a mesh-density quantity, not a tolerance: it is the cell the node bounds.
func (b *chartCover) stationIsMaterial(i, j int) bool {
	u := b.us[i] + inwardProbe(b.us, i, b.r.uPer)
	v := b.vs[j] + inwardProbe(b.vs, j, b.r.vPer)
	return b.r.covers(u, v)
}

// inwardProbe is how far a station's membership query moves off a bounded axis's end: half the gap to
// its neighbour at the first and last station, nothing anywhere else (and nothing at all on a wrapping
// axis, which has no end).
func inwardProbe(stations []float64, i int, periodic bool) float64 {
	if periodic || len(stations) < 2 {
		return 0
	}
	if i == 0 {
		return (stations[1] - stations[0]) / 2
	}
	if i == len(stations)-1 {
		return (stations[i-1] - stations[i]) / 2
	}
	return 0
}

// keepChartTriangles is the covering's classification, in the three steps it takes: the CHART's own
// verdict at each candidate translate's centroid, then the BOUNDARY's (a triangle carrying a rim
// segment lies on that chain's material side, chart_face_rim_side.go), then one translate per 3D
// triangle (chart_face_replica.go). The chart decides the interior, the boundary decides what it
// bounds, and the replica selection decides which copy ships.
//
// The chart's verdict is ONE call to covers at the centroid. A second opinion for a centroid sitting on
// a contour edge used to follow it; it decided nothing anywhere in the corpus over six decades of band
// width and is deleted (#3519, see covers in chart_face_region.go for the sweep).
func (b *chartCover) keepChartTriangles(tris [][3]int) [][3]int {
	keep := make([]bool, len(tris))
	for i, t := range tris {
		u, v := b.centroid(t)
		keep[i] = b.r.windowCandidate(u, v) && b.r.covers(u, v)
	}
	b.bindToTheRim(tris, keep)
	b.keepOneReplicaEach(tris, keep)
	return selectTriangles(tris, keep)
}
