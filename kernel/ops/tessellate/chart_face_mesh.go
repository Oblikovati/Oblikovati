// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
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

// chartFaceMesh meshes a trimmed curved face from the parametric trim it carries. ok=false when the
// face carries no chart, when its surface wraps in neither direction, or when the mesh that comes out
// is not bounded by its own boundary — the caller then reports the discarded trim rather than shipping
// a mesh nobody certified.
//
// Example: if m, ok := chartFaceMesh(f, f.Geometry(), q); ok { return m }
func chartFaceMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	r, ok := newChartRegion(f, s)
	if !ok {
		return nil, false
	}
	chains := chartBoundaryChains(f, s, r, q)
	b := newChartCover(s, r, q)
	loops := b.addChains(chains)
	b.addInterior(chains)
	kept := b.keepChartTriangles(constrainedTriangulationAll(b.xy, loops))
	if len(kept) == 0 {
		return nil, false
	}
	pos, nrm, idx := weldCoverTriangles(b.pos, b.nrm, kept)
	m := patchMeshFrom(pos, nrm, idx)
	validate.RepairFolds(m, 8)
	return m, chartMeshIsBoundedByItsRim(m, chains)
}

// chartMeshIsBoundedByItsRim accepts the mesh only when its unpaired edges are EXACTLY the boundary
// segments it was given — the same SET, not merely the same count.
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
func chartMeshIsBoundedByItsRim(m *Mesh, chains []chartChain) bool {
	extra, missing := chartRimMismatch(m, chains)
	return m != nil && m.TriangleCount() > 0 && extra == 0 && missing == 0
}

// chartRimMismatch is the gate's own comparison, in the two numbers it decides on: how many of the
// mesh's unpaired edges are no rim segment, and how many rim segments the mesh does not bound. (0, 0) is
// a patch bounded by exactly its rim.
//
// It is the ONE place a rim is keyed. The tests read the gate through it rather than counting segments
// their own way: a count taken on a different weld grid, or per chain instead of per face, is a
// different question, and a corpus row that asks a different question from the gate is not asserting the
// gate. (−1, −1) for a mesh there is nothing to compare.
func chartRimMismatch(m *Mesh, chains []chartChain) (extra, missing int) {
	if m == nil || m.TriangleCount() == 0 {
		return -1, -1
	}
	rim := chainSegmentKeys(chains, geom.ResolutionForPoints(m.Positions).Weld())
	for _, e := range weldedFreeEdgeKeys(m) {
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
func weldedFreeEdgeKeys(m *Mesh) [][2][3]int64 {
	grid := geom.ResolutionForPoints(m.Positions).Weld()
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
	s      geom.Surface
	r      chartRegion
	us, vs []float64 // the interior grid's parameter lines
	padU   float64   // how far past the branch window a replica is kept
	padV   float64
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
	for _, sh := range b.r.shifts() {
		for _, c := range chains {
			loops = append(loops, b.addChain(c.p3, c.uv, sh[0], sh[1])...)
		}
	}
	return loops
}

// addInterior lays the covering's grid nodes: every station pair the chart covers and which stands
// clear of the boundary, replicated at each period shift. One 3D point per node, shared by its
// replicas, so the seam welds exactly.
func (b *chartCover) addInterior(chains []chartChain) {
	margin := b.nodeMargin()
	for i, u := range b.us {
		for j, v := range b.vs {
			if !b.stationIsMaterial(i, j) || !b.clearOfChains(chains, u, v, margin) {
				continue
			}
			fu, fv := b.r.fold(u, v)
			p := b.s.PointAt(fu, fv)
			b.addReplicas(p, u, v)
		}
	}
}

// addReplicas adds one interior node at every period shift that lands inside the pad.
func (b *chartCover) addReplicas(p math.Point3, u, v float64) {
	for _, sh := range b.r.shifts() {
		if b.inPad(u+sh[0], v+sh[1]) {
			b.add(p, u+sh[0], v+sh[1])
		}
	}
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

// chartBoundaryClearance is how much of a boundary CHORD an interior node must keep clear of it.
//
// The region comes from the chart, which samples the boundary curve finely; the mesh's own boundary is
// the shared edge's much coarser chord polygon. Between the two lies a band where the chart's answer
// does not describe the mesh's boundary at all — the chart calls a sliver outside the chord "inside the
// hole", so the triangles there are dropped and the mesh's rim detours around the gap through interior
// nodes the neighbour face has never heard of (measured on the one-window torus complement: 32 rim
// edges where the shared oval has 28).
//
// It is a MEASURED constant, not a derived one, and saying which is the honest part. The band's width is
// bounded by the discretisation's sagitta, chord²/8ρ, and a clearance of k chords puts the first
// triangle's centroid (2/3)·k·chord out — so the sagitta argument alone is satisfied by any
// k > 3·chord/(16ρ), about 0.03 for the faces here. It does not predict what actually fails, because the
// band is not always a sagitta: where a boundary TOUCHES itself the rim is sampled coarsely right at the
// touch (the lemniscate complement carries 0.17 rad of u in one chord against the covering's own 0.0245
// stations) and an interior node lands INSIDE the chord rather than beside it. What bounds that is the
// chord itself.
//
// So it is swept, on the three bodies whose charted faces the clearance governs — the genus-1 lemniscate
// complement, RS− and RD− — at both facetings, counting failures over
// ./kernel/ops/tessellate/ ./kernel/ops/boolean/:
//
//	k          0.125  0.25  0.5  0.75  0.875  1.0  1.1  1.25  1.5  2.0  3.0  4.0
//	failures      5     4    2    0      0     0    0     1     3    4    7   16
//
// A plateau of 0.75 … 1.1, pinned at its middle. Below it the complement tears at PropertyQuality (272
// free edges, the face declined and fallen to the surface's whole domain); above it the clearance starts
// eating the interior next to a coarse boundary and the complement's own volume walks away from the
// analytic (−1.30 % at 0.875, −1.72 % at 1.25, −11.0 % at 4.0).
const chartBoundaryClearance = 0.875 // tol:mesh-density (chords; swept 0.125…4, plateau 0.75…1.1)

// chartNodeClearance is the fraction of a grid gap an interior node must keep from the boundary. A node
// ON a constraint owns no triangle and derails the segment recovery; one just inside it makes a sliver
// against the exact edge points, which this mesher may not move.
const chartNodeClearance = 0.3

// nodeMargin is that clearance in the metric-scaled (u,v), so it reads as a 3D distance on both axes.
func (b *chartCover) nodeMargin() float64 {
	gu := (b.r.uHi - b.r.uLo) / stdmath.Max(1, float64(len(b.us))) * b.su
	gv := (b.r.vHi - b.r.vLo) / stdmath.Max(1, float64(len(b.vs))) * b.sv
	return chartNodeClearance * stdmath.Min(gu, gv)
}

// clearOfChains reports whether a grid node stands at least margin from every boundary segment, on
// every period image of it (a node just inside one seam is close to a boundary just inside the other).
func (b *chartCover) clearOfChains(chains []chartChain, u, v, margin float64) bool {
	for _, sh := range b.r.shifts() {
		for _, c := range chains {
			if b.chainIsNear(c, sh, u, v, margin) {
				return false
			}
		}
	}
	return true
}

// chainIsNear reports whether the shifted chain comes within its clearance of (u,v) in the scaled (u,v).
//
// The clearance is the chain's MEAN chord, deliberately, and reading each segment's own length instead
// was tried and measured worse. A boundary is not sampled uniformly — the merged cocylindrical wall's
// notched rim carries 320 chords of 0.074 mm around its top and TWO of 4 mm down the boss's chord edges
// — but a clearance scaled to those two would clear a 4 mm disc of interior nodes off a face 4 mm tall
// and starve the region: measured, the merged band went from 10 unpaired edges at PropertyQuality to
// 469, and the figure-eight band overshot its analytic area. The mean is what the covering as a whole is
// sampled at, which is the scale the chart-versus-chord band is compared against.
func (b *chartCover) chainIsNear(c chartChain, sh [2]float64, u, v, gridMargin float64) bool {
	x, y := u*b.su, v*b.sv
	margin := stdmath.Max(gridMargin, chartBoundaryClearance*c.chord)
	if !boxIsNear(b.scaledChainBox(c, sh), x, y, margin) {
		return false
	}
	for i := 0; i+1 < len(c.uv); i++ {
		a, e := c.uv[i], c.uv[i+1]
		if distToSeg2D(x, y, (float64(a.X)+sh[0])*b.su, (float64(a.Y)+sh[1])*b.sv,
			(float64(e.X)+sh[0])*b.su, (float64(e.Y)+sh[1])*b.sv) < margin {
			return true
		}
	}
	return false
}

// scaledChainBox is a shifted chain's bounding box in the metric-scaled (u,v).
func (b *chartCover) scaledChainBox(c chartChain, sh [2]float64) [4]float64 {
	return [4]float64{(c.uMin + sh[0]) * b.su, (c.uMax + sh[0]) * b.su,
		(c.vMin + sh[1]) * b.sv, (c.vMax + sh[1]) * b.sv}
}

// boxIsNear reports whether (x,y) is within margin of the box [xLo,xHi]×[yLo,yHi].
func boxIsNear(box [4]float64, x, y, margin float64) bool {
	return x >= box[0]-margin && x <= box[1]+margin && y >= box[2]-margin && y <= box[3]+margin
}

// keepChartTriangles keeps each triangle whose centroid lies in the chart's branch window AND on its
// material side — the region's own two-part definition.
func (b *chartCover) keepChartTriangles(tris [][3]int) [][3]int {
	out := make([][3]int, 0, len(tris))
	for _, t := range tris {
		if u, v := b.centroid(t); b.r.inWindow(u, v) && b.triangleIsMaterial(t, u, v) {
			out = append(out, t)
		}
	}
	return out
}

// triangleIsMaterial answers the region for one triangle: at its centroid, and — only when that answers
// NO — by the majority of three points halfway from the centroid to each vertex.
//
// The retry is for a centroid that lands ON a contour edge, where an even-odd count answers by which
// side the ray was cast from rather than by the geometry. That is not a measure-zero curiosity here: a
// band's artificial seam can be SLANTED (the merged cocylindrical wall's runs from (0,0) to (−0.1963,10)),
// and a slope of exactly eight u-stations over the whole v range puts grid-built centroids EXACTLY on it
// — measured on that face at PropertyQuality, the centroid (−0.008181231, 0.416666667) and the seam agree
// to 1e-11, both branches read "outside", and the triangle vanished from both. Forty such holes tore the
// wall (615 unpaired edges against a rim of 578).
//
// The retry can only ADD a triangle the point test refused, never duplicate one: covers is periodic, so a
// triangle it accepts anywhere is accepted on exactly the one translate inWindow keeps. A majority, not
// "any", so a triangle that genuinely lies outside a real boundary — where at most one sub-point can fall
// the other side of the chart-versus-chord band — is still refused.
func (b *chartCover) triangleIsMaterial(t [3]int, u, v float64) bool {
	if b.r.covers(u, v) {
		return true
	}
	if fu, fv := b.r.fold(u, v); !b.r.onContour(fu, fv) {
		return false // a decided NO: the centroid is nowhere near a contour edge
	}
	votes := 0
	for _, i := range t {
		if b.r.covers((u+b.uu[i])/2, (v+b.vv[i])/2) {
			votes++
		}
	}
	return votes >= 2
}
