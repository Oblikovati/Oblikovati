// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"os"
	"testing"

	"oblikovati.org/kernel/exchange/plyfmt"
	"oblikovati.org/math"
)

// realScan is a 266k-point Artec scan; benchmarks skip when it is absent (e.g. CI).
const realScan = "/home/vmiguel/git/oblikovati-workspace/vini-scan-examples/vini scan examples/capstan vertraging.ply"

// benchCloud loads the real scan into a cloud, or skips.
func benchCloud(b *testing.B) *PointCloud {
	b.Helper()
	data, err := os.ReadFile(realScan)
	if err != nil {
		b.Skipf("no sample scan: %v", err)
	}
	doc, err := plyfmt.Parse(data)
	if err != nil {
		b.Fatal(err)
	}
	pts, err := doc.Vertices()
	if err != nil {
		b.Fatal(err)
	}
	pc := New("Scan", "scan.ply", "rid", pts)
	b.Logf("loaded %d points", len(pts))
	return pc
}

// BenchmarkDisplayedPointsUnbounded measures the per-frame cost of the full display set (transform
// every point to model space + stride), which the head currently rebuilds every frame.
func BenchmarkDisplayedPointsUnbounded(b *testing.B) {
	pc := benchCloud(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pc.DisplayedPoints()
	}
}

// BenchmarkDisplayedPointsBudgeted measures the same with a 50k display budget.
func BenchmarkDisplayedPointsBudgeted(b *testing.B) {
	pc := benchCloud(b)
	pc.SetMaximumPointCount(50000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pc.DisplayedPoints()
	}
}

// BenchmarkDisplayedPointsCropped measures it with an active crop (transform + crop test per point).
func BenchmarkDisplayedPointsCropped(b *testing.B) {
	pc := benchCloud(b)
	box := pc.RangeBox()
	c := box.Center()
	pc.AddCrop(math.NewBox(math.P3(c.X-50, c.Y-50, c.Z-50), math.P3(c.X+50, c.Y+50, c.Z+50)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pc.DisplayedPoints()
	}
}

// BenchmarkNearestModelPoint measures a snap query over the whole cloud.
func BenchmarkNearestModelPoint(b *testing.B) {
	pc := benchCloud(b)
	q := pc.RangeBox().Center()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pc.NearestModelPoint(q)
	}
}

// BenchmarkDisplayedPointsRebuild measures the worst case — the placement changes every frame (a
// drag), so the cache misses and the set is rebuilt each time.
func BenchmarkDisplayedPointsRebuild(b *testing.B) {
	pc := benchCloud(b)
	pc.SetMaximumPointCount(50000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.SetTransform(liftZBench(float64(i))) // change the placement → cache miss
		_ = pc.DisplayedPoints()
	}
}

func liftZBench(dz float64) math.Matrix4 {
	m := math.Identity4()
	c := m.Cells()
	c[11] = math.Scalar(dz)
	return math.Matrix4FromCells(c)
}
