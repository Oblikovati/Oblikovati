// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/fit"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Fit surface to points (M36-F15): least-squares fit a clean Class-A NURBS surface to a region of a
// scanned point cloud / mesh, then wrap it as a single-face surface body. Degree and span (control)
// targets keep the result clean; the achieved deviation is reported by the caller (model/analysis
// F14). This is the body-level step of reverse-engineering styling from scan data (#1291).

// FitSurfaceToPoints builds a surface body by fitting a degree×degree B-spline with nu×nv control
// points to the region points. It errors when the points do not determine a base plane or the
// control count is too high for the region (see fit.SurfaceToPoints).
func FitSurfaceToPoints(points []math.Point3, degree, nu, nv int) (*topo.Body, error) {
	surf, err := fit.SurfaceToPoints(points, degree, nu, nv)
	if err != nil {
		return nil, fmt.Errorf("ops.FitSurfaceToPoints: %w", err)
	}
	return fullDomainBody(surf, "fit-surface"), nil
}
