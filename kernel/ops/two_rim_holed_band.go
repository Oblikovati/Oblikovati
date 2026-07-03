// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Two-rim holed developable-band tessellation (Oblikovati/Oblikovati#1724, ADR-0046, cap-crossing slice 2).
// The rim-crossing cut's target WALL is a full-wrap cylinder band bounded by TWO full-period rims — a
// NOTCHED top rim (where the tool exits ACROSS the cap edge) plus the intact bottom rim — carrying the
// tool's ENTRY lens as an interior hole. Unlike the drill-through wall (holedConicWallMesh), whose split
// hands back ONE seam-bridged wrapping outer loop, this wall comes back as SEPARATE loops (the notch
// breaks the seam bridge), so BOTH rims look like full-wrap "holes" holesIntoBranch rejects, and the face
// falls through to the flat best-fit-plane CDT — which mangles a full cylinder wall (an inside-out band,
// ~140 free edges, near-zero volume). The fix stays in the tessellation layer: synthesize the missing seam
// slit that bridges the two rims into one wrapping outer loop, then mesh through the SAME proven unroll +
// metric CDT (unrolledWallCDT) the drilled wall uses — the notch is just a non-straight top edge in
// unrolled (u,v).

// seamSteps is the number of segments the synthetic rim-bridging seam is divided into. The seam is a short,
// near-axial slit on the wall, so a modest count keeps its boundary nodes dense enough for well-shaped
// triangles; the interior is refined by adaptiveInteriorNodes regardless.
const seamSteps = 16

// twoRimHoledBandMesh meshes a developable side bounded by two full-wrap rims (either may be notched)
// carrying interior lens holes, by bridging the rims at a synthetic seam and unrolling. ok=false unless
// the surface is a developable side and EXACTLY ONE of the hole loops is itself a full-wrap rim (zero →
// holedConicWallMesh already handles it; two or more → not a two-rim band this mesher understands).
func twoRimHoledBandMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	if !isDevelopableSide(s) || isPeriodic(s.UDomain()) == isPeriodic(s.VDomain()) {
		return nil, false
	}
	rims, lenses := splitWrappingHoles(s, holes3D)
	if len(rims) != 1 {
		return nil, false
	}
	wrap, ok := bridgeRimsAtSeam(s, outer3D, rims[0], lenses)
	if !ok {
		return nil, false
	}
	outerUV, umin, umax, ok := wrappedWallUV(s, wrap)
	if !ok {
		return nil, false
	}
	lensUV, ok := holesIntoBranch(s, lenses, umin, umax)
	if !ok {
		return nil, false
	}
	return unrolledWallCDT(s, q, wrap, lenses, outerUV, lensUV), true
}

// splitWrappingHoles partitions hole loops into those that themselves wrap the full period (a second rim,
// mis-demoted to a hole by faceHoleBoundaries) and the genuine non-wrapping lens holes, by their unrolled
// angular span (a full rim spans ~2π; a lens spans a small arc).
func splitWrappingHoles(s geom.Surface, holes3D [][]math.Point3) (rims, lenses [][]math.Point3) {
	for _, h := range holes3D {
		if holeWrapsPeriod(s, h) {
			rims = append(rims, h)
		} else {
			lenses = append(lenses, h)
		}
	}
	return rims, lenses
}

// holeWrapsPeriod reports whether a hole loop spans essentially the full angular period (so it is a rim,
// not a lens) — the same >2π−0.5 test holesIntoBranch uses to reject a seam-wrapping hole.
func holeWrapsPeriod(s geom.Surface, hole []math.Point3) bool {
	us := make([]float64, len(hole))
	for i, p := range hole {
		us[i], _ = s.ParamAt(p)
	}
	lo, hi := minMax(cumulativeUnwrap(us))
	return hi-lo > 2*stdmath.Pi-0.5
}

// bridgeRimsAtSeam joins two full-wrap rims into one seam-wrapping outer loop by cutting each rim at a
// shared-vertex seam and walking bottom rim → up the seam → top rim (reversed) → down the seam (a CCW
// unrolled rectangle). The seam is placed in the largest lens-free angular gap (so it never crosses a hole)
// and runs between an EXISTING vertex on each rim, so it introduces no new rim point that would T-crack the
// neighbouring cap; its interior points are interpolated ALONG the surface (never a chord through the solid)
// and are reused identically on both rectangle edges, so the two seam copies weld. ok=false if either rim is
// too short to order.
func bridgeRimsAtSeam(s geom.Surface, top3D, bot3D []math.Point3, lenses [][]math.Point3) ([]math.Point3, bool) {
	top := orderedRing(s, top3D)
	bot := orderedRing(s, bot3D)
	if len(top) < 3 || len(bot) < 3 {
		return nil, false
	}
	seamTh := clearSeamAngle(s, top, lenses) // seam in the widest gap clear of BOTH lenses and the notch
	ti := nearestAngleIndex(s, top, seamTh)  // seam anchors on EXISTING rim vertices (no new rim point → no crack)
	bi := nearestAngleIndex(s, bot, seamTh)
	topSeq := rotateRing(top, ti)
	botSeq := rotateRing(bot, bi)
	seam := seamOnSurface(s, top[ti], bot[bi], seamSteps) // interior seam points, on the surface, bottom→top

	wrap := make([]math.Point3, 0, len(topSeq)+len(botSeq)+2*len(seam)+3)
	wrap = append(wrap, botSeq...)          // bottom rim, ascending θ from the seam vertex
	wrap = append(wrap, botSeq[0])          // bottom-right corner: the +2π copy of the seam bottom vertex
	wrap = append(wrap, seam...)            // right seam edge, bottom → top (on surface)
	wrap = append(wrap, topSeq[0])          // top-right corner: the +2π copy of the seam top vertex
	wrap = appendReversed(wrap, topSeq[1:]) // top rim reversed, descending θ back toward the seam
	wrap = append(wrap, topSeq[0])          // top-left corner: the seam top vertex
	wrap = appendReversed(wrap, seam)       // left seam edge, top → bottom (same points, reversed → welds)
	return wrap, true
}

// clearSeamAngle returns an angle in the WIDEST angular gap clear of both the lens holes AND the top rim's
// notch (the low-v stretch of the outer loop where the tool exited), so the rim-bridging seam crosses no hole
// (holesIntoBranch would reject a straddled lens) and anchors on intact full-height rim well away from the
// trivalent corners (anchoring on a notch vertex tangles the corner mesh). Both are treated as keep-away
// angle clusters. With neither present any angle is clear, so it returns 0.
func clearSeamAngle(s geom.Surface, top []math.Point3, lenses [][]math.Point3) float64 {
	angs := notchAngles(s, top)
	for _, h := range lenses {
		for _, p := range h {
			angs = append(angs, normTwoPi(angleOf(s, p)))
		}
	}
	if len(angs) == 0 {
		return 0
	}
	sort.Float64s(angs)
	bestGap, bestMid := -1.0, 0.0
	for i := range angs {
		gap := angs[(i+1)%len(angs)] - angs[i]
		if i == len(angs)-1 {
			gap += 2 * stdmath.Pi
		}
		if gap > bestGap {
			bestGap, bestMid = gap, normTwoPi(angs[i]+gap/2)
		}
	}
	return bestMid
}

// notchAngles returns the angles of the top rim's notch vertices — those dipping below the rim's max v by
// more than a small fraction of the rim's own v-extent. An un-notched (constant-v) rim yields none.
func notchAngles(s geom.Surface, top []math.Point3) []float64 {
	vs := make([]float64, len(top))
	vmax := stdmath.Inf(-1)
	for i, p := range top {
		vs[i] = vParam(s, p)
		vmax = stdmath.Max(vmax, vs[i])
	}
	vmin, _ := minMax(vs)
	cut := vmax - 0.02*(vmax-vmin) // any vertex measurably below the plateau is in the notch
	var out []float64
	for i, v := range vs {
		if v < cut {
			out = append(out, normTwoPi(angleOf(s, top[i])))
		}
	}
	return out
}

// normTwoPi folds an angle into [0, 2π).
func normTwoPi(a float64) float64 {
	a = stdmath.Mod(a, 2*stdmath.Pi)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
}

// nearestAngleIndex returns the index of the ring vertex whose angle is closest to theta (shortest way
// around), so the seam bridging the two rims stays as near-axial as the rims' vertices allow.
func nearestAngleIndex(s geom.Surface, ring []math.Point3, theta float64) int {
	best, bd := 0, stdmath.Inf(1)
	for i, p := range ring {
		if d := stdmath.Abs(foldAngle(angleOf(s, p) - theta)); d < bd {
			best, bd = i, d
		}
	}
	return best
}

// seamOnSurface returns the interior points of the seam slit from the bottom rim vertex up to the top rim
// vertex, interpolated in (θ,v) and evaluated ON the surface so the slit never chords through the solid.
func seamOnSurface(s geom.Surface, top, bot math.Point3, steps int) []math.Point3 {
	ut, vt := s.ParamAt(top)
	ub, vb := s.ParamAt(bot)
	dth := foldAngle(ut - ub) // shortest angular way from bottom to top
	out := make([]math.Point3, 0, steps-1)
	for k := 1; k < steps; k++ {
		f := float64(k) / float64(steps)
		out = append(out, s.PointAt(ub+dth*f, vb+(vt-vb)*f))
	}
	return out
}

// rotateRing returns ring reordered to start at index i, preserving order (wrapping) — the seam cut point.
func rotateRing(ring []math.Point3, i int) []math.Point3 {
	out := make([]math.Point3, 0, len(ring))
	out = append(out, ring[i:]...)
	out = append(out, ring[:i]...)
	return out
}

// appendReversed appends src in reverse order to dst.
func appendReversed(dst, src []math.Point3) []math.Point3 {
	for i := len(src) - 1; i >= 0; i-- {
		dst = append(dst, src[i])
	}
	return dst
}

// foldAngle folds an angle difference into (−π, π].
func foldAngle(d float64) float64 {
	for d > stdmath.Pi {
		d -= 2 * stdmath.Pi
	}
	for d <= -stdmath.Pi {
		d += 2 * stdmath.Pi
	}
	return d
}
