// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// spineParam is a point's signed position along the fillet cylinder's axis: the projection of
// (p - cyl.Origin) onto cyl.AxisDir, i.e. Cylinder.PointAt's v parameter. This is the shared ruler
// the intact-boss setback detector (fillet_setback_detect.go) measures runout extent against.
func spineParam(p math.Point3, cyl geom.Cylinder) float64 {
	return float64(cyl.Origin.VectorTo(p).Dot(cyl.AxisDir.AsVector()))
}

// spineInterval projects a cut's two crossing points onto the fillet spine and returns the
// resulting [lo,hi] interval (lo <= hi regardless of which crossing is axially first).
func spineInterval(cut imprintCut, cyl geom.Cylinder) (lo, hi float64) {
	a, b := spineParam(cut.pMinus, cyl), spineParam(cut.pPlus, cyl)
	if a > b {
		a, b = b, a
	}
	return a, b
}
