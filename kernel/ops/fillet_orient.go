// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// orientFilletShell makes every result face's loop winding mutually consistent BEFORE the edge
// catalog builds topology: it 2-colours the shared-edge graph and reverses the loops of every face
// whose winding conflicts with its neighbours (B2). The fillet rebuild seeds cylinder/sphere blend
// patches by geometry alone (fillet_faces.go), so a concave corner can emit a patch wound the same
// way as its neighbour — two co-parallel co-edges that ops.Validate rejects. This is the topological
// analogue of the mesh-level consistentOutwardFlips pass and, like it, is a NO-OP on an already-
// consistent shell (the 34 baseline-passing corpus cases have zero conflicts, so they are untouched).
//
// Only RELATIVE consistency matters here: ops.Validate compares the two co-edges of each manifold
// edge (uses[0].Reversed != uses[1].Reversed) and never reads the surface normal or Face.reversed,
// so a uniform global flip is invisible downstream. Tessellation is winding-independent for every
// mesher EXCEPT the sphere-patch mesher (spherePatchMesh), which reads a sub-hemisphere host sphere
// loop's ABSOLUTE winding to pick which region it bounds — so a genuine host sphere (D5/E4) must be
// seeded here wound CCW-seen-from-outside; orientForSphereHost (fillet_curved_sphere_orient.go) does
// that BEFORE calling this pass. Otherwise triangles are wound to the surface normal, then
// Face.reversed is applied. A non-orientable shell (an odd cycle in the graph) is left as built and
// fails Validate loud rather than being laundered.
func orientFilletShell(faces []filletFace, rings [][][]int, classes map[[2]int]int) {
	uses := buildEdgeSides(faces, rings, classes)
	adj := filletAdjacency(len(faces), uses)
	flip := colorFilletFaces(adj)
	applyFilletFlips(faces, rings, flip)
}

// edgeSide is one face's directed traversal a→b of a shared edge — the raw material for detecting
// whether two faces cross their common edge in opposite (consistent) or the same (conflicting) sense.
type edgeSide struct {
	face int
	a, b int
}

// buildEdgeSides groups every loop segment by its reconstructed-edge key (welded pair + tangent-seam
// class, the SAME key edgeCatalog builds), recording which face used it and in which direction.
func buildEdgeSides(faces []filletFace, rings [][][]int, classes map[[2]int]int) map[seamEdgeKey][]edgeSide {
	sides := map[seamEdgeKey][]edgeSide{}
	for fi := range faces {
		for li, ring := range rings[fi] {
			ids := faces[fi].loops[li].srcE
			for k := 0; k < len(ring); k++ {
				a, b := ring[k], ring[(k+1)%len(ring)]
				key := seamEdgeKey{canon2(a, b), edgeClassOf(a, b, srcIDAt(ids, k), classes)}
				sides[key] = append(sides[key], edgeSide{fi, a, b})
			}
		}
	}
	return sides
}

// faceLink is an orientation constraint between two faces sharing a manifold edge: conflict=true when
// they currently traverse it the SAME way (so one must flip); false when already antiparallel.
type faceLink struct {
	other    int
	conflict bool
}

// filletAdjacency turns the edge-side groups into a per-face constraint graph. Only manifold edges
// (used exactly twice by two DISTINCT faces) constrain orientation: a boundary edge (one use), a
// non-manifold edge (>2), or a face's own closed seam (both uses on one face) carry no cross-face
// winding constraint and are skipped.
//
// A CLOSED edge (a==b, a full-circle rim welded to one vertex) is also skipped: its two co-edges
// trivially share a==a, so the winding conflict test can't read a sense from them — the closed-seam
// rule in edgeCatalog.use already makes the second co-edge antiparallel (B1). Including it would fire
// a false conflict on every rim and spuriously flip torus/cone bands, corrupting their full-circle
// arc through reverseFilletLoop's three-point re-fit (from==to is degenerate). This is what keeps the
// pass a true no-op on the already-consistent P2 torus bands.
func filletAdjacency(nFaces int, sides map[seamEdgeKey][]edgeSide) [][]faceLink {
	adj := make([][]faceLink, nFaces)
	for _, s := range sides {
		if len(s) != 2 || s[0].face == s[1].face || s[0].a == s[0].b {
			continue
		}
		conflict := s[0].a == s[1].a // both a→b the same way ⇒ inconsistent winding
		adj[s[0].face] = append(adj[s[0].face], faceLink{s[1].face, conflict})
		adj[s[1].face] = append(adj[s[1].face], faceLink{s[0].face, conflict})
	}
	return adj
}

// colorFilletFaces 2-colours each connected component of the constraint graph, returning which faces
// must be reversed to make all shared edges antiparallel. The seed face of each component keeps its
// winding (flip=false); a uniform global flip would be invisible to Validate anyway.
func colorFilletFaces(adj [][]faceLink) []bool {
	flip := make([]bool, len(adj))
	seen := make([]bool, len(adj))
	for seed := range adj {
		if !seen[seed] {
			bfsColorFaces(seed, adj, flip, seen)
		}
	}
	return flip
}

// bfsColorFaces breadth-first assigns flip bits from a seed: a neighbour across a conflicting edge
// takes the opposite bit (so exactly one of the pair flips), across a consistent edge the same bit.
func bfsColorFaces(seed int, adj [][]faceLink, flip, seen []bool) {
	seen[seed] = true
	for queue := []int{seed}; len(queue) > 0; {
		fi := queue[0]
		queue = queue[1:]
		for _, link := range adj[fi] {
			if !seen[link.other] {
				seen[link.other] = true
				flip[link.other] = flip[fi] != link.conflict
				queue = append(queue, link.other)
			}
		}
	}
}

// applyFilletFlips reverses the loops (and their welded rings, kept in lock-step) of every face the
// colouring marked, using the metadata-preserving reverseFilletLoop so tangent-seam identity survives.
func applyFilletFlips(faces []filletFace, rings [][][]int, flip []bool) {
	for fi := range faces {
		if !flip[fi] {
			continue
		}
		for li := range faces[fi].loops {
			faces[fi].loops[li] = reverseFilletLoop(faces[fi].loops[li])
			rings[fi][li] = reverseIntRing(rings[fi][li])
		}
	}
}

// reverseFilletLoop reverses a loop's traversal while preserving ALL provenance metadata — the
// metadata-preserving twin of reverseArcLoop (which drops srcV/srcE and so cannot be reused here).
// pts[i]/srcV[i] identify point i; curves[i]/srcE[i] identify the segment LEAVING point i, so under
// reversal the point arrays index n-i while the segment arrays index n-1-i (the segment leaving a
// point in the reversed loop was the segment ARRIVING at it forward). Arc segments are re-derived
// through their recovered midpoint in the new direction; straight segments stay nil.
func reverseFilletLoop(loop filletLoop) filletLoop {
	n := len(loop.pts)
	mids := arcMidpoints(loop)
	out := filletLoop{
		pts:    make([]math.Point3, n),
		curves: make([]geom.Curve3, n),
		srcV:   make([]uint64, n),
		srcE:   make([]uint64, n),
	}
	for i := 0; i < n; i++ {
		from, to := loop.pts[(n-i)%n], loop.pts[(n-i-1+n)%n]
		seg := (n - i - 1 + n) % n
		out.pts[i] = from
		out.srcV[i] = srcIDAt(loop.srcV, (n-i)%n)
		out.srcE[i] = srcIDAt(loop.srcE, seg)
		if c := curveAt(loop.curves, seg); c != nil {
			out.curves[i] = reverseSegmentCurve(c, from, mids[seg], to)
		}
	}
	return out
}

// reverseSegmentCurve reverses one loop segment's curve for reverseFilletLoop. An OPEN arc is re-derived
// through its recovered midpoint in the new direction (the historical path, byte-identical). A CLOSED
// self-loop rim seam (from==to — a full-turn rim, e.g. an intact runout boss wall's top rim) cannot be
// rebuilt from three points (two coincide → a degenerate zero Arc3d, the r8 cap→0 defect): a geom.Arc3d
// reverses by flipping its ORIGINAL sweep, and any OTHER closed rim curve — a geom.EllipseFull top/bottom
// rim of an oblique elliptical-cylinder wall (T7), a closed b-spline seam — is carried UNCHANGED, exactly
// as survivorCurve does for a reversed edge use (the periodic mesher rebuilds it from the surface (u,v)
// and never reads the rim's intrinsic direction, so no reversal is needed). Rebuilding such a closed rim
// via Arc3dByThreePoints collapsed the elliptical wall to a degenerate arc through the ellipse centre,
// meshing the whole boss as a cone/disk (T7 area 2381.68 → 450) — the same class as the r8 cap→0 defect.
func reverseSegmentCurve(c geom.Curve3, from, mid, to math.Point3) geom.Curve3 {
	if from == to {
		if arc, ok := c.(geom.Arc3d); ok {
			arc.StartAngle += arc.SweepAngle
			arc.SweepAngle = -arc.SweepAngle
			return arc
		}
		return c // a closed non-arc rim seam (full-turn ellipse / b-spline): reversal is a no-op for the mesher
	}
	switch geom.InnerCurve(c).(type) {
	case coneCanalSpring, geom.Polyline:
		// A cone-ruling-canal rail with no by-endpoints reconstruction: the exact plane/cone-foot SPRING
		// (coneCanalSpring), or the ⊥-axis far-cap trim, which is the ONLY geom.Polyline any fillet rail
		// ever produces (anchoredTrimPolyline, fillet_cone_far.go — CN4b-2). Reverse it generically (as
		// reverseEndSegs does with ReverseCurve3): a spring reverses exactly and a polyline reverses to the
		// same polyline traversed backwards, so ReverseCurve3 is always faithful for BOTH — never re-fit
		// either as an arc (which would replace the exact locus with a wrong circle). The geom.Polyline arm
		// of this case is therefore effectively scoped to the cone-canal far-cap even though the type match
		// is unqualified: no other fillet rail is a geom.Polyline. Crucially it does NOT match N7's
		// BSpline/spiric canal-CORNER rails, which base reverses via Arc3dByThreePoints — the N7
		// byte-identity gate; and a genuinely correct polyline reversal would be safe here regardless.
		return geom.ReverseCurve3(c)
	}
	r, _ := geom.Arc3dByThreePoints(from, mid, to) // an OPEN arc (or any pre-CN4b-2 rail): the historical path, byte-identical
	return r
}

// reverseIntRing reverses a welded-index ring with the SAME anchor convention as reverseFilletLoop
// (index 0 fixed, the rest reversed) so ring[i] stays the weld of the loop's pts[i] after the flip.
func reverseIntRing(ring []int) []int {
	n := len(ring)
	out := make([]int, n)
	for i := 0; i < n; i++ {
		out[i] = ring[(n-i)%n]
	}
	return out
}
