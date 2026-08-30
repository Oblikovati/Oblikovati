// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"math"
	"sort"
)

// Sketch decoding — ASSEMBLY and REFERENCE RESOLUTION (M48 #2229 split of sketch.go). Turning the raw
// records collectItems produced (sketch_records.go) into a Sketch: the clean path (resolveByRefs)
// reconstructs every line and circle centre exactly from the point references each curve carries, so
// any polygon and off-origin circle rebuild faithfully; the fallback (assembleSketch) uses inline
// circle centres and a convex-ring heuristic when the references don't rank-align. assembleCluster is
// the entry that tries the exact path first and falls back.

// assembleCluster reconstructs one sketch from its entities. When the cluster is clean
// (every curve's point references resolve), lines and circle centres are emitted from
// their exact referenced points — reconstructing non-convex loops and off-origin circles
// faithfully. Otherwise it falls back to inline circle centres and the convex-ring
// heuristic (good for simple convex profiles and origin circles only).
func assembleCluster(items []sketchItem) Sketch {
	var pts []idPoint
	var circs []circleEnt
	var lines []lineRefs
	var arcs []arcEnt
	var ells []ellipseEnt
	for _, it := range items {
		switch {
		case it.pt != nil:
			pts = append(pts, *it.pt)
		case it.circle != nil:
			circs = append(circs, *it.circle)
		case it.line != nil:
			lines = append(lines, *it.line)
		case it.arc != nil:
			arcs = append(arcs, *it.arc)
		case it.ellipse != nil:
			ells = append(ells, *it.ellipse)
		}
	}
	if s, ok := resolveByRefs(pts, circs, lines, arcs, ells); ok {
		s.Resolved = true
		return s
	}
	// resolveByRefs failed (its reference and point counts disagree). For a pure-line cluster,
	// try reconstructing the profile from each line's inline infinite-line geometry, which needs no
	// reference resolution — accepted only when every corner lands on a collected point (self-
	// validated), so it never emits guessed geometry. Falls through to the convex-ring heuristic.
	if s, ok := reconstructInlineLoop(items); ok {
		return s
	}
	inline := make([]Circle, len(circs))
	for i, ce := range circs {
		inline[i] = Circle{Center: ce.inline, Radius: ce.radius}
	}
	coords := make([]Point2D, len(pts))
	for i, ip := range pts {
		coords[i] = ip.p
	}
	return assembleSketch(coords, inline, len(lines))
}

// hasOriginPoint reports whether any decoded point sits at the sketch origin (0,0).
func hasOriginPoint(pts []idPoint) bool {
	for _, p := range pts {
		if absf(p.p.X) < 1e-9 && absf(p.p.Y) < 1e-9 {
			return true
		}
	}
	return false
}

// resolveByRefs reconstructs lines and circle centres from the point references every
// sketch curve carries. Rank-aligning the sorted distinct references to the sorted
// point-entity ids (both assigned in creation order, so equal rank == same point) recovers
// each point's coordinate exactly — so any polygon (convex or not) and any off-origin
// circle rebuild faithfully. Returns ok=false (caller falls back) unless the cluster is
// clean: at least one curve, every line paired, every circle referenced, and #distinct
// references == #points.
func resolveByRefs(pts []idPoint, circs []circleEnt, lines []lineRefs, arcs []arcEnt, ells []ellipseEnt) (Sketch, bool) {
	if len(lines) == 0 && len(circs) == 0 && len(arcs) == 0 && len(ells) == 0 {
		return Sketch{}, false
	}
	refSet := map[uint32]struct{}{}
	for _, l := range lines {
		if !l.paired {
			return Sketch{}, false
		}
		refSet[l.a] = struct{}{}
		refSet[l.b] = struct{}{}
	}
	for _, c := range circs {
		if !c.hasRef {
			return Sketch{}, false
		}
		refSet[c.ref] = struct{}{}
	}
	for _, a := range arcs {
		refSet[a.start] = struct{}{}
		refSet[a.end] = struct{}{}
		refSet[a.center] = struct{}{}
	}
	for _, e := range ells {
		refSet[e.ref] = struct{}{}
	}
	// The sketch origin (0,0) is an implicit centre point that geometry can reference but that
	// carries no coordinate node (so it never appears in pts). When exactly one referenced point
	// is otherwise unaccounted for and none of the decoded points is at the origin, synthesise it
	// with the lowest id — Inventor creates the origin first, so it rank-aligns to the smallest
	// reference. This makes a revolve profile that touches its centreline (a shaft) resolve
	// exactly instead of collapsing to the convex fallback.
	if len(refSet) == len(pts)+1 && !hasOriginPoint(pts) {
		pts = append([]idPoint{{id: 0, p: Point2D{0, 0}}}, pts...)
	}
	if len(refSet) != len(pts) {
		return Sketch{}, false
	}
	refs := make([]uint32, 0, len(refSet))
	for r := range refSet {
		refs = append(refs, r)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i] < refs[j] })
	verts := append([]idPoint(nil), pts...)
	sort.Slice(verts, func(i, j int) bool { return verts[i].id < verts[j].id })
	coord := make(map[uint32]Point2D, len(refs))
	for i, r := range refs {
		coord[r] = verts[i].p
	}
	var s Sketch
	for _, c := range circs {
		s.Circles = append(s.Circles, Circle{Center: coord[c.ref], Radius: c.radius})
	}
	for _, l := range lines {
		s.Lines = append(s.Lines, Line{A: coord[l.a], B: coord[l.b]})
	}
	for _, a := range arcs {
		s.Arcs = append(s.Arcs, Arc{Center: coord[a.center], Radius: a.radius, Start: coord[a.start], End: coord[a.end]})
	}
	for _, e := range ells {
		s.Ellipses = append(s.Ellipses, Ellipse{
			Center: coord[e.ref], MajorAxis: Point2D{e.axisX, e.axisY}, MajorR: e.majorR, MinorR: e.minorR,
		})
	}
	return s, true
}

// assembleSketch is the fallback assembly when endpoint refs don't resolve cleanly: a
// circle owns its centre point, a single line owns its two endpoints, a convex ring is
// ordered by angle, and whatever remains is emitted as standalone sketch points.
func assembleSketch(pts []Point2D, circles []Circle, lineCurves int) Sketch {
	remaining := make([]Point2D, 0, len(pts))
	for _, p := range pts {
		if !nearAnyCenter(p, circles) {
			remaining = append(remaining, p)
		}
	}
	s := Sketch{Circles: circles}
	switch {
	case lineCurves == 0:
		s.Points = remaining // genuine standalone points
	case lineCurves == 1 && len(remaining) == 2:
		s.Lines = []Line{{A: remaining[0], B: remaining[1]}}
	case len(remaining) >= 3 && lineCurves >= len(remaining):
		// A closed polygon we could not resolve exactly: order the N corners into a simple
		// convex ring and connect consecutive corners. Correct only for convex profiles.
		s.Lines = closeLoop(orderConvex(remaining))
	default:
		// Ambiguous endpoint set — omit rather than emit misleading geometry.
	}
	return s
}

// orderConvex sorts points into convex (CCW) order around their centroid.
func orderConvex(pts []Point2D) []Point2D {
	var cx, cy float64
	for _, p := range pts {
		cx, cy = cx+p.X, cy+p.Y
	}
	n := float64(len(pts))
	cx, cy = cx/n, cy/n
	out := append([]Point2D(nil), pts...)
	sort.SliceStable(out, func(i, j int) bool {
		return math.Atan2(out[i].Y-cy, out[i].X-cx) < math.Atan2(out[j].Y-cy, out[j].X-cx)
	})
	return out
}

func closeLoop(ring []Point2D) []Line {
	lines := make([]Line, len(ring))
	for i := range ring {
		lines[i] = Line{A: ring[i], B: ring[(i+1)%len(ring)]}
	}
	return lines
}

func nearAnyCenter(p Point2D, circles []Circle) bool {
	for _, c := range circles {
		if absf(p.X-c.Center.X) < 1e-9 && absf(p.Y-c.Center.Y) < 1e-9 {
			return true
		}
	}
	return false
}
