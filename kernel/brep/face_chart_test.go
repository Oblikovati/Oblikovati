// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/test-utilities/brepfixture"
)

// The gate for the carried chart (ADR-0063), judged by an oracle that reads no chart at all: a point on
// the operand's surface belongs to the result's face exactly when the boolean's own keep rule says so
// against the tool's half-space. The predicate is the boolean's premise, so a face whose chart disagrees
// with it covers the wrong region whatever else it satisfies.
//
// It is the measurement that chose this design over the reassembly ADR-0062 tried: on the oblique
// figure-eight the carried chart is right at every sample where the reassembled reading was wrong at
// 108 of 218.

// chartHalfSpaceCase is one boolean whose curved face the tool's half-space judges.
type chartHalfSpaceCase struct {
	name   string
	body   *topo.Body
	onFace func(math.Point3) (want, decided bool)
}

func TestFaceChartCoversTheKeptSide(t *testing.T) {
	t.Parallel()
	for _, tc := range chartHalfSpaceCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			charted, wrong, samples := 0, 0, 0
			for _, f := range tc.body.Faces() {
				contours := f.Chart()
				if contours == nil || !chartProbeSurface(f.Geometry()) {
					continue
				}
				charted++
				n, bad := probeChart(f, contours, tc.onFace)
				samples, wrong = samples+n, wrong+bad
			}
			if charted == 0 {
				t.Fatalf("%s: no curved face carries a chart — the producer stopped recording one", tc.name)
			}
			if wrong > 0 {
				t.Errorf("%s: the chart puts %d of %d surface samples on the wrong side of the tool", tc.name, wrong, samples)
			}
		})
	}
}

// probeChart samples a face's chart window and counts the points the chart places against the oracle.
func probeChart(f *topo.Face, contours [][]math.Point2, onFace func(math.Point3) (bool, bool)) (samples, wrong int) {
	u0, u1, v0, v1, ok := polyBounds(contours)
	if !ok {
		return 0, 0
	}
	uPer, vPer := surfacePeriodic(f.Geometry())
	for i := 1; i < chartProbeSteps; i++ {
		for j := 1; j < chartProbeSteps; j++ {
			u := u0 + (u1-u0)*float64(i)/chartProbeSteps
			v := v0 + (v1-v0)*float64(j)/chartProbeSteps
			want, decided := onFace(f.Geometry().PointAt(u, v))
			if !decided {
				continue
			}
			samples++
			if chartContains(contours, math.P2(u, v), uPer, vPer) != want {
				wrong++
			}
		}
	}
	return samples, wrong
}

// chartProbeSteps is the grid across a chart window. It is a sampling density, not a tolerance: the
// oracle leaves the band around the cut undecided, so no sample sits on a boundary.
const chartProbeSteps = 16

// chartProbeSurface limits the probe to the operand surfaces the half-space oracle is stated for.
func chartProbeSurface(s geom.Surface) bool {
	switch s.(type) {
	case geom.Torus, geom.Cylinder:
		return true
	}
	return false
}

// halfSpaceKeepRule is the oracle: the tool is a box whose only face inside the operand is the plane
// axis·p = cut, so a point on the operand belongs to the result exactly when the op keeps that side.
// Points within chartProbeBand of the cut are undecided — the boundary is not what this measures.
func halfSpaceKeepRule(op Op, axis math.Vector3, cut float64) func(math.Point3) (bool, bool) {
	return func(p math.Point3) (bool, bool) {
		d := float64(math.P3(0, 0, 0).VectorTo(p).Dot(axis)) - cut
		if stdmath.Abs(d) < chartProbeBand {
			return false, false
		}
		return (d > 0) == (op != Difference), true
	}
}

// chartProbeBand keeps samples clear of the cut itself. It is a model distance on bodies of extent ~10,
// so it is a coarse exclusion zone rather than a tolerance any verdict depends on.
const chartProbeBand = 0.05 // tol:sampling — probe exclusion band around the cut plane

func chartHalfSpaceCases(t *testing.T) []chartHalfSpaceCase {
	t.Helper()
	z, y := math.V3(0, 0, 1), math.V3(0, 1, 0)
	tilted := math.V3(0, 0.6, 0.8)
	big := func(lo math.Point3) *topo.Body { return brepfixture.Box(lo, 40, 40, 40) }
	mk := func(name string, op Op, target *topo.Body, lo math.Point3, axis math.Vector3, cut float64) chartHalfSpaceCase {
		res, err := Boolean(op, target, big(lo))
		if err != nil || res == nil {
			t.Fatalf("%s: Boolean: %v", name, err)
		}
		return chartHalfSpaceCase{name: name, body: res, onFace: halfSpaceKeepRule(op, axis, cut)}
	}
	return []chartHalfSpaceCase{
		mk("torus band, perpendicular", Difference, chartTorus(t, z), math.P3(-20, -20, 0), z, 0),
		mk("torus band, perpendicular kept", Intersection, chartTorus(t, z), math.P3(-20, -20, 0), z, 0),
		mk("torus spiric oval cap", Intersection, chartTorus(t, z), math.P3(-20, 6, -20), y, 6),
		mk("torus genus-1 complement", Difference, chartTorus(t, z), math.P3(-20, 6, -20), y, 6),
		mk("torus two-oval band", Intersection, chartTorus(t, z), math.P3(-20, 2, -20), y, 2),
		mk("torus oblique figure-eight kept", Intersection, chartTorus(t, tilted), math.P3(-20, -20, 1), z, 1),
		mk("torus oblique figure-eight cut", Difference, chartTorus(t, tilted), math.P3(-20, -20, 1), z, 1),
		mk("cylinder band, oblique", Difference, chartCylinder(t), math.P3(-20, -20, 4), z, 4),
	}
}

func chartTorus(t *testing.T, axis math.Vector3) *topo.Body {
	t.Helper()
	b, err := SolidTorus(math.P3(0, 0, 0), axis, 5, 2, "t")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	return b
}

func chartCylinder(t *testing.T) *topo.Body {
	t.Helper()
	c, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0.6, 0.8), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	return c
}
