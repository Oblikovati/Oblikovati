// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// LoftCanalStations lofts EXACT rolling-ball cross-section arcs — one per closed-form spine station —
// into the constant-radius canal BSplineSurface (degree-2-rational-u × spline-v). It is the ARM sibling
// of loftCanal: where loftCanal reads a MARCHED spine's vertices and PROJECTS the ball feet onto the
// host surfaces, this consumes stations whose centre and BOTH host feet are supplied EXACTLY (closed
// form). That is the recommended construction for the Cone∧Plane RULING-edge arm, whose ball-centre
// spine is an exact hyperbola and whose feet are exact plane/cone contacts (cone-host-corner-
// derivation.md §2 "Arm B", candidate 1) — no SSI marching, no convergence story. The only
// approximation is the v-interpolation BETWEEN exact stations; every station column is exact on the true
// envelope. feetA[j]/feetB[j] must sit at distance `radius` from centers[j] (asserted, weld-scaled) — a
// mis-supplied foot is a caller construction bug, declined not lofted (do-no-harm).
//
//	surf, err := geom.LoftCanalStations(centers, planeFeet, coneFeet, 10, res.Weld())
func LoftCanalStations(centers, feetA, feetB []math.Point3, radius, weld float64) (BSplineSurface, error) {
	if err := assertStationShape(centers, feetA, feetB); err != nil {
		return BSplineSurface{}, err
	}
	cols, err := exactStationColumns(centers, feetA, feetB, radius, weld)
	if err != nil {
		return BSplineSurface{}, err
	}
	return loftStationColumns(centers, cols)
}

// assertStationShape rejects a malformed station set: fewer than two stations (nothing to loft between),
// or foot rows whose lengths disagree with the centre count.
func assertStationShape(centers, feetA, feetB []math.Point3) error {
	n := len(centers)
	if n < 2 {
		return fmt.Errorf("LoftCanalStations: %d stations, need >= 2 to loft", n)
	}
	if len(feetA) != n || len(feetB) != n {
		return fmt.Errorf("LoftCanalStations: foot counts (fa=%d, fb=%d) != station count %d", len(feetA), len(feetB), n)
	}
	return nil
}

// exactStationColumns builds one cross-section arc control triple per station from the EXACT feet: it
// asserts each foot at ball distance `radius` (do-no-harm) then reuses arcControls (shoulder + cos-β
// weight) — the same rational-quadratic geometry the marched stationColumns builds, minus the surface
// projection (the feet are already exact closed forms, so no ClosestPointOnSurface).
func exactStationColumns(centers, feetA, feetB []math.Point3, radius, weld float64) ([]arcStation, error) {
	cols := make([]arcStation, len(centers))
	for j, m := range centers {
		if err := assertFootAtRadius(m, feetA[j], radius, weld, j, "A"); err != nil {
			return nil, err
		}
		if err := assertFootAtRadius(m, feetB[j], radius, weld, j, "B"); err != nil {
			return nil, err
		}
		shoulder, weight, err := arcControls(m, feetA[j], feetB[j], radius)
		if err != nil {
			return nil, fmt.Errorf("LoftCanalStations station %d (centre %v): %w", j, m, err)
		}
		cols[j] = arcStation{fa: feetA[j], shoulder: shoulder, fb: feetB[j], weight: weight}
	}
	return cols, nil
}

// assertFootAtRadius verifies a supplied host foot sits at ball distance `radius` from the station
// centre, carrying the station index/side and the offending gap on failure. The caller (the cone canal
// arm) computes the feet in closed form and asserts |T−m| == r algebraically, so a gap over weld here is
// a construction bug, not floating-point imprecision.
func assertFootAtRadius(m, foot math.Point3, radius, weld float64, j int, side string) error {
	if d := stdmath.Abs(float64(foot.DistanceTo(m)) - radius); d > weld {
		return fmt.Errorf(
			"LoftCanalStations station %d foot %s is %g off radius %g (weld %g): non-envelope foot", j, side, d, radius, weld)
	}
	return nil
}
