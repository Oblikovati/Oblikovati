// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"maps"
	"testing"

	"oblikovati.org/api/wire"
)

// chamferArgs builds a features.add(chamfer) request (edge keys carry binary bytes, so it
// must be JSON-marshalled, not string-concatenated).
func chamferArgs(t *testing.T, edge string, extra map[string]any) string {
	t.Helper()
	args := map[string]any{"edgeRefs": []string{edge}, "distance": "3 mm"}
	maps.Copy(args, extra)
	return mustJSON(t, map[string]any{"kind": "chamfer", "args": args})
}

// TestChamferTwoDistancesOverWire drives the full wire path for an asymmetric chamfer
// (M20-F03 #474): features.add(chamfer, chamferType=twoDistances) on a box edge → valid solid.
func TestChamferTwoDistancesOverWire(t *testing.T) {
	t.Parallel()
	r, s, verticals := filletBoxFixture(t)
	call(t, r, s, "features.add", chamferArgs(t, verticals[0], map[string]any{"chamferType": "twoDistances", "distance2": "6 mm"}),
		&struct {
			Bodies int `json:"bodies"`
		}{})
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("two-distance chamfered body invalid: %+v", v.Problems)
	}
}

// TestChamferDistanceAngleOverWire drives the distance-and-angle mode over the wire.
func TestChamferDistanceAngleOverWire(t *testing.T) {
	t.Parallel()
	r, s, verticals := filletBoxFixture(t)
	call(t, r, s, "features.add", chamferArgs(t, verticals[0], map[string]any{"chamferType": "distanceAndAngle", "angle": "30 deg"}),
		&struct {
			Bodies int `json:"bodies"`
		}{})
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("distance-angle chamfered body invalid: %+v", v.Problems)
	}
}

// TestChamferUnknownTypeOverWire rejects an unknown chamfer mode with a precise error.
func TestChamferUnknownTypeOverWire(t *testing.T) {
	t.Parallel()
	r, s, verticals := filletBoxFixture(t)
	if _, err := r.Handle(s, "features.add", []byte(chamferArgs(t, verticals[0], map[string]any{"chamferType": "bevel"}))); err == nil {
		t.Error("expected an error for an unknown chamferType")
	}
}
