// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/math"

// Loft skinning — CAP construction (M48 #2236 split of loft_skin.go). Extends the loft's end loops
// outward by a small overshoot so the end caps close cleanly against a neighbouring body, shifting each
// end loop along its centroid axis. The section preparation lives in loft_skin.go.

// extendEnds pushes a hole loft's first and last sections slightly OUTWARD along the loft
// direction, so a bore Cut overshoots the outer body's end caps instead of meeting them
// coplanar — a coplanar-cap Difference leaves the tube open. Open lofts only (eps<=0 is a
// no-op, e.g. for closed lofts which have no ends).
func extendEnds(loops [][]math.Point3, eps float64) [][]math.Point3 {
	m := len(loops)
	if m < 2 || eps <= 0 {
		return loops
	}
	out := make([][]math.Point3, m)
	copy(out, loops)
	out[0] = shiftLoop(loops[0], unitFromTo(loopCentroid(loops[1]), loopCentroid(loops[0])), eps)
	out[m-1] = shiftLoop(loops[m-1], unitFromTo(loopCentroid(loops[m-2]), loopCentroid(loops[m-1])), eps)
	return out
}

// loftOvershoot is the bore-extension distance for a loft of these (outer) sections: a small
// fraction of the loft's end-to-end length (with a floor), enough to clear the coplanar-cap
// degeneracy without being visible.
func loftOvershoot(outers [][]math.Point3) float64 {
	if len(outers) < 2 {
		return 0
	}
	span := float64(loopCentroid(outers[0]).DistanceTo(loopCentroid(outers[len(outers)-1])))
	if e := 0.01 * span; e > 1e-3 {
		return e
	}
	return 1e-3
}

func loopCentroid(loop []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range loop {
		sx, sy, sz = sx+float64(p.X), sy+float64(p.Y), sz+float64(p.Z)
	}
	n := float64(len(loop))
	return math.P3(math.Scalar(sx/n), math.Scalar(sy/n), math.Scalar(sz/n))
}

// unitFromTo returns the unit vector from a to b (zero when coincident).
func unitFromTo(a, b math.Point3) math.Vector3 {
	v := a.VectorTo(b)
	l := v.Length()
	if l == 0 {
		return math.V3(0, 0, 0)
	}
	return v.Scale(1 / float64(l))
}

func shiftLoop(loop []math.Point3, dir math.Vector3, d float64) []math.Point3 {
	out := make([]math.Point3, len(loop))
	delta := dir.Scale(d)
	for i, p := range loop {
		out[i] = p.TranslateBy(delta)
	}
	return out
}
