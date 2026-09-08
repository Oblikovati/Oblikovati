// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Test-only windows onto the curved-trim classification. They live in a _test.go file, so they are
// compiled into the package's tests and never ship — the external corpus test needs to see the
// unexported recognizers, and nothing else does.

// CurvedTrimRecognizerHits evaluates EVERY classification recognizer on one face and returns, in a
// fixed order, the names of the kinds whose recognizer answers true. A classification answers at most
// once; a ladder could answer several times and simply take the first, which is the property
// TestCurvedTrimKindsAreMutuallyExclusive exists to deny.
//
// kindSpherePatch is NOT here: it is the declared residual of the sphere family (a chart can hold a
// cap too, but a cap is not a patch), so an overlap test on it would be meaningless. What must not
// overlap on a sphere is the cap and the belt, and both are listed.
func CurvedTrimRecognizerHits(f *topo.Face, q Quality) []string {
	s := f.Geometry()
	outer3D, holes3D := FaceOuterBoundary(f, q), faceHoleBoundaries(f, q)
	var hits []string
	for _, v := range curvedTrimRecognizers(f, s, outer3D, holes3D, q) {
		if v.holds {
			hits = append(hits, v.kind.String())
		}
	}
	return hits
}

// curvedTrimVerdict is one recognizer's answer for one face.
type curvedTrimVerdict struct {
	kind  curvedTrimKind
	holds bool
}

// curvedTrimRecognizers pairs each special kind with its recognizer's verdict on this face, in kind
// order so the report is byte-identical across runs.
func curvedTrimRecognizers(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) []curvedTrimVerdict {
	sph, isSphere := sphereOf(s)
	_, isCone := coneApexTrimOf(f, s, outer3D, holes3D)
	_, isCap := sphereCapTrimOf(f, sph, outer3D, holes3D, q)
	_, isBelt := sphereBeltTrimOf(f, sph, q)
	_, isTube := spiricTubeTrimOf(f, s, q)
	_, isHoled := twoRimHoledTrimOf(f.Chart(), s, outer3D, holes3D)
	_, isWedge := wedgeBandTrimOf(f, s, q)
	return []curvedTrimVerdict{
		{kindConeApexFan, isCone},
		{kindSphereCapFan, isSphere && isCap},
		{kindSphereZoneBand, isSphere && isBelt},
		{kindRuledBandLoft, ruledTwoRimBandHolds(f, s, q)},
		{kindSpiricBand, isTube},
		{kindTwoRimHoledBand, isHoled},
		{kindWedgeBand, isWedge},
	}
}

// SphereCapRimFormHits evaluates the cap's three RIM FORMS independently on one face and returns the
// names of those that hold. The forms are meant to be disjoint by the boundary's own shape, and the
// cap recognizer refuses a face two of them claim rather than taking the first — this is what proves
// the refusal never has to fire.
func SphereCapRimFormHits(f *topo.Face, q Quality) []string {
	sph, isSphere := sphereOf(f.Geometry())
	if !isSphere {
		return nil
	}
	outer3D, holes3D := FaceOuterBoundary(f, q), faceHoleBoundaries(f, q)
	var hits []string
	for form := range sphereCapRimFormCount {
		if _, ok := sphereCapRimOfForm(form, f, sph, outer3D, holes3D, q); ok {
			hits = append(hits, form.String())
		}
	}
	return hits
}

// SphereCapRimIsRecognized reports whether the cap arm claims this boundary — the recognizer the
// deleted SphereCapFan used to wrap.
func SphereCapRimIsRecognized(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) bool {
	sph, isSphere := sphereOf(s)
	if !isSphere {
		return false
	}
	_, ok := sphereCapTrimOf(f, sph, outer3D, holes3D, q)
	return ok
}

// SphereCapFanMesh recognises and meshes a cap in one call, for tests that only want the mesh.
func SphereCapFanMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	sph, isSphere := sphereOf(s)
	if !isSphere {
		return nil, false
	}
	c, ok := sphereCapTrimOf(f, sph, outer3D, holes3D, q)
	if !ok {
		return nil, false
	}
	return buildSphereCap(c.sph, c.rim, c.axis, q), true
}

// SpherePatchMeshOf recognises and meshes an arc-bounded sphere patch in one call.
func SpherePatchMeshOf(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	sph, isSphere := sphereOf(s)
	if !isSphere {
		return nil, false
	}
	p, ok := spherePatchTrimOf(f, sph, outer3D, holes3D)
	if !ok {
		return nil, false
	}
	return SpherePatchMesh(p, outer3D, holes3D, q)
}

// SphereZoneBandFanOf recognises and meshes a belt in one call.
func SphereZoneBandFanOf(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	sph, isSphere := sphereOf(s)
	if !isSphere {
		return nil, false
	}
	b, ok := sphereBeltTrimOf(f, sph, q)
	if !ok {
		return nil, false
	}
	return SphereZoneBandFan(b, q), true
}

// ClassifyCurvedTrimName is the name of the kind the classification itself selects for a face.
func ClassifyCurvedTrimName(f *topo.Face, q Quality) string {
	s := f.Geometry()
	return classifyCurvedTrim(f, s, FaceOuterBoundary(f, q), faceHoleBoundaries(f, q), q).kind.String()
}

// UnchartedTrimKindName and ChartedTrimKindName name the two kinds no special recognizer claims, so
// the corpus test can say which one it expects without reaching for the constants.
func UnchartedTrimKindName() string { return kindUncharted.String() }

// ChartedTrimKindName names the kind a face with a chart and no special shape takes.
func ChartedTrimKindName() string { return kindChart.String() }

// SpherePatchTrimKindName names the sphere family's residual arm.
func SpherePatchTrimKindName() string { return kindSpherePatch.String() }

// TwoRimHoledBandVerdict reports, for one face, whether it has the two-rim HOLED band shape at all,
// whether it records a chart to mesh from, and whether the classification sends it to that arm. The
// three together are what says the conditioning gate sorts the corpus the way it claims to.
func TwoRimHoledBandVerdict(f *topo.Face, q Quality) (isShape, charted, toArm bool) {
	s := f.Geometry()
	outer3D, holes3D := FaceOuterBoundary(f, q), faceHoleBoundaries(f, q)
	_, isShape = twoRimHoledTrimOf(nil, s, outer3D, holes3D) // nil chart: the SHAPE, ungated
	_, toArm = twoRimHoledTrimOf(f.Chart(), s, outer3D, holes3D)
	return isShape, len(f.Chart()) > 0, toArm
}

// ChartMeshVerdict drives the chart-driven mesher directly and reports whether it accepted the face,
// its welded free-edge count and the rim it was given — the three numbers its acceptance gate compares.
func ChartMeshVerdict(f *topo.Face, q Quality) (ok bool, free, rim int, area float64) {
	s := f.Geometry()
	m, ok := chartFaceMesh(f, s, q)
	r, _ := newChartRegion(f, s)
	rim = chainSegmentCount(chartBoundaryChains(f, s, r, q))
	if m != nil {
		free, area = WeldedFreeEdgeCount(m), m.Area()
	}
	return ok, free, rim, area
}

// TwoRimCorridorProbe reports the closest approach between two lens windows and the boundary chord the
// gate compares it against.
func TwoRimCorridorProbe(f *topo.Face, q Quality) (gap, chord float64) {
	_, lenses := splitWrappingHoles(f.Geometry(), faceHoleBoundaries(f, q))
	return closestLensApproach(lenses), meanChainChord(FaceOuterBoundary(f, q))
}
