// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// lipArgsJSON builds a features.add(lip) request (edge keys carry binary bytes, so JSON-marshal).
func lipArgsJSON(t *testing.T, edge string, groove bool) string {
	t.Helper()
	return mustJSON(t, map[string]any{"kind": "lip", "args": map[string]any{
		"edgeRefs": []string{edge}, "width": "2 mm", "height": "2 mm", "groove": groove,
	}})
}

// TestLipOverWire runs a raised lip along a box edge end to end (M20-F10 #485) → valid solid.
func TestLipOverWire(t *testing.T) {
	r, s, verticals := filletBoxFixture(t)
	call(t, r, s, "features.add", lipArgsJSON(t, verticals[0], false), &struct {
		Bodies int `json:"bodies"`
	}{})
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("lipped body invalid: %+v", v.Problems)
	}
}

// TestLipGrooveOverWire runs the groove variant over the wire.
func TestLipGrooveOverWire(t *testing.T) {
	r, s, verticals := filletBoxFixture(t)
	call(t, r, s, "features.add", lipArgsJSON(t, verticals[0], true), &struct {
		Bodies int `json:"bodies"`
	}{})
	var v wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0}`, &v)
	if !v.Valid {
		t.Fatalf("grooved body invalid: %+v", v.Problems)
	}
}
