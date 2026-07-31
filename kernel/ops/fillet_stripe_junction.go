// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/blend"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Stripe junction crossings (simple/Y9, OCCT ChFi3d parity). A stripe's blend faces must be split
// where the SUPPORT FACES change under the contact, not where the chain's own vertices happen to
// sit: the descending edge below a junction meets the blend's contact curve at its own CROSSING
// meridian, which coincides with the junction vertex's meridian only when the descending edge runs
// through the contact foot (the box-rim case). On simple/Y9 the top face is split by chords that are
// NOT radial, so the crossing sits 1–2° past the chain vertex — DRAWEXE's band faces span exactly
// the crossing meridians (132.42°/199.78°/0° there), and building the sections at the chain vertices
// instead leaves a contact sliver on the WRONG top face and an unclosable loop. This mirrors how
// OCCT's ChFi3d_Builder re-bounds each SurfData at the support-face transition (PerformIntersectionAtEnd
// family), not at the spine's own vertex.

// resolveJunctionCrossings relocates every interior junction's section to the station where the
// descending edge actually crosses the blend contact, adjusting the adjacent segments' feet, apex,
// and contact curves. Junctions whose crossing IS the chain station (the box-rim family) are left
// byte-identical. It also records which side (shared vs wall) each descending edge lies on.
func (st *tangentStripe) resolveJunctionCrossings(sp *blend.Spine, m *blend.Marcher) error {
	n := len(st.segs)
	st.cutOnShared = make([]bool, n)
	stations := make([]float64, n)
	touched := make([]bool, n)
	for j := 0; j < n; j++ {
		stations[j], _ = sp.EdgeSpineRange(j)
	}
	for j := 0; j < n; j++ {
		if st.junction[j] == nil {
			continue
		}
		if err := st.resolveOneCrossing(sp, m, j, stations, touched); err != nil {
			return err
		}
	}
	return st.rebuildTouchedContacts(sp, m, stations, touched)
}

// resolveOneCrossing finds junction j's crossing station and, when it moved off the chain vertex
// station, rewrites the two adjacent segments' shared feet and the junction apex.
func (st *tangentStripe) resolveOneCrossing(sp *blend.Spine, m *blend.Marcher, j int, stations []float64, touched []bool) error {
	st.cutOnShared[j] = edgeHasFace(st.down[j], st.shared)
	sChain := stations[j]
	sStar, err := st.crossingStation(sp, m, j, sChain)
	if err != nil {
		return err
	}
	if stdmath.Abs(sStar-sChain) <= float64(m.Res.Weld()) {
		return nil // the descending edge runs through the foot — the shipped box-rim behavior
	}
	prev := (j - 1 + len(st.segs)) % len(st.segs)
	if !sameBlendSurface(st.segs[prev].surf, st.segs[j].surf, 10*float64(m.Res.Weld())) {
		return fmt.Errorf("fillet: stripe junction %d crossing at spine %.6g needs the neighbour segments "+
			"to continue one blend surface, and they do not", j, sStar)
	}
	if err := st.applyCrossing(sp, m, j, prev, sStar); err != nil {
		return err
	}
	stations[j] = st.wrapStation(sStar, sp)
	touched[prev], touched[j] = true, true
	return nil
}

// crossingStation solves for the spine station whose SIDE foot (shared or wall, per where the
// descending edge lies) falls on the descending edge's curve. The chain station itself is accepted
// first — the common case needs no search and never evaluates a neighbour segment's supports.
func (st *tangentStripe) crossingStation(sp *blend.Spine, m *blend.Marcher, j int, sChain float64) (float64, error) {
	tol := float64(m.Res.Weld())
	gap := st.footGapToDown(sp, m, j, j, sChain)
	if gap <= tol {
		return sChain, nil
	}
	prev := (j - 1 + len(st.segs)) % len(st.segs)
	sPrev, gPrev := st.minimizeFootGap(sp, m, j, prev, sChain-0.45*st.segLen(sp, prev), sChain)
	sNext, gNext := st.minimizeFootGap(sp, m, j, j, sChain, sChain+0.45*st.segLen(sp, j))
	s, g := sNext, gNext
	if gPrev < gNext {
		s, g = sPrev, gPrev
	}
	if g > tol {
		return 0, fmt.Errorf("fillet: stripe junction %d: the descending edge misses the blend contact by %.3g", j, g)
	}
	return s, nil
}

// footGapToDown is the distance from the side foot at spine station s (supports read from segment
// seg) to the descending edge's curve. An unreachable station reads as +Inf.
func (st *tangentStripe) footGapToDown(sp *blend.Spine, m *blend.Marcher, j, seg int, s float64) float64 {
	c, ok := m.BallCentre(sp.PointAt(st.wrapStation(s, sp)), st.shared.Geometry(), st.segs[seg].wall.Geometry(), st.r)
	if !ok {
		return stdmath.Inf(1)
	}
	side := st.segs[seg].wall.Geometry()
	if st.cutOnShared[j] {
		side = st.shared.Geometry()
	}
	foot := surfaceFoot(side, c)
	d := st.down[j].Geometry()
	t, _ := geom.CurveParamAtPoint3(d, foot)
	lo, hi := d.Domain()
	return float64(d.PointAt(clampStation(t, lo, hi)).DistanceTo(foot))
}

// minimizeFootGap ternary-searches the foot-to-descending-edge gap over [a,b] (spine stations,
// wrap-aware through footGapToDown) down to parameter exhaustion.
func (st *tangentStripe) minimizeFootGap(sp *blend.Spine, m *blend.Marcher, j, seg int, a, b float64) (float64, float64) {
	f := func(s float64) float64 { return st.footGapToDown(sp, m, j, seg, s) }
	for it := 0; it < 200 && b-a > 1e-13*sp.Length(); it++ {
		m1, m2 := a+(b-a)/3, b-(b-a)/3
		if f(m1) <= f(m2) {
			b = m2
		} else {
			a = m1
		}
	}
	s := (a + b) / 2
	return s, f(s)
}

// applyCrossing rewrites the junction's section data at the crossing station: the shared/wall feet
// become the entry feet of segment j and the exit feet of segment prev, and the apex moves with them.
func (st *tangentStripe) applyCrossing(sp *blend.Spine, m *blend.Marcher, j, prev int, sStar float64) error {
	c, ok := m.BallCentre(sp.PointAt(st.wrapStation(sStar, sp)), st.shared.Geometry(), st.segs[j].wall.Geometry(), st.r)
	if !ok {
		return fmt.Errorf("fillet: stripe junction %d: no blend section at its crossing station %.6g", j, sStar)
	}
	fS1 := surfaceFoot(st.shared.Geometry(), c)
	fW := surfaceFoot(st.segs[j].wall.Geometry(), c)
	st.segs[j].topA, st.segs[j].wallA = fS1, fW
	st.segs[prev].topB, st.segs[prev].wallB = fS1, fW
	st.apex[j] = exposedApex(c, fS1, fW, st.r)
	return nil
}

// rebuildTouchedContacts rebuilds the contact curves of every segment whose bounding station moved,
// spanning its adjusted entry/exit feet through the mid-station foot (three-point arc, or a line
// when the feet are collinear — the same rule the marcher's contactCurve uses).
func (st *tangentStripe) rebuildTouchedContacts(sp *blend.Spine, m *blend.Marcher, stations []float64, touched []bool) error {
	n := len(st.segs)
	for i := 0; i < n; i++ {
		if !touched[i] {
			continue
		}
		exit := stations[(i+1)%n]
		if i == n-1 && !st.closed {
			exit = sp.Length()
		}
		mid := st.wrapMidStation(stations[i], exit, sp)
		c, ok := m.BallCentre(sp.PointAt(mid), st.shared.Geometry(), st.segs[i].wall.Geometry(), st.r)
		if !ok {
			return fmt.Errorf("fillet: stripe segment %d: no blend section at its adjusted mid-station", i)
		}
		st.segs[i].topContact = threePointContact(st.segs[i].topA, surfaceFoot(st.shared.Geometry(), c), st.segs[i].topB)
		st.segs[i].wallContact = threePointContact(st.segs[i].wallA, surfaceFoot(st.segs[i].wall.Geometry(), c), st.segs[i].wallB)
		if st.segs[i].topContact == nil || st.segs[i].wallContact == nil {
			return fmt.Errorf("fillet: stripe segment %d: cannot rebuild its adjusted contact curves", i)
		}
	}
	return nil
}

// segLen is segment i's spine length.
func (st *tangentStripe) segLen(sp *blend.Spine, i int) float64 {
	first, last := sp.EdgeSpineRange(i)
	return last - first
}

// wrapStation maps a station onto the spine's parameter range: modulo the length for a closed loop,
// clamped for an open run.
func (st *tangentStripe) wrapStation(s float64, sp *blend.Spine) float64 {
	if !st.closed {
		return clampStation(s, 0, sp.Length())
	}
	l := sp.Length()
	s = stdmath.Mod(s, l)
	if s < 0 {
		s += l
	}
	return s
}

// wrapMidStation is the midpoint of the forward span entry→exit, wrap-aware on a closed spine.
func (st *tangentStripe) wrapMidStation(entry, exit float64, sp *blend.Spine) float64 {
	if !st.closed || exit >= entry {
		return (entry + exit) / 2
	}
	return st.wrapStation(entry+(exit+sp.Length()-entry)/2, sp)
}

// clampStation constrains x to [lo,hi].
func clampStation(x, lo, hi float64) float64 {
	return stdmath.Min(stdmath.Max(x, lo), hi)
}

// surfaceFoot is the projection of p onto s (nearest point via the surface's parameter inverse).
func surfaceFoot(s geom.Surface, p math.Point3) math.Point3 {
	u, v := s.ParamAt(p)
	return s.PointAt(u, v)
}

// threePointContact is a contact curve through three feet: the circular arc, or a line segment when
// they are collinear (a plane support) — mirroring the marcher's contactCurve rule.
func threePointContact(a, m, b math.Point3) geom.Curve3 {
	area := a.VectorTo(m).Cross(a.VectorTo(b)).Length()
	if float64(area) < 1e-12 { // collinear feet ⇒ straight contact
		return geom.NewLineSegment(a, b)
	}
	arc, err := geom.Arc3dByThreePoints(a, m, b)
	if err != nil {
		return nil
	}
	return arc
}

// sameBlendSurface reports whether two blend surfaces are geometrically the same surface, by
// round-tripping sample points of one through the other's parameter inverse.
func sameBlendSurface(a, b geom.Surface, tol float64) bool {
	for _, uv := range [][2]float64{{0.1, 0.2}, {0.5, 0.6}, {0.9, 0.3}} {
		p := a.PointAt(uv[0], uv[1])
		if float64(surfaceFoot(b, p).DistanceTo(p)) > tol {
			return false
		}
	}
	return true
}
