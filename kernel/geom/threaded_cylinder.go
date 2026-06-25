// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// ThreadedCylinder is a cylinder whose surface carries a real machined thread: the radius is
// modulated by a V thread profile that spirals along a helix at the given Pitch. It is how a
// modeled (cut) thread is represented — swapping a plain cylindrical face for this one threads
// the part with O(1) work (no boolean), and tessellating it yields the real threaded geometry.
//
// External threads cut grooves inward (radius dips to Radius−Depth at the root); internal
// (bore) threads cut outward (radius rises to Radius+Depth). The amplitude ramps to zero over
// one pitch at each end (thread runout) so the surface still meets its bounding circles.
type ThreadedCylinder struct {
	Cylinder
	Pitch       float64 // axial advance per turn (model units)
	Depth       float64 // thread height (root-to-crest)
	Internal    bool
	RightHanded bool
	VMin, VMax  float64 // axial extent of the threaded run
}

var _ Surface = ThreadedCylinder{}

// radiusAt returns the threaded radius at angle u, axial v.
func (t ThreadedCylinder) radiusAt(u, v float64) float64 {
	hand := 1.0
	if !t.RightHanded {
		hand = -1.0
	}
	phase := stdmath.Mod(v-hand*t.Pitch*u/(2*stdmath.Pi), t.Pitch)
	if phase < 0 {
		phase += t.Pitch
	}
	frac := phase / t.Pitch             // 0..1 within one thread
	groove := 1 - stdmath.Abs(2*frac-1) // V: 0 at crest (0,1), 1 at root (0.5)
	groove *= t.runout(v)
	if t.Internal {
		return t.Radius + t.Depth*groove // bore: cut outward
	}
	return t.Radius - t.Depth*groove // shaft: cut inward
}

// runout ramps the thread amplitude from 0 at each end (over one pitch) to 1 in the middle, so
// the threaded surface coincides with the plain bounding circles at v = VMin / VMax.
func (t ThreadedCylinder) runout(v float64) float64 {
	if t.Pitch <= 0 {
		return 1
	}
	return stdmath.Min(clamp01((v-t.VMin)/t.Pitch), clamp01((t.VMax-v)/t.Pitch))
}

// PointAt returns the threaded surface point at (u = angle, v = axial distance).
func (t ThreadedCylinder) PointAt(u, v float64) math.Point3 {
	r := t.radiusAt(u, v)
	return t.Origin.TranslateBy(t.AxisDir.AsVector().Scale(v)).TranslateBy(t.radial(u).Scale(r))
}

// DerivativesAt returns the partials by central differences (the thread modulation has no tidy
// closed form; finite differences are exact enough for shading/normals). Each direction uses the
// first-difference-optimal step stepD1 = ε^{1/3} scaled by its domain span — the periodic angular
// span in u and the threaded run in v — so the step stays scale-invariant rather than a fixed 1e-5
// that ignored both the [0,2π] angular scale and the part size (#1402).
func (t ThreadedCylinder) DerivativesAt(u, v float64) (du, dv math.Vector3) {
	eu := stepD1 * spanOr1(t.UDomain())
	ev := stepD1 * spanOr1(t.VDomain())
	du = t.PointAt(u-eu, v).VectorTo(t.PointAt(u+eu, v)).Scale(1 / (2 * eu))
	dv = t.PointAt(u, v-ev).VectorTo(t.PointAt(u, v+ev)).Scale(1 / (2 * ev))
	return du, dv
}

// NormalAt returns the unit normal (du×dv), aligned outward (positive radial component) to
// match the base [Cylinder.NormalAt] convention so a threaded face's orientation — and the
// divergence-theorem volume of a reversed (bore) face — comes out right.
func (t ThreadedCylinder) NormalAt(u, v float64) math.Vector3 {
	du, dv := t.DerivativesAt(u, v)
	n := du.Cross(dv)
	if n.LengthSquared() == 0 {
		return t.radial(u)
	}
	if n.Dot(t.radial(u)) < 0 { // keep it on the outward (away-from-axis) side, like a cylinder
		n = n.Scale(-1)
	}
	unit, err := math.UnitVector3FromVector(n)
	if err != nil {
		return t.radial(u)
	}
	return unit.AsVector()
}

// UDomain is the periodic angular range; VDomain is the threaded run.
func (t ThreadedCylinder) UDomain() (lo, hi float64) { return fullCircleDomain() }
func (t ThreadedCylinder) VDomain() (lo, hi float64) { return t.VMin, t.VMax }

// ParamAt inverts to the underlying cylinder's (u, v) — exact on the bounding circles (runout
// zero) and a good approximation elsewhere; the thread modulation is small.
func (t ThreadedCylinder) ParamAt(q math.Point3) (u, v float64) { return t.Cylinder.ParamAt(q) }
