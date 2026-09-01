// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"math"
	"testing"

	"oblikovati.org/model/compdef"
)

// TestPath3DFromPoints covers the polyline→path helper: a valid chain builds an open path; too
// few points or a malformed point is a clean error.
func TestPath3DFromPoints(t *testing.T) {
	t.Parallel()
	p, err := path3DFromPoints([][]float64{{0, 0, 0}, {0, 0, 2}, {0, 0, 5}})
	if err != nil {
		t.Fatalf("valid polyline: %v", err)
	}
	if p.Count() != 3 || p.IsClosed() {
		t.Errorf("path = %d points closed=%v, want 3 open", p.Count(), p.IsClosed())
	}
	if _, err := path3DFromPoints([][]float64{{0, 0, 0}}); err == nil {
		t.Error("a single-point polyline should error")
	}
	if _, err := path3DFromPoints([][]float64{{0, 0, 0}, {1, 2}}); err == nil {
		t.Error("a 2-component point should error")
	}
}

// TestSweepAlongPathPoints sweeps the profile along an explicit 3D polyline (no path sketch),
// mirroring how a loft rail consumes explicit points; it must build a healthy solid body.
func TestSweepAlongPathPoints(t *testing.T) {
	t.Parallel()
	s := profiledPart(t)
	// The profile sits on XY; a straight polyline up +Z sweeps it into a prism (no pathSketchIndex).
	args := map[string]any{
		"sketchIndex": 0, "profileIndex": 0,
		"pathPoints": [][]float64{{0, 0, 0}, {0, 0, 5}},
	}
	raw, err := applyMap(t, s, "sweep", args)
	if err != nil {
		t.Fatalf("sweep along pathPoints: %v", err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !res.Healthy || res.Bodies < 1 {
		t.Errorf("sweep result = %+v, want a healthy body from the polyline path", res)
	}
}

// TestSweepPathPointsOverrideSketch: pathPoints takes precedence, so a valid polyline sweeps even
// when the pathSketchIndex would be out of range (the sketch path is not consulted).
func TestSweepPathPointsOverridesSketch(t *testing.T) {
	t.Parallel()
	s := profiledPart(t)
	args := map[string]any{
		"sketchIndex": 0, "profileIndex": 0, "pathSketchIndex": 99,
		"pathPoints": [][]float64{{0, 0, 0}, {0, 0, 5}},
	}
	if _, err := applyMap(t, s, "sweep", args); err != nil {
		t.Fatalf("pathPoints should override the (invalid) sketch path: %v", err)
	}
}

// TestSweepIntersectOperation: the sweep accepts Inventor's INTERSECT operation (A ∩ B), not just
// new/join/cut — the geometry and parseOperation always supported it; only the schema enum withheld
// it (parity with PartFeatureOperationEnum, minus the surface op). Verified by the boolean identity
// A∩B + (A−B) = A: sweeping the same tool onto identical base boxes with intersect and with cut must
// give complementary volumes that sum to the base — proof both are real, correct booleans, and that
// intersect is now reachable over the wire.
func TestSweepIntersectOperation(t *testing.T) {
	t.Parallel()
	vBase, vInt := sweepThenVolume(t, "intersect")
	_, vCut := sweepThenVolume(t, "cut")
	if vInt <= 0 || vInt >= vBase {
		t.Errorf("intersect volume = %g, want 0 < v < base %g (a join would be ≥ base)", vInt, vBase)
	}
	if want := vBase; math.Abs((vInt+vCut)-want) > 1e-6*want {
		t.Errorf("A∩B + A−B = %g, want the base %g (intersect and cut must be complementary)", vInt+vCut, want)
	}
}

// sweepThenVolume builds a base box (extrude profile 0 up 5 cm), sweeps profile 1 up 3 cm with the
// given boolean op, and returns the base and result volumes.
func sweepThenVolume(t *testing.T, op string) (base, result float64) {
	t.Helper()
	s := profiledPart(t)
	if _, err := applyMap(t, s, "extrude", map[string]any{"sketchIndex": 0, "distance": "5 cm", "operation": "new"}); err != nil {
		t.Fatalf("base extrude: %v", err)
	}
	base = bodyVolume(t, s)
	raw, err := applyMap(t, s, "sweep", map[string]any{
		"sketchIndex": 1, "profileIndex": 0,
		"pathPoints": [][]float64{{0, 0, 0}, {0, 0, 3}}, // tool z∈[0,3] ⊂ base z∈[0,5]
		"operation":  op,
	})
	if err != nil {
		t.Fatalf("sweep %s: %v", op, err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !res.Healthy || res.Bodies != 1 {
		t.Fatalf("sweep %s result = %+v, want one healthy body", op, res)
	}
	return base, bodyVolume(t, s)
}

// TestSweepPathPointsTooFew: a polyline with fewer than two points is a clean error.
func TestSweepPathPointsTooFew(t *testing.T) {
	t.Parallel()
	s := profiledPart(t)
	args := map[string]any{
		"sketchIndex": 0, "profileIndex": 0,
		"pathPoints": [][]float64{{0, 0, 0}},
	}
	if _, err := applyMap(t, s, "sweep", args); err == nil {
		t.Error("a single-point pathPoints should error")
	}
}

// TestSweepSurfaceOperation sweeps the profile with operation:"surface" (kSurfaceOperation, #1858):
// the rectangle swept 5 up +Z builds one healthy OPEN sheet body (walls only, no end caps), not
// booleaned against anything.
func TestSweepSurfaceOperation(t *testing.T) {
	t.Parallel()
	s := profiledPart(t)
	raw, err := applyMap(t, s, "sweep", map[string]any{
		"sketchIndex": 0, "profileIndex": 0, "operation": "surface",
		"pathPoints": [][]float64{{0, 0, 0}, {0, 0, 5}},
	})
	if err != nil {
		t.Fatalf("surface sweep: %v", err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !res.Healthy || res.Bodies != 1 {
		t.Fatalf("surface sweep result = %+v, want one healthy body", res)
	}
	b := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).SurfaceBodies().Item(0)
	if b.IsSolid() {
		t.Error("surface-operation sweep produced a SOLID body, want an open sheet")
	}
}
