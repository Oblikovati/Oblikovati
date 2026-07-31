// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// The PINCHED canal loft: the sibling of LoftCanalStations for a rolling-ball band whose
// cross-section COLLAPSES TO A POINT at one or both marching ends. The geometry arises when the
// two host surfaces become mutually TANGENT along the blended edge (OCCT blend/tolblend_simple
// B4/B7/B8/C2/C3: an oblique circle-extrusion wall whose tilt equals the cone cap's slope — the
// wall ruling at the tangency azimuth passes through the cone apex, so the wedge dihedral, and
// with it the fillet section, vanishes there). The band then closes like a teardrop: both feet
// and the whole section arc converge to the single tangency point while the ball centre stays a
// full radius away — a well-posed limit, not a failure (W-F derivation, conecanal_oracle.py:
// section angle ∝ (u−u*)², feet → the rim point, station t → the rim).
//
// LoftCanalStations cannot carry that limit: arcControls rejects a zero-angle section
// (arcMinHalfAngle, correctly — a zero arc has no plane). Here the caller marks which marching
// ENDS are pinched, and those columns are synthesized as the exact degenerate limit instead:
// all three control points at the pinch point with weight 1 (cos of the vanishing half-angle) —
// the standard degenerate-boundary NURBS form (a cone-apex column). Every interior column is the
// same exact rational arc LoftCanalStations builds.

// PinchedCanalLoft is the lofted band plus the v-parameter of every station — the caller needs
// station parameters to split the boundary rails at topology-carrying stations (host seam
// azimuths) with SplitCurve.
type PinchedCanalLoft struct {
	Surf    BSplineSurface
	VParams []float64
}

// LoftPinchedCanalStations lofts exact rolling-ball cross-sections — one per station — into the
// canal band, allowing the FIRST and/or LAST column to be a pinch (a point section). A pinched
// column's feet must both sit AT the pinch point (the caller synthesizes the tangency limit) and
// the point must still be a ball radius from the centre; interior columns must be genuine arcs.
//
//	loft, err := geom.LoftPinchedCanalStations(centers, wallFeet, coneFeet, r, res.Weld(), true, true)
func LoftPinchedCanalStations(centers, feetA, feetB []math.Point3, radius, weld float64, pinchStart, pinchEnd bool) (PinchedCanalLoft, error) {
	if err := assertStationShape(centers, feetA, feetB); err != nil {
		return PinchedCanalLoft{}, err
	}
	cols, err := pinchedStationColumns(centers, feetA, feetB, radius, weld, pinchStart, pinchEnd)
	if err != nil {
		return PinchedCanalLoft{}, err
	}
	vParams, err := alphaParams(coords3(centers), 1) // chord-length spine parametrization (P&T §9.2.1)
	if err != nil {
		return PinchedCanalLoft{}, fmt.Errorf("LoftPinchedCanalStations: spine chord-length params: %w", err)
	}
	vDeg, _ := fitDegree(len(centers)) // >= 2 stations guaranteed by assertStationShape
	surf, err := assembleCanalLoft(cols, vParams, vDeg)
	if err != nil {
		return PinchedCanalLoft{}, err
	}
	return PinchedCanalLoft{Surf: surf, VParams: vParams}, nil
}

// pinchedStationColumns builds the control column of every station: the degenerate limit column
// at a pinched end, the ordinary exact rational arc elsewhere.
func pinchedStationColumns(centers, feetA, feetB []math.Point3, radius, weld float64, pinchStart, pinchEnd bool) ([]arcStation, error) {
	cols := make([]arcStation, len(centers))
	last := len(centers) - 1
	for j := range centers {
		if (j == 0 && pinchStart) || (j == last && pinchEnd) {
			col, err := pinchColumn(centers[j], feetA[j], feetB[j], radius, weld, j)
			if err != nil {
				return nil, err
			}
			cols[j] = col
			continue
		}
		col, err := interiorArcColumn(centers[j], feetA[j], feetB[j], radius, weld, j)
		if err != nil {
			return nil, err
		}
		cols[j] = col
	}
	return cols, nil
}

// pinchColumn is the tangency-limit column: both feet at the single pinch point, a ball radius
// from the centre, all three controls collapsed there with the limit weight cos(0)=1.
func pinchColumn(center, fa, fb math.Point3, radius, weld float64, j int) (arcStation, error) {
	if gap := float64(fa.DistanceTo(fb)); gap > weld {
		return arcStation{}, fmt.Errorf(
			"LoftPinchedCanalStations: pinch station %d feet are %g apart (weld %g): not a collapsed section", j, gap, weld)
	}
	if err := assertFootAtRadius(center, fa, radius, weld, j, "pinch"); err != nil {
		return arcStation{}, err
	}
	return arcStation{fa: fa, shoulder: fa, fb: fa, weight: 1}, nil
}

// interiorArcColumn is the ordinary exact-station column (LoftCanalStations semantics): both feet
// asserted at ball radius, the rational-quadratic arc controls between them.
func interiorArcColumn(center, fa, fb math.Point3, radius, weld float64, j int) (arcStation, error) {
	if err := assertFootAtRadius(center, fa, radius, weld, j, "A"); err != nil {
		return arcStation{}, err
	}
	if err := assertFootAtRadius(center, fb, radius, weld, j, "B"); err != nil {
		return arcStation{}, err
	}
	shoulder, weight, err := arcControls(center, fa, fb, radius)
	if err != nil {
		return arcStation{}, fmt.Errorf("LoftPinchedCanalStations station %d (centre %v): %w", j, center, err)
	}
	return arcStation{fa: fa, shoulder: shoulder, fb: fb, weight: weight}, nil
}
