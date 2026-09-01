// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
)

// Rebuilding a body's faces to clean Class-A NURBS (M36-F02). An imported or boolean-derived
// face often carries a dense, unevenly-knotted freeform surface; RebuildFaceSurfaces refits
// each such surface to a fresh low-degree NURBS with a small, even control net (geom.RebuildSurface)
// and swaps it in with transform.ReplaceFaceSurface, keeping the face's loops/edges. Because the rebuild
// preserves the surface parameterization within tolerance, the face's trim loops still evaluate
// onto the new surface, so no edge surgery is needed.

// RebuildFaceSurfaces rebuilds every finite-domain (freeform) face of b to a clean
// degree-(uDeg×vDeg) NURBS with nu×nv control points, returning the rebuilt body and the worst
// geometric deviation across the rebuilt faces. Analytic faces with an unbounded parameter
// domain (planes, full cylinders) are already clean and pass through unchanged; it errors when
// the body has no rebuildable face.
func RebuildFaceSurfaces(b *topo.Body, uDeg, vDeg, nu, nv, samples int) (*topo.Body, float64, error) {
	out := b
	maxDev := 0.0
	rebuilt := 0
	for _, f := range b.Faces() {
		surf := f.Geometry()
		if !finiteSurfaceDomain(surf) {
			continue
		}
		clean, dev, err := geom.RebuildSurface(surf, uDeg, vDeg, nu, nv, samples)
		if err != nil {
			return nil, 0, fmt.Errorf("ops.RebuildFaceSurfaces: rebuild face %x: %w", f.ReferenceKey(), err)
		}
		if dev > maxDev {
			maxDev = dev
		}
		if out, err = transform.ReplaceFaceSurface(out, f.ReferenceKey(), clean); err != nil {
			return nil, 0, fmt.Errorf("ops.RebuildFaceSurfaces: swap face %x: %w", f.ReferenceKey(), err)
		}
		rebuilt++
	}
	if rebuilt == 0 {
		return nil, 0, fmt.Errorf("ops.RebuildFaceSurfaces: body has no rebuildable (finite-domain) face")
	}
	return out, maxDev, nil
}

// finiteSurfaceDomain reports whether a surface has a bounded, non-degenerate parameter box in
// both directions — the precondition for sampling it for a rebuild. Analytic surfaces with an
// infinite or periodic-unbounded domain (plane, full cylinder) return false.
func finiteSurfaceDomain(s geom.Surface) bool {
	ulo, uhi := s.UDomain()
	vlo, vhi := s.VDomain()
	return isFiniteSpan(ulo, uhi) && isFiniteSpan(vlo, vhi)
}

// isFiniteSpan reports whether [lo, hi] is finite and has positive width.
func isFiniteSpan(lo, hi float64) bool {
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) || stdmath.IsNaN(lo) || stdmath.IsNaN(hi) {
		return false
	}
	return hi > lo
}
