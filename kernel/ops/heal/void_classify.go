// SPDX-License-Identifier: GPL-2.0-only

package heal

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Void-shell classification (M48/C3 #3483/#3482): whether a shell is an internal void (cavity) decided
// analytically and ORIENTATION-INDEPENDENTLY, replacing the tessellated signed-volume sign. Void-ness is a
// property of the shell RELATIVE TO THE BODY — a lone closed shell is geometrically identical whether it
// bounds solid or void — so it is decided by classifying a point inside the shell's region against the body,
// not by a per-face outward orientation (which a curved full-wrap void wall cannot supply; see the
// containment fix in kernel/brep/classify_point.go).

// shellProbeWelds is the interior-probe offset in multiples of the shell's weld tolerance: enough to clear
// the on-surface band the classifier applies (so the probe reads a decisive inside/outside, not on-surface),
// yet a tiny fraction of any real region so a thin cavity still seeds inside itself.
const shellProbeWelds = 32 // tol:relative — probe offset as a multiple of the weld tolerance

// ShellIsVoidInBody reports whether shell sh is an internal void (cavity) of body b: a shell bounds a void
// iff a point strictly inside the shell's own region lies OUTSIDE the body's material. It seeds such an
// interior point and classifies it against b with the even–odd ray-parity classifier (brep.ClassifyPointTol)
// — no tessellated signed volume and no per-face outward sign. An open shell bounds no region and is never
// a void.
//
// Example: if heal.ShellIsVoidInBody(body, sh) { /* sh is a cavity skin — its region is empty */ }
func ShellIsVoidInBody(b *topo.Body, sh *topo.Shell) bool {
	if !sh.IsClosed() {
		return false
	}
	// The tolerance scale comes from the BODY box, not the shell box: a boundaryless-sphere shell's box can
	// be unbounded, which would poison both the probe offset and the on-surface band.
	onTol := geom.ResolutionForBox(b.RangeBox()).Weld()
	p, ok := shellInteriorPoint(sh, shellProbeWelds*onTol, onTol)
	if !ok {
		return false
	}
	return brep.ClassifyPointTol(b, p, onTol) == brep.Outside
}

// shellInteriorPoint seeds a point strictly inside the region shell sh bounds. From each face's interior it
// offsets ±eps along the surface normal and returns the first offset the shell's own ray-parity classifier
// confirms is inside — orientation-free, so it needs no correct outward sense; both signs are tried and the
// first that lands inside wins. onTol is the classifier's on-surface band; eps must exceed it.
func shellInteriorPoint(sh *topo.Shell, eps, onTol float64) (math.Point3, bool) {
	if eps <= 0 {
		return math.Point3{}, false
	}
	for _, f := range sh.Faces() {
		base, n, ok := faceInteriorProbe(f)
		if !ok {
			continue
		}
		for _, s := range []float64{eps, -eps} {
			cand := base.TranslateBy(n.AsVector().Scale(math.Scalar(s)))
			if brep.ClassifyShellPoint(sh, cand, onTol) == brep.Inside {
				return cand, true
			}
		}
	}
	return math.Point3{}, false
}

// faceInteriorProbe returns a point on the interior of face f and its surface normal there.
func faceInteriorProbe(f *topo.Face) (math.Point3, math.UnitVector3, bool) {
	surf := f.Geometry()
	u, v, ok := faceInteriorUV(f, surf)
	if !ok {
		return math.Point3{}, math.UnitVector3{}, false
	}
	nrm, err := math.UnitVector3FromVector(surf.NormalAt(u, v))
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, false
	}
	return surf.PointAt(u, v), nrm, true
}

// faceInteriorUV returns interior parameters of face f. For a trimmed face it projects the average of the
// face's edge midpoints onto the surface (an interior point for a convex face) and accepts it only when it
// lands inside the trim — so a hole or non-convex face whose average falls outside is skipped and another
// face tried. A BOUNDARYLESS face (a whole sphere or torus, its own closed shell — the curved cavity a
// per-face orientation cannot sign) has no edges, so it falls back to the surface's parameter-domain centre.
func faceInteriorUV(f *topo.Face, surf geom.Surface) (u, v float64, ok bool) {
	var sum math.Vector3
	n := 0
	for _, e := range f.Edges() {
		lo, hi := e.Geometry().Domain()
		sum = sum.Add(e.Geometry().PointAt((lo + hi) / 2).AsVector())
		n++
	}
	if n > 0 {
		avg := sum.Scale(math.Scalar(1.0 / float64(n)))
		pu, pv := surf.ParamAt(math.P3(avg.X, avg.Y, avg.Z))
		if brep.PointInFaceTrim(f, surf.PointAt(pu, pv)) {
			return pu, pv, true
		}
	}
	u0, u1 := surf.UDomain()
	v0, v1 := surf.VDomain()
	if stdmath.IsInf(u0, 0) || stdmath.IsInf(u1, 0) || stdmath.IsInf(v0, 0) || stdmath.IsInf(v1, 0) {
		return 0, 0, false
	}
	return (u0 + u1) / 2, (v0 + v1) / 2, true
}
