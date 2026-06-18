// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestAddSketchEntities: a drawing sketch renders its line/circle/rectangle entities as curves.
func TestAddSketchEntities(t *testing.T) {
	c := NewContent()
	ss := c.Sheets().Active().Sketches()
	sk := ss.Add("S1")
	if sk.Name() != "S1" {
		t.Fatalf("sketch name = %q, want S1", sk.Name())
	}
	if _, err := ss.AddEntity("S1", types.SketchLineEntity, [][2]float64{{10, 10}, {60, 10}}, 0); err != nil {
		t.Fatalf("AddEntity line: %v", err)
	}
	if _, err := ss.AddEntity("S1", types.SketchRectangleEntity, [][2]float64{{10, 20}, {60, 50}}, 0); err != nil {
		t.Fatalf("AddEntity rectangle: %v", err)
	}
	if _, err := ss.AddEntity("S1", types.SketchCircleEntity, [][2]float64{{100, 100}}, 12); err != nil {
		t.Fatalf("AddEntity circle: %v", err)
	}
	if sk.EntityCount() != 3 {
		t.Errorf("entity count = %d, want 3", sk.EntityCount())
	}
	// 1 line + 4 rectangle sides + a circle polyline (>1 segment) = many curves.
	if sk.CurveCount() < 1+4+8 {
		t.Errorf("curve count = %d, want a line + 4 rect sides + a circle polyline", sk.CurveCount())
	}
}

// TestSketchEntityValidation: malformed entities are rejected.
func TestSketchEntityValidation(t *testing.T) {
	c := NewContent()
	ss := c.Sheets().Active().Sketches()
	ss.Add("S1")
	if _, err := ss.AddEntity("S1", types.SketchLineEntity, [][2]float64{{0, 0}}, 0); err == nil {
		t.Error("a line with 1 point = ok, want error")
	}
	if _, err := ss.AddEntity("S1", types.SketchCircleEntity, [][2]float64{{0, 0}}, 0); err == nil {
		t.Error("a circle with radius 0 = ok, want error")
	}
	if _, err := ss.AddEntity("NOPE", types.SketchLineEntity, [][2]float64{{0, 0}, {1, 1}}, 0); err == nil {
		t.Error("AddEntity on a missing sketch = ok, want error")
	}
}

// TestSketchPersists: a drawing sketch's entities survive a save/open round-trip and re-render.
func TestSketchPersists(t *testing.T) {
	c := NewContent()
	ss := c.Sheets().Active().Sketches()
	ss.Add("S1")
	if _, err := ss.AddEntity("S1", types.SketchRectangleEntity, [][2]float64{{10, 20}, {60, 50}}, 0); err != nil {
		t.Fatalf("AddEntity: %v", err)
	}
	data, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	restored := NewContent()
	if err := restored.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	rs := restored.Sheets().Active().Sketches()
	if rs.Count() != 1 || rs.Item(0).EntityCount() != 1 || rs.Item(0).CurveCount() != 4 {
		t.Fatalf("restored sketch = %d sketches / %d entities / %d curves, want 1/1/4 (rect re-rendered)",
			rs.Count(), sketchEntityCount(rs), sketchCurveCount(rs))
	}
}

func sketchEntityCount(ss *DrawingSketches) int {
	if ss.Count() == 0 {
		return 0
	}
	return ss.Item(0).EntityCount()
}

func sketchCurveCount(ss *DrawingSketches) int {
	if ss.Count() == 0 {
		return 0
	}
	return ss.Item(0).CurveCount()
}
