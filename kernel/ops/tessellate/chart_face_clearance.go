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
// THE LAST TWO COLUMNS PREDATE chartClearanceCellCap, and reading them as this constant's shipped cost
// is the trap this paragraph exists to close (#3527). #3517 caps the chord the clearance is read from at
// a fraction of the covering's own cell, and on the complement's oval apex — where an uncapped chord is
// many cells long — the CAP binds, so k = 0.90 does not reach what it asks for. At the shipped constant
// the deficit measures 1.1097 % (201.642061 against the analytic 203.904871) with a face area of
// 263.73402 mm², which happens to equal what the table recorded at k = 0.3 — the same number, not the
// same clearance: shipped k = 0.3 reads 1.0936 %.
//
// THE SELECTION ARGUMENT BELOW DOES NOT SURVIVE THAT, and this is the part a later sweeper must not
// read past. Its rule is "take the LARGEST value the cost ceiling admits", and it names 0.92 as the
// first swept value over the 1.39 % ceiling at 1.4136 %. Re-measured under the cap, one constant edit
// at a time (#3527 review 1):
//
//	k        table's cost   measured under the cap   face area
//	0.10         1.0936 %                 1.0936 %   263.76393
//	0.30         1.1097 %                 1.0936 %   263.76393
//	0.90         1.3497 %                 1.1097 %   263.73402   (shipped)
//	0.92         1.4136 %                 1.1097 %   263.73402
//	1.20         1.5979 %                 1.1148 %   263.72994
//	1.24         1.7156 %                 1.1148 %   263.72994
//	1.50         1.8616 %                 1.1148 %           —
//	2.00         2.7038 %                 1.2708 %           —
//	3.00         5.8686 %                 1.5979 %           —
//
// So 0.92 is NOT over the ceiling — it reads a fifth of the way to it — and every swept k from 0.10 to
// 1.50 clears 1.39 % with room, the whole range 0.10–1.24 spanning 0.021 percentage points. The cost
// premise no longer DISCRIMINATES over the admissible range, so the rule as written points at the last k
// before the corpus tears (≈1.24), not at 0.90: the shipped constant is not what its own stated rule
// selects. What 0.90 rests on today is the tear edge above it and the larger-is-better premise alone.
//
// The tear column is a separate question and was NOT re-measured here; the argument above holds either
// way. If the tear edge is still 1.25 the rule selects 1.24; if the cap moved that too, nothing is known
// about what the rule selects. Restoring an argument for this constant needs the 33-value sweep re-run
// with the cap in place — a task of its own, not a comment — and until it is run, treat the value as
// inherited rather than derived.
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
// TWO premises select 0.90, and the second one is the half a cost ceiling cannot supply on its own.
// READ THE CAP PARAGRAPH ABOVE FIRST: every cost figure in the rest of this receipt is a pre-cap
// reading, and the first premise no longer discriminates under the cap. What follows is the argument as
// it was made, kept because the second premise still stands and because a re-sweep has to know what it
// is replacing.
//
// The CEILING is 1.39 %: the deficit the deleted window mesher achieved on this body, which
// chart_face_mesh_test.go's own row exists to have beaten. It admits every k up to 0.91 — 0.92 is the
// first swept value over it, at 1.4136 % (pre-cap: 0.92 reads 1.1097 % on the shipped tree, well under
// the ceiling) — and the round that sat at 0.935 was over it at 1.4137 %. A ceiling defines an
// admissible SET, so on its own it would select the cheapest admissible value, k = 0.1 at 1.0936 %.
//
// The second premise is why LARGER is better inside that set, and it is the paragraph above stated as a
// rule rather than as history: a smaller clearance lets an interior node approach the rim, and the two
// failures this sweep used to show below 0.70 were both nodes that reached it. Those particular
// failures are fixed, but what the clearance buys is unchanged — distance between the covering's
// interior and a boundary the chart samples differently — and nothing measures how much of that is
// enough, because no swept value fails any more. So the rule is: take the LARGEST value the cost
// ceiling admits, which buys the most of the thing the guard exists for at a price the corpus's own
// claim still tolerates. That selected 0.90 on the pre-cap column (0.91 builds the identical mesh; 0.92
// was over the ceiling there). Under the cap the same rule selects ≈1.24 instead, which is why the
// constant now needs a re-sweep rather than a re-reading.
//
// Both premises are weaker than a measured failure edge and the comment says so rather than dressing
// them up. In particular the ceiling can never be RE-taken: the mesher that produced it is deleted, and
// the reading is historical (commit 95cdd21e records 201.07 against an analytic 203.90 on this body,
// a 1.388 % deficit, with its raw operands). The row's own comparison has eroded across this wave —
// 1.11 % at 95cdd21e, 1.2982 % at 0.875, 1.3497 % here — and 0.90 sits 0.04 percentage points under
// the ceiling. A later sweep that wants to move this constant has to say what it does to that claim,
// and a change that retires the claim leaves this constant with no argument at all: re-derive it then,
// do not inherit it.
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
	margin := stdmath.Max(gridMargin, chartBoundaryClearance*stdmath.Min(c.chord, chartClearanceCellCap*b.coverCell()))
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

// chartClearanceCellCap caps the chord the boundary clearance is read from at this many of the
// covering's OWN cells — because past about a cell, a clearance stops protecting the mesh and starts
// deleting it.
//
// The clearance keeps interior grid nodes out of the band where the CHART's boundary and the mesh's
// chord polygon disagree, and it is measured in chords because what fails at a coarse rim is a node
// landing inside the chord rather than beside it. But a chord is only evidence of that band while it
// is comparable to the grid: a chord many cells long clears many cells of interior, and the
// triangulation then bridges the hole with edges that span the same distance — which is the chord
// error the clearance was protecting. Measured on occtparity W8 face 2 before the routing fix, and on
// six more faces after it, that bridging IS the chord defect.
//
// Swept, reading three things that fail in different directions: the genus-1 complement's torus face
// at DefaultQuality (the case the chord term exists for — below the edge it is DECLINED and falls to
// the surface's whole domain), the count of failing rows in ./kernel/ops/tessellate, and the worst
// chord sagitta ÷ PropertyQuality's tolerance on the pinned corpus's charted faces:
//
//	cap     complement D          rows  J3 f00  A6 f06  K2 f03  J5 f00  I9 f00  K1 f05
//	0.125   294.428, 28 free       7    1.396   0.971   0.856   1.465   0.941   0.565
//	0.25    294.428, 28 free       7    1.396   0.971   0.856   1.465   0.941   0.565
//	0.375   294.428, 28 free       7    —       —       —       —       —       —
//	0.5     294.428, 28 free       7    —       —       —       —       —       —
//	0.625   294.428, 28 free       7    —       —       —       —       —       —
//	0.75    263.734,  0 free       3    —       —       —       —       —       —
//	0.875   263.734,  0 free       2    1.530   0.971   3.632   1.465   0.941   0.565
//	1.0     263.734,  0 free       2    1.581   0.971   4.119   1.465   0.941   0.565
//	1.25    263.6,    0 free       2    4.221   1.553   4.389   4.059   3.765   2.259
//	1.5     263.6,    0 free       2    4.221   1.553   4.964   4.059   3.765   2.259
//	2.0+    263.555,  0 free       2    4.221   1.882   4.964   4.059   3.765   2.259
//
// Bounded on BOTH sides, which is what makes 0.875 a choice rather than an edge: at 0.625 and below
// the complement loses its own region entirely, and at 1.25 and above every chord ratio reverts to the
// uncapped reading. The window is [0.75, 1.0] and 0.875 is its midpoint, 1.4× the largest failing
// value. The two rows that remain in the "rows" column are pins this change MOVES and which are
// re-measured with it: the complement's own face area (263.55487 → 263.73402, against an analytic
// 264.88981 — it moves TOWARD the oracle) and RODB∩, which the routing fix moves, not this cap.
const chartClearanceCellCap = 0.875 // tol:mesh-density (covering cells; swept 0.125…2.0 above)

// coverCell is the covering's own cell size, as a 3D length: the smaller of its two mean station gaps.
func (b *chartCover) coverCell() float64 {
	gu := (b.r.uHi - b.r.uLo) / stdmath.Max(1, float64(len(b.us))) * b.su
	gv := (b.r.vHi - b.r.vLo) / stdmath.Max(1, float64(len(b.vs))) * b.sv
	return stdmath.Min(gu, gv)
}
