//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
)

func vcCamera() scene.Camera {
	c := scene.NewCamera(800, 600)
	c.Eye, c.Target, c.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	return c
}

func TestRolledViewRotatesUpKeepingFraming(t *testing.T) {
	cam := vcCamera()
	rolled := rolledView(cam, true)
	if rolled.Eye != cam.Eye || rolled.Target != cam.Target {
		t.Error("roll must keep the eye/target (framing) — only the up vector turns")
	}
	if stdmath.Abs(rolled.Up.Dot(cam.Up)) > 1e-9 {
		t.Errorf("roll should turn the up vector 90° (perpendicular): up·up' = %v", rolled.Up.Dot(cam.Up))
	}
	if rolledView(cam, false).Up.IsEqualTo(rolled.Up, 1e-9) {
		t.Error("CW and CCW roll should give opposite up vectors")
	}
}

func TestAdjacentViewSnapsToNeighbourFace(t *testing.T) {
	// A FRONT view (camera looking from −Y, up +Z): the up arrow should land on TOP (+Z).
	cam := scene.NewCamera(800, 600)
	cam.Eye, cam.Target, cam.Up = math.P3(0, -10, 0), math.P3(0, 0, 0), math.V3(0, 0, 1)
	o := doc.IdentityCubeOrient()

	up := adjacentView(cam, AdjacentUp, o, math.P3(0, 0, 0))
	if up.Eye.Z <= 0 {
		t.Errorf("up arrow from FRONT should look from above (+Z): eye=%v", up.Eye)
	}
	for _, dir := range []AdjacentDir{AdjacentUp, AdjacentDown, AdjacentLeft, AdjacentRight} {
		adj := adjacentView(cam, dir, o, math.P3(0, 0, 0))
		if adj.Eye.IsEqualTo(cam.Eye, 1e-9) {
			t.Errorf("adjacent %d should move the eye to the neighbour face", dir)
		}
		if d := adj.Eye.DistanceTo(adj.Target) - cam.Eye.DistanceTo(cam.Target); d > 1e-6 || d < -1e-6 {
			t.Errorf("adjacent %d must preserve the eye–target distance (Δ=%v)", dir, d)
		}
	}
}

func TestHitViewCubeArrowZones(t *testing.T) {
	p := cubePlacement{cx: 100, cy: 100, r: 20}
	adj := p.r * viewCubeAdjOff
	if h := hitViewCubeArrow(p.cx, p.cy-adj, p); h.kind != arrowAdjacent || h.dir != AdjacentUp {
		t.Errorf("top zone = %+v, want adjacent-up", h)
	}
	if h := hitViewCubeArrow(p.cx+adj, p.cy, p); h.kind != arrowAdjacent || h.dir != AdjacentRight {
		t.Errorf("right zone = %+v, want adjacent-right", h)
	}
	roll := p.r * viewCubeRollOff
	if h := hitViewCubeArrow(p.cx+roll*0.62, p.cy-roll*0.95, p); h.kind != arrowRoll || !h.ccw {
		t.Errorf("upper roll zone = %+v, want roll-ccw", h)
	}
	if h := hitViewCubeArrow(p.cx, p.cy, p); h.kind != arrowNone { // cube centre: no arrow
		t.Errorf("cube centre = %+v, want no arrow", h)
	}
}

// fakeArrowSession captures the animated camera for the arrow-action test.
type fakeArrowSession struct {
	cam      scene.Camera
	animated scene.Camera
	did      bool
}

func (f *fakeArrowSession) Camera() scene.Camera                      { return f.cam }
func (f *fakeArrowSession) SetCamera(c scene.Camera)                  { f.cam = c }
func (f *fakeArrowSession) AnimateCameraTo(c scene.Camera, _ float64) { f.animated, f.did = c, true }
func (f *fakeArrowSession) CubeOrientation() doc.CubeOrient           { return doc.IdentityCubeOrient() }
func (f *fakeArrowSession) ViewCubePivot() math.Point3                { return math.P3(0, 0, 0) }

func TestApplyViewCubeArrowAnimates(t *testing.T) {
	f := &fakeArrowSession{cam: vcCamera()}
	applyViewCubeArrow(f, cubeArrowHit{kind: arrowRoll, ccw: true}, 800, 600)
	if !f.did {
		t.Fatal("a roll arrow should animate the camera")
	}
	if stdmath.Abs(f.animated.Up.Dot(vcCamera().Up)) > 1e-9 {
		t.Error("the roll arrow should animate to a 90°-rolled view")
	}
}
