// SPDX-License-Identifier: GPL-2.0-only

package predicates

import "math"

// epsilon is the binary64 unit roundoff 2^-53. The static-filter error bounds
// below are Shewchuk's a-priori forward-error bounds specialized to binary64;
// they are exact once the machine format is fixed, so they are constants.
const epsilon = 1.1102230246251565e-16 // 2^-53

// o3dFilterA bounds the forward error of the plain floating-point orient3d
// estimate relative to its "permanent" (the sum of magnitudes of the estimate's
// terms). When |det| exceeds this bound the estimate's sign is certified correct.
const o3dFilterA = (7.0 + 56.0*epsilon) * epsilon

// o2dFilterA is the corresponding bound for orient2d, relative to detsum.
const o2dFilterA = (3.0 + 16.0*epsilon) * epsilon

// rounded forces its (already binary64) argument through an explicit conversion.
// The Go spec guarantees this rounds to binary64 precision, which blocks the
// compiler from fusing a preceding multiply with a following add/sub into an FMA
// (Oblikovati#2020: Go fuses a*b+c on arm64 but not amd64). The static-filter
// error bound is only valid if every multiply is separately rounded, so EVERY
// product that feeds an add/sub must pass through here. It is a no-op move at
// run time. See doc.go.
func rounded(x float64) float64 { return float64(x) }

// filterOrient2D returns (estimate, certified). When certified is true the sign of
// estimate is the exact sign of orient2d; when false the caller must fall back to
// the exact rational determinant.
func filterOrient2D(ax, ay, bx, by, cx, cy float64) (det float64, certified bool) {
	detleft := rounded((ax - cx) * (by - cy))
	detright := rounded((ay - cy) * (bx - cx))
	det = detleft - detright
	// Opposite-signed products cannot cancel catastrophically: the sign is safe.
	if (detleft > 0) != (detright > 0) || detleft == 0 || detright == 0 {
		return det, true
	}
	detsum := math.Abs(detleft + detright)
	errbound := o2dFilterA * detsum
	return det, math.Abs(det) >= errbound
}

// filterOrient3D returns (estimate, certified) for orient3d, analogously.
func filterOrient3D(ax, ay, az, bx, by, bz, cx, cy, cz, dx, dy, dz float64) (det float64, certified bool) {
	adx, ady, adz := ax-dx, ay-dy, az-dz
	bdx, bdy, bdz := bx-dx, by-dy, bz-dz
	cdx, cdy, cdz := cx-dx, cy-dy, cz-dz

	bdxcdy, cdxbdy := rounded(bdx*cdy), rounded(cdx*bdy)
	cdxady, adxcdy := rounded(cdx*ady), rounded(adx*cdy)
	adxbdy, bdxady := rounded(adx*bdy), rounded(bdx*ady)

	t1 := rounded(adz * (bdxcdy - cdxbdy))
	t2 := rounded(bdz * (cdxady - adxcdy))
	t3 := rounded(cdz * (adxbdy - bdxady))
	det = t1 + t2 + t3

	permanent := (math.Abs(bdxcdy)+math.Abs(cdxbdy))*math.Abs(adz) +
		(math.Abs(cdxady)+math.Abs(adxcdy))*math.Abs(bdz) +
		(math.Abs(adxbdy)+math.Abs(bdxady))*math.Abs(cdz)
	errbound := o3dFilterA * permanent
	return det, math.Abs(det) >= errbound
}
