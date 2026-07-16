// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A, Task 5.3 — the loop machinery behind retrimCurvedHost. It reads a host face's
// original outer loop, splits its edges where the arm contact rails land, and returns the surviving
// "far" path (the boundary away from the trihedral vertex) so the caller can splice the rails in.
// A circular split (the wall bottom-rim sub-arc) is re-emitted as an exact Arc3d edge, never a chord,
// so the assembled mesh cannot bulge or crack (tessellation-first).

// originalHostSegs reads the host's OUTER loop as an ordered ring of endSegs (each from→to carrying
// the edge's Arc3d curve oriented to the traversal, or nil for a straight edge). The corner bite is
// always cut from the outer boundary; any inner (hole) loop is untouched by the retrim and is carried
// through separately (see innerHostLoops) rather than read here.
func originalHostSegs(host *topo.Face) []endSeg {
	loop := outerHostLoop(host)
	if loop == nil {
		return nil
	}
	return segsFromLoop(loop)
}

// outerHostLoop returns the host face's outer boundary loop, found by topo.Loop.IsOuter() rather than
// assumed to sit at Loops()[0] — a face's inner (hole) loops can be stored at any index, and reading
// index 0 unconditionally would retrim a hole instead of the boundary on such a face (review finding,
// T5.3). Returns nil if the face carries no outer loop (malformed topology).
func outerHostLoop(host *topo.Face) *topo.Loop {
	for _, l := range host.Loops() {
		if l.IsOuter() {
			return l
		}
	}
	return nil
}

// innerHostLoops carries every INNER (hole) loop of the host through unchanged into the retrimmed
// result: the corner bite only re-clips the outer boundary (§B), so a host with a genuine hole must
// keep it verbatim, else the retrim would silently erase it. Kept for the far-runout / weld callers
// (fillet_curved_farrunout.go, fillet_curved_weld.go) that still only ever retrim the outer loop; on
// a single-outer-loop host it is exactly loopsExcept(host, that outer loop).
func innerHostLoops(host *topo.Face) []filletLoop {
	return loopsExcept(host, outerHostLoop(host))
}

// bittenLoop is the loop whose vertex nearest the corner-sphere centre c is globally minimal — the
// wire the trihedral corner actually bites, which may be the OUTER rim (B3) or an INNER notch window
// (N7's boolean-cut wall, where the corner lands on the hole loop, not the boundary). Generalizes the
// T5.3 "retrim the outer loop" assumption (derivation R.0). Rejects (false) when two loops tie for
// nearest within tol (an ambiguous symmetric part — do-no-harm) or the face carries no loops.
func bittenLoop(host *topo.Face, c math.Point3, tol float64) (*topo.Loop, bool) {
	var best *topo.Loop
	bestD, tie := stdmath.Inf(1), false
	for _, l := range host.Loops() {
		d := loopMinDistToCentre(l, c)
		switch {
		case d < bestD-tol:
			best, bestD, tie = l, d, false
		case stdmath.Abs(d-bestD) <= tol && l != best:
			tie = true
		}
	}
	if best == nil || tie {
		return nil, false // no loop, or an ambiguous nearest-loop tie: cannot pick the bitten wire
	}
	return best, true
}

// loopMinDistToCentre is the distance from c to the loop's nearest vertex.
func loopMinDistToCentre(l *topo.Loop, c math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, u := range l.EdgeUses() {
		if d := float64(useFromVertex(u).Point().DistanceTo(c)); d < best {
			best = d
		}
	}
	return best
}

// segsFromLoop turns one loop's edge uses into endSegs (generalizes originalHostSegs, which retrimmed
// only the outer loop, to any loop bittenLoop selects).
func segsFromLoop(l *topo.Loop) []endSeg {
	uses := l.EdgeUses()
	segs := make([]endSeg, 0, len(uses))
	for _, u := range uses {
		segs = append(segs, endSegFromUse(u))
	}
	return segs
}

// loopsExcept carries every loop of host except keep through unchanged (generalizes innerHostLoops:
// once the bitten loop can be inner, the carried-through set is "all others", not "all inner").
func loopsExcept(host *topo.Face, keep *topo.Loop) []filletLoop {
	var out []filletLoop
	for _, l := range host.Loops() {
		if l != keep {
			out = append(out, unchangedLoop(l))
		}
	}
	return out
}

// unchangedLoop copies a topo loop into a filletLoop with no modification, preserving each vertex's
// source identity and the source edge leaving it (mirrors transformLoop's "unchanged survivor" case,
// fillet_faces.go) so the assembler's welder keeps the loop's original topo provenance.
func unchangedLoop(l *topo.Loop) filletLoop {
	var fl filletLoop
	for _, u := range l.EdgeUses() {
		v := useFromVertex(u)
		fl.addID(v.Point(), survivorCurve(u), v.ID(), u.Edge().ID())
	}
	return fl
}

// endSegFromUse turns one edge use into an endSeg, carrying an Arc3d curve (with its midpoint, for
// re-orientation and splitting) or leaving the curve nil for a straight edge.
func endSegFromUse(u *topo.EdgeUse) endSeg {
	from, to := useFromVertex(u).Point(), useToVertex(u).Point()
	if arc, ok := survivorCurve(u).(geom.Arc3d); ok {
		return endSeg{from: from, to: to, curve: arc, mid: arc.PointAt(0.5), arc: true}
	}
	return endSeg{from: from, to: to}
}

// useToVertex returns the to-vertex of an edge use (honouring reversal) — the sibling of
// useFromVertex.
func useToVertex(u *topo.EdgeUse) *topo.Vertex {
	if u.Reversed() {
		return u.Edge().StartVertex()
	}
	return u.Edge().EndVertex()
}

// bittenVertex is the original loop vertex nearest the corner ball centre C — the trihedral corner
// the retrim bites away. The surviving far path is the loop side that does NOT contain it.
func bittenVertex(segs []endSeg, c math.Point3) math.Point3 {
	best, bestD := segs[0].from, segs[0].from.DistanceTo(c)
	for _, s := range segs {
		if d := s.from.DistanceTo(c); d < bestD {
			best, bestD = s.from, d
		}
	}
	return best
}

// farPathSegs returns the original-loop boundary from fromP to toP that avoids the bitten vertex v —
// the "far" side kept by the retrim, with the two rail landing points spliced in as new vertices.
func farPathSegs(segs []endSeg, fromP, toP, v math.Point3, tol float64) ([]endSeg, bool) {
	ring := insertSplits(segs, []math.Point3{fromP, toP}, tol)
	i, j := indexOfSegFrom(ring, fromP, tol), indexOfSegFrom(ring, toP, tol)
	if i < 0 || j < 0 {
		return nil, false // a rail landing point does not lie on the original loop — cannot close
	}
	if fwd := segsForward(ring, i, j); !pathHasVertex(fwd, v, tol) {
		return fwd, true
	}
	return reverseEndSegs(segsForward(ring, j, i)), true // the other way, oriented fromP→toP
}

// insertSplits rebuilds the ring so every point in pts that lies interior to an edge splits it.
func insertSplits(segs []endSeg, pts []math.Point3, tol float64) []endSeg {
	var out []endSeg
	for _, s := range segs {
		out = append(out, splitSeg(s, pts, tol)...)
	}
	return out
}

// splitSeg splits one edge at every pts point lying interior to it, ordered along the edge.
func splitSeg(s endSeg, pts []math.Point3, tol float64) []endSeg {
	on := onSegPoints(s, pts, tol)
	if len(on) == 0 {
		return []endSeg{s}
	}
	chain := append(append([]math.Point3{s.from}, on...), s.to)
	out := make([]endSeg, 0, len(chain)-1)
	for k := 0; k+1 < len(chain); k++ {
		out = append(out, subSeg(s, chain[k], chain[k+1]))
	}
	return out
}

// onSegPoints returns the pts strictly interior to edge s, ordered by their parameter along it.
func onSegPoints(s endSeg, pts []math.Point3, tol float64) []math.Point3 {
	type keyed struct {
		p   math.Point3
		key float64
	}
	var found []keyed
	for _, p := range pts {
		if key, ok := segParam(s, p, tol); ok {
			found = append(found, keyed{p, key})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].key < found[j].key })
	out := make([]math.Point3, len(found))
	for i, f := range found {
		out[i] = f.p
	}
	return out
}

// segParam returns p's fractional parameter along edge s (0..1), ok only when p lies strictly
// interior to the edge (endpoints excluded), on its line or arc within tol.
func segParam(s endSeg, p math.Point3, tol float64) (float64, bool) {
	if float64(p.DistanceTo(s.from)) <= tol || float64(p.DistanceTo(s.to)) <= tol {
		return 0, false
	}
	if !s.arc {
		return lineParam(s.from, s.to, p, tol)
	}
	return arcParam(s.curve.(geom.Arc3d), p, tol)
}

// lineParam is p's parameter t∈(0,1) on segment a→b, ok only when p lies on the segment within tol.
func lineParam(a, b, p math.Point3, tol float64) (float64, bool) {
	d := a.VectorTo(b)
	l2 := float64(d.Dot(d))
	if l2 == 0 {
		return 0, false
	}
	t := float64(a.VectorTo(p).Dot(d)) / l2
	if t <= 0 || t >= 1 {
		return 0, false
	}
	if float64(a.TranslateBy(d.Scale(math.Scalar(t))).DistanceTo(p)) > tol {
		return 0, false
	}
	return t, true
}

// arcParam is p's fractional parameter on arc within (0,1), ok only when p lies on the arc's circle
// (radius within tol) and inside its sweep.
func arcParam(arc geom.Arc3d, p math.Point3, tol float64) (float64, bool) {
	if stdmath.Abs(float64(p.DistanceTo(arc.Center))-arc.Radius) > tol {
		return 0, false
	}
	w := arc.Center.VectorTo(p)
	bin := arc.Normal.Cross(arc.RefDir)
	raw := stdmath.Atan2(float64(w.Dot(bin)), float64(w.Dot(arc.RefDir.AsVector())))
	delta := wrapToSweep(raw-arc.StartAngle, arc.SweepAngle)
	frac := delta / arc.SweepAngle
	if frac <= 0 || frac >= 1 {
		return 0, false
	}
	return frac, true
}

// wrapToSweep brings the angular offset delta into the interval swept by sweep: [0,2π) for a
// positive sweep, (−2π,0] for a negative one, so delta/sweep lands in [0,1) for an on-arc point.
func wrapToSweep(delta, sweep float64) float64 {
	for delta < 0 {
		delta += 2 * stdmath.Pi
	}
	for delta >= 2*stdmath.Pi {
		delta -= 2 * stdmath.Pi
	}
	if sweep < 0 {
		delta -= 2 * stdmath.Pi
	}
	return delta
}

// subSeg returns the portion of edge s between from and to (both on s): a straight sub-segment, or
// an exact Arc3d sub-arc through the circle's angular midpoint (never a chord).
func subSeg(s endSeg, from, to math.Point3) endSeg {
	if !s.arc {
		return endSeg{from: from, to: to}
	}
	arc := s.curve.(geom.Arc3d)
	mid := arcMidBetween(arc.Center, arc.Radius, from, to)
	sub, err := geom.Arc3dByThreePoints(from, mid, to)
	if err != nil {
		return endSeg{from: from, to: to}
	}
	return endSeg{from: from, to: to, curve: sub, mid: mid, arc: true}
}

// arcMidBetween is the point on the circle (centre, radius) on the shorter arc between from and to —
// the bisector direction unit(from̂+tô) scaled to the radius. Used to re-fit a split sub-arc.
func arcMidBetween(center math.Point3, radius float64, from, to math.Point3) math.Point3 {
	a, b := center.VectorTo(from), center.VectorTo(to)
	bis, err := math.UnitVector3FromVector(a.Add(b))
	if err != nil {
		return from.Midpoint(to) // near-antipodal: degrade to the chord midpoint (not hit in Slice A)
	}
	return center.TranslateBy(bis.AsVector().Scale(math.Scalar(radius)))
}

// indexOfSegFrom returns the ring index whose seg starts at p, or −1.
func indexOfSegFrom(ring []endSeg, p math.Point3, tol float64) int {
	for i, s := range ring {
		if float64(s.from.DistanceTo(p)) <= tol {
			return i
		}
	}
	return -1
}

// segsForward collects the ring segments from index i up to (excluding) index j, cyclically.
func segsForward(ring []endSeg, i, j int) []endSeg {
	n := len(ring)
	var out []endSeg
	for k := i; k != j; k = (k + 1) % n {
		out = append(out, ring[k])
	}
	return out
}

// pathHasVertex reports whether v is one of the path's junctions (any seg endpoint within tol).
func pathHasVertex(path []endSeg, v math.Point3, tol float64) bool {
	for _, s := range path {
		if float64(s.from.DistanceTo(v)) <= tol || float64(s.to.DistanceTo(v)) <= tol {
			return true
		}
	}
	return false
}
