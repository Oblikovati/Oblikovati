// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// This file is the shared leg the obstacle body-level tests drive: it runs the REAL fillet pipeline on a
// real imported B-rep up to the point where the ObstacleFeature (and its surf-rst canal payload, or the
// nil that means the tier declined) exists, so a test can assert on the pipeline's own values instead of
// on a hand-built stand-in.

// corpusFixtureDir is the occtparity corpus's STEP fixture root, read directly from kernel/ops (the
// occtparity package itself imports kernel/ops, so importing it back would cycle — u4StepPath's
// precedent, fillet_obstacle_dual_test.go).
const corpusFixtureDir = "../../../model/feature/occtparity/fixtures"

// obstacleFeatureFor runs body's real fillet at the pick (edge midpoint + radius) up to
// buildObstacleFeature and returns the edgeFillet, the feature, the shared wing cross-section geometry,
// and the model tol.Resolution. It fails the test when the body carries no single-host obstacle there: every
// caller's premise is that it does.
func obstacleFeatureFor(t *testing.T, body *topo.Body, name string, mid math.Point3, radius float64) (edgeFillet, *ObstacleFeature, obstacleGeom, tol.Resolution) {
	t.Helper()
	ef, res := singleEdgeFillet(t, body, name, mid, radius)
	d, ok := detectObstacle(ef, res)
	if !ok {
		t.Fatalf("%s: detectObstacle found no single-host obstacle at the pick %v r=%g", name, mid, radius)
	}
	of, og, ok := buildObstacleFeature(ef, d, res)
	if !ok {
		t.Fatalf("%s: buildObstacleFeature declined", name)
	}
	return ef, of, og, res
}

// singleEdgeFillet resolves one body's single filleted edge into the edgeFillet the obstacle detector
// consumes, running the same sequence FilletEdges runs internally (u4Fillet's pattern).
func singleEdgeFillet(t *testing.T, body *topo.Body, name string, mid math.Point3, radius float64) (edgeFillet, tol.Resolution) {
	t.Helper()
	edge := edgeAtMidpoint(body, mid)
	if edge == nil {
		t.Fatalf("%s: filleted edge (midpoint %v) not found", name, mid)
	}
	return resolveSingleEdgeFillet(t, body, edge, name, radius), tol.ForBody(body)
}

// resolveSingleEdgeFillet is singleEdgeFillet's pick→corner→fillet leg, split out to keep both functions
// inside the 20-line budget. It insists on exactly one edgeFillet: every caller here is a single pick.
func resolveSingleEdgeFillet(t *testing.T, body *topo.Body, edge *topo.Edge, name string, radius float64) edgeFillet {
	t.Helper()
	picks, err := resolveFilletPicks(body, filletPicksFor([][]byte{edge.ReferenceKey()}, radius))
	if err != nil {
		t.Fatalf("%s: resolveFilletPicks: %v", name, err)
	}
	blends, miters, err := computeCorners(body, picks)
	if err != nil {
		t.Fatalf("%s: computeCorners: %v", name, err)
	}
	fils, err := computeFillets(body, picks, blends, miters, FillConcaveOutward, nil)
	if err != nil || len(fils) != 1 {
		t.Fatalf("%s: computeFillets = (%d fillets, %v), want (1, nil)", name, len(fils), err)
	}
	return fils[0]
}

// importCorpusBody imports one corpus fixture as a real B-rep, exactly as occtparity's own importInput
// does (TargetUnitMM 1, one solid).
func importCorpusBody(t *testing.T, rel string) *topo.Body {
	t.Helper()
	return importStepSolid(t, filepath.Join(corpusFixtureDir, rel))
}

// importStepSolid imports a single-solid STEP file, the pattern every ops fillet fixture uses so the
// topology under test is a real imported B-rep rather than a hand-welded stub.
func importStepSolid(t *testing.T, path string) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import %s: %v (n=%d)", path, err, len(bodies))
	}
	return bodies[0]
}

// dipSpineStations is of.RimArcPts projected on the fillet spine — the row obstacleCanalRimFeet requires
// to be strictly monotone. Exposed to tests so a fixture's premise ("this dip re-enters the band") can be
// asserted rather than assumed.
func dipSpineStations(ef edgeFillet, of *ObstacleFeature) []float64 {
	e := ef.cyl.AxisDir.AsVector()
	ss := make([]float64, len(of.RimArcPts))
	for i, q := range of.RimArcPts {
		ss[i] = float64(ef.cyl.Origin.VectorTo(q).Dot(e))
	}
	return ss
}
