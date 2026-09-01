// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"
	"slices"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Sharp-corner run-outs (Oblikovati#2083). fillet_stripe_caps.go closes an open run with a flat cap
// face in the terminal section plane, on the premise — stated in its own header — that the plane
// "never intrudes on the neighbouring geometry". That holds when the run stops part-way along a rim,
// where the section plane is genuinely new. It does NOT hold when the run stops at a SHARP CORNER:
// there the spine reaches the end of the rim, and the section plane is the plane of the side face
// already sitting at that corner.
//
// Building a cap there produced the right point set by cancellation and the wrong B-rep: a reversed
// cap face lying on an untrimmed neighbour (measured on a 4 cm box, r=0.25: two such pairs, each
// double-covering 0.0134 cm2), plus two zero-area spikes per terminal where the corner connectors
// doubled back along a boundary they were collinear with. Every edge was still used twice, so the
// topology gate passed it, and the flaps cancel in the volume integral, so the OCCT volume oracle
// passed it too — but the surface area over-counted and every face-based consumer (render, STEP,
// boolean) saw two coincident opposite faces.
//
// The fix is to recognise the case and let the side face carry the run-out: it is re-trimmed along
// the section arc, the corner vertex is consumed, and no cap face or connector is built at all.

// stripeEnd is a run-out that lands ON an existing planar face. endFace is that face; topEdge and
// wallEdge are the two of its boundary edges that meet at the consumed corner — one shared with the
// stripe's shared face, one with the terminal segment's wall. All three are nil when the terminal's
// section plane is genuinely new, which is the case the flat cap is for.
type stripeEnd struct {
	endFace  *topo.Face
	topEdge  *topo.Edge
	wallEdge *topo.Edge
}

// active reports whether this terminal lands on an existing face, so the cap must be skipped.
func (e stripeEnd) active() bool { return e.endFace != nil }

// resolveStripeEnds classifies both run-outs of an open stripe. A terminal is only taken as landing
// on a face when EVERY part of the substitution is available — the coincident planar face and both
// of its boundary edges at the corner — so anything unusual keeps the existing cap path rather than
// producing a half-rebuilt corner.
//
// Example: ends := resolveStripeEnds(st, weld); if ends[0].active() { /* no cap at terminal 0 */ }
func resolveStripeEnds(st *tangentStripe, weld float64) [2]stripeEnd {
	var out [2]stripeEnd
	if st.closed {
		return out
	}
	for t := range 2 {
		out[t] = resolveOneStripeEnd(st, t, weld)
	}
	return out
}

// resolveOneStripeEnd classifies terminal t, returning the zero stripeEnd (keep the flat cap) unless
// every piece of the substitution is there.
func resolveOneStripeEnd(st *tangentStripe, t int, weld float64) stripeEnd {
	tm := st.term[t]
	wall := st.segs[tm.seg].wall
	origin, normal, ok := terminalSectionPlane(tm)
	if !ok {
		return stripeEnd{}
	}
	face := coplanarFaceAt(tm.vertex, origin, normal, st.shared, wall, weld)
	if face == nil {
		return stripeEnd{}
	}
	top, wallE := edgeSharedBy(tm.vertex, st.shared, face), edgeSharedBy(tm.vertex, wall, face)
	if top == nil || wallE == nil {
		return stripeEnd{}
	}
	return stripeEnd{endFace: face, topEdge: top, wallEdge: wallE}
}

// terminalSectionPlane is the plane of the run-out section, taken from the three points that already
// define the cap arc. Orientation is irrelevant here — this plane is only ever compared with a face's,
// never used to decide which side the solid is on.
func terminalSectionPlane(tm stripeTerm) (math.Point3, math.Vector3, bool) {
	n := tm.topA.VectorTo(tm.apex).Cross(tm.topA.VectorTo(tm.wallA))
	l := float64(n.Length())
	if l == 0 {
		return math.Point3{}, math.Vector3{}, false // a collapsed section: no plane to compare
	}
	return tm.topA, n.Scale(math.Scalar(1 / l)), true
}

// coplanarFaceAt returns the planar face at v, other than skipA/skipB, whose plane is the one through
// origin with unit normal n.
func coplanarFaceAt(v *topo.Vertex, origin math.Point3, n math.Vector3, skipA, skipB *topo.Face, weld float64) *topo.Face {
	return soleCoplanarFace(facesAt(v), origin, n, skipA, skipB, weld)
}

// soleCoplanarFace picks the ONE candidate among faces, or nil when there is none or more than one.
// Ambiguity is declined rather than resolved: two faces in the section plane at one corner is not the
// simple sharp run-out this path rebuilds, and the flat cap is the safer answer there.
func soleCoplanarFace(faces []*topo.Face, origin math.Point3, n math.Vector3, skipA, skipB *topo.Face, weld float64) *topo.Face {
	var found *topo.Face
	for _, f := range faces {
		if f == skipA || f == skipB || !planeMatches(f, origin, n, weld) {
			continue
		}
		if found != nil {
			return nil
		}
		found = f
	}
	return found
}

// planeMatches reports whether f is planar and its plane is the one through origin with normal n.
// The normal test is on the SINE of the angle between the two normals, so it carries no length; the
// offset test is a distance, so it is compared against the model-relative weld.
func planeMatches(f *topo.Face, origin math.Point3, n math.Vector3, weld float64) bool {
	pl, planar := f.Geometry().(geom.Plane)
	if !planar {
		return false
	}
	if float64(pl.Normal().Cross(n).Length()) > planeParallelSin {
		return false
	}
	return stdmath.Abs(float64(pl.Origin.VectorTo(origin).Dot(n))) <= weld
}

// planeParallelSin is the largest sine of the angle between two normals still called parallel
// (~0.06°). tol:angular — a sine is dimensionless, so it carries no model scale (ADR-0042).
const planeParallelSin = 1e-3

// facesAt returns the distinct faces meeting at v, walking its edges.
func facesAt(v *topo.Vertex) []*topo.Face {
	seen := map[*topo.Face]bool{}
	out := make([]*topo.Face, 0, 4)
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if !seen[f] {
				seen[f], out = true, append(out, f)
			}
		}
	}
	return out
}

// edgeSharedBy returns the single edge at v bounding both a and b.
func edgeSharedBy(v *topo.Vertex, a, b *topo.Face) *topo.Edge {
	return soleEdgeBounding(v.Edges(), a, b)
}

// soleEdgeBounding picks the ONE edge bounding both a and b, or nil when there is not exactly one —
// the same "leave it to the cap path" refusal as soleCoplanarFace.
func soleEdgeBounding(edges []*topo.Edge, a, b *topo.Face) *topo.Edge {
	var found *topo.Edge
	for _, e := range edges {
		if !edgeBounds(e, a) || !edgeBounds(e, b) {
			continue
		}
		if found != nil {
			return nil
		}
		found = e
	}
	return found
}

// edgeBounds reports whether e is one of f's boundary edges.
func edgeBounds(e *topo.Edge, f *topo.Face) bool {
	return slices.Contains(e.Faces(), f)
}

// curveBetween restricts c to the span from a to b, presented over [0,1] in that direction.
//
// It is not enough to hand the parent curve to a shorter edge: an edge's curve must span exactly that
// edge, because tessellate.SampleEdgeCurve walks the curve's WHOLE domain and only snaps the two end samples to
// the vertices. A curved boundary cut back to a section foot would otherwise be tessellated along its
// original sweep — the same class of defect as the canal sub-edge TrimmedCurve3 was written for.
//
// Example: sub, err := curveBetween(rim.Geometry(), foot.Point(), far.Point(), weld)
func curveBetween(c geom.Curve3, a, b math.Point3, weld float64) (geom.Curve3, error) {
	ta, err := paramOn(c, a, weld)
	if err != nil {
		return nil, err
	}
	tb, err := paramOn(c, b, weld)
	if err != nil {
		return nil, err
	}
	return geom.TrimmedCurve3{Base: c, Lo: ta, Hi: tb}, nil
}

// paramOn is p's parameter on c, refusing a point that is not ON c. CurveParamAtPoint3 PROJECTS —
// a Line returns the foot of the perpendicular and calls it unique — so the answer has to be checked
// by evaluating it back, or a corner this path misread would be cut back to an invented point.
func paramOn(c geom.Curve3, p math.Point3, weld float64) (float64, error) {
	t, nature := geom.CurveParamAtPoint3(c, p)
	if nature != geom.UniqueSolution {
		return 0, fmt.Errorf("no unique parameter for %v on the boundary curve, so it cannot be cut back", p)
	}
	if d := float64(c.PointAt(t).DistanceTo(p)); d > weld {
		return 0, fmt.Errorf("%v is %.3g off the boundary curve (weld %.3g), so it cannot be cut back", p, d, weld)
	}
	return t, nil
}

// --- the rebuild half: what stripeBuild does once a terminal is classified active ---

// addEndRemnants cuts back the two end-face boundary edges that met at a consumed terminal corner,
// each to its section foot. The remnant carries the ORIGINAL curve restricted to the surviving span,
// so a curved boundary stays curved — an edge's curve must span exactly that edge (tessellate.SampleEdgeCurve
// walks the whole domain), which is why this trims rather than reusing the parent outright.
func (g *stripeBuild) addEndRemnants(t int, feet [2]*topo.Vertex, lin func(string, int) topo.Lineage) error {
	var err error
	if g.endTopE[t], err = g.endRemnant(t, g.ends[t].topEdge, feet[0], lin("etop", t)); err != nil {
		return err
	}
	g.endWallE[t], err = g.endRemnant(t, g.ends[t].wallEdge, feet[1], lin("ewall", t))
	return err
}

// endRemnant is one cut-back edge: e's surviving span, from the section foot to whichever of e's
// vertices is NOT the consumed corner.
func (g *stripeBuild) endRemnant(t int, e *topo.Edge, foot *topo.Vertex, lin topo.Lineage) (*topo.Edge, error) {
	far := otherVertex(e, g.st.term[t].vertex)
	sub, err := curveBetween(e.Geometry(), foot.Point(), far.Point(), g.weld)
	if err != nil {
		return nil, fmt.Errorf("fillet: stripe terminal %d: %w", t, err)
	}
	return g.bld.AddEdge(sub, foot, g.verts[far], lin), nil
}

// endEdgeSide identifies e as one of an active terminal's end-face boundary edges, returning that
// terminal and which side of the tube it bounds.
func (g *stripeBuild) endEdgeSide(e *topo.Edge) (term, side int, ok bool) {
	for t := range 2 {
		if !g.ends[t].active() {
			continue
		}
		if e == g.ends[t].topEdge {
			return t, endTopSide, true
		}
		if e == g.ends[t].wallEdge {
			return t, endWallSide, true
		}
	}
	return 0, 0, false
}

// Which side of the tube an end-face boundary edge bounds: the stripe's shared face, or the terminal
// segment's wall.
const (
	endTopSide = iota
	endWallSide
)

// termFeet are terminal t's two section feet — shared-face side, then wall side.
func (g *stripeBuild) termFeet(t int) [2]*topo.Vertex {
	if t == 0 {
		return [2]*topo.Vertex{g.vS1[0], g.vW[0]}
	}
	return [2]*topo.Vertex{g.vEndS1, g.vEndW}
}

// mapEndUse replaces a consumed corner's end-face boundary edge with its cut-back remnant. On the END
// FACE itself the section arc is spliced in as well, so that face's boundary now runs
// remnant → arc → remnant across the corner it used to turn at — which is the whole point: the arc
// bounds the end face directly instead of a separate cap face lying on top of it (#2083).
//
// The arc is emitted by whichever of the two sides ARRIVES at the corner, so it lands exactly once,
// exactly between them.
func (g *stripeBuild) mapEndUse(f *topo.Face, u *topo.EdgeUse, t, side int) []topo.Use {
	feet := g.termFeet(t)
	rem, foot := g.endTopE[t], feet[endTopSide]
	if side == endWallSide {
		rem, foot = g.endWallE[t], feet[endWallSide]
	}
	if useFromVertex(u) == g.st.term[t].vertex {
		return []topo.Use{dirUse(rem, foot)} // ran corner → far; now foot → far
	}
	walk := dirUse(rem, otherVertex(rem, foot)) // ran far → corner; now far → foot
	if f != g.ends[t].endFace {
		return []topo.Use{walk}
	}
	return []topo.Use{walk, dirUse(g.cap[t], foot)}
}
