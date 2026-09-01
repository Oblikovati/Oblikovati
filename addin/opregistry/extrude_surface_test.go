// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"oblikovati.org/model/compdef"
)

// TestExtrudeSurfaceOperation drives the extrude tool with operation:"surface" (Inventor's
// kSurfaceOperation) end-to-end through the registry: the result is one healthy OPEN sheet body —
// non-solid, walls only (four side faces of the 4×3 rectangle, no start/end caps), and NOT
// booleaned against anything. This is the surfacing workflow's entry point (#1858).
func TestExtrudeSurfaceOperation(t *testing.T) {
	t.Parallel()
	s := profiledPart(t)
	raw, err := applyMap(t, s, "extrude", map[string]any{
		"sketchIndex": 0, "distance": "5 mm", "operation": "surface",
	})
	if err != nil {
		t.Fatalf("surface extrude: %v", err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !res.Healthy || res.Bodies != 1 {
		t.Fatalf("surface extrude result = %+v, want one healthy body", res)
	}
	b := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).SurfaceBodies().Item(0)
	if b.IsSolid() {
		t.Error("surface-operation extrude produced a SOLID body, want an open sheet")
	}
	if got := len(b.Faces()); got != 4 {
		t.Errorf("surface extrude has %d faces, want 4 walls (no caps)", got)
	}
}

// TestExtrudeSurfaceThenJoinKeepsBodies: a surface extrude does not boolean, so a following
// join extrude of the other profile leaves the sheet untouched and adds the solid alongside it —
// two bodies, one sheet + one solid.
func TestExtrudeSurfaceThenSolidCoexist(t *testing.T) {
	t.Parallel()
	s := profiledPart(t)
	if _, err := applyMap(t, s, "extrude", map[string]any{"sketchIndex": 0, "distance": "5 mm", "operation": "surface"}); err != nil {
		t.Fatalf("surface extrude: %v", err)
	}
	if _, err := applyMap(t, s, "extrude", map[string]any{"sketchIndex": 1, "distance": "5 mm", "operation": "new"}); err != nil {
		t.Fatalf("solid extrude: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if got := def.SurfaceBodies().Count(); got != 2 {
		t.Fatalf("want 2 bodies (sheet + solid), got %d", got)
	}
	var sheets, solids int
	for _, b := range def.SurfaceBodies().All() {
		if b.IsSolid() {
			solids++
		} else {
			sheets++
		}
	}
	if sheets != 1 || solids != 1 {
		t.Errorf("want 1 sheet + 1 solid, got %d sheet(s) + %d solid(s)", sheets, solids)
	}
}

// TestExtrudeUnknownOperationErrors: an unknown operation name is still a clean error after
// adding "surface" (guards the parseOperation switch's default).
func TestExtrudeUnknownOperationErrors(t *testing.T) {
	t.Parallel()
	s := profiledPart(t)
	if _, err := applyMap(t, s, "extrude", map[string]any{"sketchIndex": 0, "distance": "5 mm", "operation": "bogus"}); err == nil {
		t.Error("unknown operation should error")
	}
}
