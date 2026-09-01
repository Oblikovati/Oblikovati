// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Whether a CLOSED elliptic-rim canal band actually FITS on the two host faces that must carry it.
// This is the do-no-harm floor that keeps the T5/U2 shapes out: their fillet foot ring runs OFF the
// plate the boss stands on, which OCCT answers with a CLIPPED, multi-piece band (5 pieces on T5, 8 on
// U2, with the host plane face split too) — a materially different construction this slice does not
// build. Failing these gates makes ellipticClosedRimArmEdge return handled=false, so the edge falls
// through to the byte-identical curvedAdjacentError refusal rather than shipping a band that pokes
// through a neighbour face.

// ellipticRimFootsFit gates the band on the two faces that must CARRY it: every plane foot must land
// on the cap face's real trimmed region (outer loop, clear of holes) and every wall foot must stay
// inside the wall's own axial extent. A band that spills off the cap (T5/U2: the fillet runs out over
// the plate's edge, which OCCT answers with a clipped, multi-piece band) or overruns the wall is a
// DIFFERENT construction — decline it honestly instead of shipping a band poking through a neighbour.
func ellipticRimFootsFit(spine ellipticRimSpine, st ellipticRimStations, e *topo.Edge, pl geom.Plane, wallF, capF *topo.Face) bool {
	if !capFaceContainsFeet(capF, pl, st.planeFeet) {
		return false
	}
	return ellipticRimWallSpanFits(spine, st, e, wallF)
}

// capFaceContainsFeet reports whether EVERY plane foot lands inside the cap face's outer loop and
// clear of its holes. It cannot reuse planeFootOnTrimmedFace: that projects the loop through
// loopUVPolygon, which emits one point per edge (plus an arc midpoint), so a cap bounded by a SINGLE
// closed ELLIPSE degenerates to a one-point polygon and every test fails. Here each loop is sampled
// densely along its real curves instead, which is what a closed conic boundary needs.
func capFaceContainsFeet(capF *topo.Face, pl geom.Plane, feet []math.Point3) bool {
	uv := func(p math.Point3) math.Point2 {
		d := pl.Origin.VectorTo(p)
		return math.P2(d.Dot(pl.UAxis.AsVector()), d.Dot(pl.VAxis.AsVector()))
	}
	polys := make([][]math.Point2, 0, len(capF.Loops()))
	outer := make([]bool, 0, len(capF.Loops()))
	for _, l := range capF.Loops() {
		polys = append(polys, sampledLoopUVPolygon(l, uv))
		outer = append(outer, l.IsOuter())
	}
	for _, f := range feet {
		if !uvPointInTrimmedRegion(uv(f), polys, outer) {
			return false
		}
	}
	return true
}

// uvPointInTrimmedRegion reports whether q is inside some OUTER loop and inside no hole.
func uvPointInTrimmedRegion(q math.Point2, polys [][]math.Point2, outer []bool) bool {
	in := false
	for i, poly := range polys {
		if !probe.PointInLoop2D(q, poly) {
			continue
		}
		if !outer[i] {
			return false // inside a hole
		}
		in = true
	}
	return in
}

// ellipticRimLoopSamples is the per-edge sampling density of sampledLoopUVPolygon. A closed conic
// boundary is one edge, so the count must resolve the WHOLE curve, not a short span; 64 chords hold an
// ellipse to ~1e-3 of its semi-minor axis, far finer than the fillet offsets being classified.
const ellipticRimLoopSamples = 64

// sampledLoopUVPolygon projects a loop into the plane's (u,v) frame by sampling each edge's own curve
// across its parameter domain — the dense counterpart of loopUVPolygon, for loops whose edges are
// closed conics rather than short line/arc segments.
func sampledLoopUVPolygon(l *topo.Loop, uv func(math.Point3) math.Point2) []math.Point2 {
	var poly []math.Point2
	for _, u := range l.EdgeUses() {
		c := u.Edge().Geometry()
		lo, hi := c.Domain()
		for i := range ellipticRimLoopSamples {
			t := lo + (hi-lo)*float64(i)/float64(ellipticRimLoopSamples)
			if u.Reversed() {
				t = hi - (hi-lo)*float64(i)/float64(ellipticRimLoopSamples)
			}
			poly = append(poly, uv(c.PointAt(t)))
		}
	}
	return poly
}

// ellipticRimWallSpanFits reports whether every wall foot stays between the rim and the wall's FAR
// boundary. The far boundary is the wall's other CLOSED edge; its own far face must be planar (the
// oblique prism's opposite cap, the pipe's far cap), which lets the far limit be evaluated per station
// in closed form — v_far(u) = (c₂ − n̂₂·S(u,0)) / (n̂₂·â) — rather than assumed constant. A wall with no
// planar far boundary declines: this slice does not model a fillet running into a curved neighbour.
func ellipticRimWallSpanFits(spine ellipticRimSpine, st ellipticRimStations, e *topo.Edge, wallF *topo.Face) bool {
	n2, c2, ok := ellipticWallFarPlane(e, wallF)
	if !ok {
		return false
	}
	den2 := float64(n2.AsVector().Dot(spine.ec.AxisDir.AsVector()))
	if stdmath.Abs(den2) < ellipticRimAxisTiltTol {
		return false
	}
	origin := math.P3(0, 0, 0)
	for _, f := range st.wallFeet {
		u, v := spine.ec.ParamAt(f)
		s0 := float64(origin.VectorTo(spine.ec.PointAt(u, 0)).Dot(spine.nPl.AsVector()))
		vRim := (spine.cPl - s0) / spine.den
		vFar := (c2 - float64(origin.VectorTo(spine.ec.PointAt(u, 0)).Dot(n2.AsVector()))) / den2
		if (v-vRim)*(vFar-vRim) < 0 || stdmath.Abs(v-vRim) > stdmath.Abs(vFar-vRim) {
			return false // the foot ran past the rim's own side, or past the wall's far boundary
		}
	}
	return true
}

// ellipticWallFarPlane returns the unit normal and signed offset of the plane bounding the wall at its
// FAR end: the plane of the face across the wall's other CLOSED edge. ok=false when the wall has no
// such edge or its far neighbour is not planar.
func ellipticWallFarPlane(e *topo.Edge, wallF *topo.Face) (math.UnitVector3, float64, bool) {
	for _, we := range wallF.Edges() {
		if we == e || we.StartVertex() != we.EndVertex() {
			continue
		}
		for _, f := range we.Faces() {
			p, isPlane := f.Geometry().(geom.Plane)
			if !isPlane {
				continue
			}
			n := p.Normal().AsUnit()
			return n, float64(math.P3(0, 0, 0).VectorTo(p.Origin).Dot(n.AsVector())), true
		}
	}
	return math.UnitVector3{}, 0, false
}
