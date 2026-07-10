// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// SurfaceBoundConstraint is a 3D geometric constraint bound to an external surface (a part face) by
// reference key. Because model/sketch cannot resolve a face itself, such a constraint restores
// frozen carrying only the key; a host rebinds a live surface source after a reload — exactly as
// projected geometry rebinds (#1839). While unbound the constraint is inactive.
type SurfaceBoundConstraint interface {
	SurfaceRef() string
	BindSurface(SurfaceSource)
}

// OnFace3D holds a 3D sketch point on a referenced surface (a part face): the solver drives the
// point's signed distance to the surface to zero, removing one degree of freedom (Inventor
// OnFaceConstraint3D, #1839). The surface is reached through the [SurfaceSource] seam so the
// constraint tracks recompute. It contributes no equation while unbound (restored but not yet
// rebound) or when the reference is lost, so a saved sketch loads without a dangling face pin.
type OnFace3D struct {
	constraintBase
	P       *Point3D
	surface SurfaceSource
	ref     string // the face reference key, kept for serialize + rebind
}

// NewOnFace3D constrains point p onto the surface source; ref is the face reference key persisted
// for a later [OnFace3D.BindSurface].
//
//	c := NewOnFace3D(p, compdef.NewFaceRefSource(part, key), key)
func NewOnFace3D(p *Point3D, surface SurfaceSource, ref string) *OnFace3D {
	return &OnFace3D{constraintBase: newConstraint(), P: p, surface: surface, ref: ref}
}

// SurfaceRef returns the constrained face's reference key (for serialize + rebind).
func (c *OnFace3D) SurfaceRef() string { return c.ref }

// BindSurface attaches a live surface source after a reload, reactivating the constraint.
func (c *OnFace3D) BindSurface(s SurfaceSource) { c.surface = s }

// Variables are the constrained point's three coordinate DOFs — none while inactive, so an unbound
// or lost constraint drops out of the solve entirely (mirrors a driven dimension).
func (c *OnFace3D) Variables() []*math.Scalar {
	if c.activeSurface() == nil {
		return nil
	}
	return []*math.Scalar{&c.P.X, &c.P.Y, &c.P.Z}
}

// Residuals is the point's signed distance to the surface, driven to zero; nil while inactive.
func (c *OnFace3D) Residuals() []float64 {
	surf := c.activeSurface()
	if surf == nil {
		return nil
	}
	p := c.P.Position()
	u, v, foot := geom.ClosestPointOnSurface(surf, p)
	return []float64{float64(foot.VectorTo(p).Dot(surf.NormalAt(u, v)))}
}

// Partials is the surface unit normal at the closest foot — the exact gradient of the point's
// signed distance to the surface (∂d/∂p = n); nil while inactive.
func (c *OnFace3D) Partials() [][]float64 {
	surf := c.activeSurface()
	if surf == nil {
		return nil
	}
	u, v, _ := geom.ClosestPointOnSurface(surf, c.P.Position())
	n := surf.NormalAt(u, v)
	return [][]float64{{float64(n.X), float64(n.Y), float64(n.Z)}}
}

// activeSurface resolves the bound source's current surface, or nil when unbound/lost.
func (c *OnFace3D) activeSurface() geom.Surface {
	if c.surface == nil {
		return nil
	}
	return c.surface.Surface()
}
