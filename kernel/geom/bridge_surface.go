// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Bridge surface (M36-F09) — a clean NURBS transition spanning two edges, meeting each side at a
// chosen continuity (G0/G1/G2). The everyday Class-A "connect these two panels" move. The bridge is
// degree(boundary)×5: six control rows in the bridge direction, enough to match G2 on BOTH sides
// (rows 0–2 hold continuity to surface A at v=0, rows 3–5 to surface B at v=1, without overlap). The
// two boundary curves are made compatible first (F01), then the seam rows are set by F05's
// knot-correct MatchSurface.

// bridgeVRows is the bridge-direction control-row count: 6 (degree 5) supports up to G2 on each
// side (3 rows per side). G3 on both sides would need 8 rows — a follow-up.
const bridgeVRows = 6
const bridgeVDeg = 5

// BridgeSurface builds the NURBS bridge from curve cA (on surface sA, its edgeA boundary) to cB (on
// sB, edgeB), matching sA at orderA and sB at orderB (0=G0 position only, 1=G1 tangent, 2=G2
// curvature). cA and cB must be the shared boundary iso-curves of sA/sB and run the same direction
// (the caller orients them). It errors when an order exceeds G2 or a match is invalid.
func BridgeSurface(cA, cB BSplineCurve, sA, sB BSplineSurface, edgeA, edgeB Boundary, orderA, orderB int) (BSplineSurface, error) {
	if orderA > 2 || orderB > 2 {
		return BSplineSurface{}, fmt.Errorf("geom.BridgeSurface: continuity order %d/%d exceeds G2 (the 6-row bridge's limit)", orderA, orderB)
	}
	ca, cb, err := MakeCompatible(cA, cB)
	if err != nil {
		return BSplineSurface{}, fmt.Errorf("geom.BridgeSurface: %w", err)
	}
	bridge, err := bridgeNet(ca, cb)
	if err != nil {
		return BSplineSurface{}, err
	}
	if orderA > 0 {
		if bridge, err = MatchSurface(bridge, sA, VMinEdge, edgeA, orderA); err != nil {
			return BSplineSurface{}, fmt.Errorf("geom.BridgeSurface: side A: %w", err)
		}
	}
	if orderB > 0 {
		if bridge, err = MatchSurface(bridge, sB, VMaxEdge, edgeB, orderB); err != nil {
			return BSplineSurface{}, fmt.Errorf("geom.BridgeSurface: side B: %w", err)
		}
	}
	return bridge, nil
}

// bridgeNet builds the initial degree(ca)×5 ruled net: row 0 = ca (v=0 boundary), row 5 = cb (v=1),
// the interior rows linearly interpolated. MatchSurface later overwrites rows 1–2 / 3–4 for G1/G2;
// the ruled init is just a well-conditioned starting net (and the final shape for an all-G0 bridge).
func bridgeNet(ca, cb BSplineCurve) (BSplineSurface, error) {
	nu := len(ca.Ctrl)
	ctrl := make([][]math.Point3, nu)
	w := make([][]float64, nu)
	for i := 0; i < nu; i++ {
		ctrl[i] = make([]math.Point3, bridgeVRows)
		w[i] = make([]float64, bridgeVRows)
		for r := 0; r < bridgeVRows; r++ {
			ctrl[i][r] = lerpP(ca.Ctrl[i], cb.Ctrl[i], float64(r)/float64(bridgeVRows-1))
			w[i][r] = 1
		}
	}
	vk := make([]float64, bridgeVRows+bridgeVDeg+1) // clamped degree-5 Bézier knots: 6 zeros, 6 ones
	for k := bridgeVRows; k < len(vk); k++ {
		vk[k] = 1
	}
	return NewBSplineSurface(ca.Degree, bridgeVDeg, ctrl, w, ca.Knots, vk)
}
