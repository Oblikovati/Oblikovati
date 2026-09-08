// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// severingSlotBars are four through-slots cut from a 10×10×2 plate that free the square island
// [4,6]×[4,6] from the rest of it. They are laid out so the island and the remainder still KISS along
// the vertical edge through (6,6): the right slot stops at y=6 and the top slot at x=6, so the plate's
// material fills the +x+y quadrant of that edge and nothing separates the two lumps there.
//
// That kiss is the whole point of the fixture. The severed island's boundary passes through a point
// that is also on the remainder's boundary, and the orientation pass used to classify the island at
// exactly such a point — see TestCutThatSeversAKissingLumpWindsBothLumpsOutward.
func severingSlotBars() [][2]m.Point3 {
	return [][2]m.Point3{
		{m.P3(3, 3, -1), m.P3(6, 4, 3)},
		{m.P3(3, 4, -1), m.P3(4, 6, 3)},
		{m.P3(6, 3, -1), m.P3(7, 6, 3)},
		{m.P3(4, 6, -1), m.P3(6, 7, 3)},
	}
}

// severedIslandPlate cuts severingSlotBars out of the 10×10×2 plate, one at a time, and returns the
// two-lump result.
func severedIslandPlate(t *testing.T) *topo.Body {
	t.Helper()
	body, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(10, 10, 2), "plate")
	if err != nil {
		t.Fatalf("plate: %v", err)
	}
	for i, bar := range severingSlotBars() {
		tool, err := brep.SolidBlock(bar[0], bar[1], "bar")
		if err != nil {
			t.Fatalf("bar %d: %v", i, err)
		}
		if body, err = brep.Boolean(brep.Difference, body, tool); err != nil {
			t.Fatalf("cut bar %d: %v", i, err)
		}
	}
	return body
}

// TestCutThatSeversAKissingLumpWindsBothLumpsOutward is the corpus row for
// Oblikovati/Oblikovati#3512: a Cut that severs a lump which still touches the rest of the body.
//
// The orientation pass groups a result's faces into shells and asks, per shell, whether it is a VOID of
// the shells already oriented — a void's faces carry the opposite outward sense. It used to ask that at
// a loop VERTEX of the shell's first face, which is a point ON the shell's own boundary and, at a kiss,
// on the other shell's boundary too: every ray's nearest crossing is then the self-hit at t≈0 and the
// side read from it is a coin flip. Here it came up "inside", the severed island was taken for a void,
// and all six of its faces had their stored sense inverted against their own loop winding — the
// emission post-condition brep.FaceWindingConsistent exists to catch, which made the Inventor
// multipoint disk's Extrusion4 decline with boolean.winding-reject and go sick.
//
// The volume is Requicha-exact: 10·10·2 − (3+2+3+2)·1·2 = 180, of which the island is 2·2·2 = 8. Both
// lumps must integrate POSITIVE — a void would integrate negative and the total would come out 164.
func TestCutThatSeversAKissingLumpWindsBothLumpsOutward(t *testing.T) {
	body := severedIslandPlate(t)
	if !Validate(body).ValidSolid() {
		t.Fatalf("severed plate is not a valid closed solid: %v", Validate(body).Issues)
	}
	if got, want := len(body.Faces()), 22; got != want {
		t.Errorf("face census = %d, want %d (16 on the remainder, 6 on the island)", got, want)
	}
	assertEveryFaceWoundOutward(t, body)
	assertSeveredLumpVolumes(t, body)
	assertMeshVolumeAtBothFacetings(t, body)
}

// assertEveryFaceWoundOutward checks the emission post-condition the boolean's own acceptance gate
// applies: no face may be wound against its outward normal.
func assertEveryFaceWoundOutward(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, f := range body.Faces() {
		if ok, certain := brep.FaceWindingConsistent(f); certain && !ok {
			t.Errorf("face %q is wound against its outward normal", f.ReferenceKey())
		}
	}
}

// assertSeveredLumpVolumes checks the Requicha volume and that BOTH shells are lumps: a shell mistaken
// for a void integrates negative, which a whole-body total alone can hide.
func assertSeveredLumpVolumes(t *testing.T, body *topo.Body) {
	t.Helper()
	if got, want := len(body.Shells()), 2; got != want {
		t.Fatalf("shell count = %d, want %d (the remainder and the severed island)", got, want)
	}
	total := 0.0
	for _, s := range body.Shells() {
		v, ok := query.AnalyticShellVolume(s)
		if !ok {
			t.Fatalf("shell %d has no analytic volume", s.ID())
		}
		if v <= 0 {
			t.Errorf("shell %d integrates %g — a lump must be positive; a void's sense was stored", s.ID(), v)
		}
		total += v
	}
	if stdmath.Abs(total-severedPlateVolume) > 1e-9 { // tol:calibrated — exact planar volume, only rounding
		t.Errorf("volume = %g, want %g (Requicha: plate − four through-slots)", total, severedPlateVolume)
	}
}

// severedPlateVolume is 10·10·2 less the four through-slots (3+2+3+2 in plan area, 2 thick).
const severedPlateVolume = 180.0

// assertMeshVolumeAtBothFacetings meshes the result at both samplings: a gate that holds at one
// faceting is measuring the tessellation, not the geometry. A lump stored with a void's sense meshes
// inside-out, so the mesh volume catches the defect independently of the analytic integrator.
//
// The watertightness metric is NOT the gate here. It counts edges not shared by exactly two triangles,
// and the kiss this fixture is built around is a legitimate FOUR-triangle edge — the two lumps meet
// along it — so a correct mesh of this body reports it as free. Volume is the metric that means the
// same thing at a kiss as anywhere else.
func assertMeshVolumeAtBothFacetings(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, gq := range gateQualities() {
		mesh, _ := tessellate.TessellateBody(body, gq.q)
		v := tessellate.MeshGeometryProperties(mesh).Volume
		if stdmath.Abs(v-severedPlateVolume) > 1e-6 { // tol:calibrated — planar facets are exact here
			t.Errorf("%s faceting: mesh volume = %g, want %g", gq.name, v, severedPlateVolume)
		}
	}
}
