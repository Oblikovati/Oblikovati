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
// RE-SWEPT for #3519, over a wider range and a wider corpus, and the earlier table's two structural
// claims are BOTH withdrawn by the measurement. It said the failure edge rests on ONE face (the
// complement's oval apex at PropertyQuality, for k ≤ 0.55) and that the upper end is no failure boundary
// at all. Neither holds: the complement at PropertyQuality is watertight at EVERY k swept here,
// including 0.1 — measured with the deleted incidence retry restored as well, so it is not this slice's
// doing but something the wave fixed after the table was written — and the corpus does tear above the
// plateau.
//
// The corpus is the 21 classification-corpus bodies, the merged cocylindrical wall and six
// torus-cut-by-its-tangent-plane pieces, at BOTH facetings. Four faces decide; every other row in the
// corpus is watertight at every k from 0.1 to 3.0. Free edges of the whole body, and the charted face's
// own meshed area (a full-domain area is the decline, not a mesh):
//
//	k        complement D        R=5 r=2 ∩ D         R=5 r=2.5 ∩ D       R=5 r=2 − D
//	0.2      28 / 294.42794      44 / 392.57058      48 / 490.71323       0 / 281.68147
//	0.25      0 / 263.74953      44 / 392.57058      48 / 490.71323       0 / 281.68147
//	0.3       0 / 263.73402      44 / 392.57058      48 / 490.71323       0 / 281.69358
//	0.35      0 / 263.73402       0 / 110.94830      48 / 490.71323       0 / 281.69147
//	0.5       0 / 263.72994       0 / 110.94725      48 / 490.71323       0 / 281.61993
//	0.7       0 / 263.60871       0 / 110.87427      48 / 490.71323       0 / 281.57144
//	0.705     0 / 263.60871       0 / 110.87427       0 / 158.45510       0 / 281.57144
//	0.8       0 / 263.56345       0 / 110.86598       0 / 158.46089       0 / 281.54775
//	0.875     0 / 263.55487       0 / 110.87045       0 / 158.47274       0 / 281.51437
//	0.935     0 / 263.45103       0 / 110.87045       0 / 158.47274       0 / 281.52457
//	1.0       0 / 263.42317       0 / 110.86804       0 / 158.44277       0 / 281.54574
//	1.24      0 / 263.28202       0 / 110.84685       0 / 158.32769       0 / 281.49772
//	1.25      0 / 263.28202       0 / 110.84685       0 / 158.32769       2 / 281.58399
//	1.5       0 / 263.15360       0 / 110.55359       0 / 158.25096       2 / 281.74082
//	3.0       0 / 260.51898       0 / 108.77724       0 / 156.61430       0 / 280.72340
//
// 29 values were swept in all: 0.1, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.45, 0.5, 0.55, 0.6, 0.7, 0.705,
// 0.75, 0.8, 0.85, 0.875, 0.92, 0.935, 0.95, 1.0, 1.1, 1.2, 1.24, 1.25, 1.3, 1.4, 1.5, 3.0. The rows
// left out of the table above fall inside the steps it shows.
//
// So, stated as what was measured rather than as a property of the interval: EVERY ONE of those 29
// values from 0.705 to 1.24 inclusive meshes every corpus row watertight, and the two swept values that
// bracket that run both tear — 0.70 by 48 free edges on the R=5 r=2.5 intersect piece, 1.25 by 2 on the
// R=5 r=2 cut piece. The first draft of this table said the plateau was [0.75, 1.2] "and nowhere wider",
// which was a claim about where the sweep had put its steps, not about the geometry; the edges are
// actually bracketed at (0.70, 0.705] and [1.24, 1.25). Nothing here says the run is unbroken between
// two swept values.
//
// BOTH brackets are a self-touching boundary: the figure-eight pinch below (R=5 r=2 declines for
// k ≤ 0.3 and R=5 r=2.5 for k ≤ 0.70, torus_square_pinch_test.go) and the same family's other piece
// above. The complement's apex, which the old table thought was the whole edge, is the weakest of the
// four — it only fails for k ≤ 0.2.
//
// 0.935 is the LOG-CENTRE of the bracketed run, √(0.70 × 1.25) = 0.93541 to three decimals, and it is
// 1.336× either bracketing failure. The first draft kept 0.875 (1.25× below, 1.43× above) on the
// argument that "the cost above the edge is monotone, so the smallest value with a real margin is what
// the cost argument wants" — which is an argument the same sweep refutes, because the upper end is a
// TEAR and not only a cost. With a failure at both ends the cost term cannot break the tie and the
// centre does.
//
// Centring is not free, and the price is a body VOLUME rather than the face area: the complement's own
// DefaultQuality mesh volume goes from 201.258 to 201.022 against an analytic 203.905, so its coarse
// chord deficit rises from 1.298 % to 1.414 % — just past the 1.39 % the deleted window mesher used to
// achieve on that body, which chart_face_mesh_test.go's own comment cites. Its face area costs
// 0.104 mm² of 263.6 (0.039 %), and no gate moves: every bound in the corpus is 1 % or looser and the
// PropertyQuality reading of the same body is 0.017 %. It is recorded here because a rise in a measured
// deficit is a price, not a rounding.
//
// The complement's face area is pinned two-sided at the value this k gives (chart_face_mesh_test.go),
// so the constant cannot move to any value that changes that mesh without saying so.
const chartBoundaryClearance = 0.935 // tol:mesh-density (chords; 29 values swept 0.10…3.00 above, tears at 0.70 and 1.25)

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
