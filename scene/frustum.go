// SPDX-License-Identifier: GPL-2.0-only

package scene

import (
	stdmath "math"

	"oblikovati.org/math"
)

// cullPlane is a half-space with an INWARD normal: a point p is inside when n·p + d ≥ 0.
type cullPlane struct {
	n math.Vector3
	d float64
}

// keeps reports whether the box is NOT entirely behind the plane. The box corner farthest along
// the inward normal (its support point) is the one most likely inside, so if even that corner is
// behind the plane the whole box is outside. This is the conservative half of the AABB/frustum
// test — it never reports a straddling box as outside.
func (pl cullPlane) keeps(b math.Box) bool {
	support := b.FarthestPoint(pl.n)
	return pl.n.Dot(support.AsVector())+pl.d >= 0
}

// Frustum is the camera's view volume as inward-facing half-spaces: a near plane plus the four
// sides. The far plane is omitted on purpose — the viewport's far distance is dynamic (skybox /
// scene dependent) and culling distant geometry is not the win F1 targets. A box is visible when
// it is inside every plane; the test is conservative (a box straddling a frustum corner may be
// kept though off-screen, but a visible box is never dropped, so nothing pops). See M34-F1.
type Frustum struct {
	planes []cullPlane
}

// IntersectsBox reports whether the world AABB is potentially visible — the broad-phase a
// renderer uses to skip off-screen instances before the GPU upload.
//
//	if cam.Frustum().IntersectsBox(inst.WorldBox()) { draw(inst) }
func (f Frustum) IntersectsBox(b math.Box) bool {
	if b.IsEmpty() {
		return false
	}
	for _, pl := range f.planes {
		if !pl.keeps(b) {
			return false
		}
	}
	return true
}

// Frustum returns the camera's view volume for culling: a near plane at the eye plus four side
// planes — through the eye for perspective, parallel to the view for orthographic. All normals
// face inward.
func (c Camera) Frustum() Frustum {
	fwd := c.Forward()
	right := unit(fwd.Cross(c.Up))
	up := right.Cross(fwd) // re-orthogonalized against forward/right
	near := cullPlane{n: fwd, d: -fwd.Dot(c.Eye.AsVector())}
	if c.Orthographic {
		return Frustum{planes: c.orthoPlanes(right, up, near)}
	}
	return Frustum{planes: c.perspectivePlanes(fwd, right, up, near)}
}

// perspectivePlanes builds the near plus four side planes from the corner-ray directions (forward
// ± right·tanH ± up·tanV), each oriented inward by sidePlane.
func (c Camera) perspectivePlanes(fwd, right, up math.Vector3, near cullPlane) []cullPlane {
	tanV := stdmath.Tan(c.FOV / 2)
	tanH := tanV * c.aspect()
	tl := fwd.Sub(right.Scale(tanH)).Add(up.Scale(tanV))
	tr := fwd.Add(right.Scale(tanH)).Add(up.Scale(tanV))
	bl := fwd.Sub(right.Scale(tanH)).Sub(up.Scale(tanV))
	br := fwd.Add(right.Scale(tanH)).Sub(up.Scale(tanV))
	return []cullPlane{
		near,
		c.sidePlane(bl, tl, fwd), // left
		c.sidePlane(tr, br, fwd), // right
		c.sidePlane(tl, tr, fwd), // top
		c.sidePlane(br, bl, fwd), // bottom
	}
}

// sidePlane builds a frustum side plane through the eye spanned by two corner-ray directions,
// orienting its normal toward the interior (the forward axis lies inside every side plane, so a
// normal with a positive forward component points inward).
func (c Camera) sidePlane(a, b, fwd math.Vector3) cullPlane {
	n := a.Cross(b)
	if n.Dot(fwd) < 0 {
		n = n.Negate()
	}
	return cullPlane{n: n, d: -n.Dot(c.Eye.AsVector())}
}

// orthoPlanes builds the four parallel side planes of an orthographic view, offset from the eye by
// the half-extent the FOV spans at the target depth (so toggling projection keeps the same scale).
func (c Camera) orthoPlanes(right, up math.Vector3, near cullPlane) []cullPlane {
	dist := float64(c.Eye.DistanceTo(c.Target))
	halfH := stdmath.Tan(c.FOV/2) * dist
	halfW := halfH * c.aspect()
	planeAt := func(inward, offset math.Vector3) cullPlane {
		p0 := c.Eye.TranslateBy(offset).AsVector()
		return cullPlane{n: inward, d: -inward.Dot(p0)}
	}
	return []cullPlane{
		near,
		planeAt(right, right.Scale(-halfW)), // left boundary, inward = +right
		planeAt(right.Negate(), right.Scale(halfW)), // right boundary, inward = −right
		planeAt(up, up.Scale(-halfH)),               // bottom boundary, inward = +up
		planeAt(up.Negate(), up.Scale(halfH)),       // top boundary, inward = −up
	}
}

// aspect is the viewport width/height ratio, defaulting to 1 for a degenerate (zero-height) size.
func (c Camera) aspect() float64 {
	if c.Height <= 0 {
		return 1
	}
	return float64(c.Width) / float64(c.Height)
}
