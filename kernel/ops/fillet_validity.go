// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// filletFitEps guards the two direction-cosine degeneracies of the recession formula: near-flat
// faces (1−c→0, the edge is barely a crease) and razor edges (1+c→0, anti-parallel faces). These
// are absolute because c=nA·nB is a dimensionless cosine, not a length (ADR-0050 P2 / #1800).
const filletFitEps = 1e-9

// validateFilletRadii rejects picks whose radius exceeds the geometric maximum the local planar
// geometry admits — the closed-form planar specialization of OCCT's seed-section solvability
// (ChFiDS_StartsolFailure): past r_max the rolling-ball tangent band overruns the finite face and
// the fillet self-intersects, which the topological Validate cannot catch (#1800). Only edges
// between two planar faces are bounded here; curved-neighbour edges get their r_max from the
// marcher's seed solve in Phase 4. Radii are unchanged — this fails honestly, it does not clamp.
func validateFilletRadii(picks []filletPick, concave ConcaveFill) error {
	for _, p := range picks {
		a, b, nA, nB, err := edgePlanarFaces(p.edge)
		if err != nil {
			continue // non-planar edge: not on the analytic planar path
		}
		// The rolling ball recedes INTO each finite wall. For a convex edge that recession points
		// into the face; for a concave OUTWARD fill (the default pocket fill) the frame negates the
		// normals/offset (see filletFrame), so the width must be measured in that negated direction —
		// otherwise the ray escapes the face and the bound is +Inf, letting an over-large concave
		// radius overrun the pocket (rampam #1800 concave gap). A concave INWARD recess lands its
		// tangent points off the bounded faces by design, so it is left to the assembly's validity
		// gate rather than bounded here.
		concaveOutward := ClassifyEdgeConvexity(p.edge) == EdgeConcave && concave == FillConcaveOutward
		if ClassifyEdgeConvexity(p.edge) == EdgeConcave && !concaveOutward {
			continue
		}
		rMax, bindingFace, bindingW, ok := maxFilletRadius(p, a, b, nA, nB, picks, concaveOutward)
		if !ok {
			continue // degenerate dihedral: leave to the assembly's own validity gate
		}
		if r := p.maxRadius(); r > rMax*(1+1e-6) {
			phi := stdmath.Acos(-clampUnit(float64(nA.Dot(nB)))) * 180 / stdmath.Pi
			return fmt.Errorf(
				"fillet: radius %g exceeds geometric maximum %g on edge %d (available in-face width %g on face %d, dihedral %.1f°); reduce the radius or use a smaller value",
				r, rMax, p.edge.ID(), bindingW, bindingFace, phi)
		}
	}
	return nil
}

// maxFilletRadius returns the largest constant radius that fits between the two planar faces of
// pick p, r_max = cot(α/2)·min(W_A,W_B), with W the available in-face width toward the nearest
// competing boundary (reduced by any co-filleted neighbour's own band). ok=false on a degenerate
// dihedral (near-flat or razor faces). Mirrors the derivation in ADR-0050 Phase 2.
func maxFilletRadius(p filletPick, a, b *topo.Face, nA, nB math.Vector3, all []filletPick, concaveOutward bool) (rMax float64, bindingFace uint64, bindingW float64, ok bool) {
	c := float64(nA.Dot(nB))
	if 1-c < filletFitEps || 1+c < filletFitEps {
		return 0, 0, 0, false
	}
	offDir := nA.Add(nB).Scale(math.Scalar(-1 / (1 + c)))
	if concaveOutward {
		// Match the outward-fill frame (filletFrame): the ball sits in the void, so the recession
		// runs the other way — negate the offset and the wall normals so availableWidth casts INTO
		// the pocket walls. c (and thus k) is unchanged: (−nA)·(−nB) = nA·nB.
		offDir, nA, nB = offDir.Scale(-1), nA.Scale(-1), nB.Scale(-1)
	}
	k := stdmath.Sqrt((1 + c) / (1 - c)) // cot(α/2): r_max = k · W
	wA := availableWidth(p.edge, a, nA, offDir, all)
	wB := availableWidth(p.edge, b, nB, offDir, all)
	if wA <= wB {
		return k * wA, a.ID(), wA, true
	}
	return k * wB, b.ID(), wB, true
}

// availableWidth is the smallest in-face clearance from edge e into face F perpendicular to e,
// found by casting rays from samples along e toward the tangent-recession direction and taking
// the nearest boundary hit. When the nearest boundary is itself a filleted edge, its own band is
// subtracted so the two bands do not collide (constraint (b) of the ADR-0050 P2 derivation).
//
// The clearance w(t) along the spine is a pointwise minimum of affine functions (each competing
// edge's crossing distance is affine in t because the ray origin moves along e while the ray
// direction is fixed), hence CONCAVE — so its minimum is attained at a spine endpoint, never
// strictly inside. The two endpoints are therefore load-bearing and must be sampled EXACTLY at the
// stored vertices: interpolating A+(B−A) reintroduces ~u_mach·|coords| of roundoff (≈4e-13 at
// 2000-unit scale), which makes a boundary edge sharing that endpoint vertex cross the ray at
// x≈5e-15 instead of exactly 0 — slipping past rayClearance's x≤0 guard and poisoning the width to
// ~0 (the G2 "radius exceeds geometric maximum 0" bug). Sampling the exact vertex plus excluding
// the edges incident to that vertex (they meet e at the corner, where the crossing is x=0 by
// construction, not a real opposing wall) removes the phantom exactly for polygonal faces while
// still catching a reflex fold via the interior samples. See geometry-math-advisor derivation.
func availableWidth(e *topo.Edge, f *topo.Face, nF, offDir math.Vector3, all []filletPick) float64 {
	mF := offDir.Add(nF) // = tangent-recession direction, already in F's plane (offDir·nF = −1)
	if l := float64(mF.Length()); l > 0 {
		mF = mF.Scale(math.Scalar(1 / l))
	}
	yAxis := nF.Cross(mF) // in-plane, ⟂ to the ray
	graze := grazeFloor * float64(e.StartVertex().Point().VectorTo(e.EndVertex().Point()).Length())
	best := stdmath.Inf(1)
	for _, sp := range widthSamples(e) {
		for _, g := range f.Edges() {
			if g == e || sharesVertex(g, sp.corner) {
				continue // skip e itself and, at an endpoint sample, the edges meeting e there
			}
			s, hit := rayClearance(sp.point, mF, yAxis, g.StartVertex().Point(), g.EndVertex().Point())
			if !hit {
				continue
			}
			// An edge meeting the fillet edge at a shared vertex is part of the SAME-side boundary
			// (the fillet edge's own straight or tangent-curved continuation), not an opposing wall.
			// A straight one grazes to x=0 only at the shared vertex (caught by sp.corner above); a
			// tangent CURVED one hugs the ray and grazes sub-ε along the whole span (N5: a near-line
			// arc tangent to the fillet edge, x~1e-10). Drop those grazes. A genuine opposing wall —
			// including a real thin ligament — shares NO vertex with e, so this never blinds the
			// #1800 self-intersection gate. Threshold is model-scaled to the fillet edge length.
			if s < graze && incidentToFilletEdge(g, e) {
				continue
			}
			best = stdmath.Min(best, s-neighbourBand(g, all))
		}
	}
	return stdmath.Max(best, 0)
}

// widthSample is one ray origin along the fillet edge. corner is the stored vertex when the sample
// sits exactly on an edge endpoint (so incident boundary edges are excluded there), nil for the
// interior samples.
type widthSample struct {
	point  math.Point3
	corner *topo.Vertex
}

// widthSamples returns the ray origins used to probe a face's in-face width: the two edge
// endpoints taken EXACTLY from the stored vertices (concave-clearance minima live there, and exact
// coordinates keep a shared-corner crossing at exactly x=0), plus evenly spaced interior points.
func widthSamples(e *topo.Edge) []widthSample {
	sv, ev := e.StartVertex(), e.EndVertex()
	s, t := sv.Point(), ev.Point()
	const n = 5
	out := make([]widthSample, n)
	out[0] = widthSample{point: s, corner: sv}
	out[n-1] = widthSample{point: t, corner: ev}
	for i := 1; i < n-1; i++ {
		out[i] = widthSample{point: s.TranslateBy(s.VectorTo(t).Scale(math.Scalar(float64(i) / float64(n-1))))}
	}
	return out
}

// sharesVertex reports whether edge g is incident to vertex v (nil v ⇒ false, for interior
// samples). Compared by vertex identity so a boundary edge meeting the fillet edge at a corner is
// recognised regardless of coordinate roundoff.
func sharesVertex(g *topo.Edge, v *topo.Vertex) bool {
	if v == nil {
		return false
	}
	return g.StartVertex().ID() == v.ID() || g.EndVertex().ID() == v.ID()
}

// grazeFloor scales, relative to the fillet edge length, the clearance below which a boundary edge
// SHARING a fillet-edge vertex is treated as the same-side continuation (a straight or tangent-arc
// grazing) rather than an opposing wall. Sits far above float roundoff (~1e-9·L worst case) and far
// below any real in-face ligament (≥ ~1e-4·L). See geometry-math-advisor ε valley.
const grazeFloor = 1e-6

// incidentToFilletEdge reports whether g meets the fillet edge e at either of e's endpoints.
func incidentToFilletEdge(g, e *topo.Edge) bool {
	return sharesVertex(g, e.StartVertex()) || sharesVertex(g, e.EndVertex())
}

// rayClearance returns the distance s>0 at which the in-plane ray from origin O along xAxis
// crosses segment [Q0,Q1] (coordinates taken in the (xAxis,yAxis) plane basis), or ok=false when
// the segment does not straddle the ray on the forward side.
func rayClearance(origin math.Point3, xAxis, yAxis math.Vector3, q0, q1 math.Point3) (float64, bool) {
	y0 := float64(origin.VectorTo(q0).Dot(yAxis))
	y1 := float64(origin.VectorTo(q1).Dot(yAxis))
	if (y0 > 0) == (y1 > 0) {
		return 0, false // both endpoints on the same side ⇒ no crossing of the y=0 ray line
	}
	u := y0 / (y0 - y1) // parameter along the segment where y=0
	x0 := float64(origin.VectorTo(q0).Dot(xAxis))
	x1 := float64(origin.VectorTo(q1).Dot(xAxis))
	x := x0 + u*(x1-x0)
	if x <= 0 {
		return 0, false // crossing is behind the ray origin
	}
	return x, true
}

// neighbourBand is the in-face recession of edge g's own fillet when g is one of the picks, else
// 0 — how far g's band eats into the shared face, subtracted from the clearance.
func neighbourBand(g *topo.Edge, all []filletPick) float64 {
	for _, p := range all {
		if p.edge != g {
			continue
		}
		_, _, nA, nB, err := edgePlanarFaces(g)
		if err != nil {
			return 0
		}
		c := float64(nA.Dot(nB))
		if 1+c < filletFitEps {
			return 0
		}
		return p.maxRadius() * stdmath.Sqrt((1-c)/(1+c)) // d(r) = r·√((1−c)/(1+c))
	}
	return 0
}

// maxRadius is the largest radius applied along a pick (both ends plus any intermediate stops) —
// the recession is deepest there, so the fit bound is checked against it.
func (p filletPick) maxRadius() float64 {
	m := stdmath.Max(p.r0, p.r1)
	for _, rp := range p.mids {
		m = stdmath.Max(m, rp.R)
	}
	return m
}

// clampUnit constrains x to [-1,1] so Acos of a rounded dot product never yields NaN.
func clampUnit(x float64) float64 {
	if x < -1 {
		return -1
	}
	if x > 1 {
		return 1
	}
	return x
}
