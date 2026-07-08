// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"
)

// TestPath3DFromPoints covers the polyline→path helper: a valid chain builds an open path; too
// few points or a malformed point is a clean error.
func TestPath3DFromPoints(t *testing.T) {
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
	s := profiledPart(t)
	args := map[string]any{
		"sketchIndex": 0, "profileIndex": 0, "pathSketchIndex": 99,
		"pathPoints": [][]float64{{0, 0, 0}, {0, 0, 5}},
	}
	if _, err := applyMap(t, s, "sweep", args); err != nil {
		t.Fatalf("pathPoints should override the (invalid) sketch path: %v", err)
	}
}

// TestSweepPathPointsTooFew: a polyline with fewer than two points is a clean error.
func TestSweepPathPointsTooFew(t *testing.T) {
	s := profiledPart(t)
	args := map[string]any{
		"sketchIndex": 0, "profileIndex": 0,
		"pathPoints": [][]float64{{0, 0, 0}},
	}
	if _, err := applyMap(t, s, "sweep", args); err == nil {
		t.Error("a single-point pathPoints should error")
	}
}
