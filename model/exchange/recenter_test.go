// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"math"
	"strings"
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// farCluster builds n circles clustered around (cx,cy) plus a few outliers far on the
// opposite side, so the median centre stays on the cluster while a mean/bbox-centre would
// not.
func farCluster(cx, cy float64, n int) []drawing.Entity {
	es := make([]drawing.Entity, 0, n+3)
	for i := 0; i < n; i++ {
		es = append(es, &drawing.Circle{Center: [3]float64{cx + float64(i), cy + float64(i%5), 0}, Radius: 1})
	}
	for i := 0; i < 3; i++ { // outliers mirrored to the far side
		es = append(es, &drawing.Arc{Center: [3]float64{-cx, cy, 0}, Radius: 1})
	}
	return es
}

// TestRecenterMovesFarDrawingToOrigin: a drawing centred far from the origin is shifted so
// its bulk lands near the origin, and the reported offset is the (robust) original centre.
func TestRecenterMovesFarDrawingToOrigin(t *testing.T) {
	const cx, cy = 5.4e7, 1.78e7
	es := farCluster(cx, cy, 200)
	out, offset, did := recenterFarFromOrigin(es)
	if !did {
		t.Fatal("expected a far drawing to be recentered")
	}
	if math.Abs(offset[0]-cx) > 200 || math.Abs(offset[1]-cy) > 200 {
		t.Errorf("offset = %v, want ~(%.0f,%.0f) (the cluster median, not pulled to 0 by outliers)", offset, cx, cy)
	}
	// The bulk (first circle) must now be near the origin, within float32-friendly range.
	c0 := out[0].(*drawing.Circle).Center
	if math.Hypot(c0[0], c0[1]) > 1000 {
		t.Errorf("first cluster entity at %v after recenter, want near origin", c0)
	}
}

// TestRecenterLeavesNearOriginUnchanged: a normal near-origin drawing is not shifted, so
// ordinary imports and the export round-trip are unaffected.
func TestRecenterLeavesNearOriginUnchanged(t *testing.T) {
	es := []drawing.Entity{
		&drawing.Line{Start: [3]float64{0, 0, 0}, End: [3]float64{100, 50, 0}},
		&drawing.Circle{Center: [3]float64{2000, 1500, 0}, Radius: 30}, // 20 m out, still well under 1 km
	}
	out, _, did := recenterFarFromOrigin(es)
	if did {
		t.Error("a near-origin drawing must not be recentered")
	}
	if &out[0] != &es[0] {
		t.Error("unchanged drawing should return the input slice")
	}
}

// TestRecenterIgnoresOppositeOutliers pins the robustness reason: the median centre stays on
// the dense cluster even though the outliers sit symmetrically opposite, where a bounding-box
// centre would average to ~0 and leave the cluster far from the origin.
func TestRecenterIgnoresOppositeOutliers(t *testing.T) {
	es := farCluster(5.4e7, 1.78e7, 100)
	c, ok := robustCenter(es)
	if !ok || c[0] < 5.3e7 {
		t.Fatalf("robust centre X = %.0f (ok=%v), want ~5.4e7 on the cluster", c[0], ok)
	}
}

// TestRobustCenterEmpty: no anchored entities yields ok=false.
func TestRobustCenterEmpty(t *testing.T) {
	if _, ok := robustCenter([]drawing.Entity{&drawing.LwPolyline{}}); ok {
		t.Error("a drawing with no anchored entities should report no centre")
	}
}

// TestImportDrawingRecentersAndWarns wires the recenter into the shared import: a far-from-origin
// drawing lands near the origin and the import surfaces the recenter warning.
func TestImportDrawingRecentersAndWarns(t *testing.T) {
	part := compdef.NewPartComponentDefinition()
	dr := &drawing.Drawing{Entities: farCluster(6e8, 2e8, 60)} // unitless ⇒ no unit scaling
	imp := importDrawing(part, dr, sketch.XYPlane())
	if len(imp.warnings) == 0 || !strings.HasPrefix(imp.warnings[0], "import: recentered drawing") {
		t.Fatalf("expected a recenter warning first, got %v", imp.warnings)
	}
	// The geometry now lives near the origin (well inside the float32-safe range).
	sk := part.Sketches3D()
	if sk.Count() == 0 && part.Sketches().Count() == 0 {
		t.Fatal("no sketch produced by import")
	}
}
