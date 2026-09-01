// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/renderer"
)

// The scene value types are the router-facing mirror of the renderer/scene internals (audit
// B10, #1621). These tests pin the wall converters as bijections and check the Session
// accessors round-trip through them, so a dropped field surfaces here rather than as a
// half-populated add-in reply.

// TestLightValueRoundTrip: renderLight∘lightValue is identity on a fully non-zero light.
func TestLightValueRoundTrip(t *testing.T) {
	t.Parallel()
	in := renderer.SceneLight{
		Kind: renderer.SpotLight, On: true, Color: [3]float32{0.1, 0.2, 0.3}, Intensity: 2,
		Direction: [3]float32{0, 0, 1}, Position: [3]float32{4, 5, 6},
		SpotInner: 0.3, SpotOuter: 0.6, Attenuation: [3]float32{1, 0.1, 0.01},
	}
	if got := renderLight(lightValue(in)); got != in {
		t.Errorf("light round-trip = %+v, want %+v", got, in)
	}
}

// TestShadowRigRoundTrip: renderShadowRig∘shadowRigValue is identity.
func TestShadowRigRoundTrip(t *testing.T) {
	t.Parallel()
	in := renderer.ShadowSettings{
		GroundShadows: true, GroundXRay: true, ObjectShadows: true,
		AmbientShadows: true, Density: 0.7, Softness: 0.4,
	}
	if got := renderShadowRig(shadowRigValue(in)); got != in {
		t.Errorf("shadow round-trip = %+v, want %+v", got, in)
	}
}

// TestEnvironmentValueRoundTrip: a preset and a file environment both survive the wall.
func TestEnvironmentValueRoundTrip(t *testing.T) {
	t.Parallel()
	preset := renderer.Environment{Preset: renderer.EnvStudio, Rotation: 1, Intensity: 2, ShowImage: true}
	if got := renderEnvironment(environmentValue(preset)); got != preset {
		t.Errorf("preset env round-trip = %+v, want %+v", got, preset)
	}
	file := renderer.Environment{Preset: renderer.EnvNone, FilePath: "/tmp/x.hdr", Intensity: 1, ShowImage: true}
	if got := renderEnvironment(environmentValue(file)); got != file {
		t.Errorf("file env round-trip = %+v, want %+v", got, file)
	}
}

// TestEnvironmentStateIsActive matches renderer.Environment.IsActive across the wall.
func TestEnvironmentStateIsActive(t *testing.T) {
	t.Parallel()
	cases := []EnvironmentState{{Preset: "None"}, {Preset: "Sky"}, {FilePath: "/x.hdr"}, {}}
	for _, e := range cases {
		want := renderEnvironment(e).IsActive()
		if e.IsActive() != want {
			t.Errorf("IsActive(%+v) = %v, want %v", e, e.IsActive(), want)
		}
	}
}

// TestCameraFrameRoundTrip: renderCamera∘cameraFrameValue is identity.
func TestCameraFrameRoundTrip(t *testing.T) {
	t.Parallel()
	in := renderCamera(CameraFrame{FOV: 0.8, Width: 1280, Height: 800, Orthographic: true})
	if got := renderCamera(cameraFrameValue(in)); got != in {
		t.Errorf("camera round-trip = %+v, want %+v", got, in)
	}
}

// TestSessionLightingValueSeam round-trips the value types through the Session accessors.
func TestSessionLightingValueSeam(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.SetShadowSettings(ShadowRig{ObjectShadows: true, Density: 0.5})
	if !s.ShadowSettings().ObjectShadows {
		t.Error("shadow rig did not round-trip through the Session")
	}
	l, err := s.AddLight(types.SpotLight)
	if err != nil {
		t.Fatalf("AddLight: %v", err)
	}
	if l.Definition != types.SpotLight {
		t.Errorf("AddLight returned definition %v, want SpotLight", l.Definition)
	}
}

// TestSceneGalleriesAreNamed: the app-typed galleries expose the renderer gallery names.
func TestSceneGalleriesAreNamed(t *testing.T) {
	t.Parallel()
	if len(LightingStyleGallery()) != len(renderer.LightingStyleGallery()) {
		t.Error("lighting-style gallery lost entries crossing the wall")
	}
	if got := EnvironmentGallery(); len(got) == 0 || got[0].Name == "" {
		t.Errorf("environment gallery is empty or unnamed: %+v", got)
	}
}
