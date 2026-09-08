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
	kept := b.keepCanonical(constrainedTriangulationAll(b.xy, loops))
	if len(kept) == 0 {
		return nil, false
	}
	pos, nrm, idx := weldCoverTriangles(b.pos, b.nrm, kept)
	m := patchMeshFrom(pos, nrm, idx)
	validate.RepairFolds(m, 8)
	return m, chartMeshIsBoundedByItsRim(m, chains)
}

// chartMeshIsBoundedByItsRim accepts the mesh only when its unpaired edges number no more than the
// boundary segments it was given — a patch whose only free edges ARE its rim. A charted face that came
// out torn (a seam that did not close, a region the triangulation lost) is DECLINED, so the router's
// defect reporter speaks instead of the tear shipping quietly.
func chartMeshIsBoundedByItsRim(m *Mesh, chains []chartChain) bool {
	return m != nil && m.TriangleCount() > 0 && WeldedFreeEdgeCount(m) <= chainSegmentCount(chains)
}

// chartCover accumulates the covering: exact 3D positions with their surface normals, the
// metric-scaled (u,v) the triangulation runs in, and the UNFOLDED (u,v) the canonical selection reads.
type chartCover struct {
	s      geom.Surface
	r      chartRegion
	su, sv float64   // the (u,v) metric, so the triangulation runs in a space isometric to 3D
	us, vs []float64 // the interior grid's parameter lines
	padU   float64   // how far past the branch window a replica is kept
	padV   float64
	pos    []math.Point3
	nrm    []math.Vector3
	xy     [][2]float64
	uu, vv []float64
}

// chartCoverPadStations is how many grid gaps of replicated covering are kept either side of the branch
// window. The point set within it is then EXACTLY periodic out past any triangle's circumcircle, so a
// seam-spanning triangle is built identically on both sides and exactly one of its replicas is kept.
const chartCoverPadStations = 3

// newChartCover sizes the covering: the trim-local (u,v) metric, the interior grid's parameter lines
// (the SAME adaptive breakpoints the full-domain grid uses, so a charted face is faceted at the density
// the quality asks for and not at one of this mesher's own), and the replication pad.
func newChartCover(s geom.Surface, r chartRegion, q Quality) *chartCover {
	su, sv := trimMetricScale(s, r.contours[0])
	us := chartStations(s, r, q, true)
	vs := chartStations(s, r, q, false)
	return &chartCover{s: s, r: r, su: su, sv: sv, us: us, vs: vs,
		padU: chartPad(len(us), r.uLo, r.uHi), padV: chartPad(len(vs), r.vLo, r.vHi)}
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

// chartPad is the replication pad in one axis: chartCoverPadStations grid gaps.
func chartPad(stations int, lo, hi float64) float64 {
	if stations < 2 {
		return hi - lo
	}
	return chartCoverPadStations * (hi - lo) / float64(stations)
}

// add records one covering vertex: its exact 3D point, the surface normal at its FOLDED parameters
// (the branch a replica lives on is a bookkeeping fact, not a geometric one), and both the scaled and
// the raw (u,v).
func (b *chartCover) add(p math.Point3, u, v float64) int {
	fu, fv := b.r.fold(u, v)
	i := len(b.pos)
	b.pos = append(b.pos, p)
	b.nrm = append(b.nrm, b.s.NormalAt(fu, fv))
	b.xy = append(b.xy, [2]float64{u * b.su, v * b.sv})
	b.uu, b.vv = append(b.uu, u), append(b.vv, v)
	return i
}

// inPad reports whether a replicated (u,v) is close enough to the branch window to be worth carrying.
func (b *chartCover) inPad(u, v float64) bool {
	if b.r.uPer && (u < b.r.uLo-b.padU || u > b.r.uHi+b.padU) {
		return false
	}
	return !b.r.vPer || (v >= b.r.vLo-b.padV && v <= b.r.vHi+b.padV)
}

// addChains lays every boundary chain into the covering at each period shift, returning the constraint
// pairs. A chain is constrained SEGMENT by segment rather than as a closed loop because a boundary that
// wraps a period does not close in the covering space; per-segment pairs constrain it either way, and
// the caller classifies triangles itself rather than by the loop-parity flood.
func (b *chartCover) addChains(chains []chartChain) [][]int {
	var loops [][]int
	for _, sh := range b.r.shifts() {
		for _, c := range chains {
			loops = append(loops, b.addChain(c, sh)...)
		}
	}
	return loops
}

// addChain adds one shifted chain, keeping only the points inside the pad, and returns its segments.
func (b *chartCover) addChain(c chartChain, sh [2]float64) [][]int {
	idx := make([]int, len(c.uv))
	for i, p := range c.uv {
		u, v := p[0]+sh[0], p[1]+sh[1]
		idx[i] = -1
		if b.inPad(u, v) {
			idx[i] = b.add(c.p3[i], u, v)
		}
	}
	return chainConstraints(idx)
}

// chainConstraints is the chain's consecutive index pairs, skipping any segment an end of which fell
// outside the pad — that segment's own replica carries it.
func chainConstraints(idx []int) [][]int {
	var segs [][]int
	for i := 0; i+1 < len(idx); i++ {
		if idx[i] >= 0 && idx[i+1] >= 0 {
			segs = append(segs, []int{idx[i], idx[i+1]})
		}
	}
	return segs
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
// the shared edge's much coarser chord polygon. Between the two lies a band, as wide as the edge
// discretisation's sagitta, where the chart's answer does not describe the mesh's boundary at all — the
// chart calls a sliver outside the chord "inside the hole", so the triangles there are dropped and the
// mesh's rim detours around the gap through interior nodes the neighbour face has never heard of
// (measured on the one-window torus complement: 32 rim edges where the shared oval has 28).
//
// Keeping every interior node half a chord clear of the boundary puts the band entirely inside the
// FIRST triangle off the boundary, whose centroid is then a third of a chord away — outside a band that
// is at most chord²/8ρ wide, since a discretisation whose chord is not far shorter than the curve's own
// radius would not have been accepted. It is a mesh-density quantity, not a tolerance: where the shared
// edge is finely sampled the chord is small and the grid clearance below governs instead.
const chartBoundaryClearance = 0.5

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
func (b *chartCover) chainIsNear(c chartChain, sh [2]float64, u, v, gridMargin float64) bool {
	x, y := u*b.su, v*b.sv
	margin := stdmath.Max(gridMargin, chartBoundaryClearance*c.chord)
	if !boxIsNear(b.scaledChainBox(c, sh), x, y, margin) {
		return false
	}
	for i := 0; i+1 < len(c.uv); i++ {
		a, e := c.uv[i], c.uv[i+1]
		if distToSeg2D(x, y, (a[0]+sh[0])*b.su, (a[1]+sh[1])*b.sv, (e[0]+sh[0])*b.su, (e[1]+sh[1])*b.sv) < margin {
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

// keepCanonical keeps each triangle whose centroid lies in the chart's branch window AND on its
// material side. Replication gives every seam-spanning triangle exactly one translate with its centroid
// in the window, so this de-duplicates the seam without ever cutting the mesh at it.
func (b *chartCover) keepCanonical(tris [][3]int) [][3]int {
	out := make([][3]int, 0, len(tris))
	for _, t := range tris {
		u := (b.uu[t[0]] + b.uu[t[1]] + b.uu[t[2]]) / 3
		v := (b.vv[t[0]] + b.vv[t[1]] + b.vv[t[2]]) / 3
		if b.r.inWindow(u, v) && b.r.covers(u, v) {
			out = append(out, t)
		}
	}
	return out
}
