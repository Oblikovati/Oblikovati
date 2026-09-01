// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Where the B-spline host canal's stations are PLACED along the spine (split out of
// fillet_bspline_host_band.go for #2218).
//
// A closed spine wraps and needs its anchors spaced round the loop; an open one is cut at both
// ends, prolonged past the cut so the loft has material to trim, then clamped back and deduped.
// The plan is the only thing the loft above needs to know about that difference.

// bsplineHostAnchorPlan is one march's anchor list with the two EDGE-END station indices:
// stations [iEdge0, iEdge1] anchor on the picked edge itself; indices outside are prolong
// stations riding the hosts' natural extension (prolong-then-trim,
// BRepBlend_SurfRstLineBuilder.cxx). arcs holds each anchor's signed arc coordinate
// (0 at the edge start, negative before it) for the end-window searches.
type bsplineHostAnchorPlan struct {
	anchors []geom.CanalEdgeAnchor
	arcs    []float64
	iEdge0  int
	iEdge1  int
}

// newBsplineHostAnchorPlan builds the section anchors for one march: n+1 arc-length-
// uniform on-edge anchors, GEOMETRICALLY refined toward both open ends (the cap-trim
// region needs station spacing far below the uniform step to hold the envelope bound on
// the retained sliver), plus geometric prolong anchors out to bsplineHostOverrunFrac·r.
// A closed edge gets the uniform loop with the seam-tangent closure fix instead.
func newBsplineHostAnchorPlan(spec bsplineHostMarchSpec, dir float64, n int) bsplineHostAnchorPlan {
	if spec.closed {
		return closedAnchorPlan(spec, dir, n)
	}
	arcs := openAnchorArcs(spec.arcTable.length, spec.cut, n)
	return openAnchorPlan(spec, dir, arcs)
}

// closedAnchorPlan is the closed-rim anchor loop: uniform, with the closure station in
// the IDENTICAL section plane as station 0 — one-sided polyline tangents at the seam
// differ by the discretization, which would open an r·Δangle closure gap.
func closedAnchorPlan(spec bsplineHostMarchSpec, dir float64, n int) bsplineHostAnchorPlan {
	on := spec.arcTable.uniformAnchors(n+1, dir)
	seamT := spec.arcTable.closedSeamTangent(dir)
	on[0].T, on[len(on)-1].T = seamT, seamT
	on[len(on)-1].P = on[0].P
	arcs := make([]float64, len(on))
	for k := range arcs {
		arcs[k] = spec.arcTable.length * float64(k) / float64(n)
	}
	return bsplineHostAnchorPlan{anchors: on, arcs: arcs, iEdge0: 0, iEdge1: len(on) - 1}
}

// openAnchorArcs is the sorted signed arc-coordinate ladder of an open edge's anchors
// over the retained cut window: uniform interior + geometric refinement (step halvings
// down to step/64) toward the cut ends AND the edge ends (where the hosts' extension
// kinks the foot rails) + geometric prolong runs out to the exact cut bounds.
func openAnchorArcs(length float64, cut [2]float64, n int) []float64 {
	onLo, onHi := stdmath.Max(0, cut[0]), stdmath.Min(length, cut[1])
	step := (onHi - onLo) / float64(n)
	arcs := make([]float64, 0, n+64)
	for k := 0; k <= n; k++ {
		arcs = append(arcs, onLo+step*float64(k))
	}
	for d := step / 2; d >= step/64; d /= 2 {
		arcs = append(arcs, onLo+d, onHi-d, d, length-d)
	}
	arcs = appendProlongArcs(arcs, cut, length, step)
	arcs = clampArcsToCut(arcs, cut)
	sort.Float64s(arcs)
	return dedupArcs(arcs, step/256)
}

// appendProlongArcs adds the geometric prolong runs past each edge end (fine near the
// end, coarse toward the cut bound) plus the exact cut endpoints.
func appendProlongArcs(arcs []float64, cut [2]float64, length, step float64) []float64 {
	if cut[0] < 0 {
		for d := step / 64; d < -cut[0]; d *= 2 {
			arcs = append(arcs, -d)
		}
		arcs = append(arcs, cut[0])
	}
	if cut[1] > length {
		for d := step / 64; d < cut[1]-length; d *= 2 {
			arcs = append(arcs, length+d)
		}
		arcs = append(arcs, cut[1])
	}
	return arcs
}

// clampArcsToCut drops ladder values outside the retained cut window (an edge-end
// refinement value can overshoot a cut that begins inside the edge).
func clampArcsToCut(arcs []float64, cut [2]float64) []float64 {
	out := arcs[:0]
	for _, a := range arcs {
		if a >= cut[0] && a <= cut[1] {
			out = append(out, a)
		}
	}
	return out
}

// dedupArcs drops near-duplicate arc coordinates (a refinement value landing on a uniform
// station) so no two section planes coincide.
func dedupArcs(arcs []float64, tol float64) []float64 {
	out := arcs[:1]
	for _, a := range arcs[1:] {
		if a-out[len(out)-1] > tol {
			out = append(out, a)
		}
	}
	return out
}

// openAnchorPlan realizes the arc ladder into anchors: on-edge arcs evaluate the arc
// table; prolong arcs extrapolate along the end tangents keeping the END section normal
// (the walking line continues in the last section family). dir=−1 reverses the whole
// march order (and the section normals with it).
func openAnchorPlan(spec bsplineHostMarchSpec, dir float64, arcs []float64) bsplineHostAnchorPlan {
	L := spec.arcTable.length
	p0, t0 := spec.arcTable.at(0)
	p1, t1 := spec.arcTable.at(L)
	plan := bsplineHostAnchorPlan{anchors: make([]geom.CanalEdgeAnchor, len(arcs)), arcs: arcs, iEdge0: -1}
	for k, s := range arcs {
		plan.anchors[k] = openAnchorAt(spec, s, L, p0, t0, p1, t1)
		if s >= 0 && plan.iEdge0 < 0 {
			plan.iEdge0 = k
		}
		if s <= L {
			plan.iEdge1 = k
		}
	}
	if dir < 0 {
		reverseAnchorPlan(&plan)
	}
	return plan
}

// openAnchorAt evaluates one anchor of the open plan: on-edge from the table, prolong by
// end-tangent extrapolation.
func openAnchorAt(spec bsplineHostMarchSpec, s, length float64, p0 math.Point3, t0 math.Vector3, p1 math.Point3, t1 math.Vector3) geom.CanalEdgeAnchor {
	if s < 0 {
		return geom.CanalEdgeAnchor{P: p0.TranslateBy(t0.Scale(math.Scalar(s))), T: t0}
	}
	if s > length {
		return geom.CanalEdgeAnchor{P: p1.TranslateBy(t1.Scale(math.Scalar(s - length))), T: t1}
	}
	p, t := spec.arcTable.at(s)
	return geom.CanalEdgeAnchor{P: p, T: t}
}

// reverseAnchorPlan flips the march order in place: anchors reversed, tangents negated,
// arc coordinates mirrored, edge indices swapped.
func reverseAnchorPlan(plan *bsplineHostAnchorPlan) {
	n := len(plan.anchors)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		plan.anchors[i], plan.anchors[j] = plan.anchors[j], plan.anchors[i]
		plan.arcs[i], plan.arcs[j] = plan.arcs[j], plan.arcs[i]
	}
	for i := range plan.anchors {
		plan.anchors[i].T = plan.anchors[i].T.Scale(-1)
	}
	plan.iEdge0, plan.iEdge1 = n-1-plan.iEdge1, n-1-plan.iEdge0
}
