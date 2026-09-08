// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A face's chart is the face's OWN (u, v) boundary (ADR-0063). Every vertex the face's loops carry has
// to be a vertex of that contour, at its own parameters — re-based by whole periods, which map back to
// the same 3-D point, and by nothing else.
//
// This is a POST-CONDITION over the boolean's corpus rather than a row about one body, because the
// class of defect is what needs guarding: the merged cocylindrical band's chart came back rigidly
// rotated by 0.0497 rad — half a sampling step — against its own edges, which nothing then compared
// (ADR-0061 stage 5, round 3). Comparing in 3-D is what makes the rule period-blind: PointAt of a
// vertex a whole turn away is the same point, and PointAt of a rotated one is not.

// TestEveryChartCarriesItsFacesOwnVertices walks the corpus.
func TestEveryChartCarriesItsFacesOwnVertices(t *testing.T) {
	t.Parallel()
	for _, tc := range chartCorpus(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertChartsCarryTheirVertices(t, tc.body)
		})
	}
}

// chartCorpusRow is one body of the post-condition's corpus.
type chartCorpusRow struct {
	name string
	body *topo.Body
}

// chartCorpus is the cheap booleans that record charts: the cocylindrical merge this guards, the two
// coaxial unions it generalised, a bore continuing a bore, and a drilled block.
func chartCorpus(t *testing.T) []chartCorpusRow {
	t.Helper()
	return []chartCorpusRow{
		{"cocylindrical boss on a wall", cocylindricalBossBody(t)},
		{"rod on rod, abutting", unionOfCoaxialRods(t, 4, 3)},
		{"rod on rod, overlapping", unionOfCoaxialRods(t, 3, 4)},
		{"a bore continuing a bore", continuedBoreBody(t)},
		{"a drilled block", drilledBlockBody(t)},
	}
}

// assertChartsCarryTheirVertices checks every charted face of the body.
func assertChartsCarryTheirVertices(t *testing.T, b *topo.Body) {
	t.Helper()
	charted := 0
	for _, f := range b.Faces() {
		if len(f.Chart()) == 0 {
			continue
		}
		charted++
		assertFaceChartCarriesItsVertices(t, f)
	}
	if charted == 0 {
		t.Fatal("no face of this body carries a chart; the row guards nothing")
	}
}

// assertFaceChartCarriesItsVertices requires every loop vertex of the face to be a chart vertex.
func assertFaceChartCarriesItsVertices(t *testing.T, f *topo.Face) {
	t.Helper()
	res := geom.ResolutionForBox(f.RangeBox())
	for _, p := range loopVertexPoints(f) {
		if chartHoldsPoint(f, p, res.Weld()) {
			continue
		}
		u, v := f.Geometry().ParamAt(p)
		t.Errorf("face %q carries the vertex %v (u=%.9f v=%.9f) but its chart has no vertex there: the "+
			"chart is not this face's own boundary", string(f.ReferenceKey()), p, u, v)
	}
}

// loopVertexPoints is every vertex the face's loops walk through.
func loopVertexPoints(f *topo.Face) []math.Point3 {
	var out []math.Point3
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			out = append(out, u.Edge().StartVertex().Point(), u.Edge().EndVertex().Point())
		}
	}
	return out
}

// chartHoldsPoint reports whether some chart vertex maps back onto p. Mapping through PointAt is what
// makes the test period-blind and rotation-sensitive at once.
func chartHoldsPoint(f *topo.Face, p math.Point3, tol float64) bool {
	for _, c := range f.Chart() {
		for _, q := range c {
			if float64(f.Geometry().PointAt(float64(q.X), float64(q.Y)).DistanceTo(p)) <= tol {
				return true
			}
		}
	}
	return false
}

// cocylindricalBossBody is the #2167 shape from primitives: a boss cocylindrical with its host, one
// side flattened, so the merged wall's second rim is notched.
func cocylindricalBossBody(t *testing.T) *topo.Body {
	t.Helper()
	host := mustChartCylinder(t, math.P3(0, 0, 0), 3, 6)
	upper := mustChartCylinder(t, math.P3(0, 0, 6), 3, 4)
	chop, err := SolidBlock(math.P3(2.4, -4, 5), math.P3(5, 4, 11), "chop")
	if err != nil {
		t.Fatalf("chop block: %v", err)
	}
	return mustChartBoolean(t, Union, host, mustChartBoolean(t, Difference, upper, chop))
}

// unionOfCoaxialRods unions the r=2 rod over z∈[0,4] with one of height h based at baseZ.
func unionOfCoaxialRods(t *testing.T, baseZ, h float64) *topo.Body {
	t.Helper()
	return mustChartBoolean(t, Union, mustChartCylinder(t, math.P3(0, 0, 0), 2, 4),
		mustChartCylinder(t, math.P3(0, 0, math.Scalar(baseZ)), 2, math.Scalar(h)))
}

// continuedBoreBody drills a blind bore and continues it to the bottom with a coaxial equal drill.
func continuedBoreBody(t *testing.T) *topo.Body {
	t.Helper()
	blk, err := SolidBlock(math.P3(-3, -3, 0), math.P3(3, 3, 6), "blk")
	if err != nil {
		t.Fatalf("block: %v", err)
	}
	step := mustChartBoolean(t, Difference, blk, mustChartCylinder(t, math.P3(0, 0, 3), 1, 4))
	return mustChartBoolean(t, Difference, step, mustChartCylinder(t, math.P3(0, 0, -1), 1, 4))
}

// drilledBlockBody is the plain through-drill, the charted case with no merge in it at all.
func drilledBlockBody(t *testing.T) *topo.Body {
	t.Helper()
	blk, err := SolidBlock(math.P3(-3, -3, 0), math.P3(3, 3, 4), "blk")
	if err != nil {
		t.Fatalf("block: %v", err)
	}
	return mustChartBoolean(t, Difference, blk, mustChartCylinder(t, math.P3(0, 0, -1), 1.2, 6))
}

// mustChartCylinder builds an axial cylinder or fails the test.
func mustChartCylinder(t *testing.T, at math.Point3, r, h math.Scalar) *topo.Body {
	t.Helper()
	b, err := SolidCylinder(at, math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder(%v, %v, %v): %v", at, r, h, err)
	}
	return b
}

// mustChartBoolean runs the boolean or fails the test.
func mustChartBoolean(t *testing.T, op Op, a, b *topo.Body) *topo.Body {
	t.Helper()
	out, err := Boolean(op, a, b)
	if err != nil {
		t.Fatalf("%v: %v", op, err)
	}
	return out
}
