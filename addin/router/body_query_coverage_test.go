// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestBodyQueries drives the body query handlers against the extruded box (4×3×5 cm):
// locate-by-point across entity kinds, ray hit, point containment, convexity, validate,
// and range box.
func TestBodyQueries(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)

	for _, kind := range []string{"face", "edge", "vertex"} {
		_ = tryCall(t, r, s, "body.locateUsingPoint", mustJSON(t, wire.LocateUsingPointArgs{
			BodyIndex: 0, Point: []float64{0, 0, 0}, EntityKind: kind, ProximityTolerance: 0.5,
		}))
	}

	_ = tryCall(t, r, s, "body.findUsingRay", mustJSON(t, wire.FindUsingRayArgs{
		BodyIndex: 0, Origin: []float64{2, 1.5, -1}, Direction: []float64{0, 0, 1}, FindFirstOnly: true,
	}))

	var inside wire.IsPointInsideResult
	call(t, r, s, "body.isPointInside", mustJSON(t, wire.IsPointInsideArgs{BodyIndex: 0, Point: []float64{2, 1.5, 2.5}}), &inside)

	for _, coll := range []string{"all", "concave", "convex"} {
		_ = tryCall(t, r, s, "body.convexityEdges", mustJSON(t, wire.ConvexityEdgesArgs{BodyIndex: 0, Collection: coll}))
	}

	call(t, r, s, "body.validate", `{"bodyIndex":0}`, nil)
	call(t, r, s, "body.rangeBox", `{"bodyIndex":0}`, nil)

	// Out-of-range body index is a clean error.
	if err := tryCall(t, r, s, "body.validate", `{"bodyIndex":9}`); err == nil {
		t.Error("body.validate on an out-of-range body should error")
	}
}
