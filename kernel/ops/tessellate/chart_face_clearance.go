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
// RE-SWEPT for #3519, three times: over a wider range, then finer at the edges, then AGAIN after
// #3551, which moved the edges by fixing the mesher rather than the constant. The current table is the
// third; the two claims the ORIGINAL receipt made are both withdrawn and are recorded here so a later
// sweep is not checked against them. It said the failure edge rests on ONE face — the complement's oval
// apex at PropertyQuality, for k ≤ 0.55 — and that the upper end is no failure boundary at all. Neither
// holds: the complement at PropertyQuality is watertight at every k swept here, and the corpus does tear
// above.
//
// The corpus is the 21 classification-corpus bodies, the merged cocylindrical wall and six
// torus-cut-by-its-tangent-plane pieces, at BOTH facetings. The columns are the whole body's free edges
// and the complement's own two numbers, which are what this constant is paid for and paid in: the
// DefaultQuality mesh volume as a fraction short of the analytic 203.904871, and the torus face's own
// meshed area.
//
//	k        corpus tears                          complement D deficit   complement D face
//	0.1      none                                              1.0936 %        263.76393
//	0.2      none                                              1.0953 %        263.75923
//	0.3      none                                              1.1097 %        263.73402
//	0.5      none                                              1.1148 %        263.72994
//	0.70     none                                              1.2708 %        263.60871
//	0.875    none                                              1.2982 %        263.55487
//	0.90     none                                              1.3497 %        263.49469
//	0.91     none                                              1.3497 %        263.49469
//	0.92     none                                              1.4136 %        263.45103
//	1.2      none                                              1.5979 %        263.33288
//	1.24     none                                              1.7156 %        263.28202
//	1.25     R=5 r=2 cut D, 2 free edges                       1.7156 %        263.28202
//	1.3      R=5 r=2 cut D, 2                                  1.8462 %        263.18783
//	1.5      R=5 r=2 cut D, 2                                  1.8616 %        263.15360
//	2.0      R=5 r=2 cut D, 2; R=5 r=2.5 cut D, 2              2.7038 %        262.61631
//	3.0      R=5 r=2.5 cut D, 2; R=10 r=3 cut P, 2             5.8686 %        260.51898
//
// (0.15, 0.25, 0.35, 0.4, 0.45, 0.55, 0.6, 0.705, 0.75, 0.8, 0.85, 0.88, 0.89, 0.935, 0.95, 1.0, 1.1
// and 1.4 were swept too and fall inside the steps above.)
//
// THE LOWER FAILURE EDGE IS GONE, and that is the most important line here. Two rounds ago the corpus
// tore below k = 0.70 — the R=5 r=2.5 figure-eight intersect piece by 48 free edges at every k ≤ 0.70,
// and the complement itself by 28 at every k ≤ 0.2 — and the constant was placed to clear them. Neither
// was the chart-versus-chord band this clearance is for. Both were #3551: the covering laid two vertices
// at one location, and a constrained triangulation cannot recover a constraint incident to either, which
// a low clearance makes more likely because it lets an interior node reach the rim. The covering merges
// coincident locations now, and every swept k from 0.1 upward meshes the whole corpus watertight.
//
// So what is left is ONE edge and one monotone cost. The edge is above: the R=5 r=2 cut piece tears from
// k = 1.25 (measured watertight at 1.24), and two more rows join it further up. The cost is the table's
// third column — the clearance only ever REMOVES interior nodes, so the complement's coarse mesh loses
// volume as k rises, monotonically, from 1.09 % at 0.1 to 5.87 % at 3.0.
//
// 0.90 is therefore chosen BY THE COST, not by centring, and that is the whole of its justification: it
// is the largest swept value whose cost is under 1.39 %, the deficit the deleted window mesher used to
// achieve on that body and which chart_face_mesh_test.go's own row exists to have beaten. 0.92 is over
// it at 1.4136 %, and the round that sat at 0.935 was over it at 1.4137 %. Below 0.90 nothing in the
// corpus argues for or against any value, so the claim the row makes is what decides, and a later sweep
// that wants to move this constant has to say what it does to that claim.
//
// The margin above is 1.389× the smallest k that tears (1.25). There is no margin to state below,
// because no swept value from 0.1 up fails — which is a statement about a mesher that no longer breaks
// there, not about a constant that no longer matters.
//
// The complement's face area is pinned two-sided at the value this k gives (chart_face_mesh_test.go), so
// the constant cannot move to any value that changes that mesh without saying so.
const chartBoundaryClearance = 0.90 // tol:mesh-density (chords; 33 values swept 0.10…3.00 above, tears from 1.25)

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
