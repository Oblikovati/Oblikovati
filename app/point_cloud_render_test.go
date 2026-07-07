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

// TestPointCloudItemsRendersVisibleClouds: a visible attached cloud yields one Lines marker batch
// (6 vertices per point); hiding it removes the batch (M17-F06, #645).
func TestPointCloudItemsRendersVisibleClouds(t *testing.T) {
	s, def := emptyPartSession(t)
	if got := s.PointCloudItems(s.Camera(), 0.5); len(got) != 0 {
		t.Fatalf("a part with no clouds yields %d items, want 0", len(got))
	}

	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("scan")})
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 1), math.P3(2, 2, 2)}
	pc, err := def.PointClouds().Add("Cloud1", "c.xyz", rid, pts)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}

	items := s.PointCloudItems(s.Camera(), 0.5)
	if len(items) != 1 || items[0].Primitive != renderer.Lines {
		t.Fatalf("items = %+v, want one Lines batch", items)
	}
	if len(items[0].Positions) != len(pts)*6 {
		t.Errorf("positions = %d, want %d (6 per point)", len(items[0].Positions), len(pts)*6)
	}

	pc.SetVisible(false)
	if got := s.PointCloudItems(s.Camera(), 0.5); len(got) != 0 {
		t.Errorf("a hidden cloud still rendered %d items, want 0", len(got))
	}
}

// TestRGBModeDoesNotReamplifyNormalisedColor is the #1787 regression: colour arrives at the renderer
// already normalised to 0..1 by the reader (a dark 16-bit point is ~0.003, not 200), so RGB mode must
// pass it through with a clamp — no per-point bit-depth guessing that divided a dark point by 255 and
// rendered it ~256× too bright. A dark point stays dark; a saturated point stays white.
func TestRGBModeDoesNotReamplifyNormalisedColor(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("scan")})
	dark := [3]float32{200.0 / 65535, 200.0 / 65535, 200.0 / 65535} // a dark 16-bit point, decoded
	pc, err := def.PointClouds().AddWithSamples("Cloud1", "c.las", rid, []pointcloud.PointSample{
		{Point: math.P3(0, 0, 5), HasRGB: true, RGB: dark},
		{Point: math.P3(1, 0, 5), HasRGB: true, RGB: [3]float32{1, 1, 1}},
	})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	pc.SetDisplayMode(types.PointCloudDisplayModeRGB)
	items := s.PointCloudItems(s.Camera(), 0.5)
	if len(items) != 1 {
		t.Fatalf("items = %+v, want one batch", items)
	}
	if got := items[0].Colors[0]; got != [4]float32{dark[0], dark[1], dark[2], 1} {
		t.Errorf("dark 16-bit point rendered %v, want it dim %v (not re-amplified)", got, dark)
	}
	if got := items[0].Colors[6]; got != [4]float32{1, 1, 1, 1} {
		t.Errorf("saturated point rendered %v, want white {1,1,1,1}", got)
	}
}

// TestPointCloudBounds returns the model-space extent of the visible clouds — the box the viewport
// far plane consults to enclose a large or distant scan (#1789). No clouds, or only hidden ones,
// yield an empty box.
func TestPointCloudBounds(t *testing.T) {
	s, def := emptyPartSession(t)
	if !s.PointCloudBounds().IsEmpty() {
		t.Fatal("a part with no clouds must report an empty point-cloud box")
	}

	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("scan")})
	pts := []math.Point3{math.P3(-100, -200, -300), math.P3(100, 200, 300)}
	pc, err := def.PointClouds().Add("Cloud1", "c.xyz", rid, pts)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}

	box := s.PointCloudBounds()
	if box.Min != math.P3(-100, -200, -300) || box.Max != math.P3(100, 200, 300) {
		t.Errorf("cloud bounds = %v..%v, want (-100,-200,-300)..(100,200,300)", box.Min, box.Max)
	}

	pc.SetVisible(false)
	if !s.PointCloudBounds().IsEmpty() {
		t.Error("a hidden cloud must not contribute to the point-cloud box")
	}
}

// TestPointCloudItemsColorsByDisplayMode checks the draw batch carries per-point colors when a
// cloud is set to RGB or intensity display.
func TestPointCloudItemsColorsByDisplayMode(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("scan")})
	// Keep all four points within the emptyPartSession 200×200 viewport at z=5 so none is frustum-
	// clipped from the batch (a point at x=3 falls off-screen, dropping the intensity max sample).
	pc, err := def.PointClouds().AddWithSamples("Cloud1", "c.xyz", rid, []pointcloud.PointSample{
		{Point: math.P3(0, 0, 5), HasRGB: true, RGB: [3]float32{1, 0, 0}},
		{Point: math.P3(1, 0, 5), HasRGB: true, RGB: [3]float32{0, 1, 0}},
		{Point: math.P3(0, 1, 5), HasIntensity: true, Intensity: 10},
		{Point: math.P3(1, 1, 5), HasIntensity: true, Intensity: 20},
	})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	pc.SetDisplayMode(types.PointCloudDisplayModeDefault)
	items := s.PointCloudItems(s.Camera(), 0.5)
	if len(items) != 1 || len(items[0].Colors) != len(items[0].Positions) {
		t.Fatalf("default item colors = %+v", items)
	}
	if items[0].Colors[0] != renderer.PointCloudColor {
		t.Errorf("default color = %v, want %v", items[0].Colors[0], renderer.PointCloudColor)
	}

	pc.SetDisplayMode(types.PointCloudDisplayModeRGB)
	items = s.PointCloudItems(s.Camera(), 0.5)
	if len(items) != 1 || len(items[0].Colors) != len(items[0].Positions) {
		t.Fatalf("rgb item colors = %+v", items)
	}
	if items[0].Colors[0] != [4]float32{1, 0, 0, 1} || items[0].Colors[6] != [4]float32{0, 1, 0, 1} {
		t.Errorf("rgb colors = %+v", items[0].Colors[:12])
	}

	pc.SetDisplayMode(types.PointCloudDisplayModeIntensity)
	items = s.PointCloudItems(s.Camera(), 0.5)
	if len(items) != 1 || len(items[0].Colors) != len(items[0].Positions) {
		t.Fatalf("intensity item colors = %+v", items)
	}
	// The intensity samples are points 2 and 3 (indices 0,1 carry only RGB, so they stay the default
	// color in intensity mode). Each point expands to 6 marker vertices, so point 2's color is at
	// index 12 (intensity 10 = range min → ramp low red) and point 3's at index 18
	// (20 = max → ramp high yellow).
	if items[0].Colors[12] != [4]float32{1, 0, 0, 1} || items[0].Colors[18] != [4]float32{1, 1, 0, 1} {
		t.Errorf("intensity colors = %+v", items[0].Colors[12:24])
	}
}

// TestPointCloudItemsApplyRenderDensity keeps the CPU marker path aligned with the retained native
// point-buffer path: density is render-only, but every point-cloud draw path must honor it.
func TestPointCloudItemsApplyRenderDensity(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("scan")})
	if _, err := def.PointClouds().Add("Cloud1", "c.xyz", rid, []math.Point3{
		math.P3(0, 0, 5), math.P3(1, 0, 5), math.P3(0, 1, 5),
	}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	s.SetPointCloudRenderDensity(0)
	if got := s.PointCloudItems(s.Camera(), 0.5); len(got) != 0 {
		t.Fatalf("0%% render density produced %d marker batches, want none", len(got))
	}
}
