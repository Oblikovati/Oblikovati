// SPDX-License-Identifier: GPL-2.0-only

package opregistry

// Regression for Oblikovati#2075. The live MCP driver builds a mitered corner whose faces
// interpenetrate at the bend root; the model/feature miter fixtures do not reproduce it because
// they bypass the sheet-metal STYLE and use a different gauge. This drives the same operations the
// router does, on the same 40x30 mm style-defaulted sheet, and gates at level 2 — every other
// sheet-metal fixture gates on ops.Validate, which is topology only and cannot see two faces
// passing through each other (#2077).

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
)

// miterEdgeKeys returns the reference keys of the two adjacent top edges the live driver picks:
// the X-running edge at max Y, and the Y-running edge at max X. They share a corner, so the second
// flange has something to miter onto.
func miterEdgeKeys(t *testing.T, s *app.Session) (string, string) {
	t.Helper()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	b := def.SurfaceBodies().Item(0)
	var alongX, alongY string
	bestY, bestX := -1e30, -1e30
	for _, e := range b.Edges() {
		p, q := e.StartVertex().Point(), e.EndVertex().Point()
		if float64(p.Z) < 0.05 || float64(q.Z) < 0.05 {
			continue // the top face only, matching smTopEdges
		}
		switch {
		case nearlyEqual(float64(p.Y), float64(q.Y)) && float64(p.Y) > bestY:
			bestY, alongX = float64(p.Y), string(e.ReferenceKey())
		case nearlyEqual(float64(p.X), float64(q.X)) && float64(p.X) > bestX:
			bestX, alongY = float64(p.X), string(e.ReferenceKey())
		}
	}
	if alongX == "" || alongY == "" {
		t.Fatalf("no adjacent top edge pair: alongX=%q alongY=%q", alongX, alongY)
	}
	return alongX, alongY
}

func nearlyEqual(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

// TestAutoMiterAtStyleDefaultsDoesNotInterpenetrate sweeps the miter gap across the sheet
// thickness. The live failure is at 0.5 mm — half the 1 mm gauge — and the existing fixtures only
// ever tried 0 and a gap wider than the sheet, so they stepped straight over it.
func TestAutoMiterAtStyleDefaultsDoesNotInterpenetrate(t *testing.T) {
	t.Parallel()
	for _, gap := range []string{"0.2 mm", "0.5 mm", "0.8 mm", "1 mm", "3 mm"} {
		s, _ := seedSheetMetalSheet(t)
		edgeX, edgeY := miterEdgeKeys(t, s)
		if _, err := applyMap(t, s, "sheetMetalFlange", map[string]any{
			"edge": edgeX, "height": "10 mm", "radius": "2 mm",
		}); err != nil {
			t.Fatalf("gap %s: first flange: %v", gap, err)
		}
		if _, err := applyMap(t, s, "sheetMetalFlange", map[string]any{
			"edge": edgeY, "height": "10 mm", "radius": "2 mm",
			"applyAutoMiter": true, "miterGap": gap,
		}); err != nil {
			t.Fatalf("gap %s: mitered flange: %v", gap, err)
		}
		assertNoInterpenetration(t, s, gap)
	}
}

// assertNoInterpenetration checks every body of the part at ops.CheckGeometry.
func assertNoInterpenetration(t *testing.T, s *app.Session, gap string) {
	t.Helper()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	for i := 0; i < def.SurfaceBodies().Count(); i++ {
		b := def.SurfaceBodies().Item(i)
		ok, problems := ops.ValidateBodyEntities(b, ops.CheckGeometry, ops.DefaultQuality())
		if !ok {
			t.Errorf("gap %s: body %d interpenetrates (%d problems), first: %s",
				gap, i, len(problems), problems[0].Issue)
		}
	}
}
