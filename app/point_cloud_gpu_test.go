// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
	"oblikovati.org/renderer"
)

// attachTestCloud attaches a 3-point cloud to the session's active part and returns it.
func attachTestCloud(t *testing.T) *Session {
	t.Helper()
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	if _, err := def.PointClouds().Add("Scan", "s.xyz", rid,
		[]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	return s
}

// TestPointCloudGPUVerticesInterleave: n points yield n*7 floats, positions first. RGB is the
// default mode, but a cloud without RGB data still falls back to the host marker color.
func TestPointCloudGPUVerticesInterleave(t *testing.T) {
	s := attachTestCloud(t)
	verts, n := s.PointCloudGPUVertices()
	if n != 3 {
		t.Fatalf("count = %d, want 3", n)
	}
	if len(verts) != n*pointCloudGPUStride {
		t.Fatalf("len(verts) = %d, want %d", len(verts), n*pointCloudGPUStride)
	}
	// First vertex is the origin point in the fallback marker color.
	if verts[0] != 0 || verts[1] != 0 || verts[2] != 0 {
		t.Errorf("first pos = (%v,%v,%v), want origin", verts[0], verts[1], verts[2])
	}
	if got := colorAtGPUVertex(verts, 0); got != renderer.PointCloudColor {
		t.Errorf("first color = %v, want default marker color %v", got, renderer.PointCloudColor)
	}
}

// TestPointCloudGPUVerticesColorsByDisplayMode checks the retained GL-points upload path carries
// the same RGB and intensity colors as the old CPU marker path, not the marker fallback.
func TestPointCloudGPUVerticesColorsByDisplayMode(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("scan")})
	pc, err := def.PointClouds().AddWithSamples("Scan", "s.xyz", rid, []pointcloud.PointSample{
		{Point: math.P3(0, 0, 0), HasRGB: true, RGB: [3]float32{255, 0, 0}},
		{Point: math.P3(1, 0, 0), HasRGB: true, RGB: [3]float32{0, 255, 0}},
		{Point: math.P3(0, 1, 0), HasIntensity: true, Intensity: 10},
		{Point: math.P3(1, 1, 0), HasIntensity: true, Intensity: 20},
	})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}

	pc.SetDisplayMode(types.PointCloudDisplayModeRGB)
	verts, n := s.PointCloudGPUVertices()
	if n != 4 {
		t.Fatalf("rgb count = %d, want 4", n)
	}
	if got := colorAtGPUVertex(verts, 0); got != [4]float32{1, 0, 0, 1} {
		t.Errorf("rgb first color = %v, want red", got)
	}
	if got := colorAtGPUVertex(verts, 1); got != [4]float32{0, 1, 0, 1} {
		t.Errorf("rgb second color = %v, want green", got)
	}

	pc.SetDisplayMode(types.PointCloudDisplayModeIntensity)
	verts, n = s.PointCloudGPUVertices()
	if n != 4 {
		t.Fatalf("intensity count = %d, want 4", n)
	}
	if got := colorAtGPUVertex(verts, 2); got != [4]float32{1, 0, 0, 1} {
		t.Errorf("intensity min color = %v, want red", got)
	}
	if got := colorAtGPUVertex(verts, 3); got != [4]float32{1, 1, 0, 1} {
		t.Errorf("intensity max color = %v, want yellow", got)
	}
}

func colorAtGPUVertex(verts []float32, i int) [4]float32 {
	base := i*pointCloudGPUStride + 3
	return [4]float32{verts[base], verts[base+1], verts[base+2], verts[base+3]}
}

// TestStrideForCapThinsAndPreserves: over-cap slices thin to ~max at an even stride; within-cap and
// non-positive-cap slices pass through untouched.
func TestStrideForCapThinsAndPreserves(t *testing.T) {
	s := make([]pointcloud.PointSample, 1000)
	if got := strideForCap(s, 100); len(got) < 90 || len(got) > 100 {
		t.Errorf("cap 100 gave %d, want ~100", len(got))
	}
	if got := strideForCap(s, 5000); len(got) != 1000 {
		t.Errorf("within-cap thinned to %d, want 1000", len(got))
	}
	if got := strideForCap(s, 0); len(got) != 1000 {
		t.Errorf("cap 0 thinned to %d, want passthrough 1000", len(got))
	}
}

// TestCapForRenderRespectsExplicitBudget: a cloud the user budgeted is returned unchanged (the model
// already applied that budget); only an unbudgeted cloud is subject to the render cap.
func TestCapForRenderRespectsExplicitBudget(t *testing.T) {
	pc := pointcloud.New("Scan", "s.xyz", "rid", []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)})
	pc.SetMaximumPointCount(1)
	s := make([]pointcloud.PointSample, 10)
	if got := capForRender(pc, s); len(got) != 10 {
		t.Errorf("budgeted cloud was re-capped to %d, want the model's set unchanged (10)", len(got))
	}
}

// TestPointCloudDisplayKeyStable: identical state hashes identically (so the head skips re-upload).
func TestPointCloudDisplayKeyStable(t *testing.T) {
	s := attachTestCloud(t)
	if a, b := s.PointCloudDisplayKey(), s.PointCloudDisplayKey(); a != b {
		t.Errorf("key not stable across identical calls: %d then %d", a, b)
	}
}

// TestPointCloudDisplayKeyChangesOnBudget: changing the display budget changes the key (forces a
// re-upload of the newly-thinned set), while the empty scene stays non-zero (0 = "always upload").
func TestPointCloudDisplayKeyChangesOnBudget(t *testing.T) {
	empty, _ := emptyPartSession(t)
	if empty.PointCloudDisplayKey() == 0 {
		t.Error("empty-scene key is 0, which the renderer reserves for always-upload")
	}
	s := attachTestCloud(t)
	before := s.PointCloudDisplayKey()
	s.PickablePointClouds()[0].SetMaximumPointCount(2)
	if after := s.PointCloudDisplayKey(); after == before {
		t.Error("key unchanged after the display budget changed")
	}
}

// TestPointCloudDisplayKeyChangesOnDisplayMode pins the retained-upload invalidation path: display
// mode changes bake new colors into the GPU vertices, so they must force one re-upload.
func TestPointCloudDisplayKeyChangesOnDisplayMode(t *testing.T) {
	s := attachTestCloud(t)
	pc := s.PickablePointClouds()[0]
	before := s.PointCloudDisplayKey()
	pc.SetDisplayMode(types.PointCloudDisplayModeDefault)
	def := s.PointCloudDisplayKey()
	if def == before {
		t.Fatal("key unchanged after switching to default display mode")
	}
	pc.SetDisplayMode(types.PointCloudDisplayModeIntensity)
	if after := s.PointCloudDisplayKey(); after == def {
		t.Error("key unchanged after switching to intensity display mode")
	}
}

// TestPointCloudRenderDensityDefaultAndClamp pins the session-level viewport density knob: it
// starts at full density and clamps UI/API input to the valid percentage range.
func TestPointCloudRenderDensityDefaultAndClamp(t *testing.T) {
	s := NewSession()
	if got := s.PointCloudRenderDensity(); got != 100 {
		t.Fatalf("default render density = %g, want 100", got)
	}
	s.SetPointCloudRenderDensity(-12)
	if got := s.PointCloudRenderDensity(); got != 0 {
		t.Errorf("negative render density = %g, want clamp to 0", got)
	}
	s.SetPointCloudRenderDensity(125)
	if got := s.PointCloudRenderDensity(); got != 100 {
		t.Errorf("oversize render density = %g, want clamp to 100", got)
	}
	s.SetPointCloudRenderDensity(37.5)
	if got := s.PointCloudRenderDensity(); got != 37.5 {
		t.Errorf("render density = %g, want 37.5", got)
	}
}

// TestPointCloudPointSizeDefaultAndClamp pins the session-level native point-size knob: it starts
// at one pixel and clamps UI/API input to the supported point-size range.
func TestPointCloudPointSizeDefaultAndClamp(t *testing.T) {
	s := NewSession()
	if got := s.PointCloudPointSize(); got != 1 {
		t.Fatalf("default point size = %g, want 1", got)
	}
	s.SetPointCloudPointSize(-4)
	if got := s.PointCloudPointSize(); got != 1 {
		t.Errorf("negative point size = %g, want clamp to 1", got)
	}
	s.SetPointCloudPointSize(12)
	if got := s.PointCloudPointSize(); got != 10 {
		t.Errorf("oversize point size = %g, want clamp to 10", got)
	}
	s.SetPointCloudPointSize(4.5)
	if got := s.PointCloudPointSize(); got != 4.5 {
		t.Errorf("point size = %g, want 4.5", got)
	}
}

// TestPointCloudIntensityRampDefaultAndSet pins the global intensity colors: low defaults red,
// high defaults yellow, and alpha remains opaque even if a caller passes a transparent value.
func TestPointCloudIntensityRampDefaultAndSet(t *testing.T) {
	s := NewSession()
	low, high := s.PointCloudIntensityRamp()
	if low != [4]float32{1, 0, 0, 1} || high != [4]float32{1, 1, 0, 1} {
		t.Fatalf("default intensity ramp = %v/%v, want red/yellow", low, high)
	}
	s.SetPointCloudIntensityRamp([4]float32{-1, 0.25, 2, 0}, [4]float32{0.1, 0.2, 0.3, 0})
	low, high = s.PointCloudIntensityRamp()
	if low != [4]float32{0, 0.25, 1, 1} || high != [4]float32{0.1, 0.2, 0.3, 1} {
		t.Fatalf("normalized intensity ramp = %v/%v", low, high)
	}
}

// TestPointCloudDisplayKeyChangesOnIntensityRamp proves ramp edits invalidate the retained point
// buffer because intensity colors are baked into uploaded vertices.
func TestPointCloudDisplayKeyChangesOnIntensityRamp(t *testing.T) {
	s := attachTestCloud(t)
	s.PickablePointClouds()[0].SetDisplayMode(types.PointCloudDisplayModeIntensity)
	before := s.PointCloudDisplayKey()
	s.SetPointCloudIntensityRamp([4]float32{0, 0, 1, 1}, [4]float32{0, 1, 0, 1})
	if after := s.PointCloudDisplayKey(); after == before {
		t.Error("key unchanged after point-cloud intensity ramp changed")
	}
}

// TestDensityFilteredSamplesStableAndApproximate checks that render density keeps a deterministic
// random-looking subset rather than a prefix or a per-frame random draw.
func TestDensityFilteredSamplesStableAndApproximate(t *testing.T) {
	pc := pointcloud.New("Scan", "s.xyz", "rid", denseTestPoints(10000))
	samples := pc.DisplayedSamples()
	a := densityFilteredSamples(pc, samples, 25)
	b := densityFilteredSamples(pc, samples, 25)
	if len(a) < 2200 || len(a) > 2800 {
		t.Fatalf("25%% density kept %d samples, want about 2500", len(a))
	}
	if len(a) != len(b) {
		t.Fatalf("density filter was unstable: %d then %d samples", len(a), len(b))
	}
	for i := range a {
		if a[i].Point != b[i].Point {
			t.Fatalf("density filter changed sample %d: %v then %v", i, a[i].Point, b[i].Point)
		}
	}
	if a[0].Point == samples[0].Point && a[1].Point == samples[1].Point && a[2].Point == samples[2].Point {
		t.Error("density filter appears to keep a prefix, want randomly distributed samples")
	}
	if got := densityFilteredSamples(pc, samples, 0); len(got) != 0 {
		t.Errorf("0%% density kept %d samples, want 0", len(got))
	}
	if got := densityFilteredSamples(pc, samples, 100); len(got) != len(samples) {
		t.Errorf("100%% density kept %d samples, want all %d", len(got), len(samples))
	}
}

// TestDensityFilteredSamplesIgnoresName pins that the density filter seeds on ResourceID only:
// renaming a cloud must not change which samples it keeps, because the display key (and thus the
// retained GPU buffer) never sees the name (#645).
func TestDensityFilteredSamplesIgnoresName(t *testing.T) {
	pc := pointcloud.New("Scan", "s.xyz", "rid", denseTestPoints(10000))
	samples := pc.DisplayedSamples()
	before := densityFilteredSamples(pc, samples, 25)
	pc.SetName("Renamed Scan")
	after := densityFilteredSamples(pc, samples, 25)
	if len(before) != len(after) {
		t.Fatalf("rename changed filtered count: %d then %d", len(before), len(after))
	}
	for i := range before {
		if before[i].Point != after[i].Point {
			t.Fatalf("rename changed kept sample %d: %v then %v", i, before[i].Point, after[i].Point)
		}
	}
}

// TestPointCloudGPUVerticesApplyRenderDensity verifies the native point-upload path receives the
// density-filtered point count, not the full cloud.
func TestPointCloudGPUVerticesApplyRenderDensity(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("scan")})
	if _, err := def.PointClouds().Add("Scan", "s.xyz", rid, denseTestPoints(1000)); err != nil {
		t.Fatalf("attach: %v", err)
	}
	s.SetPointCloudRenderDensity(10)
	verts, n := s.PointCloudGPUVertices()
	if n < 70 || n > 130 {
		t.Fatalf("10%% density uploaded %d points, want about 100", n)
	}
	if len(verts) != n*pointCloudGPUStride {
		t.Errorf("len(verts) = %d, want %d", len(verts), n*pointCloudGPUStride)
	}
	s.SetPointCloudRenderDensity(0)
	verts, n = s.PointCloudGPUVertices()
	if n != 0 || len(verts) != 0 {
		t.Errorf("0%% density uploaded %d points / %d floats, want none", n, len(verts))
	}
}

// TestPointCloudDisplayKeyChangesOnRenderDensity proves slider edits invalidate the retained
// point buffer exactly like budget/display-mode edits do.
func TestPointCloudDisplayKeyChangesOnRenderDensity(t *testing.T) {
	s := attachTestCloud(t)
	before := s.PointCloudDisplayKey()
	s.SetPointCloudRenderDensity(50)
	if after := s.PointCloudDisplayKey(); after == before {
		t.Error("key unchanged after point-cloud render density changed")
	}
}

func denseTestPoints(n int) []math.Point3 {
	pts := make([]math.Point3, n)
	for i := range pts {
		pts[i] = math.P3(math.Scalar(i%97), math.Scalar((i*37)%101), math.Scalar(i/97))
	}
	return pts
}
