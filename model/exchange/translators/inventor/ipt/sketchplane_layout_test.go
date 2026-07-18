// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestRigidPlacementCannotProveOrientation pins the LIMIT of isRigidPlacement, so nobody reads it
// as proof that the axes are the matrix's columns.
//
// A rotation's transpose is also a rotation, so a transposed read — axes taken as ROWS — is just as
// orthonormal and passes identically. This test asserts that blind spot on purpose: if a future
// change makes isRigidPlacement able to tell the two apart, this fails and the comment on it (and
// this test) should be rewritten to say what the stronger oracle proves.
//
// What actually settles the layout is TestTranslationLivesInTheLastColumn below.
func TestRigidPlacementCannotProveOrientation(t *testing.T) {
	// A 90° rotation about Z, column-vector convention, with a translation in the last column.
	m := [4][4]float64{
		{0, -1, 0, 1.5},
		{1, 0, 0, 2.5},
		{0, 0, 1, 3.5},
		{0, 0, 0, 1},
	}
	var mt [4][4]float64
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			mt[r][c] = m[c][r]
		}
	}
	mt[0][3], mt[1][3], mt[2][3], mt[3][3] = m[0][3], m[1][3], m[2][3], 1
	if !isRigidPlacement(m) {
		t.Fatalf("isRigidPlacement rejected a valid rigid placement")
	}
	if !isRigidPlacement(mt) {
		t.Fatalf("isRigidPlacement rejected the TRANSPOSED read — it is now stronger than documented; " +
			"update its comment and this test to state what it proves")
	}
	// The two disagree about where the sketch's X axis points, and the oracle cannot referee.
	if (m[0][0] == mt[0][0]) && (m[1][0] == mt[1][0]) && (m[2][0] == mt[2][0]) {
		t.Fatalf("test is vacuous: pick a rotation whose transpose differs")
	}
}

// TestTranslationLivesInTheLastColumn is the real proof of the matrix convention, and the reason
// sketchPlane reads the axes as COLUMNS.
//
// Unlike the rotation block, the translation is not transpose-symmetric: it sits in the last column
// (p' = M·p) or the last row (row-vector convention), never both. Measured over the whole corpus,
// all 517 sketch transforms agree — 346 with a translation carry it in the last column, 0 in the
// last row (171 are pure rotations/identity and say nothing either way). This pins one real
// placement so the convention cannot be silently transposed later: a transposed decode would put
// the origin at the last row's cells and read a wrong (yet still orthonormal) frame.
func TestTranslationLivesInTheLastColumn(t *testing.T) {
	// CompressionRollerArmActuatorScrew's sk1: the screwdriver-slot plane, origin (0.7, 0, 0),
	// X = -Z, Y = +Y, hence normal +X — a placement whose normal is perpendicular to world Z, the
	// case the old direction rule was blind to.
	d := openDoc(t, "real_screw_slot.ipt")
	sks := GraphSketches(d)
	if len(sks) < 2 {
		t.Fatalf("decoded %d sketches, want at least 2", len(sks))
	}
	s := sks[1]
	if !s.PlaneOK {
		t.Fatalf("sk1 has no decoded placement")
	}
	wantOrigin := [3]float64{0.7, 0, 0}
	wantX := [3]float64{0, 0, -1}
	wantY := [3]float64{0, 1, 0}
	for i := range wantOrigin {
		if abs(s.Plane.Origin[i]-wantOrigin[i]) > 1e-9 {
			t.Errorf("sk1 origin = %v, want %v (translation read from the wrong place?)", s.Plane.Origin, wantOrigin)
			break
		}
	}
	for i := range wantX {
		if abs(s.Plane.XAxis[i]-wantX[i]) > 1e-9 || abs(s.Plane.YAxis[i]-wantY[i]) > 1e-9 {
			t.Errorf("sk1 axes = X%v Y%v, want X%v Y%v (axes read as rows instead of columns?)",
				s.Plane.XAxis, s.Plane.YAxis, wantX, wantY)
			break
		}
	}
	// The normal this frame implies is what an extrude grows along; it must be +X, not -X.
	x, y := s.Plane.XAxis, s.Plane.YAxis
	n := [3]float64{x[1]*y[2] - x[2]*y[1], x[2]*y[0] - x[0]*y[2], x[0]*y[1] - x[1]*y[0]}
	if abs(n[0]-1) > 1e-9 || abs(n[1]) > 1e-9 || abs(n[2]) > 1e-9 {
		t.Errorf("sk1 normal = %v, want (+1,0,0)", n)
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
