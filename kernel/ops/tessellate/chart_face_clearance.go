// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
)

// The chart mesher's boundary clearance (ADR-0061): how far an interior grid node must stand from the
// face's own rim, and why. Moved out of chart_face_mesh.go unchanged when that file reached the size
// limit; the measurements below are the ones the constants were chosen on.

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
// band is not always a sagitta: at the oval's APEX, where the rim's chord is coarsest in u (the genus-1
// complement carries 0.17 rad of u in one chord there against the covering's own 0.0245 stations), an
// interior node lands INSIDE the chord rather than beside it. What bounds that is the chord itself.
//
// (This used to say the complement's boundary "touches itself" on a lemniscate. It does not: the torus
// R=5 r=1.5 cut by the plane x = R keeps ONE loop of two spiric arcs, (5,0,−1.5) → (5,0,+1.5), a single
// smooth oval, and the two (u,v) points once cited as its touch, (3π/2, π/2) and (3π/2, 3π/2), are the
// oval's top and bottom, 2r = 3 mm apart. The lemniscate is the figure-eight fixture, d = R − r, a
// different body — final fix wave, finding 4.)
//
// So it is measured. Swept on the three bodies whose charted faces the clearance governs — the genus-1
// complement, RS− and RD− — reading each body's free edges and its torus FACE's own area at BOTH
// facetings, so a plateau is flat in the numbers and not merely in a pass/fail count:
//
//	k       complement D      complement P        RS− D       RS− P       RD− D       RD− P
//	0.50    0 / 263.72994     272 / 296.06212     0 / 236.36  0 / 237.87  0 / 290.29  0 / 291.88
//	0.55    0 / 263.72994     272 / 296.06212     — as 0.50 —
//	0.60    0 / 263.68219     0 / 264.87124       0 / 236.36  0 / 237.87  0 / 290.29  0 / 291.88
//	0.70    0 / 263.60871     0 / 264.87119       0 / 236.10  0 / 237.87  0 / 290.29  0 / 291.88
//	0.875   0 / 263.55487     0 / 264.87111       0 / 236.10  0 / 237.87  0 / 290.29  0 / 291.88
//	1.00    0 / 263.42317     0 / 264.87104       0 / 236.10  0 / 237.87  0 / 290.29  0 / 291.88
//	1.10    0 / 263.33288     0 / 264.87097       0 / 235.44  0 / 237.87  0 / 290.28  0 / 291.88
//	1.50    0 / 263.15360     0 / 264.87045       — watertight, area falling —
//	3.00    0 / 260.51898     0 / 264.86644       — watertight, area falling —
//
// Only ONE body and ONE faceting ever fails: the complement at PropertyQuality, for k ≤ 0.55, where the
// face is declined and falls to the surface's whole domain (296.062 against the 264.871 it builds). The
// upper end is not a failure boundary at all — it is a monotone COST, the clearance removing interior
// nodes next to a coarse boundary, and every area above falls with k as a rule that only ever removes
// nodes must.
//
// 0.875 is therefore chosen, not centred: as small as the cost argument wants, with a real margin over
// the edge. It is 1.6× the largest k that fails and 1.46× the smallest that passes, and it costs
// 0.13 mm² of 263.7 — 0.05% — against sitting at 0.6. The complement's face area is pinned two-sided at
// the value this k gives (chart_face_mesh_test.go), so the constant cannot move without saying so.
const chartBoundaryClearance = 0.875 // tol:mesh-density (chords; swept 0.50…3.00 above, fails at k ≤ 0.55)

// chartNodeClearance is the fraction of a grid gap an interior node must keep from the boundary. A node
// ON a constraint owns no triangle and derails the segment recovery; one just inside it makes a sliver
// against the exact edge points, which this mesher may not move.
//
// It is a FLOOR under the chord clearance — chainIsNear takes the larger of the two — and 0.3 is the
// value at which it is only that. Swept over the whole chart corpus (RS−, RD−, the three RODB rows, the
// complement, both figure-eight pieces, the merged band) at both facetings (final fix wave, finding 5):
//
//	k       bodies not watertight   faces whose area moved from k = 0    chains where the floor wins the max
//	0.0     none                    —                                    0
//	0.1     none                    none                                 0
//	0.2     none                    none                                 0
//	0.3     none                    none                                 3  (RODB∪, RODB−, RODB∩ rod walls at PropertyQuality)
//	0.4     none                    RODB∪/RODB− rod wall D 24.85688 → 24.85677   5
//	0.5     none                    RODB∪/RODB− rod wall D 24.85688 → 24.85680   8
//	0.75    none                    RD− torus D, both rod walls D, both lens patches D   12
//
// Every face is byte-identical from k = 0 to k = 0.3: where the floor wins the max (three lens-window
// chains whose chords are shorter than 0.3 of a grid gap, at PropertyQuality) it culls no node the chord
// clearance had not already culled. The first node goes at 0.4. So the floor never decides a corpus mesh
// today; it is kept because the failure it guards is real (a node on a constraint) and its cost here is
// nothing. It is not centred: 0.3 is the largest swept value at which nothing moves.
const chartNodeClearance = 0.3 // tol:mesh-density (fraction of a grid gap; swept 0…0.75, first node culled at 0.4)

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
