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
// long triangles across the curvature and get the area wrong.) A trim that is neither an iso
// rectangle nor a shape one of the wrapping meshers recognises is meshed from the parametric trim the
// face CARRIES (ADR-0061/ADR-0063, chart_face_mesh.go): region from the chart, points from the shared
// edges. Only a face that carries no chart at all still falls to the surface's whole domain, and that
// degradation is reported (trim_ignored.go).
//
// This file is the ROUTER. The pieces it dispatches to are split by responsibility:
// tessellate_trim_grid.go (the structured (u,v) grid construction),
// tessellate_trim_boundary.go (the trim-boundary → (u,v) imprint and the boundary patch mesher),
// tessellate_trim_special.go (the surface-specific special-case meshers and the cone-apex fans),
// tessellate_trim_policy.go (the full-domain quality/deflection fallback),
// chart_face_mesh.go (the chart-driven covering-space mesher this router ends at).

// tessellateCurvedFace meshes a curved face's trimmed region (see file doc).
func tessellateCurvedFace(f *topo.Face, q Quality) *Mesh {
	s := f.Geometry()
	if m := splineFaceMesh(f, s, q); m != nil {
		return m // M25: a B-spline face via the metric-aware (u,v) triangulation
	}
	outer3D := FaceOuterBoundary(f, q)
	holes3D := faceHoleBoundaries(f, q)
	if len(outer3D) < 3 {
		// A face with hole loops but NO outer loop wraps the whole closed surface minus those windows —
		// the genus-1 complement of a cap (a torus minus an oval, a sphere minus a lens). Its region is
		// exactly what the chart records, so it is meshed from the chart (ADR-0061/ADR-0063).
		return chartedTrimMesh(f, s, q, "")
	}
	m, special, refused := specialCurvedMesh(f, s, outer3D, holes3D, q)
	if special {
		return m // a cone-apex/sphere fan or cap, sphere box-cut patch, or notched-rim band
	}
	outerUV, holesUV, ok := ToUVLoops(s, outer3D, holes3D)
	if !ok {
		return meshSeamCrossingFace(f, s, outer3D, holes3D, q, refused) // a loop wrapping the seam: band/cap fallbacks
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
// closedDomainMesh; and a face no wrapping mesher reduces is meshed from the region it CARRIES
// (chartedTrimMesh / chartFaceMesh, ADR-0061/ADR-0063), not from the surface's whole domain — only a
// face recording no chart still falls that far, and that degradation is reported.
//
// That last rule now covers the SINGLY-periodic surfaces too. A cylinder band whose notched rim steps
// axially is not a v(u) graph, so the ruled loft declines it and it used to land on the best-fit-plane
// CDT, which flattens a wrapping band: measured on the merged cocylindrical wall a D-prism leaves on a
// cylinder of its own radius, 61 free edges. A charted band is meshed from its chart like every other
// charted face; the best-fit-plane CDT stays for the sphere cap straddling the pole, which records none.
func meshSeamCrossingFace(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality, refused string) *Mesh {
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
		// A doubly-periodic band that isn't two circles + a seam: the chart says which region it is.
		return chartedTrimMesh(f, s, q, refused)
	}
	if IsPeriodic(s.UDomain()) != IsPeriodic(s.VDomain()) {
		return singlyPeriodicWrapMesh(f, s, outer3D, holes3D, q)
	}
	// A seam-wrapping face no wrapping mesher reduced: the chart carries its region (ADR-0063), so the
	// chart-driven mesher takes it; only a face that carries NO chart falls through to the defect.
	return chartedTrimMesh(f, s, q, refused)
}

// singlyPeriodicWrapMesh meshes a seam-wrapping face on a cylinder, cone or sphere that no wrapping
// mesher reduced: from the region it RECORDS when it carries one, else through the best-fit-plane CDT,
// which is the sphere cap straddling the pole (the full-domain grid tears there) and which reports the
// wall wrap it could not mesh.
func singlyPeriodicWrapMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) *Mesh {
	if m, ok := chartFaceMesh(f, s, q); ok {
		return m
	}
	m := trimmedPatchMesh(s, outer3D, holes3D)
	recordUnmeshedWallWrap(m, s, outer3D, len(holes3D))
	return m
}

// chartedTrimMesh is the single classification at the end of the curved-face router: a trimmed face
// that CARRIES a parametric trim is meshed from it (region from the chart, points from the shared
// edges); one that carries none — or whose chart the mesher cannot take — falls to the surface's whole
// parametric domain, and that degradation is reported, never silent (ADR-0061 stage 5).
func chartedTrimMesh(f *topo.Face, s geom.Surface, q Quality, refused string) *Mesh {
	if m, ok := chartFaceMesh(f, s, q); ok {
		return m
	}
	return recordIgnoredTrim(fullDomainGridMesh(s, q), s, len(f.Loops()), refused)
}
