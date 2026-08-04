// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/topo"
)

// The Cylinder∧Cylinder seam WELD dispatch: the group classifier the curved-arm router's
// single-runout prefix consults (fillet_curved_single_runout.go), and the sequential rebuild that
// turns one op's seam picks into the final solid. A multi-seam op (bfuseblend/B4: the bore's entry
// AND exit loops in one blend) rebuilds one seam at a time — each later pick's edge is re-located
// on the intermediate body by its stable reference key (rebuildRim preserves lineage on every
// untouched edge) and re-solved there, so the second band's topology is built against the body the
// first band produced. The seams of one op are disjoint closed loops (a shared vertex would be a
// miter/corner, which never reaches this payload), so sequential composition is exact.

// cylCylSeamGroupOf returns the op's seam picks in fils order when EVERY fillet carries the
// cyl∧cyl seam payload, else nil (mixed or foreign ops keep their existing dispatch, do-no-harm).
func cylCylSeamGroupOf(fils []edgeFillet) []cylCylSeamOpPick {
	group := make([]cylCylSeamOpPick, 0, len(fils))
	for _, ef := range fils {
		band, ok := ef.armSurface.(*cylCylSeamBand)
		if !ok {
			return nil
		}
		group = append(group, cylCylSeamOpPick{
			key: ef.edge.ReferenceKey(), r: band.r, line: band.line,
			start: ef.edge.StartVertex().Point(), end: ef.edge.EndVertex().Point(),
		})
	}
	return group
}

// cylCylSeamGroupBody rebuilds the body with every seam of the group rounded, sequentially: each
// pick is re-located on the CURRENT body by reference key (identity on the original body for the
// first pick) and re-solved there — the seams of one op are disjoint, so composition is exact and
// the re-solve is the correctness point (the intermediate body's faces are new entities). An
// empty reason means the returned body carries all bands; a non-empty reason names the failing
// seam and obstruction and the body is nil (never a partial body — the router floors).
func cylCylSeamGroupBody(body *topo.Body, band *cylCylSeamBand, res Resolution) (*topo.Body, string) {
	if len(band.group) == 0 {
		return nil, "cyl∧cyl seam weld: empty pick group (dispatch stash missing)"
	}
	current := body
	for _, pick := range band.group {
		var reason string
		if current, reason = rebuildLocatedCylCylSeam(current, pick, res); reason != "" {
			return nil, reason
		}
	}
	return current, ""
}

// rebuildLocatedCylCylSeam re-locates one seam pick on the current body, re-solves it there, and
// rebuilds through its mode's weld: the closed-rim rebuild for a loop, the single-arm runout for
// an equal-parallel valley line.
func rebuildLocatedCylCylSeam(body *topo.Body, pick cylCylSeamOpPick, res Resolution) (*topo.Body, string) {
	e, ok := locateCylCylSeamPick(body, pick)
	if !ok {
		return nil, fmt.Sprintf("cyl∧cyl seam weld: seam edge (key %x, %v→%v) not found on the rebuilt body",
			pick.key, pick.start, pick.end)
	}
	if pick.line {
		return rebuildCylCylValleyLine(body, e, pick.r, res)
	}
	wrapF, otherF, ok := cylCylSeamHosts(e)
	if !ok {
		return nil, fmt.Sprintf("cyl∧cyl seam weld: re-located edge %d no longer borders two wrap-classified cylinders", e.ID())
	}
	rim, _, reason := solveCylCylSeamRim(body, e, wrapF, otherF, pick.r)
	if reason != "" {
		return nil, reason
	}
	return rebuildCylCylSeam(body, rim)
}

// locateCylCylSeamPick finds a pick's seam edge on the CURRENT body: by reference key first
// (survives the closed-rim rebuild, which copies lineage), else by its two endpoint points — the
// seams of one op are disjoint and untouched by each other's welds, so the endpoints are exact;
// the tolerance is a fraction of the seam's own span. Both-cylinder adjacency is re-checked by
// the caller's solve, not here.
func locateCylCylSeamPick(body *topo.Body, pick cylCylSeamOpPick) (*topo.Edge, bool) {
	if e, ok := body.FindEdgeByKey(pick.key); ok {
		return e, true
	}
	tol := 1e-6 * float64(pick.start.DistanceTo(pick.end))
	for _, e := range body.Edges() {
		s, en := e.StartVertex().Point(), e.EndVertex().Point()
		forward := float64(s.DistanceTo(pick.start)) <= tol && float64(en.DistanceTo(pick.end)) <= tol
		backward := float64(s.DistanceTo(pick.end)) <= tol && float64(en.DistanceTo(pick.start)) <= tol
		if forward || backward {
			return e, true
		}
	}
	return nil, false
}

// rebuildCylCylValleyLine re-solves one equal-parallel valley LINE pick on the current body and
// welds it through the EXISTING single-arm runout assembly (fillet_curved_single_runout.go) with
// the plain exact cylinder arm — the same body-builder a convex Plane∧Cylinder line arm takes.
// An intermediate wrong solid cannot slip through silently: the router certifies the FINAL body,
// and each seam's own weld declines are relayed here.
func rebuildCylCylValleyLine(body *topo.Body, e *topo.Edge, r float64, res Resolution) (*topo.Body, string) {
	ef, ok := cylCylParallelValleyArmEdge(body, e, filletPick{edge: e, r0: r, r1: r})
	if !ok {
		return nil, fmt.Sprintf("cyl∧cyl seam weld: valley line edge %d no longer solves on the rebuilt body", e.ID())
	}
	band := ef.armSurface.(*cylCylSeamBand)
	ef.armSurface = band.Surface // unwrap: hand the runout weld the plain exact cylinder arm
	return singleArmRunoutBody(body, ef, res)
}

// rebuildCylCylSeam routes one solved seam through the host-agnostic closed-rim rebuild — the
// proven watertight closed-band weld (rebuildRim), concave/convex selected by the solve.
func rebuildCylCylSeam(body *topo.Body, rim *rimFillet) (*topo.Body, string) {
	rebuild := rebuildWithRimFillet
	if rim.concave {
		rebuild = rebuildWithConcaveRimFillet
	}
	b, err := rebuild(body, rim)
	if err != nil {
		return nil, fmt.Sprintf("cyl∧cyl seam rebuild declined: %v", err)
	}
	return b, ""
}
