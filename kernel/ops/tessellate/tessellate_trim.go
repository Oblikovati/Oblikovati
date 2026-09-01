// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Trimmed curved-face tessellation (third piece of the curved-B-rep stack). A curved
// face is meshed over its trim region, not the surface's whole UV domain, by mapping its
// boundary loops (shared edge discretization, so they match neighbours exactly) into
// (u,v) space with Surface.ParamAt. When that region is an iso-aligned rectangle whose
// opposite edges sample identically — the shape of every analytic fillet/blend face and
// of axial cylinder/cone walls — a STRUCTURED grid of thin iso quads tessellates it
// watertight and with correct curved area. (Ear-clipping the boundary instead would chord
// long triangles across the curvature and get the area wrong.) Anything else — holes, a
// non-rectangular trim, a seam-crossing periodic loop — falls back to the full-domain grid
// (a follow-up; needs a constrained triangulation).
//
// This file is the ROUTER. The pieces it dispatches to are split by responsibility:
// tessellate_trim_grid.go (the structured (u,v) grid construction),
// tessellate_trim_boundary.go (the trim-boundary → (u,v) imprint and the boundary patch mesher),
// tessellate_trim_special.go (the surface-specific special-case meshers and the cone-apex fans),
// tessellate_trim_policy.go (the full-domain quality/deflection fallback).

// tessellateCurvedFace meshes a curved face's trimmed region (see file doc).
func tessellateCurvedFace(f *topo.Face, q Quality) *Mesh {
	s := f.Geometry()
	if m := splineFaceMesh(f, s, q); m != nil {
		return m // M25: a B-spline face via the metric-aware (u,v) triangulation
	}
	outer3D := FaceOuterBoundary(f, q)
	holes3D := faceHoleBoundaries(f, q)
	if t, ok := s.(geom.Torus); ok && len(outer3D) < 3 && len(holes3D) > 0 {
		// A torus face with hole loops but NO outer loop wraps the whole closed surface minus the holes —
		// the genus-1 COMPLEMENT of an oval cap (a torus-minus-disk). The full-domain grid would ignore the
		// hole; torusComplementMesh charts the torus minus the oval window instead (Oblikovati#1375).
		return torusComplementMesh(t, holes3D, q)
	}
	if len(outer3D) < 3 {
		return fullDomainGridMesh(s, q)
	}
	if m, special := specialCurvedMesh(f, s, outer3D, holes3D, q); special {
		return m // a cone-apex/sphere fan or cap, sphere box-cut patch, or notched-rim band
	}
	outerUV, holesUV, ok := ToUVLoops(s, outer3D, holes3D)
	if !ok {
		return meshSeamCrossingFace(f, s, outer3D, holes3D, q) // a loop wrapping the seam: band/cap fallbacks
	}
	if us, vs, isRect := isoRectangleGrid(outerUV); len(holesUV) == 0 && isRect {
		return structuredGridMesh(s, us, vs) // cylinder/cone wall, fillet face: exact area
	}
	if us, vs, skip, isCells := IsoRectilinearGrid(outerUV); len(holesUV) == 0 && isCells {
		// A band the obstacle imprint notched (fillet_band_imprint.go): still bounded entirely by
		// iso-lines, so it is a union of grid cells and needs no triangulator — see
		// tessellate_rectilinear.go for what the generic CDT does with it instead.
		return structuredGridMeshSkip(s, us, vs, skip)
	}
	return nonRectangularMesh(s, q, outer3D, holes3D, outerUV, holesUV)
}

// splineFaceMesh meshes a B-spline face through the metric-aware (u,v) triangulation (M25), or nil when
// the face is not a B-spline (so the caller falls through to the analytic-surface paths).
func splineFaceMesh(f *topo.Face, s geom.Surface, q Quality) *Mesh {
	if _, isSpline := s.(geom.BSplineSurface); !isSpline {
		return nil
	}
	// The CLOSED elliptic-rim canal band (two closed rails + a seam used twice) has no usable planar
	// (u,v) trim, so it is lofted rail-to-rail instead. Gated on that exact loop shape, which no open
	// canal arm or corner patch has.
	if m, ok := canalRimBandMesh(f, s, q); ok {
		return m
	}
	// The PINCHED canal band (EllipticalCylinder∧Cone host tangency, tolblend B4..C3): its
	// cross-section collapses to a point, so the trim/pcurve paths degenerate there; it is lofted
	// rail-to-rail with a shared pinch vertex instead (W-F, pinched_band_loft.go). Gated on the
	// zero-width v-end column + all-iso boundary — no other B-spline face has that shape.
	if m, ok := pinchedCanalBandMesh(f, s, q); ok {
		return m
	}
	// A closed-in-u (periodic) B-spline face whose trim straddles the seam tangles the planar seam-cut
	// loop; the covering-space periodic CDT un-seams it. It defers (nil,false) for the ordinary open patch.
	if m, ok := periodicNurbsFaceMesh(f, q); ok {
		return m
	}
	return NurbsPcurveMesh(f, q)
}

// meshSeamCrossingFace meshes a curved face whose boundary loop wraps the periodic seam (so toUVLoops
// can't unwrap it): a full cylinder/cone side or a torus rim-fillet band closes the seam watertight via
// closedDomainMesh; a singly-periodic sphere cap straddling the pole goes through the best-fit-plane CDT
// (the full-domain grid tears there); a doubly-periodic torus we can't reduce keeps the full-domain grid.
func meshSeamCrossingFace(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) *Mesh {
	if us, vs, isBand := periodicBandGrid(s, outer3D, holes3D); isBand {
		if m, ok := unequalRimBandMesh(f, s, bandGridStations(s, us, vs), q); ok {
			return m // rims at DIFFERENT station counts: loft rim-to-rim so each keeps its own shared-edge
			// discretization — the one grid would re-tile the coarser rim and crack it (band_rim_stations.go)
		}
		return closedDomainMesh(s, us, vs) // full cylinder/cone side with circular rims: grid the period
	}
	if m, ok := HoledConicWallMesh(s, outer3D, holes3D, q); ok {
		return m // a drilled cylinder/cone wall: full-period side with lens holes — unroll + metric CDT
	}
	if m, ok := saddleBandLoftMesh(f, s, q); ok {
		return m // a cylinder/cone band with non-circular (saddle) rims — a crossing cylinder: loft v(u)
	}
	if _, _, isBand := doublyPeriodicBandGrid(s, outer3D, holes3D); isBand {
		if m, ok := closedBandLoftMesh(f, s, q); ok {
			return m // torus rim-fillet band: loft so each edge ring keeps its own (differing) tessellation
		}
		if m, ok := torusTubeBandLoftMesh(f, s, q); ok {
			return m // spiric closed-rim HOST (J3/A4): a TUBE-wrapping band (meridian circle + canal rail + seam)
		}
		return fullDomainGridMesh(s, q) // shouldn't reach: a doubly-periodic band that isn't two circles + a seam
	}
	if IsPeriodic(s.UDomain()) != IsPeriodic(s.VDomain()) {
		m := trimmedPatchMesh(s, outer3D, holes3D) // sphere cap on the pole: CDT in the best-fit plane
		recordUnmeshedWallWrap(m, s, outer3D, len(holes3D))
		return m
	}
	return fullDomainGridMesh(s, q) // doubly-periodic / aperiodic seam face we can't reduce
}
