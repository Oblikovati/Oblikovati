// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The canal far-end THROUGH-VERTEX extension edges (M6' C4 W3c/F2, derivation: .superpowers/sdd/
// canal-far-runout-derivation.md §5-6). A canal arm's terminal section ends on its F_far face, but its
// wall-side host rail can run PAST F_far's far vertex q — landing off F_far's original loop (measured F1
// floor: the wall far span "will not close — outer ends off the bitten loop"). The fix (derivation §5):
// a new edge along host∩F_far from q to that off-loop foot, keeping q as a loop vertex. It is a CO-CIRCULAR
// arc when F_far ⊥ the cylinder axis (wall∩{z=80}, s_4, sweep asin(1/9)) or a COLLINEAR segment when F_far
// ∥ the axis (wall∩{x=80}, s_5, length r). ONE object is shared between the F_far imprint face and the
// wall's far path (identity contract §7.3); both sample it identically. s_10 (both hosts planar, no
// cylinder host) needs no extension — its terminal's feet already land on the F_far loop (verbatim splice).

// canalArmExtension builds the arm's through-vertex extension edge (derivation §5), returning it with the
// host face (the wall cylinder) it lies on. Returns ok=false (extHost nil, no extension) when the arm has
// no cylinder host (s_10), F_far is unavailable (a bare-face fixture), the section cannot be built, or the
// wall foot coincides with the far vertex q (the B3 zero-length collapse → the verbatim splice). The
// extension is oriented q→foot; its foot end equals the wall host rail's outer end (shared-edge identity).
func canalArmExtension(arm edgeFillet, h0, h1 endSeg, wi cornerWeld, res Resolution) (endSeg, *topo.Face, bool) {
	ffar, ok := canalFarFace(arm, wi)
	if !ok {
		return endSeg{}, nil, false
	}
	host, cyl, foot, ok := cylinderHostFoot(arm, h0, h1)
	if !ok {
		return endSeg{}, nil, false
	}
	tol := res.Weld() * wi.radius
	q := farEndVertex(arm.edge, wi.center).Point()
	if float64(q.DistanceTo(foot)) <= tol {
		return endSeg{}, nil, false // wall foot ON the far vertex — no extension (verbatim splice)
	}
	ext, ok := canalWallExtension(cyl, ffar, q, foot, tol)
	if !ok {
		return endSeg{}, nil, false
	}
	return ext, host, true
}

// cylinderHostFoot returns the arm's CYLINDER (wall) host, its surface, and its rail's outer foot — the
// side whose runout falls off the F_far loop and needs a through-vertex extension (both N7 extensions are
// wall-side). h0 lies on arm.a, h1 on arm.b (canalArmHostRails' order), so the foot is the matching rail's
// outer end. Returns ok=false when neither host is a cylinder (s_10, both hosts planar → no extension).
func cylinderHostFoot(arm edgeFillet, h0, h1 endSeg) (*topo.Face, geom.Cylinder, math.Point3, bool) {
	if cyl, ok := arm.a.Geometry().(geom.Cylinder); ok {
		return arm.a, cyl, h0.from, true
	}
	if cyl, ok := arm.b.Geometry().(geom.Cylinder); ok {
		return arm.b, cyl, h1.from, true
	}
	return nil, geom.Cylinder{}, math.Point3{}, false
}

// canalWallExtension builds the extension edge on the cylinder host ∩ F_far (derivation §5), from the far
// vertex q to the off-loop foot: a CO-CIRCULAR arc when F_far ⊥ the cylinder axis (the intersection is a
// circle of radius cyl.Radius, e.g. s_4's z=80) or a COLLINEAR segment when F_far ∥ the axis (the
// intersection is a ruling, e.g. s_5's x=80). Declines (false) when F_far is neither ⊥ nor ∥ the axis
// (out of the closed-form scope) or q/foot do not lie on that section (a defect, never snapped).
func canalWallExtension(cyl geom.Cylinder, ffar geom.Plane, q, foot math.Point3, tol float64) (endSeg, bool) {
	if planePerpToDir(ffar, cyl.AxisDir) {
		return coCircularExtension(cyl, ffar, q, foot, tol)
	}
	if planeContainsDir(ffar, cyl.AxisDir) {
		return collinearExtension(cyl, q, foot, tol)
	}
	return endSeg{}, false
}

// coCircularExtension builds the co-circular extension arc on cyl∩ffar (a circle of radius cyl.Radius
// centred where the axis pierces ffar), the SHORT arc from q to foot through the circle's angular
// midpoint. Both q and foot must lie on that circle (dist to centre = cyl.Radius within tol), else the
// section is not the assumed circle and it declines.
func coCircularExtension(cyl geom.Cylinder, ffar geom.Plane, q, foot math.Point3, tol float64) (endSeg, bool) {
	centre := axisPointOnPlane(cyl, ffar)
	if stdmath.Abs(float64(centre.DistanceTo(q))-cyl.Radius) > tol ||
		stdmath.Abs(float64(centre.DistanceTo(foot))-cyl.Radius) > tol {
		return endSeg{}, false
	}
	mid := arcMidBetween(centre, cyl.Radius, q, foot)
	arc, err := geom.Arc3dByThreePoints(q, mid, foot)
	if err != nil {
		return endSeg{}, false
	}
	return endSeg{from: q, to: foot, curve: arc, mid: mid, arc: true}, true
}

// collinearExtension builds the collinear extension segment on cyl∩ffar (a ruling parallel to the axis),
// from q to foot. It asserts q and foot are on the same ruling — both on the cylinder (radial distance
// cyl.Radius) and q→foot parallel to the axis — else the foot is not on host∩F_far and it declines.
func collinearExtension(cyl geom.Cylinder, q, foot math.Point3, tol float64) (endSeg, bool) {
	if !onCylinderRuling(cyl, q, tol) || !onCylinderRuling(cyl, foot, tol) {
		return endSeg{}, false
	}
	dir, err := math.UnitVector3FromVector(q.VectorTo(foot))
	if err != nil {
		return endSeg{}, false
	}
	if stdmath.Abs(float64(dir.AsVector().Dot(cyl.AxisDir.AsVector()))) < 1-sinFloor {
		return endSeg{}, false // q→foot not along the axis — not a single ruling
	}
	return endSeg{from: q, to: foot}, true
}

// onCylinderRuling reports whether p lies on the cylinder wall (its perpendicular distance from the axis
// equals cyl.Radius within tol).
func onCylinderRuling(cyl geom.Cylinder, p math.Point3, tol float64) bool {
	axis := cyl.AxisDir.AsVector()
	d := cyl.Origin.VectorTo(p)
	perp := d.Sub(axis.Scale(d.Dot(axis)))
	return stdmath.Abs(float64(perp.Length())-cyl.Radius) <= tol
}

// axisPointOnPlane is where the cylinder axis pierces the axis-perpendicular plane ffar — the centre of
// the circle cyl∩ffar the co-circular extension arc lives on. The line∩plane parameter is scale-invariant
// in the plane normal, so ffar.Normal() need not be unit. Requires |axis·n̂| ≠ 0 (guaranteed by the ⊥
// classification in canalWallExtension).
func axisPointOnPlane(cyl geom.Cylinder, ffar geom.Plane) math.Point3 {
	n := ffar.Normal()
	axis := cyl.AxisDir.AsVector()
	t := float64(cyl.Origin.VectorTo(ffar.Origin).Dot(n)) / float64(axis.Dot(n))
	return cyl.Origin.TranslateBy(axis.Scale(math.Scalar(t)))
}

// planeContainsDir reports whether direction d runs PARALLEL to plane pl (its normal ⊥ d, |n̂·d̂| ≈ 0) —
// the axis-parallel case where cyl∩pl is a ruling and the extension is a collinear segment. The sibling
// of planePerpToDir; both use the scale-free sinFloor (ADR-0042 angular exemption).
func planeContainsDir(pl geom.Plane, d math.UnitVector3) bool {
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return false
	}
	return stdmath.Abs(float64(n.AsVector().Dot(d.AsVector()))) < sinFloor
}

// extensionsOnHost returns the through-vertex extension edges whose host is `host` (the wall) — the arms
// whose wall-side runout ran off host's window loop (derivation §5). The wall's far path anchors on its
// window loop AUGMENTED by these (removing the F1-era "outer ends off the bitten loop" floor).
func extensionsOnHost(host *topo.Face, bundles []canalArmBundle) []endSeg {
	var out []endSeg
	for _, b := range bundles {
		if b.extHost == host {
			out = append(out, b.ext)
		}
	}
	return out
}

// canalFarSpan closes the wall's inner bite with its surviving far span, anchored on the window loop
// AUGMENTED by the through-vertex extensions (derivation §5-6): each inner-bite outer end (a wall rail's
// off-loop foot) reaches the loop via its extension edge (foot↔far-vertex q), then the far path runs the
// original loop between the two far vertices, avoiding the bitten trihedral corner. With no extensions it
// is exactly farPathSegs (the on-loop feet case, unchanged). Runs outerB→…→outerA (to close after `inner`,
// which runs outerA→…→outerB). Declines (false) when an outer end neither anchors on the loop nor has a
// matching extension, or the far path does not close between the anchors.
func canalFarSpan(segs []endSeg, outerA, outerB math.Point3, exts []endSeg, bittenV math.Point3, tol float64) ([]endSeg, bool) {
	if len(exts) == 0 {
		return farPathSegs(segs, outerB, outerA, bittenV, tol)
	}
	extB, qB := resolveOuterAnchor(exts, outerB, tol)
	extA, qA := resolveOuterAnchor(exts, outerA, tol)
	ring, ok := farPathSegs(segs, qB, qA, bittenV, tol) // the original loop qB→qA, avoiding the corner
	if !ok {
		return nil, false
	}
	far := prependExtension(extB, ring) // outerB→qB→…
	return appendExtension(far, extA), true
}

// resolveOuterAnchor maps an inner-bite outer end to its loop ANCHOR: when an extension's foot equals the
// outer end, the anchor is the extension's far vertex q (and the extension bridges outer↔q); otherwise the
// outer end is assumed to lie on the loop itself and is its own anchor (the on-loop foot case). Returns the
// matching extension (zero endSeg when none) and the anchor point farPathSegs walks between.
func resolveOuterAnchor(exts []endSeg, outer math.Point3, tol float64) (endSeg, math.Point3) {
	for _, e := range exts {
		if float64(e.to.DistanceTo(outer)) <= tol {
			return e, e.from
		}
	}
	return endSeg{}, outer
}

// prependExtension prepends the far span with the outerB→qB traversal of ext (ext is stored qB→outerB, so
// it is reversed here); a zero ext (the on-loop case) leaves the ring untouched.
func prependExtension(ext endSeg, ring []endSeg) []endSeg {
	if ext.curve == nil && ext.from == ext.to {
		return ring
	}
	return append([]endSeg{reversedEndSeg(ext)}, ring...)
}

// appendExtension appends the far span with the qA→outerA traversal of ext (stored qA→outerA, used as-is);
// a zero ext (the on-loop case) leaves the span untouched.
func appendExtension(far []endSeg, ext endSeg) []endSeg {
	if ext.curve == nil && ext.from == ext.to {
		return far
	}
	return append(far, ext)
}
