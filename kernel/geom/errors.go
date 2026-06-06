// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati/math"
)

// CollinearPointsError reports that three points expected to define a circle or
// arc are collinear, so no finite circumscribed circle exists. It carries the
// offending points for diagnosis.
type CollinearPointsError struct {
	A, B, C math.Point2
}

func (e *CollinearPointsError) Error() string {
	return fmt.Sprintf("geom: points %v, %v, %v are collinear; a circle/arc through three points needs them non-collinear", e.A, e.B, e.C)
}

// InvalidHelixError reports a helix built with a non-positive turn count, which has no
// well-defined sweep. It carries the offending turn count for diagnosis.
type InvalidHelixError struct {
	Turns float64
}

func (e *InvalidHelixError) Error() string {
	return fmt.Sprintf("geom: helix turns %g is not positive; a helix needs turns > 0", e.Turns)
}

// CollinearPoints3dError is the 3D analogue of [CollinearPointsError].
type CollinearPoints3dError struct {
	A, B, C math.Point3
}

func (e *CollinearPoints3dError) Error() string {
	return fmt.Sprintf("geom: points %v, %v, %v are collinear; a circle/arc through three points needs them non-collinear", e.A, e.B, e.C)
}
