// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"oblikovati.org/math"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/sketch"
)

// Reconstructing the boundary a region loop describes, so a rebuilt cell can be tested for lying
// INSIDE it (#26). The loop is an unordered bag of curves, so the boundary is chained head-to-tail;
// the arcs themselves are exact, trimmed by ipt.trimCircleEdge from the file's own point1AI /
// point2AI / posDir, so nothing here guesses which way round a circle the face runs.

// polyTol is the distance (cm) under which two region-boundary endpoints are the same point when
// chaining a loop's edges head-to-tail for the CONTAINMENT test. It is used ONLY to decide which
// cells lie inside a region (regionBoundary/joinChains); it never touches built geometry.
//
// It cannot be float-noise-tight: a loop's edges do NOT always meet exactly. A thin (sub-mm) ribbon
// where an end-cap arc meets the two side lines carries a real ~0.05 mm spread between the arc's
// trimmed endpoint and the line's endpoint (TapePath's tape-guide loops: three curves meet at a
// corner within a ~0.01 cm triangle). At 1e-6 those loops were UNCHAINABLE, so containment declined
// and the caller fell back to the curve-set rule — which over-selected 13 cells (4.83 of 20.49 cm²)
// and built the extrude at 7.6x its true volume.
//
// 7e-3 cm sits just above the real ~5e-3 cm endpoint spread and below the ~1.1e-2 cm separation of
// genuinely-distinct corner points, so it joins the meant-to-coincide endpoints without collapsing a
// real corner (a looser 1e-2 mis-closed TapePath's loop[2], halving its enclosed area). It stays far
// below any feature size — negligible for a point-in-polygon test whose cell interiors sit well
// inside the boundary. Only parts whose loops FAIL to chain at 1e-6 can change; a loop that already
// closes is unaffected.
const polyTol = 7e-3 // tol:calibrated — region containment endpoint join; see sketch arrMergeTol

// arcSegments is how finely an arc is sampled for the containment test. Containment only asks which
// side of the boundary a cell sits on — it is well inside or well outside — so this is a test
// polyline, not model geometry, and never reaches the built body.
const arcSegments = 64

// containedProfileIndices selects the cells whose INTERIOR lies inside the region — the rule the
// file actually means. Inventor's region is a face bounded by its material loops with its Cut loops
// as holes; any minimal cell inside that face belongs to it, whatever else the sketch draws through
// it. ok=false when a loop's boundary cannot be chained, so the caller keeps the old curve-set rule
// rather than guessing.
//
// The curve-set rule it supersedes ("every curve bounding a cell must be named") is wrong in BOTH
// directions, and no rule phrased over curve sets can be right: inside/outside is not decidable from
// WHICH curves bound a cell. ReadWriteHead's ex0 proves both halves — it REJECTED the two cells that
// ARE the region (a rectangle + its circular segment, 3.739 of 3.785 cm²) because one unnamed line
// divides them, and ACCEPTED a segment lying outside the region because the circle and top line
// bounding it both happen to be named. It built the wrong 17% of the part.
func containedProfileIndices(sk *sketch.Sketch, region []ipt.RegionLoop) ([]int, bool) {
	outers, holes, ok := regionBoundaries(region)
	if !ok || len(outers) == 0 {
		return nil, false
	}
	profiles := sk.Profiles()
	var out []int
	for i := 0; i < profiles.Count(); i++ {
		p := profiles.Item(i)
		if !p.IsClosed() {
			continue // an open chain bounds no area; see regionProfileIndices
		}
		q, ok := profileInteriorPoint(p)
		if !ok {
			// A cell with no placeable interior point is a degenerate sliver of the arrangement (a
			// zero-width wedge between near-coincident curves); it bounds no material, so skip it
			// rather than declining the WHOLE region. Declining sent the entire sketch to the
			// curve-set fallback whenever one of many cells was degenerate — TapePath's 60-cell tape
			// path had such slivers, so its thin ribbons were never containment-selected and the
			// fallback over-built it 7.6x.
			continue
		}
		if !insideRegion(q, outers, holes) {
			continue
		}
		// A cell belongs to the region only if it FITS the loop that holds its test point. A single
		// interior point being inside a loop is necessary but not sufficient: a large cell can have a
		// corner poke into a small loop while the rest lies far outside it. FlangeReelMotor's +-shaped
		// keep cell (108 cm²) had its point land in a 13.7 cm² edge scallop, so a through-cut selected
		// the whole + and gutted the flange. A cell far larger than its containing loop cannot be that
		// loop's interior, so it is rejected (the generous 1.5x slack passes reconstruction noise while
		// catching this 7.9x mismatch).
		if a, ok := smallestContainingArea(q, outers); ok && polygonArea(p.OuterLoop().Polygon()) > cellFitSlack*a {
			continue
		}
		out = append(out, i)
	}
	return out, true
}

// cellFitSlack is how much larger than its containing region loop a cell may be and still count as
// inside it — headroom for arc-sampling and boundary-chaining noise, well below any real over-select.
const cellFitSlack = 1.5

// profileInteriorPoint returns a point inside the cell's MATERIAL — inside its outer loop and
// outside its own holes.
//
// Testing the outer loop alone is not enough and silently mis-selects: an ANNULAR cell's outer
// polygon contains its hole, so the test point can land in the hole — which is not the cell at all.
// CompressionRollerArmActuatorScrew regressed that way. Its bore-cut names a small disc, and the
// annulus around that disc got a test point sitting in its own middle, i.e. inside the named disc,
// so the annulus counted as part of the region and the CUT removed four times too much (1.008x ->
// 0.784x). #27.
func profileInteriorPoint(p *sketch.Profile) (math.Point2, bool) {
	inner := p.InnerLoops()
	outside := func(q math.Point2) bool {
		for _, h := range inner {
			if pointInPoly(q, h.Polygon()) {
				return false
			}
		}
		return true
	}
	q, ok := interiorPointAvoiding(p.OuterLoop().Polygon(), outside)
	return q, ok
}

// regionBoundaries reconstructs the region's material outlines and its holes.
//
// A MATERIAL loop that will not chain into a closed polygon is an OPEN sliver — a degenerate
// fragment that bounds no area, so it can be no face and is dropped rather than failing the whole
// region (TapePath's tape-guide patch carries one such 2-edge stub — a ~0.005 cm arc joined to a
// line at a single shared vertex — alongside five real thin-ribbon faces; failing on it sent the
// whole region to the wrong curve-set rule, which over-selected 13 cells and built the extrude at
// 7.6x volume). A CUT loop is NOT dropped: an unreadable hole would silently over-fill the face, so
// its region still declines to containment and keeps the safer fallback.
func regionBoundaries(region []ipt.RegionLoop) (outers, holes [][]math.Point2, ok bool) {
	for _, l := range region {
		poly, ok := regionBoundary(l)
		if !ok || len(poly) < 3 {
			if l.Cut {
				return nil, nil, false // a hole we cannot read: decline rather than over-fill
			}
			continue // an open material sliver bounds no face
		}
		if l.Cut {
			holes = append(holes, poly)
		} else {
			outers = append(outers, poly)
		}
	}
	return outers, holes, len(outers) > 0
}

// insideRegion reports whether q is inside some material outline and outside every hole.
func insideRegion(q math.Point2, outers, holes [][]math.Point2) bool {
	in := false
	for _, o := range outers {
		if pointInPoly(q, o) {
			in = true
			break
		}
	}
	if !in {
		return false
	}
	for _, h := range holes {
		if pointInPoly(q, h) {
			return false
		}
	}
	return true
}

// regionBoundary chains a loop's curves into its closed boundary polygon. ok=false when they do not
// chain (an unexpected shape), so the caller falls back rather than guessing.
func regionBoundary(l ipt.RegionLoop) ([]math.Point2, bool) {
	chains, whole := loopChains(l)
	if len(chains) == 0 {
		if len(whole) == 1 {
			return whole[0], true // a lone full circle: the face is the disc
		}
		return nil, false
	}
	for {
		merged, joined := joinChains(chains)
		chains = merged
		if !joined {
			break
		}
	}
	if len(chains) != 1 || !sameXY(chains[0][0], last(chains[0])) {
		return nil, false // did not close: decline
	}
	return chains[0][:len(chains[0])-1], true
}

// loopChains turns each curve into a polyline. An arc is already trimmed to the span the face uses,
// so it chains like any other edge; only a circle the file left WHOLE (p1 == p2, as a bore's hole
// loop is) stands alone.
func loopChains(l ipt.RegionLoop) (chains [][]math.Point2, whole [][]math.Point2) {
	for _, e := range l.Edges {
		switch e.Kind {
		case ipt.EdgeLine:
			chains = append(chains, []math.Point2{pt2(e.Line.A), pt2(e.Line.B)})
		case ipt.EdgeArc:
			chains = append(chains, arcPolyline(e.Arc))
		case ipt.EdgeCircle:
			whole = append(whole, sampleWholeCircle(e.Circle))
		case ipt.EdgeEllipse:
			if e.Ellipse.Start == (ipt.Point2D{}) && e.Ellipse.End == (ipt.Point2D{}) {
				whole = append(whole, ellipseArcPolyline(e.Ellipse)) // a whole ellipse stands alone, like a bore
			} else {
				chains = append(chains, ellipseArcPolyline(e.Ellipse))
			}
		}
	}
	return chains, whole
}

// joinChains merges any two chains that share an endpoint, once. The caller repeats to fixpoint.
func joinChains(chains [][]math.Point2) ([][]math.Point2, bool) {
	for i := 0; i < len(chains); i++ {
		for j := i + 1; j < len(chains); j++ {
			if m, ok := joinPair(chains[i], chains[j]); ok {
				out := [][]math.Point2{m}
				for k := range chains {
					if k != i && k != j {
						out = append(out, chains[k])
					}
				}
				return out, true
			}
		}
	}
	return chains, false
}

// joinPair concatenates two chains at whichever ends coincide.
func joinPair(a, b []math.Point2) ([]math.Point2, bool) {
	switch {
	case sameXY(last(a), b[0]):
		return append(append([]math.Point2{}, a...), b[1:]...), true
	case sameXY(last(a), last(b)):
		return append(append([]math.Point2{}, a...), reversePts(b)[1:]...), true
	case sameXY(a[0], b[0]):
		return append(reversePts(a), b[1:]...), true
	case sameXY(a[0], last(b)):
		return append(append([]math.Point2{}, b...), a[1:]...), true
	}
	return nil, false
}
