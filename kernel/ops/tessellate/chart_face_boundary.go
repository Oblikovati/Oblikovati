// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

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
	uv                     [][2]float64
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
	c := chartChain{p3: append(append([]math.Point3(nil), loop...), loop[0]), uv: make([][2]float64, len(cu))}
	for i := range cu {
		c.uv[i] = [2]float64{cu[i] + du, cv[i] + dv}
	}
	c.uMin, c.uMax, c.vMin, c.vMax = chainBox(c.uv)
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

// chainBox is a lifted chain's (u,v) bounding box.
func chainBox(uv [][2]float64) (uMin, uMax, vMin, vMax float64) {
	uMin, vMin = stdmath.Inf(1), stdmath.Inf(1)
	uMax, vMax = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, p := range uv {
		uMin, uMax = stdmath.Min(uMin, p[0]), stdmath.Max(uMax, p[0])
		vMin, vMax = stdmath.Min(vMin, p[1]), stdmath.Max(vMax, p[1])
	}
	return uMin, uMax, vMin, vMax
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

// chainSegmentCount is how many boundary segments a set of chains carries — the number of unpaired
// mesh edges a correctly meshed patch bounded by them has, and so the acceptance bound.
func chainSegmentCount(chains []chartChain) int {
	n := 0
	for _, c := range chains {
		n += len(c.uv) - 1
	}
	return n
}
