//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
)

// TestViewportFarPlaneEnclosesPointCloud: a large point cloud viewed from far enough that the fixed
// 5000 far plane would clip it must get a far plane beyond the farthest point. Clouds render from a
// separate retained-buffer draw the instanced/overlay frameBounds never see, so without unioning
// their bounds a scan-only part would fall entirely behind the far plane and render nothing (#1789).
func TestViewportFarPlaneEnclosesPointCloud(t *testing.T) {
	s := framedSession()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("scan")})
	// A 100k-unit-wide cloud centred at the origin.
	if _, err := def.PointClouds().Add("Scan", "s.xyz", rid,
		[]math.Point3{math.P3(-50000, -50000, -50000), math.P3(50000, 50000, 50000)}); err != nil {
		t.Fatalf("attach: %v", err)
	}

	// Camera pulled back 150k on Z with no body/overlay geometry (hasGeom=false), so only the cloud
	// bounds can extend the far plane.
	cam := scene.Camera{Eye: math.P3(0, 0, 150000), Target: math.P3(0, 0, 0), Up: math.V3(0, 1, 0), FOV: 0.8, Width: inWinW, Height: inWinH}
	far := viewportFarPlane(s, cam, [3]float32{}, [3]float32{}, false)
	if far <= viewportFar {
		t.Fatalf("scan-only far = %v, expected it to extend past the fixed %v", far, viewportFar)
	}
	farthest := cam.Eye.DistanceTo(math.P3(50000, 50000, 50000))
	if far < farthest {
		t.Errorf("far plane %v does not enclose the farthest cloud point at %v", far, farthest)
	}
}

// TestViewportFarPlaneKeepsDefaultWithoutClouds: a part with no clouds and no framed geometry keeps
// the fixed far plane, so the cloud-bounds union leaves ordinary scenes unchanged.
func TestViewportFarPlaneKeepsDefaultWithoutClouds(t *testing.T) {
	s := framedSession()
	cam := scene.Camera{Eye: math.P3(40, 40, 40), Target: math.P3(0, 0, 0), Up: math.V3(0, 1, 0), FOV: 0.8, Width: inWinW, Height: inWinH}
	if got := viewportFarPlane(s, cam, [3]float32{}, [3]float32{}, false); got != viewportFar {
		t.Errorf("no-cloud far = %v, want fixed %v", got, viewportFar)
	}
}
