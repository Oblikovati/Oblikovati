// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Test-only windows onto the curved-trim classification. They live in a _test.go file, so they are
// compiled into the package's tests and never ship — the external corpus test needs to see the
// unexported predicates, and nothing else does.

// CurvedTrimPredicateHits evaluates EVERY classification predicate on one face and returns, in a fixed
// order, the names of the kinds whose predicate answers true. A classification answers at most once; a
// ladder could answer several times and simply take the first, which is the property
// TestCurvedTrimKindsAreMutuallyExclusive exists to deny.
func CurvedTrimPredicateHits(f *topo.Face, q Quality) []string {
	s := f.Geometry()
	outer3D, holes3D := FaceOuterBoundary(f, q), faceHoleBoundaries(f, q)
	var hits []string
	for _, p := range curvedTrimPredicates(f, s, outer3D, holes3D, q) {
		if p.holds {
			hits = append(hits, p.kind.String())
		}
	}
	return hits
}

// curvedTrimVerdict is one predicate's answer for one face.
type curvedTrimVerdict struct {
	kind  curvedTrimKind
	holds bool
}

// curvedTrimPredicates pairs each special kind with its predicate's verdict on this face, in kind
// order so the report is byte-identical across runs.
func curvedTrimPredicates(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) []curvedTrimVerdict {
	return []curvedTrimVerdict{
		{kindConeApexFan, isConeApexTrim(f, s, outer3D, holes3D)},
		{kindSphereCapFan, isSphereCapTrim(f, s, outer3D, holes3D, q)},
		{kindSphereZoneBand, isSphereBeltTrim(f, s, outer3D, holes3D, q)},
		{kindSpherePatch, isSpherePatchTrim(f, s, outer3D, holes3D, q)},
		{kindRuledBandLoft, isRuledTwoRimBand(f, s, q)},
		{kindSpiricBand, isSpiricTubeBand(f, s, q)},
		{kindTwoRimHoledBand, isTwoRimHoledBand(s, holes3D)},
		{kindWedgeBand, isWedgeBandTrim(f, s, q)},
	}
}

// ClassifyCurvedTrimName is the name of the kind the classification itself selects for a face.
func ClassifyCurvedTrimName(f *topo.Face, q Quality) string {
	s := f.Geometry()
	return classifyCurvedTrim(f, s, FaceOuterBoundary(f, q), faceHoleBoundaries(f, q), q).String()
}

// UnchartedTrimKindName and ChartedTrimKindName name the two kinds no special predicate claims, so the
// corpus test can say which one it expects without reaching for the constants.
func UnchartedTrimKindName() string { return kindUncharted.String() }

// ChartedTrimKindName names the kind a face with a chart and no special shape takes.
func ChartedTrimKindName() string { return kindChart.String() }
