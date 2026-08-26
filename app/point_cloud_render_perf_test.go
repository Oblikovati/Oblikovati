// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
	"oblikovati.org/scene"
)

// TestFrustumClipDropsOffScreen: a point at the view centre survives the clip; a far off-screen one
// is dropped (#645 perf).
func TestFrustumClipDropsOffScreen(t *testing.T) {
	s, _ := emptyPartSession(t) // camera at (0,0,10) looking down at the XY plane, 200×200
	cam := s.Camera()
	if got := frustumClip(samplesAt(math.P3(0, 0, 0)), cam); len(got) != 1 {
		t.Errorf("on-screen point clipped away: %d, want 1", len(got))
	}
	if got := frustumClip(samplesAt(math.P3(1e6, 0, 0)), cam); len(got) != 0 {
		t.Errorf("off-screen point kept: %d, want 0", len(got))
	}
}

// samplesAt wraps model-space points as PointSamples for the sample-based LOD/clip helpers.
func samplesAt(pts ...math.Point3) []pointcloud.PointSample {
	out := make([]pointcloud.PointSample, len(pts))
	for i, p := range pts {
		out[i] = pointcloud.PointSample{Point: p}
	}
	return out
}

// TestLODThinByScreenArea: a small on-screen footprint thins the set; a large one keeps it all.
func TestLODThinByScreenArea(t *testing.T) {
	pts := make([]pointcloud.PointSample, 1000)
	thin := lodThin(pts, screenBox{minX: 0, minY: 0, maxX: 10, maxY: 10}) // 100 px² × 0.5 → ~50
	if len(thin) < 40 || len(thin) > 60 {
		t.Errorf("LOD thinned to %d, want ~50 for a 10×10 footprint", len(thin))
	}
	full := lodThin(pts, screenBox{minX: 0, minY: 0, maxX: 1000, maxY: 1000}) // huge area → no thin
	if len(full) != 1000 {
		t.Errorf("large footprint thinned to %d, want 1000", len(full))
	}
}

// TestVisibleDisplayPointsClips: a cloud moved off-screen yields no draw points; on-screen yields
// its full small set (no LOD thinning at that size) (#645 perf).
func TestVisibleDisplayPointsClips(t *testing.T) {
	s, def := emptyPartSession(t)
	cam := s.Camera()
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if got := visibleDisplaySamples(pc, cam); len(got) != 3 {
		t.Errorf("on-screen visible = %d, want 3", len(got))
	}
	pc.SetTransform(farX(1e6)) // shove the cloud far off-screen
	if got := visibleDisplaySamples(pc, cam); len(got) != 0 {
		t.Errorf("off-screen visible = %d, want 0 (clipped)", len(got))
	}
}

// farX is a +dx translation matrix.
func farX(dx float64) math.Matrix4 {
	m := math.Identity4()
	c := m.Cells()
	c[3] = math.Scalar(dx)
	return math.Matrix4FromCells(c)
}

// gridCloud builds a synthetic cloud of n×n points on the XY plane for render benchmarks.
func gridCloud(t testing.TB, n int) []math.Point3 {
	pts := make([]math.Point3, 0, n*n)
	for i := range n {
		for j := range n {
			pts = append(pts, math.P3(math.Scalar(i), math.Scalar(j), 0))
		}
	}
	return pts
}

// BenchmarkVisibleDisplayPoints measures the per-frame render prep (cached display + LOD + clip) for
// a ~250k-point cloud fully on screen — the work done each frame to build the marker batch.
func BenchmarkVisibleDisplayPoints(b *testing.B) {
	pc := pointcloud.New("Scan", "s.xyz", "rid", gridCloud(b, 500)) // 250k points
	cam := scene.NewCamera(200, 200)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	_ = pc.DisplayedPoints() // warm the display cache
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = visibleDisplaySamples(pc, cam)
	}
}

// BenchmarkVisibleDisplayPointsBudgeted: the realistic per-frame prep for a large scan the user has
// budgeted to 50k, partly off screen (the clip runs over the budgeted set).
func BenchmarkVisibleDisplayPointsBudgeted(b *testing.B) {
	pc := pointcloud.New("Scan", "s.xyz", "rid", gridCloud(b, 500))
	pc.SetMaximumPointCount(50000)
	cam := scene.NewCamera(200, 200)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	_ = pc.DisplayedPoints()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = visibleDisplaySamples(pc, cam)
	}
}

// BenchmarkVisibleDisplayPointsFullyVisible: a cloud framed fully in view — the fast path skips the
// per-point clip entirely (only the 8 box corners are projected).
func BenchmarkVisibleDisplayPointsFullyVisible(b *testing.B) {
	pc := pointcloud.New("Scan", "s.xyz", "rid", gridCloud(b, 500))
	cam := scene.NewCamera(200, 200)
	cam.Eye, cam.Target, cam.Up = math.P3(250, 250, 900), math.P3(250, 250, 0), math.V3(0, 1, 0) // frames the whole grid
	_ = pc.DisplayedPoints()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = visibleDisplaySamples(pc, cam)
	}
}
