// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A, Task 5.4 — the HOST side of the nine-face weld. Each original body face is routed to its
// treatment: a face carrying two corner arms (their host-tangent point on it) is the circular corner
// retrim (T5.3 retrimCurvedHost, e.g. wall / cap / radial); a face bitten only by arms' FAR cross-
// sections is the far-runout retrim (this file — the y=0 radial cut two arms run out to, and the bottom
// cap the through-arm exits, §B.5); an untouched face passes through transformFace verbatim. The far-
// runout bite reuses the arm's own far cross-section arc (armRails.far) so the two sides weld watertight.

// curvedHostFaces retrims every body face for the trihedral corner: corner hosts via the circular
// retrim (T5.3), far-runout hosts via spliceCornerBite with the arms' far cross-section arcs, and any
// face neither touches passes through unchanged. Declines with a diagnostic reason on any retrim failure.
func curvedHostFaces(body *topo.Body, arms []edgeFillet, bundles []armRails, w cornerWeld, res Resolution) ([]filletFace, string) {
	corner := cornerHostSet(arms)
	tol := res.Weld() * w.radius
	out := make([]filletFace, 0, len(body.Faces()))
	for _, f := range body.Faces() {
		ff, reason := curvedHostFace(f, corner, bundles, w, res, tol)
		if reason != "" {
			return nil, reason
		}
		out = append(out, ff)
	}
	return out, ""
}

// curvedHostFace routes one host face to its treatment (corner retrim / far-runout retrim / pass-through)
// and returns its rebuilt face, or a diagnostic reason on a retrim decline.
func curvedHostFace(f *topo.Face, corner map[*topo.Face]bool, bundles []armRails, w cornerWeld, res Resolution, tol float64) (filletFace, string) {
	if corner[f] {
		ff, ok := retrimCurvedHost(f, w, res)
		if !ok {
			return filletFace{}, fmt.Sprintf("corner host retrim declined (%T)", f.Geometry())
		}
		return ff, ""
	}
	bites := farArcsBiting(f, bundles, tol)
	if len(bites) == 0 {
		return transformFace(f, nil, nil, nil, nil), "" // untouched by the corner — carried through verbatim
	}
	ff, ok := farRunoutFace(f, bites, tol)
	if !ok {
		return filletFace{}, fmt.Sprintf("far-runout retrim declined (%T, %d bites)", f.Geometry(), len(bites))
	}
	return ff, ""
}

// cornerHostSet is the set of host faces the corner arms roll on (each arm's two hosts ef.a/ef.b) — the
// faces whose corner bite is the circular T5.3 retrim.
func cornerHostSet(arms []edgeFillet) map[*topo.Face]bool {
	set := make(map[*topo.Face]bool, 2*len(arms))
	for _, ef := range arms {
		set[ef.a] = true
		set[ef.b] = true
	}
	return set
}

// farArcsBiting returns the arms' far cross-section arcs whose BOTH endpoints lie on host f's surface —
// the arms that run out onto f (§B.5: the y=0 cut receives two, the through-arm's exit cap one).
func farArcsBiting(f *topo.Face, bundles []armRails, tol float64) []endSeg {
	var out []endSeg
	surf := f.Geometry()
	for _, b := range bundles {
		if onHostSurface(surf, b.far.from, tol) && onHostSurface(surf, b.far.to, tol) {
			out = append(out, b.far)
		}
	}
	return out
}

// farRunoutFace retrims a far-runout host: each arm far arc bites off the rectangle/quarter-disk corner
// it spans, spliced in as an exact arc (never a chord). Any inner (hole) loop is carried through
// unchanged (mirrors retrimCurvedHost). Declines when a bite cannot be spliced onto the outer loop.
func farRunoutFace(host *topo.Face, bites []endSeg, tol float64) (filletFace, bool) {
	segs := originalHostSegs(host)
	if len(segs) < 3 {
		return filletFace{}, false
	}
	for _, bite := range bites {
		spliced, ok := spliceCornerBite(segs, bite, tol)
		if !ok {
			return filletFace{}, false
		}
		segs = spliced
	}
	loops := append([]filletLoop{loopFromSegs(segs)}, innerHostLoops(host)...)
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}

// spliceCornerBite replaces the loop corner an arm far arc spans with the arc itself: it splits the loop
// at the arc's two endpoints, then removes the SMALLER-AREA span between them (the bitten corner) and
// closes the kept span with the arc, oriented to keep the ring continuous. Enclosed AREA — not segment
// COUNT — is the principled criterion: a fillet run-out bites a SMALL corner off the host, never the
// bulk of the face, so the removed region is the one the bite arc encloses the smaller area with. The
// prior len(fwd)<=len(bwd) was only a proxy — a single long edge, or a rim already split into several
// sub-edges, gives a span more/fewer segments than its area warrants, and would splice the wrong side
// (a wrong pick still cracks the shell to the do-no-harm floor, never a silent wrong solid).
func spliceCornerBite(segs []endSeg, bite endSeg, tol float64) ([]endSeg, bool) {
	ring := insertSplits(segs, []math.Point3{bite.from, bite.to}, tol)
	i, j := indexOfSegFrom(ring, bite.from, tol), indexOfSegFrom(ring, bite.to, tol)
	if i < 0 || j < 0 {
		return nil, false // a bite endpoint does not lie on the original loop — cannot splice
	}
	fwd := segsForward(ring, i, j) // bite.from → bite.to
	bwd := segsForward(ring, j, i) // bite.to   → bite.from
	if cornerBiteArea(fwd, bite) <= cornerBiteArea(bwd, bite) {
		return append(bwd, bite), true // remove the smaller fwd corner; close bite.from→bite.to
	}
	return append(fwd, reverseEndSegs([]endSeg{bite})...), true // remove the smaller bwd corner
}

// cornerBiteArea is the area of the region a loop span encloses when closed by the bite arc — the corner
// spliceCornerBite removes if it drops that span. The span (whose two endpoints ARE the bite's endpoints,
// in either order) and the closing arc bulge are sampled into a point ring — each arc contributes its
// true curvature, not a chord — and the area is the Newell area-vector magnitude, exact on the planar
// far-runout host. This lets the splice pick the small bitten corner geometrically, independent of how
// many edges each span was cut into.
func cornerBiteArea(span []endSeg, bite endSeg) float64 {
	if len(span) == 0 {
		return 0
	}
	ring := segPolyline(span)                       // span[0].from … (approaching span's last vertex)
	end := span[len(span)-1].to                     // the span's far bite endpoint
	ring = append(ring, end)                        // close the span's own last vertex explicitly
	ring = append(ring, biteArcBulge(bite, end)...) // the arc bulge from `end` back toward ring[0]
	return float64(newellNormal(ring).Length()) / 2
}

// segPolyline expands an ordered, endpoint-connected endSeg path into a 3D point list, sampling each arc
// edge with biteArcSamples chords so a curved edge contributes its true shape to a downstream Newell
// area/winding computation. It emits each seg's `from` (plus that seg's arc interior); the caller appends
// the path's final `to` when a closed ring is wanted.
func segPolyline(segs []endSeg) []math.Point3 {
	var pts []math.Point3
	for _, s := range segs {
		pts = append(pts, s.from)
		if arc, ok := s.curve.(geom.Arc3d); ok && s.arc {
			pts = append(pts, arcInteriorPoints(arc, s.from, biteArcSamples)...)
		}
	}
	return pts
}

// biteArcBulge samples the bite arc's interior points ordered from `from` (one bite endpoint) toward the
// other, so they fill the arc's curvature between the span's two ends when appended to the area ring. A
// straight bite (no arc) contributes no bulge — its chord already closes the ring.
func biteArcBulge(bite endSeg, from math.Point3) []math.Point3 {
	arc, ok := bite.curve.(geom.Arc3d)
	if !ok || !bite.arc {
		return nil
	}
	return arcInteriorPoints(arc, from, biteArcSamples)
}

// biteArcSamples is the per-arc chord count used to develop a bite/rim arc into the area polyline. A
// far-runout bite spans well under a semicircle, so 24 chords put the developed area within ~0.05% of
// the true segment — far tighter than the corner-vs-face area gap the splice discriminates.
const biteArcSamples = 24

// arcInteriorPoints samples an arc edge's interior points (endpoints excluded), oriented from the arc's
// end nearest `from` toward the other end, so the samples follow the edge's traversal direction.
func arcInteriorPoints(arc geom.Arc3d, from math.Point3, k int) []math.Point3 {
	fwd := arc.PointAt(0).DistanceTo(from) <= arc.PointAt(1).DistanceTo(from)
	pts := make([]math.Point3, 0, k-1)
	for i := 1; i < k; i++ {
		t := float64(i) / float64(k)
		if !fwd {
			t = 1 - t
		}
		pts = append(pts, arc.PointAt(t))
	}
	return pts
}
