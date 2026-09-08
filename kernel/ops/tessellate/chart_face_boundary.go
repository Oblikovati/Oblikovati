// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The POINTS half of the chart-driven curved-face mesher (ADR-0061).
//
// The region comes from the chart; the mesh vertices ON the boundary come from the SHARED edges and
// from nowhere else. A face's boundary must be discretised identically on both sides of every edge, and
// the chart's own sampling is not the tessellated edge's — a first attempt that took both from the
// chart got the region right and tore the mesh everywhere it touched (measured: the drilled-plate
// control went from 0 to 68 free edges).
//
// So each boundary loop arrives as its exact 3D discretisation and is LIFTED into the covering space:
// each point's parameters, made continuous along the loop, then the whole trace shifted by whole
// periods onto the chart's branch. A lifted point differs from the chart's own sample of the same
// boundary only by whole periods, which is what makes one shift per chain enough.

// chartChain is one boundary loop of a face in the covering space: the loop's EXACT 3D points (p3) and
// their continuous (u,v) trace (uv), both with the FIRST point repeated at the end — at its continued
// parameters, so a loop that wraps a period is an open chain across the covering space and its closing
// segment is a constraint like any other.
type chartChain struct {
	p3                     []math.Point3
	uv                     []math.Point2
	uMin, uMax, vMin, vMax float64 // the chain's own (u,v) box, so a clearance query rejects it cheaply
	chord                  float64 // its mean 3D chord — the scale of its own discretisation
}

// chartBoundaryChains lifts every boundary loop of a face onto the chart's branch, outer loop first.
func chartBoundaryChains(f *topo.Face, s geom.Surface, r chartRegion, q Quality) []chartChain {
	loops := append([][]math.Point3{FaceOuterBoundary(f, q)}, faceHoleBoundaries(f, q)...)
	out := make([]chartChain, 0, len(loops))
	for _, loop := range loops {
		if c, ok := liftLoopOntoChart(s, r, loop); ok {
			out = append(out, c)
		}
	}
	return out
}

// liftLoopOntoChart lifts one 3D boundary loop into the covering space on the chart's branch. ok=false
// for a loop of fewer than three points, which bounds nothing.
func liftLoopOntoChart(s geom.Surface, r chartRegion, loop []math.Point3) (chartChain, bool) {
	if len(loop) < 3 {
		return chartChain{}, false
	}
	us, vs := surfaceParamsOfLoop(s, loop)
	cu, cv := continuousTrace(us, r.uPer), continuousTrace(vs, r.vPer)
	du, dv := r.branchOffset(cu, cv)
	c := chartChain{p3: append(append([]math.Point3(nil), loop...), loop[0]), uv: make([]math.Point2, len(cu))}
	for i := range cu {
		c.uv[i] = math.P2(cu[i]+du, cv[i]+dv)
	}
	c.uMin, c.uMax, c.vMin, c.vMax = uvBBox(c.uv)
	c.chord = meanChainChord(c.p3)
	return c, true
}

// continuousTrace unwraps a closed loop's parameter into a chain of len+1 values: the samples made
// continuous, then the FIRST sample continued past the last. That last step is where a loop which
// wraps a whole period shows its winding instead of jumping back to where it started.
func continuousTrace(a []float64, periodic bool) []float64 {
	if !periodic {
		return append(append(make([]float64, 0, len(a)+1), a...), a[0])
	}
	cu := cumulativeUnwrap(a)
	return append(cu, cu[len(cu)-1]+probe.WrapPi(a[0]-a[len(a)-1]))
}

// surfaceParamsOfLoop inverts each of a loop's 3D points to the surface's own (u,v), as parallel
// slices — the raw, per-point parameters every developing path starts from.
func surfaceParamsOfLoop(s geom.Surface, loop []math.Point3) (us, vs []float64) {
	us, vs = make([]float64, len(loop)), make([]float64, len(loop))
	for i, p := range loop {
		us[i], vs[i] = s.ParamAt(p)
	}
	return us, vs
}

// meanChainChord is a lifted chain's mean 3D segment length — the scale at which the shared edge was
// discretised, and so the width the chart-versus-chord band can reach (see chartBoundaryClearance).
func meanChainChord(p3 []math.Point3) float64 {
	if len(p3) < 2 {
		return 0
	}
	sum := 0.0
	for i := 1; i < len(p3); i++ {
		sum += float64(p3[i-1].DistanceTo(p3[i]))
	}
	return sum / float64(len(p3)-1)
}

// chainSegmentCount is how many boundary segments a set of chains carries an ODD number of times — the
// number of unpaired mesh edges a correctly meshed patch bounded by them has, and so the acceptance
// bound.
//
// Odd, not all, because a face's boundary may walk an artificial SLIT twice: the piston head's merged
// cocylindrical wall arrives as ONE wrapping loop of 56 points — its bottom circle (32), its notched rim
// (22) and the seam that bridges them, up and back down. Those two seam segments are the same two 3D
// points in opposite order; a correct patch welds them into an interior edge and bounds 54, not 56. The
// gate read 56, declined a mesh that was right, and the wall fell to the flat-patch CDT (57.913 mm²
// where 173.811 is the region's own area, and the body reported a 32-edge tear).
func chainSegmentCount(chains []chartChain) int {
	n := 0
	for _, c := range chains {
		n += len(chainSegmentKeys([]chartChain{c}, geom.ResolutionForPoints(c.p3).Weld()))
	}
	return n
}

// chainSegmentKeys is the SET of boundary segments a set of chains carries an odd number of times, keyed
// the way the mesh's own welded edges are — the rim a correctly meshed patch must have, exactly.
func chainSegmentKeys(chains []chartChain, grid float64) map[[2][3]int64]bool {
	used := map[[2][3]int64]int{}
	for _, c := range chains {
		for i := 0; i+1 < len(c.p3); i++ {
			used[orderedSegmentKey(c.p3[i], c.p3[i+1], grid)]++
		}
	}
	out := make(map[[2][3]int64]bool, len(used))
	for k, n := range used {
		if n%2 == 1 {
			out[k] = true
		}
	}
	return out
}

// orderedSegmentKey is a boundary segment's identity: its two welded endpoints, smaller first, so a
// segment and its reverse are the same key.
func orderedSegmentKey(a, b math.Point3, grid float64) [2][3]int64 {
	ka, kb := WeldKey(a, grid), WeldKey(b, grid)
	if ka[0] > kb[0] || (ka[0] == kb[0] && (ka[1] > kb[1] || (ka[1] == kb[1] && ka[2] > kb[2]))) {
		ka, kb = kb, ka
	}
	return [2][3]int64{ka, kb}
}
