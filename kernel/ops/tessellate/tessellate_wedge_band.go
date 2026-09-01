// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Oblique-ended wedge-band mesher (cluster-A planar corners, tolblend A1/D4). A pyramid slant-edge
// fillet is a cylinder band between two straight rails whose END cross-sections are OBLIQUE to the
// axis (the base plane / the corner arc are not ⊥ the slant edge), so the trim is not an iso
// rectangle and used to fall to the generic metric CDT. That CDT fills the shallow sliver between
// the end chain's chords with boundary-only triangles; their 3D lift is FLAT in the end plane,
// double-covering the neighbour's ear triangles and colliding on the same interior diagonal — the
// deg-4 "free" edges A1/D4 leaked at default quality (3 and 6, measured; 0 at property by
// combinatorial luck).
//
// A cylinder is RULED in v, so the band needs no interior nodes at all: one zipped strip between
// the two end chains is exact along every ruling, its only deviation the u-chord sagitta already
// set by the ends' shared discretization. Boundary conformity by construction: each end chain is
// tiled at exactly its shared-edge stations, and each rail is tiled as the single segment its
// planar host tiles (the gate requires the rail to discretize to its 2 endpoints).

// wedgeBandLoftMesh meshes an open oblique-ended cylinder wedge band as one zipped strip, or
// ok=false for any face that is not exactly that shape (holes, a seam wrap, a curved rail, a
// densified rail, iso ends — all fall through to the shipped paths, byte-identical).
func wedgeBandLoftMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	cyl, isCyl := s.(geom.Cylinder)
	if !isCyl || len(f.Loops()) != 1 {
		return nil, false
	}
	ends, ok := wedgeBandEndChains(f, cyl, q)
	if !ok {
		return nil, false
	}
	m := &Mesh{}
	a := addRow(m, s, ends[0].pts, ends[0].ang)
	b := addRow(m, s, ends[1].pts, ends[1].ang)
	zipOpenRows(m, a, b)
	return m, true
}

// wedgeEndChain is one end cross-section chain: its exact shared-edge points and their unwrapped
// angular stations, ascending.
type wedgeEndChain struct {
	pts []math.Point3
	ang []float64
}

// wedgeBandEndChains reads and certifies the band's boundary by WALKING ITS LOOP: exactly 2 straight
// axis-parallel RAIL edges that discretize to their endpoints, separating exactly 2 maximal end-chain
// runs (a run may be several edges — a miter seam stored as chord segments), each u-monotone, at
// least one oblique (non-iso in v) — all spanning less than a half turn. ok=false otherwise.
func wedgeBandEndChains(f *topo.Face, cyl geom.Cylinder, q Quality) ([2]wedgeEndChain, bool) {
	if len(seamEdgesOf(f)) != 0 {
		return [2]wedgeEndChain{}, false
	}
	runs, ok := wedgeChainRuns(f, cyl, q)
	if !ok || len(runs) != 2 {
		return [2]wedgeEndChain{}, false
	}
	var chains []wedgeEndChain
	for _, run := range runs {
		ch, chOK := wedgeChainAngles(cyl, run)
		if !chOK {
			return [2]wedgeEndChain{}, false
		}
		chains = append(chains, ch)
	}
	wedgeAlignChainBranches(chains)
	if !wedgeSpanBelowHalfTurn(chains) || !wedgeEndsOblique(cyl, chains) {
		return [2]wedgeEndChain{}, false
	}
	return [2]wedgeEndChain{chains[0], chains[1]}, true
}

// wedgeAlignChainBranches re-anchors chains[1]'s angles onto chains[0]'s periodic branch (shifts by
// the nearest multiple of 2π), in place. Each chain unwraps monotone WITHIN itself
// (wedgeChainAngles), but cyl.ParamAt's principal value gives no cross-chain phase continuity: on
// a band whose two ends sit at the SAME physical angular window (A1/D4's corner band, both ends an
// arc of a K-gon rim), one end's raw stations can land a whole 2π above the other's (measured: near
// end [2π, 2π+1.226], far end [0, 1.226] — same 1.226-radian span, different branch). Unaligned, the
// half-turn span check sees a bogus ~7.5-radian range and declines, and — worse — zipOpenRows'
// a.ang/b.ang comparisons would silently misorder without this, since they assume one shared axis.
func wedgeAlignChainBranches(chains []wedgeEndChain) {
	shift := 2 * stdmath.Pi * stdmath.Round((chains[0].ang[0]-chains[1].ang[0])/(2*stdmath.Pi))
	if shift == 0 {
		return
	}
	for i := range chains[1].ang {
		chains[1].ang[i] += shift
	}
}

// wedgeChainRuns walks the outer loop splitting it at rail edges (straight, axis-parallel, tiled at
// their 2 endpoints), pooling consecutive non-rail edges' discretized points into end-chain runs.
// ok=false unless there are exactly 2 rails and every rail conforms (2 stations).
func wedgeChainRuns(f *topo.Face, cyl geom.Cylinder, q Quality) ([][]math.Point3, bool) {
	runs, cur, rails, ok := wedgeChainWalkLoop(cyl, f.Loops()[0].EdgeUses(), q)
	if !ok {
		return nil, false
	}
	return wedgeChainCloseTrailingRun(runs, cur, rails), rails == 2
}

// wedgeChainWalkLoop is wedgeChainRuns' per-edge pass: classify each use as a rail (splits the
// current run) or a chain point run (accumulates), in loop order.
func wedgeChainWalkLoop(cyl geom.Cylinder, uses []*topo.EdgeUse, q Quality) (runs [][]math.Point3, cur []math.Point3, rails int, ok bool) {
	for _, u := range uses {
		pts, okPts := wedgeChainUsePoints(u, q)
		if !okPts {
			return nil, nil, 0, false
		}
		if !wedgeRailEdge(cyl, pts) {
			cur = appendChainPoints(cur, pts)
			continue
		}
		if len(pts) != 2 {
			return nil, nil, 0, false // a densified rail cannot tile as one segment against its host
		}
		rails++
		if len(cur) > 0 {
			runs, cur = append(runs, cur), nil
		}
	}
	return runs, cur, rails, true
}

// wedgeChainUsePoints discretizes one loop use in LOOP-WALK order (reversed if the use is reversed)
// so consecutive runs' points concatenate directly.
func wedgeChainUsePoints(u *topo.EdgeUse, q Quality) ([]math.Point3, bool) {
	pts := DiscretizeEdge(u.Edge(), q)
	if len(pts) < 2 {
		return nil, false
	}
	if u.Reversed() {
		pts = append([]math.Point3(nil), pts...)
		reversePointsInPlace(pts)
	}
	return pts, true
}

// wedgeChainCloseTrailingRun folds a trailing accumulator into runs: merged into the first run when
// the loop started mid-chain (both rails already closed a run pair), else appended as its own run.
func wedgeChainCloseTrailingRun(runs [][]math.Point3, cur []math.Point3, rails int) [][]math.Point3 {
	if len(cur) == 0 {
		return runs
	}
	if len(runs) == 2 && rails == 2 {
		runs[0] = appendChainPoints(cur, runs[0])
		return runs
	}
	return append(runs, cur)
}

// appendChainPoints concatenates a chain segment's points onto run, dropping a duplicated joint.
func appendChainPoints(run, pts []math.Point3) []math.Point3 {
	for _, p := range pts {
		if n := len(run); n > 0 && run[n-1].DistanceTo(p) < 1e-9 {
			continue
		}
		run = append(run, p)
	}
	return run
}

// wedgeRailEdge reports whether a discretized edge is a straight segment PARALLEL to the cylinder
// axis (a band rail; a miter-seam chord is straight but oblique and belongs to an end chain).
func wedgeRailEdge(cyl geom.Cylinder, pts []math.Point3) bool {
	if !wedgeRailStraight(pts) {
		return false
	}
	d := pts[0].VectorTo(pts[len(pts)-1])
	l := d.Length()
	if l == 0 {
		return false
	}
	return stdmath.Abs(d.Dot(cyl.AxisDir.AsVector()))/l > 1-1e-9
}

// wedgeRailStraight reports whether a discretized edge is a straight segment (every interior point
// collinear with the ends within a length-relative tolerance).
func wedgeRailStraight(pts []math.Point3) bool {
	d := pts[0].VectorTo(pts[len(pts)-1])
	l := d.Length()
	if l == 0 {
		return false
	}
	u := d.Scale(1 / l)
	for _, p := range pts[1 : len(pts)-1] {
		w := pts[0].VectorTo(p)
		if w.Sub(u.Scale(w.Dot(u))).Length() > 1e-9*l {
			return false
		}
	}
	return true
}

// wedgeChainAngles maps one end chain to unwrapped, strictly monotone angles about the cylinder
// axis, normalized ascending. ok=false on a fold in u (the zip requires monotone stations).
func wedgeChainAngles(cyl geom.Cylinder, pts []math.Point3) (wedgeEndChain, bool) {
	ang := make([]float64, len(pts))
	for i, p := range pts {
		u, _ := cyl.ParamAt(p)
		ang[i] = u
	}
	for i := 1; i < len(ang); i++ {
		for ang[i]-ang[i-1] > stdmath.Pi {
			ang[i] -= 2 * stdmath.Pi
		}
		for ang[i]-ang[i-1] < -stdmath.Pi {
			ang[i] += 2 * stdmath.Pi
		}
	}
	if ang[len(ang)-1] < ang[0] {
		reversePointsInPlace(pts)
		reverseFloats(ang)
	}
	for i := 1; i < len(ang); i++ {
		if ang[i] <= ang[i-1] {
			return wedgeEndChain{}, false
		}
	}
	return wedgeEndChain{pts: pts, ang: ang}, true
}

// wedgeSpanBelowHalfTurn requires both chains inside one common half-turn window (no seam wrap,
// the open-wedge shape this mesher is derived for).
func wedgeSpanBelowHalfTurn(chains []wedgeEndChain) bool {
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, c := range chains {
		lo = stdmath.Min(lo, c.ang[0])
		hi = stdmath.Max(hi, c.ang[len(c.ang)-1])
	}
	return hi-lo < stdmath.Pi-1e-9
}

// wedgeEndsOblique requires at least one end chain NOT perpendicular to the axis: an iso-ended band
// is the structured-grid path's (box fillets) and must stay byte-identical there.
func wedgeEndsOblique(cyl geom.Cylinder, chains []wedgeEndChain) bool {
	axis := cyl.AxisDir.AsVector()
	for _, c := range chains {
		vmin, vmax := stdmath.Inf(1), stdmath.Inf(-1)
		for _, p := range c.pts {
			v := cyl.Origin.VectorTo(p).Dot(axis)
			vmin = stdmath.Min(vmin, v)
			vmax = stdmath.Max(vmax, v)
		}
		if vmax-vmin > 1e-6*(1+stdmath.Abs(vmax)+stdmath.Abs(vmin)) {
			return true
		}
	}
	return false
}

// zipOpenRows triangulates the open strip between two angle-monotone rows: a monotone merge that
// always advances the side whose NEXT station is smaller, so every triangle connects the two rows
// (no intra-row chord is ever emitted — the invariant that kills the deg-4 diagonal coincidence).
func zipOpenRows(m *Mesh, a, b bandRow) {
	i, j := 0, 0
	for i+1 < len(a.idx) || j+1 < len(b.idx) {
		advanceA := j+1 >= len(b.idx) || (i+1 < len(a.idx) && a.ang[i+1] <= b.ang[j+1])
		if advanceA {
			emitClosedTri(m, a.idx[i], a.idx[i+1], b.idx[j])
			i++
			continue
		}
		emitClosedTri(m, a.idx[i], b.idx[j+1], b.idx[j])
		j++
	}
}

// reversePointsInPlace reverses a point slice in place (chain normalization).
func reversePointsInPlace(pts []math.Point3) {
	for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
		pts[i], pts[j] = pts[j], pts[i]
	}
}

// reverseFloats reverses a float slice in place.
func reverseFloats(xs []float64) {
	for i, j := 0, len(xs)-1; i < j; i, j = i+1, j-1 {
		xs[i], xs[j] = xs[j], xs[i]
	}
}
