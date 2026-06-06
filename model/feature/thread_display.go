// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// ThreadDisplayCurves returns one helix polyline (model units) per cosmetic thread in the
// engine, drawn on its threaded cylindrical face at the thread pitch — the geometry the head
// renders so a cosmetic thread is visible (Inventor's thread display). A thread whose face is
// unresolved or non-cylindrical contributes nothing.
func ThreadDisplayCurves(fs *PartFeatures) [][]math.Point3 {
	bodies := fs.Result()
	var out [][]math.Point3
	for i := 0; i < fs.Count(); i++ {
		tf, ok := fs.Item(i).Definition().(*ThreadFeature)
		if !ok || tf.spec == nil || tf.def.Cut { // a cut thread is real geometry, no cosmetic helix
			continue
		}
		for _, b := range bodies {
			f, ok := b.FindFaceByKey(tf.def.FaceKey)
			if !ok {
				continue
			}
			if h := threadHelix(f, tf.spec); len(h) > 1 {
				out = append(out, h)
			}
			break
		}
	}
	return out
}

// threadHelix samples a helix on the face's cylinder over the face's axial extent, spiralling
// once per thread pitch (left-handed reverses the spin). The line sits on the surface so it
// reads as the thread groove.
func threadHelix(face *topo.Face, spec *ThreadSpec) []math.Point3 {
	cyl, ok := face.Geometry().(geom.Cylinder)
	if !ok {
		return nil
	}
	vMin, vMax := axialExtent(face.RangeBox(), cyl)
	length := vMax - vMin
	pitch := spec.Pitch / 10 // designation pitch is mm; the model is cm
	if pitch <= 0 || length <= 0 {
		return nil
	}
	turns := length / pitch
	sign := 1.0
	if !spec.RightHanded {
		sign = -1
	}
	steps := int(turns*16) + 2
	pts := make([]math.Point3, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		pts[i] = cyl.PointAt(sign*t*turns*2*stdmath.Pi, vMin+t*length)
	}
	return pts
}

// axialExtent projects a face's range-box corners onto the cylinder axis to bound the threaded
// length.
func axialExtent(box math.Box, cyl geom.Cylinder) (float64, float64) {
	axis := cyl.AxisDir.AsVector()
	vMin, vMax := stdmath.Inf(1), stdmath.Inf(-1)
	for _, c := range box.Corners() {
		v := cyl.Origin.VectorTo(c).Dot(axis)
		vMin, vMax = stdmath.Min(vMin, v), stdmath.Max(vMax, v)
	}
	return vMin, vMax
}

// bodyHasMaterialOutside reports whether the body has material just outside the cylinder radius
// at axial v — true for a bore (the thread cuts outward), false for a shaft (it cuts inward).
func bodyHasMaterialOutside(body *topo.Body, cyl geom.Cylinder, v, depth float64) bool {
	axisPt := cyl.Origin.TranslateBy(cyl.AxisDir.AsVector().Scale(v))
	radial := axisPt.VectorTo(cyl.PointAt(0, v)) // outward, length = cyl.Radius
	if radial.LengthSquared() == 0 {
		return false
	}
	out := stdmath.Max(depth, 0.02) // probe this far past the surface
	probe := axisPt.TranslateBy(radial.Scale((cyl.Radius + out) / cyl.Radius))
	return ops.PointInsideBody(body, probe)
}
