// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
)

// bodyVolume sums the divergence-theorem volume of a merged mesh (positive = outward-oriented closed).
func bodyVolume(m *Mesh) float64 {
	v := 0.0
	for t := 0; t+2 < len(m.Indices); t += 3 {
		a := math.Point3{}.VectorTo(m.Positions[m.Indices[t]])
		b := math.Point3{}.VectorTo(m.Positions[m.Indices[t+1]])
		c := math.Point3{}.VectorTo(m.Positions[m.Indices[t+2]])
		v += float64(a.Dot(b.Cross(c))) / 6.0
	}
	return v
}

// TestOrientHealRestoresFlippedFace pins the M25 import-orientation heal: an imported solid whose B-rep
// gave a face an inverted sense tessellates with that face wound inward (a Normal-Debug red face).
// orientFacesOutward must flip it back via edge-adjacency 2-colouring. Simulated by flipping one face's
// mesh of a clean cylinder, then asserting the heal restores a fully outward (positive-volume) mesh.
func TestOrientHealRestoresFlippedFace(t *testing.T) {
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	faces := cyl.Faces()
	fm := make([]*Mesh, len(faces))
	for i, f := range faces {
		fm[i] = TessellateFace(f, DefaultQuality())
	}
	clean := &Mesh{}
	for _, m := range fm {
		mergeMesh(clean, m)
	}
	want := bodyVolume(clean) // correctly-oriented reference volume

	// Flip the face with the largest volume contribution (a zero-contribution cap through the origin
	// would not change the divergence volume), simulating one face imported with inverted sense.
	flip := 0
	for i := range fm {
		if stdmath.Abs(meshSignedVolume(fm[i], false)) > stdmath.Abs(meshSignedVolume(fm[flip], false)) {
			flip = i
		}
	}
	reverseMesh(fm[flip])
	broken := &Mesh{}
	for _, m := range fm {
		mergeMesh(broken, m)
	}
	if stdmath.Abs(bodyVolume(broken)-want) < 1e-6 {
		t.Fatalf("setup: flipping a face should change the divergence volume, but it did not")
	}

	orientFacesOutward(fm)
	healed := &Mesh{}
	for _, m := range fm {
		mergeMesh(healed, m)
	}
	if got := bodyVolume(healed); stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("orientFacesOutward did not restore the flipped face: volume %g, want %g", got, want)
	}
}

// TestOrientHealLeavesCleanSolidUnchanged pins that the heal NEVER flips a correctly-oriented face: a
// clean solid's tessellated volume is unchanged (still positive) after the heal.
func TestOrientHealLeavesCleanSolidUnchanged(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 10)
	faces := cyl.Faces()
	fm := make([]*Mesh, len(faces))
	for i, f := range faces {
		fm[i] = TessellateFace(f, DefaultQuality())
	}
	before := &Mesh{}
	for _, m := range fm {
		mergeMesh(before, m)
	}
	orientFacesOutward(fm)
	after := &Mesh{}
	for _, m := range fm {
		mergeMesh(after, m)
	}
	if stdmath.Abs(bodyVolume(before)-bodyVolume(after)) > 1e-6 {
		t.Errorf("heal changed a clean solid: volume %g → %g", bodyVolume(before), bodyVolume(after))
	}
}
