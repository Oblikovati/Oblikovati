// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"math"
	"strings"
	"testing"

	omath "oblikovati.org/math"
)

// TestASCIIReaderParsesXYZAndSkipsNoise: the ASCII reader reads x y z lines, ignores blank,
// comment, and extra-column lines, and skips a leading PTS count header.
func TestASCIIReaderParsesXYZAndSkipsNoise(t *testing.T) {
	src := "3\n# a comment\n\n1 2 3\n4.5 6 7 128 255 0 0\n// trailing comment\n-1 -2 -3\n"
	pts, _, err := NewASCIIReader().Read([]byte(src))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(pts) != 3 {
		t.Fatalf("parsed %d points, want 3: %+v", len(pts), pts)
	}
	if pts[0] != omath.P3(1, 2, 3) || pts[1] != omath.P3(4.5, 6, 7) || pts[2] != omath.P3(-1, -2, -3) {
		t.Errorf("points = %+v, want (1,2,3),(4.5,6,7),(-1,-2,-3)", pts)
	}
}

// TestASCIIReaderWarnsOnMalformedLine: a non-coordinate line that is not a leading count header
// is skipped with a warning citing the line — the DWG decoder's warn-and-continue policy
// (#1646) — while a scan where NOTHING decodes still errors, naming the bad record.
func TestASCIIReaderWarnsOnMalformedLine(t *testing.T) {
	pts, warns, err := NewASCIIReader().Read([]byte("1 2 3\nx y z\n"))
	if err != nil || len(pts) != 1 {
		t.Fatalf("Read = %d points, err %v; want the good point kept", len(pts), err)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "line 2") || !strings.Contains(warns[0], "x y z") {
		t.Errorf("warns = %v, want one naming line 2 and its content", warns)
	}
	if _, _, err := NewASCIIReader().Read([]byte("x y z\n")); err == nil {
		t.Error("a scan with zero decodable points should error")
	}
}

// TestNewCloudDefaults: a fresh cloud is visible, unit-scaled, identity-placed, uncapped.
func TestNewCloudDefaults(t *testing.T) {
	pc := New("scan", "/s/room.xyz", "uuid-1", []omath.Point3{omath.P3(0, 0, 0)})
	if !pc.Visible() || pc.Scale() != 1 || pc.MaximumPointCount() != 0 {
		t.Errorf("defaults = visible %v scale %v max %d, want true/1/0", pc.Visible(), pc.Scale(), pc.MaximumPointCount())
	}
	if pc.SourceFullFileName() != "/s/room.xyz" || pc.ResourceID() != "uuid-1" {
		t.Errorf("source/resource = %q/%q", pc.SourceFullFileName(), pc.ResourceID())
	}
	if pc.Transform() != omath.Identity4() {
		t.Error("a fresh cloud should be identity-placed")
	}
}

// TestToFromModelSpaceRoundTrip: a scaled, translated placement maps cloud→model and back.
func TestToFromModelSpaceRoundTrip(t *testing.T) {
	pc := New("s", "", "", nil)
	pc.SetScale(2)
	pc.SetTransform(translation(10, 20, 30))

	p := omath.P3(1, 1, 1)
	m := pc.ToModelSpace(p)
	if m != omath.P3(12, 22, 32) { // 1*2 + 10, etc.
		t.Fatalf("ToModelSpace = %+v, want (12,22,32)", m)
	}
	back, ok := pc.FromModelSpace(m)
	if !ok || !near(back, p) {
		t.Errorf("FromModelSpace = %+v (ok=%v), want %+v", back, ok, p)
	}
}

// TestSetScaleRejectsNonPositive: scale must stay positive (it divides in FromModelSpace).
func TestSetScaleRejectsNonPositive(t *testing.T) {
	pc := New("s", "", "", nil)
	if pc.SetScale(0) || pc.SetScale(-3) {
		t.Error("SetScale accepted a non-positive factor")
	}
	if pc.Scale() != 1 {
		t.Errorf("scale = %v after rejected sets, want 1", pc.Scale())
	}
}

// TestDisplayBudgetStridesEvenly: a display cap strides across the scan rather than truncating,
// and the displayed count honours the cap.
func TestDisplayBudgetStridesEvenly(t *testing.T) {
	var pts []omath.Point3
	for i := 0; i < 100; i++ {
		pts = append(pts, omath.P3(omath.Scalar(i), 0, 0))
	}
	pc := New("s", "", "", pts)
	pc.SetMaximumPointCount(10)

	if pc.TotalPointCount() != 100 || pc.DisplayedPointCount() != 10 {
		t.Fatalf("counts = total %d displayed %d, want 100/10", pc.TotalPointCount(), pc.DisplayedPointCount())
	}
	shown := pc.DisplayedPoints()
	if len(shown) != 10 {
		t.Fatalf("DisplayedPoints len = %d, want 10", len(shown))
	}
	// Strided (every 10th), so the last shown x is 90, not 9 (which truncation would give).
	if shown[len(shown)-1].X != 90 {
		t.Errorf("last displayed x = %v, want 90 (even stride, not a prefix)", shown[len(shown)-1].X)
	}
}

// TestRangeBoxInModelSpace: the model-space range box reflects scale + placement.
func TestRangeBoxInModelSpace(t *testing.T) {
	pc := New("s", "", "", []omath.Point3{omath.P3(0, 0, 0), omath.P3(1, 1, 1)})
	pc.SetScale(3)
	box := pc.RangeBox()
	if box.IsEmpty() || box.Diagonal() != omath.V3(3, 3, 3) {
		t.Errorf("model range box diagonal = %+v, want (3,3,3)", box.Diagonal())
	}
}

// translation builds a pure-translation Matrix4 via the cloud transform setter's expectations.
func translation(x, y, z omath.Scalar) omath.Matrix4 {
	m := omath.Identity4()
	cells := m.Cells()
	cells[3], cells[7], cells[11] = x, y, z // column-major translation column (row-major last col)
	return omath.Matrix4FromCells(cells)
}

// near reports whether two points are within a small tolerance (float round-trip).
func near(a, b omath.Point3) bool {
	return math.Abs(float64(a.X-b.X)) < 1e-9 && math.Abs(float64(a.Y-b.Y)) < 1e-9 && math.Abs(float64(a.Z-b.Z)) < 1e-9
}

// TestNearestModelPoint: the closest scan point to a query is returned in model space, honouring
// the placement scale; an empty cloud reports not-found (#645).
func TestNearestModelPoint(t *testing.T) {
	pc := New("s", "", "", []omath.Point3{omath.P3(0, 0, 0), omath.P3(2, 0, 0), omath.P3(0, 2, 0)})
	got, ok := pc.NearestModelPoint(omath.P3(1.9, 0.1, 0))
	if !ok || !near(got, omath.P3(2, 0, 0)) {
		t.Errorf("nearest = %v (ok=%v), want (2,0,0)", got, ok)
	}
	pc.SetScale(3) // (0,2,0) → (0,6,0); a query near it snaps there
	if got, ok := pc.NearestModelPoint(omath.P3(0, 5.5, 0)); !ok || !near(got, omath.P3(0, 6, 0)) {
		t.Errorf("scaled nearest = %v (ok=%v), want (0,6,0)", got, ok)
	}
	if _, ok := New("e", "", "", nil).NearestModelPoint(omath.P3(0, 0, 0)); ok {
		t.Error("empty cloud should report not-found")
	}
}

// TestDisplayCacheInvalidates: the displayed set is cached but re-derives whenever the placement,
// scale, budget, or crops change (#645 perf).
func TestDisplayCacheInvalidates(t *testing.T) {
	pc := New("s", "", "", []omath.Point3{omath.P3(0, 0, 0), omath.P3(2, 0, 0), omath.P3(0, 2, 0), omath.P3(2, 2, 0)})
	base := pc.DisplayedPoints()
	if len(base) != 4 {
		t.Fatalf("displayed = %d, want 4", len(base))
	}
	if &pc.DisplayedPoints()[0] != &base[0] {
		t.Error("a second call with no change should return the cached slice")
	}
	pc.SetTransform(translation(0, 0, 10))
	if moved := pc.DisplayedPoints(); moved[0].Z != 10 {
		t.Errorf("after move, first point z = %v, want 10 (cache should have re-derived)", moved[0].Z)
	}
	pc.SetMaximumPointCount(2)
	if got := len(pc.DisplayedPoints()); got != 2 {
		t.Errorf("after budget=2, displayed = %d, want 2", got)
	}
	pc.SetMaximumPointCount(0)
	pc.AddCrop(omath.NewBox(omath.P3(-1, -1, 9), omath.P3(1, 1, 11)))
	if got := len(pc.DisplayedPoints()); got != 1 {
		t.Errorf("after crop, displayed = %d, want 1 (only the cropped point)", got)
	}
}
