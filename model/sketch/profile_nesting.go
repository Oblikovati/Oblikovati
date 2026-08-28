// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Profile extraction — CONTAINMENT / NESTING classification (M48 #2242 split of profile.go). Groups the
// traced loops into profiles by containment (which loop is an outer boundary, which are holes inside it),
// merges abutting loops that share a region, and the point-in-polygon / loop-containment predicates. The
// loop tracing lives in profile_trace.go.

// groupRegions builds profiles from closed loops by even–odd nesting: a loop contained in an even
// number of others is an outer boundary; loops one level deeper inside it are its holes.
//
// As an exception, an odd-nested loop the user actually DREW also bounds its own selectable region
// (the area inside it), so nested loops yield BOTH the annulus AND the inner face, matching how
// Inventor lets you select either. A circle inside a rectangle offers the disk (#1526), and a
// rounded-rectangle (or any polygon) inside another offers its inner face (#2165 — the offset of a
// rounded profile left the inner region unselectable). Synthetic text-glyph counters carry NO
// entities (they are bare polygons from TextProfiles), so they stay holes only and the glyph region
// detection is unchanged; the grill boolean uses ClosedLoops (#863) and is likewise untouched.
//
// The extra inner regions are APPENDED after every outer region, so a loop the exception newly exposes
// never shifts an outer region's index — index-based extrude/loft selections in older documents keep
// pointing at the same face (newer ones resolve by seed point, #region-seed).
func groupRegions(loops []Loop) []*Profile {
	depth := make([]int, len(loops))
	for i := range loops {
		for j := range loops {
			if i != j && containsLoop(loops[j], loops[i]) {
				depth[i]++
			}
		}
	}
	var profiles []*Profile
	for i, l := range loops {
		if depth[i]%2 == 0 {
			profiles = append(profiles, &Profile{outer: l, inner: holesOf(i, loops, depth)})
		}
	}
	abutting := abuttingLoops(loops)
	for i, l := range loops {
		if depth[i]%2 == 1 && loopBoundsOwnRegion(l) && !abutting[i] {
			profiles = append(profiles, &Profile{outer: l, inner: holesOf(i, loops, depth)})
		}
	}
	return profiles
}

// loopBoundsOwnRegion reports whether an odd-nested (hole) loop ALSO bounds a face the user can
// select in its own right. A loop the user drew — one that carries sketch entities, whether a single
// circle/ellipse or a multi-entity polygon/rounded-rectangle chain — does; a synthetic text-glyph
// counter (a bare polygon with no entities, produced by TextProfiles) does not, so the hole in an
// 'O'/'A'/'B' stays a hole. See groupRegions (#1526, #2165).
func loopBoundsOwnRegion(l Loop) bool {
	return len(l.entities) >= 1
}

// abuttingLoops flags each loop that shares an edge with another loop — the minimal cells of one
// SUBDIVIDED island (a grid of crossing grill bars carves an interior into many abutting cells). Such
// a cell is NOT a region the user drew, so it stays a hole (merged into one outline by holesOf, #863);
// only a STANDALONE inner loop — one sharing no edge with any sibling, i.e. a genuinely nested circle,
// rectangle or rounded rectangle — bounds its own selectable region (#2165). Edges are welded so two
// cells tracing the same boundary points register as shared.
func abuttingLoops(loops []Loop) []bool {
	edgeUses := map[[2]int]int{}
	loopEdges := make([][][2]int, len(loops))
	w := newLoopWelder()
	for li, l := range loops {
		loopEdges[li] = weldedLoopEdges(w, l)
		for _, key := range loopEdges[li] {
			edgeUses[key]++
		}
	}
	abut := make([]bool, len(loops))
	for li := range loops {
		for _, key := range loopEdges[li] {
			if edgeUses[key] > 1 {
				abut[li] = true
				break
			}
		}
	}
	return abut
}

// holesOf returns the holes of outer (index oi): the loops one nesting level inside it. A
// disconnected island whose interior is itself subdivided (e.g. a grid of crossing bars inside
// the boundary) arrives as many abutting minimal cells at that level; they are merged into the
// island's outline loop(s) so the profile carries clean, non-abutting holes rather than a tiling
// of overlapping cells (#863 — abutting hole loops are degenerate downstream).
func holesOf(oi int, loops []Loop, depth []int) []Loop {
	var holes []Loop
	for j, l := range loops {
		if j != oi && depth[j] == depth[oi]+1 && containsLoop(loops[oi], l) {
			holes = append(holes, l)
		}
	}
	return mergeAbuttingLoops(holes)
}

// mergeAbuttingLoops fuses loops that share edges (the minimal cells of one connected island)
// into their boundary outline: a directed edge traversed by two abutting cells appears in both
// directions and cancels; the surviving edges chain into the outline. Disjoint loops (separate
// islands) share no edges and pass through unchanged.
func mergeAbuttingLoops(loops []Loop) []Loop {
	if len(loops) <= 1 {
		return loops
	}
	w := newLoopWelder()
	dir := map[[2]int]int{}
	for _, l := range loops {
		idx := make([]int, len(l.polygon))
		for i, p := range l.polygon {
			idx[i] = w.add(p)
		}
		for i := range idx {
			if a, b := idx[i], idx[(i+1)%len(idx)]; a != b {
				dir[[2]int{a, b}]++
			}
		}
	}
	rings := chainLoopBoundary(boundaryDirEdges(dir), w.points)
	if len(rings) == 0 {
		return loops // chaining failed (unexpected); keep the originals rather than drop holes
	}
	out := make([]Loop, len(rings))
	for i, r := range rings {
		out[i] = Loop{polygon: r, closed: true}
	}
	return out
}

// containsLoop reports whether outer contains inner: every vertex of inner lies
// inside outer's polygon. (Testing all vertices, not just the centroid, avoids a
// false positive when a hole is centered on the region's centroid.)
func containsLoop(outer, inner Loop) bool {
	if len(inner.polygon) == 0 {
		return false
	}
	for _, v := range inner.polygon {
		if !pointInPolygon(v, outer.polygon) {
			return false
		}
	}
	return true
}

// pointInPolygon is the even–odd ray-casting test.
func pointInPolygon(p math.Point2, poly []math.Point2) bool {
	inside := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		yi, yj := poly[i].Y, poly[j].Y
		if (yi > p.Y) != (yj > p.Y) {
			x := poly[i].X + (p.Y-yi)/(yj-yi)*(poly[j].X-poly[i].X)
			if p.X < x {
				inside = !inside
			}
		}
	}
	return inside
}
