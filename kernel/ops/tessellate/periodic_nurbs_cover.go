// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/math"
)

// coverShifts are the three period offsets (−P, 0, +P) replicated into the covering space, so every
// canonical triangle — including one spanning the seam — has all its neighbour nodes present.
var coverShifts = []float64{-1, 0, 1}

// coveringPeriodicMesh triangulates the closed-in-u band over three u-periods and keeps one canonical
// period (#1510). The rims and (whole) mouths are replicated across the shifts, a single constrained
// Delaunay covers them, and a triangle is kept iff its centroid lies in the canonical period AND in the
// material region (inside the band, outside every mouth). Period-shifted boundary vertices coincide in
// 3D, so welding the kept triangles closes the seam.
func coveringPeriodicMesh(s geom.BSplineSurface, q Quality, ulo, uhi float64, rims, mouths []cylLoop) *Mesh {
	period := uhi - ulo
	su, sv := MetricScale(s)
	vBot, vTop, vmin, vmax := rimBand(rims, ulo, period)
	b := &coverBuilder{s: s, su: su, sv: sv, ulo: ulo, period: period}
	loops := b.replicateBoundary(rims, mouths)
	b.addInteriorNodes(q, ulo, uhi, vmin, vmax, vBot, vTop, mouths)
	tris := constrainedTriangulationAll(b.xy, loops)
	if len(tris) == 0 {
		return nil
	}
	kept := b.selectCanonical(tris, ulo, period, vBot, vTop, mouths)
	if len(kept) == 0 {
		return nil
	}
	pos, nrm, idx := weldCoverTriangles(b.pos, b.nrm, kept)
	m := patchMeshFrom(pos, nrm, idx)
	validate.RepairFolds(m, 8)
	return m
}

// replicateBoundary lays each rim and mouth into the covering at all three period shifts, returning the
// constraint loops: rims as open per-segment chains (they do not close across the seam), whole mouths as
// closed loops.
func (b *coverBuilder) replicateBoundary(rims, mouths []cylLoop) [][]int {
	var loops [][]int
	for _, sh := range coverShifts {
		off := sh * b.period
		for _, r := range rims {
			loops = append(loops, b.addChain(r, off)...)
		}
		for _, m := range mouths {
			loops = append(loops, b.addRing(m, off))
		}
	}
	return loops
}

// coverBuilder accumulates the covering's vertices: exact 3D positions + surface normals, the metric-
// scaled (u,v) the CDT triangulates in, and the unwrapped u/v used to select the canonical period.
type coverBuilder struct {
	s           geom.BSplineSurface
	su, sv      float64
	ulo, period float64
	pos         []math.Point3
	nrm         []math.Vector3
	xy          [][2]float64
	uu, vv      []float64
}

func (b *coverBuilder) add(p math.Point3, u, v float64) int {
	i := len(b.pos)
	b.pos = append(b.pos, p)
	b.nrm = append(b.nrm, b.s.NormalAt(canonU(u, b.ulo, b.period), v))
	b.xy = append(b.xy, [2]float64{u * b.su, v * b.sv})
	b.uu = append(b.uu, u)
	b.vv = append(b.vv, v)
	return i
}

// addChain adds a shifted rim loop as an OPEN chain (it does not close across the seam) and returns its
// per-segment 2-vertex constraints, so the triangulation aligns to the rim without a spurious closing edge.
func (b *coverBuilder) addChain(l cylLoop, off float64) [][]int {
	idx := make([]int, len(l.p3))
	for i := range l.p3 {
		idx[i] = b.add(l.p3[i], l.u[i]+off, l.v[i])
	}
	segs := make([][]int, 0, len(idx)-1)
	for i := 0; i+1 < len(idx); i++ {
		segs = append(segs, []int{idx[i], idx[i+1]})
	}
	return segs
}

// addRing adds a shifted mouth loop as a closed-loop constraint (constrain wraps the last edge to the
// first), returning its vertex index sequence.
func (b *coverBuilder) addRing(l cylLoop, off float64) []int {
	idx := make([]int, len(l.p3))
	for i := range l.p3 {
		idx[i] = b.add(l.p3[i], l.u[i]+off, l.v[i])
	}
	return idx
}

// addInteriorNodes lays a curvature-adaptive staggered grid over the canonical band (between the rims,
// clear of the mouths) and replicates each kept node across the period shifts, so the interior refines to
// the chord tolerance on every copy and triangles can form across the seam.
func (b *coverBuilder) addInteriorNodes(q Quality, ulo, uhi, vmin, vmax float64, vBot, vTop rimFunc, mouths []cylLoop) {
	stepU, stepV := adaptiveStep(b.s, ulo, uhi, vmin, vmax, q)
	if stepU <= 0 || stepV <= 0 {
		return
	}
	margin := 0.3 * stdmath.Min(stepU, stepV)
	row := 0
	for v := vmin + stepV/2; v < vmax; v += stepV {
		offU := 0.0
		if row%2 == 1 {
			offU = stepU / 2
		}
		row++
		for u := ulo + stepU/2 + offU; u < uhi; u += stepU {
			if !materialPoint(u, v, b.period, vBot, vTop, mouths, margin) {
				continue
			}
			p := b.s.PointAt(u, v)
			for _, sh := range coverShifts {
				b.add(p, u+sh*b.period, v)
			}
		}
	}
}

// selectCanonical keeps each triangle whose centroid lies in the canonical period [ulo, ulo+P) and in the
// material region (inside the band, outside every mouth). Period replication means each periodic triangle
// has exactly one translate with centroid in the canonical period, so this de-duplicates without splitting.
func (b *coverBuilder) selectCanonical(tris [][3]int, ulo, period float64, vBot, vTop rimFunc, mouths []cylLoop) [][3]int {
	var out [][3]int
	for _, t := range tris {
		cu := (b.uu[t[0]] + b.uu[t[1]] + b.uu[t[2]]) / 3
		if cu < ulo || cu >= ulo+period {
			continue
		}
		cv := (b.vv[t[0]] + b.vv[t[1]] + b.vv[t[2]]) / 3
		if !materialPoint(canonU(cu, ulo, period), cv, period, vBot, vTop, mouths, 0) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// rimFunc gives the band boundary v at a canonical u (linearly interpolated along a rim).
type rimFunc func(u float64) float64

// rimBand returns the bottom and top rim v-functions and the band's v-extent. The two rims are ordered by
// mean v (bottom = smaller). Each rim is sampled as v(canonical u), single-valued for a cap-cut rim.
func rimBand(rims []cylLoop, ulo, period float64) (vBot, vTop rimFunc, vmin, vmax float64) {
	a, b := rimSamples(rims[0], ulo, period), rimSamples(rims[1], ulo, period)
	if meanV(rims[0]) > meanV(rims[1]) {
		a, b = b, a
	}
	vBot, vTop = interpRim(a, period), interpRim(b, period)
	vmin, vmax = stdmath.Inf(1), stdmath.Inf(-1)
	for _, l := range rims {
		for _, v := range l.v {
			vmin, vmax = stdmath.Min(vmin, v), stdmath.Max(vmax, v)
		}
	}
	return vBot, vTop, vmin, vmax
}

// rimSamples projects a rim's points to canonical u and sorts them, so interpRim can interpolate v(u).
func rimSamples(l cylLoop, ulo, period float64) [][2]float64 {
	pts := make([][2]float64, len(l.u))
	for i := range l.u {
		pts[i] = [2]float64{canonU(l.u[i], ulo, period), l.v[i]}
	}
	insertionSortByU(pts)
	return pts
}

// interpRim builds a PERIODIC linear interpolant v(u) over the sorted rim samples.
//
// rimSamples folds every rim point to canonical u and sorts it, so the segment from the LAST sample
// (u ≈ ulo + period − δ) back to the FIRST (u ≈ ulo + ε) crosses the seam and is not in the sorted list.
// Clamping flat there — what this did before — misplaces a cap-cut rim's band boundary near the seam by
// up to the rim's v-variation over one discretization step. That is a latent defect, not an observed
// one: on the single face in the tree that reaches this path (cand_radial's B-spline barrel, #1510) BOTH
// rims are constant-v — measured vspan 1.9e-17 and 0 — so the flat clamp was exact there and closing the
// wrap changes no shipped mesh. It is closed because a cap-cut rim would expose it silently.
func interpRim(pts [][2]float64, period float64) rimFunc {
	return func(u float64) float64 {
		if len(pts) == 0 {
			return 0
		}
		last := len(pts) - 1
		if u <= pts[0][0] { // the seam segment, approached from below: last sample − period → first
			return lerpV(u, pts[last][0]-period, pts[last][1], pts[0][0], pts[0][1])
		}
		for i := range last {
			if u <= pts[i+1][0] {
				return lerpV(u, pts[i][0], pts[i][1], pts[i+1][0], pts[i+1][1])
			}
		}
		return lerpV(u, pts[last][0], pts[last][1], pts[0][0]+period, pts[0][1]) // the same seam segment, from above
	}
}

// lerpV linearly interpolates v between two rim samples, returning v0 for a degenerate u-interval.
func lerpV(u, u0, v0, u1, v1 float64) float64 {
	if u1 == u0 {
		return v0
	}
	return v0 + (v1-v0)*(u-u0)/(u1-u0)
}

// materialPoint reports whether (u,v) at canonical u is inside the band and outside every mouth, with an
// optional inward margin (used to keep interior grid nodes clear of the boundary).
func materialPoint(u, v, period float64, vBot, vTop rimFunc, mouths []cylLoop, margin float64) bool {
	if v <= vBot(u)+margin || v >= vTop(u)-margin {
		return false
	}
	for _, m := range mouths {
		if pointInMouth(u, v, m, period, margin) {
			return false
		}
	}
	return true
}

// pointInMouth reports whether (u,v) lies in mouth m (or within margin of it), testing m and its ±period
// copies so a query in the canonical branch matches a mouth that straddles the seam.
func pointInMouth(u, v float64, m cylLoop, period, margin float64) bool {
	for _, sh := range coverShifts {
		if pointInPolyUV(u, v, m.u, m.v, sh*period) {
			return true
		}
		if margin > 0 && nearLoopUV(u, v, m.u, m.v, sh*period, margin) {
			return true
		}
	}
	return false
}

// pointInPolyUV is the even-odd ray test for (u,v) against a loop given by parallel u/v slices shifted by
// off in u.
func pointInPolyUV(u, v float64, lu, lv []float64, off float64) bool {
	in := false
	n := len(lu)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		ui, uj := lu[i]+off, lu[j]+off
		vi, vj := lv[i], lv[j]
		if (vi > v) != (vj > v) && u < (uj-ui)*(v-vi)/(vj-vi)+ui {
			in = !in
		}
	}
	return in
}

// nearLoopUV reports whether (u,v) is within margin of any edge of the shifted loop — the grid-node
// clearance test around a mouth boundary.
func nearLoopUV(u, v float64, lu, lv []float64, off, margin float64) bool {
	n := len(lu)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		if distToSeg2D(u, v, lu[j]+off, lv[j], lu[i]+off, lv[i]) < margin {
			return true
		}
	}
	return false
}

func distToSeg2D(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return stdmath.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / l2
	t = stdmath.Max(0, stdmath.Min(1, t))
	return stdmath.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

// weldTriangles compacts the covering's vertices to only those used by the kept triangles, welding
// coincident 3D positions (period-shifted seam copies) into one vertex so the face mesh is a properly
// shared-vertex, watertight patch.
func weldCoverTriangles(pos []math.Point3, nrm []math.Vector3, tris [][3]int) ([]math.Point3, []math.Vector3, [][3]int) {
	grid := weldGrid([][]math.Point3{pos})
	canon := map[[3]int64]int{}
	var outPos []math.Point3
	var outNrm []math.Vector3
	remap := func(i int) int {
		k := quantizePoint(pos[i], grid)
		if c, ok := canon[k]; ok {
			return c
		}
		c := len(outPos)
		canon[k] = c
		outPos = append(outPos, pos[i])
		outNrm = append(outNrm, nrm[i])
		return c
	}
	out := make([][3]int, 0, len(tris))
	for _, t := range tris {
		a, b, c := remap(t[0]), remap(t[1]), remap(t[2])
		if a != b && b != c && a != c {
			out = append(out, [3]int{a, b, c})
		}
	}
	return outPos, outNrm, out
}

// canonU wraps a parameter into the canonical period [ulo, ulo+P).
func canonU(u, ulo, period float64) float64 {
	return u - period*stdmath.Floor((u-ulo)/period)
}

func meanV(l cylLoop) float64 {
	s := 0.0
	for _, v := range l.v {
		s += v
	}
	return s / float64(len(l.v))
}

func insertionSortByU(pts [][2]float64) {
	for i := 1; i < len(pts); i++ {
		for j := i; j > 0 && pts[j-1][0] > pts[j][0]; j-- {
			pts[j-1], pts[j] = pts[j], pts[j-1]
		}
	}
}

// weldGrid derives a model-scaled weld grid (length) from all the given point groups.
func weldGrid(groups [][]math.Point3) float64 {
	var all []math.Point3
	for _, g := range groups {
		all = append(all, g...)
	}
	w := geom.ResolutionForPoints(all).Weld()
	if w <= 0 {
		w = 1e-9
	}
	return w
}

// quantizePoint snaps a 3D point to the weld grid, so coincident points (period-shifted seam copies)
// hash to one key. It reuses the per-coordinate quantize (csg_body.go).
func quantizePoint(p math.Point3, grid float64) [3]int64 {
	return [3]int64{mesh.Quantize(float64(p.X), grid), mesh.Quantize(float64(p.Y), grid), mesh.Quantize(float64(p.Z), grid)}
}
