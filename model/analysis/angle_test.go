// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// TestAngleBox checks angle measurement on a 4×3×5 cm block: every face is 90° from each adjacent
// face and 180° from its opposite, every meeting edge pair is 90°, and the three edges at a corner
// vertex make 90° three-point angles.
func TestAngleBox(t *testing.T) {
	block, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(4, 3, 5), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	q := ops.DefaultQuality()
	faces := block.Faces()

	sawOpposite := false
	for i := range faces {
		for j := i + 1; j < len(faces); j++ {
			deg, err := AngleDegrees(MeasureEntity{Face: faces[i]}, MeasureEntity{Face: faces[j]}, q)
			if err != nil {
				t.Fatalf("AngleDegrees(face%d,face%d): %v", i, j, err)
			}
			if !near(deg, 90) && !near(deg, 180) {
				t.Errorf("angle(face%d,face%d) = %g°, want 90 or 180", i, j, deg)
			}
			if near(deg, 180) {
				sawOpposite = true
			}
		}
	}
	if !sawOpposite {
		t.Error("no opposite face pair measured 180°")
	}

	// A vertex has no direction → entity angle rejects it.
	if _, err := AngleDegrees(MeasureEntity{Vertex: block.Vertices()[0]}, MeasureEntity{Face: faces[0]}, q); err == nil {
		t.Error("AngleDegrees(vertex, face) = ok, want error")
	}
}

// TestAngleEdgesAndThreePoint checks a perpendicular edge pair and a corner three-point angle.
func TestAngleEdgesAndThreePoint(t *testing.T) {
	block, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(4, 3, 5), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	q := ops.DefaultQuality()

	// Adjacent box edges meet at right angles: at least one edge pair measures 90°.
	edges := block.Edges()
	sawRight := false
	for i := range edges {
		for j := i + 1; j < len(edges); j++ {
			deg, err := AngleDegrees(MeasureEntity{Edge: edges[i]}, MeasureEntity{Edge: edges[j]}, q)
			if err != nil {
				t.Fatalf("edge angle: %v", err)
			}
			if near(deg, 90) {
				sawRight = true
			}
		}
	}
	if !sawRight {
		t.Error("no perpendicular edge pair measured 90°")
	}

	// Three box corners sharing axes: apex (0,0,0) to (4,0,0) and (0,3,0) is a right angle.
	apex := vertexAt(t, block, 0, 0, 0)
	px := vertexAt(t, block, 4, 0, 0)
	py := vertexAt(t, block, 0, 3, 0)
	if deg := ThreePointAngleDegrees(px, apex, py); !near(deg, 90) {
		t.Errorf("three-point angle = %g°, want 90", deg)
	}
}

func vertexAt(t *testing.T, b *topo.Body, x, y, z float64) *topo.Vertex {
	t.Helper()
	want := gmath.P3(x, y, z)
	for _, v := range b.Vertices() {
		if v.Point().DistanceTo(want) < 1e-6 {
			return v
		}
	}
	t.Fatalf("no vertex at (%g,%g,%g)", x, y, z)
	return nil
}

func near(got, want float64) bool { return math.Abs(got-want) < 1e-6 }

// TestAngleCurvedFacesUseAnalyticNormal: the angle of a CURVED face comes from its analytic surface
// normal at a representative boundary point (#3456), not from a mesh triangle. A cylinder's side is
// radial there, so it stands at 90° from both caps.
func TestAngleCurvedFacesUseAnalyticNormal(t *testing.T) {
	cyl, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	side, caps := cylinderSideAndCaps(t, cyl)
	for i, c := range caps {
		deg, err := AngleDegrees(MeasureEntity{Face: side}, MeasureEntity{Face: c}, ops.DefaultQuality())
		if err != nil {
			t.Fatalf("AngleDegrees(side, cap%d): %v", i, err)
		}
		if !near(deg, 90) {
			t.Errorf("angle(cylinder side, cap%d) = %g°, want 90", i, deg)
		}
	}
}

// cylinderSideAndCaps splits a cylinder's faces into its cylindrical side and its two planar caps.
func cylinderSideAndCaps(t *testing.T, cyl *topo.Body) (side *topo.Face, caps []*topo.Face) {
	t.Helper()
	for _, f := range cyl.Faces() {
		if _, isPlane := f.Geometry().(geom.Plane); isPlane {
			caps = append(caps, f)
			continue
		}
		side = f
	}
	if side == nil || len(caps) != 2 {
		t.Fatalf("cylinder split into side=%v and %d caps, want 1 + 2", side != nil, len(caps))
	}
	return side, caps
}

// TestAngleBoundarylessFaceUsesDomainMidpoint: a whole sphere has NO boundary edge to sample, so the
// representative point is the surface's parameter-domain midpoint. Its normal is radial, so against a
// box's six axis-aligned faces it must read 0° once, 180° once and 90° four times.
func TestAngleBoundarylessFaceUsesDomainMidpoint(t *testing.T) {
	ball, err := brep.SolidSphere(gmath.P3(0, 0, 0), 2, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	block, err := brep.SolidBlock(gmath.P3(10, 10, 10), gmath.P3(14, 13, 15), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	counts := map[float64]int{}
	for _, f := range block.Faces() {
		deg, err := AngleDegrees(MeasureEntity{Face: ball.Faces()[0]}, MeasureEntity{Face: f}, ops.DefaultQuality())
		if err != nil {
			t.Fatalf("AngleDegrees(sphere, box face): %v", err)
		}
		counts[math.Round(deg)]++
	}
	if counts[0] != 1 || counts[180] != 1 || counts[90] != 4 {
		t.Errorf("sphere-vs-box angles = %v, want one 0°, one 180° and four 90° (a radial normal)", counts)
	}
}

// TestFaceNormalFollowsReversedFace: a cavity's skin is a REVERSED face, so its material normal is the
// negated surface normal — it points into the void, away from the material.
func TestFaceNormalFollowsReversedFace(t *testing.T) {
	block, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	ball, err := brep.SolidSphere(gmath.P3(5, 5, 5), 2, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	body, err := ops.Boolean(ops.Cut, block, ball)
	if err != nil {
		t.Fatalf("cut cavity: %v", err)
	}
	skin := reversedSphereFace(t, body)
	n, err := faceNormal(skin)
	if err != nil {
		t.Fatalf("faceNormal(cavity skin): %v", err)
	}
	p, _ := faceRepresentativePoint(skin)
	if toCentre := p.VectorTo(gmath.P3(5, 5, 5)); float64(n.Dot(toCentre)) <= 0 {
		t.Errorf("cavity-skin normal %v does not point into the void at %v", n, p)
	}
}

// reversedSphereFace returns the body's reversed spherical face — the cavity skin.
func reversedSphereFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range body.Faces() {
		if _, isSphere := f.Geometry().(geom.Sphere); isSphere && f.Reversed() {
			return f
		}
	}
	t.Fatal("cut body has no reversed spherical face")
	return nil
}
