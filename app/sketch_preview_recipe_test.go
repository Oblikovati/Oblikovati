// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/app/cmdline"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// coordOf is the command-engine coordinate for a sketch-plane point.
func coordOf(p math.Point2) cmdline.Coord {
	return cmdline.Coord{X: float64(p.X), Y: float64(p.Y)}
}

// startWithPoints starts tool with its first n points already placed, by feeding them through
// the same command path a typed coordinate uses.
func startWithPoints(t *testing.T, s *Session, tool Tool, pts ...math.Point2) {
	t.Helper()
	s.StartTool(tool)
	ct, ok := tool.(interface {
		SubmitToken(*Session, CommandToken) error
	})
	if !ok {
		t.Fatalf("%T does not accept coordinate tokens", tool)
	}
	for _, p := range pts {
		if err := ct.SubmitToken(s, CommandToken{Kind: CoordToken, Coord: coordOf(p)}); err != nil {
			t.Fatalf("submit %v: %v", p, err)
		}
	}
}

// TestEveryCreateToolPreviews is the coverage gate for #2014: before it, only five of the
// seventeen Create tools drew anything while placing.
func TestEveryCreateToolPreviews(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		tool    Tool
		placed  []math.Point2
		cursor  math.Point2
		minEnts int
	}{
		{"line", NewLineTool(), []math.Point2{math.P2(0, 0)}, math.P2(5, 3), 1},
		{"rectangle", NewRectangleTool(), []math.Point2{math.P2(0, 0)}, math.P2(10, 8), 4},
		{"three-point rectangle", NewThreePointRectangleTool(), []math.Point2{math.P2(0, 0), math.P2(10, 0)}, math.P2(10, 8), 4},
		{"centre rectangle", NewCenterRectangleTool(), []math.Point2{math.P2(0, 0)}, math.P2(5, 4), 6},
		{"circle", NewCircleTool(), []math.Point2{math.P2(0, 0)}, math.P2(5, 0), 1},
		{"three-point circle", NewThreePointCircleTool(), []math.Point2{math.P2(0, 0), math.P2(10, 0)}, math.P2(5, 5), 1},
		{"arc", NewArcTool(), []math.Point2{math.P2(0, 0), math.P2(10, 0)}, math.P2(5, 5), 1},
		{"centre-point arc", NewCenterPointArcTool(), []math.Point2{math.P2(0, 0), math.P2(5, 0)}, math.P2(0, 5), 1},
		{"ellipse", NewEllipseTool(), []math.Point2{math.P2(0, 0), math.P2(5, 0)}, math.P2(0, 3), 1},
		{"polygon", NewPolygonTool(6), []math.Point2{math.P2(0, 0)}, math.P2(5, 0), 7},
		{"slot", NewSketchSlotTool(2), []math.Point2{math.P2(0, 0)}, math.P2(10, 0), 5},
		{"centre-point arc slot", NewCenterPointArcSlotTool(2), []math.Point2{math.P2(0, 0), math.P2(10, 0)}, math.P2(0, 10), 4},
		{"three-point arc slot", NewThreePointArcSlotTool(2), []math.Point2{math.P2(10, 0), math.P2(7, 7)}, math.P2(0, 10), 4},
		{"spline", NewSplineTool(), []math.Point2{math.P2(0, 0), math.P2(3, 4)}, math.P2(7, 1), 1},
		{"control-vertex spline", NewControlVertexSplineTool(), []math.Point2{math.P2(0, 0), math.P2(3, 4)}, math.P2(7, 1), 1},
		{"point", NewPointTool(), nil, math.P2(2, 2), 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := sketchSession(t)
			startWithPoints(t, s, c.tool, c.placed...)
			r, ok := s.ActiveToolRecipe(c.cursor)
			if !ok {
				t.Fatal("tool must preview once its points are placed")
			}
			if len(r.Entities) < c.minEnts {
				t.Errorf("preview entities = %d, want at least %d", len(r.Entities), c.minEnts)
			}
			if curves := s.ActiveToolPreviewCurves(c.cursor); len(curves) == 0 && c.name != "point" {
				t.Error("preview produced no drawable curves")
			}
		})
	}
}

// The preview must describe the SAME shape the commit creates — that equivalence is the whole
// point of both reading one recipe, and it is what a separate preview implementation kept
// getting wrong.
func TestPreviewMatchesCommit(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	tool := NewRectangleTool()
	startWithPoints(t, s, tool, math.P2(0, 0))
	preview, ok := s.ActiveToolRecipe(math.P2(10, 8))
	if !ok {
		t.Fatal("preview expected")
	}
	if err := tool.SubmitToken(s, CommandToken{Kind: CoordToken, Coord: coordOf(math.P2(10, 8))}); err != nil {
		t.Fatal(err)
	}
	if err := tool.Commit(s); err != nil {
		t.Fatal(err)
	}
	// Every non-construction preview curve must appear in the committed geometry.
	committed := len(sk.Entities())
	if committed != len(preview.Entities) {
		t.Errorf("committed %d entities, preview showed %d — they must describe the same shape",
			committed, len(preview.Entities))
	}
	for i, e := range preview.Entities {
		if e.Kind != sketch.RecipeLine {
			t.Errorf("preview entity %d is kind %d, want a line", i, e.Kind)
		}
	}
}

// Construction geometry must be distinguishable in the preview, so the head can draw the
// centre rectangle's diagonals dashed rather than as solid edges.
func TestPreviewSeparatesConstructionGeometry(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	startWithPoints(t, s, NewCenterRectangleTool(), math.P2(0, 0))
	curves := s.ActiveToolPreviewCurves(math.P2(5, 4))
	solid, construction := 0, 0
	for _, c := range curves {
		if c.Construction {
			construction++
			continue
		}
		solid++
	}
	if solid != 4 || construction != 2 {
		t.Errorf("preview curves = %d solid, %d construction; want 4 and 2", solid, construction)
	}
}
