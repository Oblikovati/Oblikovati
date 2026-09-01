// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestGeometricFaceShellBindsAndHollows proves the face path (M8/ADR-0040): a shell whose
// removed face is given only by a geometric descriptor binds and hollows the body.
func TestGeometricFaceShellBindsAndHollows(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	box := geomRefBox()
	NewBaseFeatures(fs).AddBase(box)

	ref := topo.DescribeFace(box.Faces()[0])
	pf := NewDressUpFeatures(fs).addShell(&ShellDefinition{
		GeomFaces: []topo.GeometricFaceRef{ref},
		Thickness: constFloat(0.5),
	})
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("geometric-face shell is not healthy: %+v", pf.Health())
	}
	boxVol := query.BodyGeometryProperties(geomRefBox(), ops.DefaultQuality()).Volume
	gotVol := query.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume
	if !(gotVol < boxVol) {
		t.Errorf("shell did not hollow the body: got %g, box %g", gotVol, boxVol)
	}
}

// TestGeometricFaceHoleBindsAndDrills proves a hole placed on a geometrically-described
// face binds and removes material.
func TestGeometricFaceHoleBindsAndDrills(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	box := geomRefBox()
	NewBaseFeatures(fs).AddBase(box)

	ref := topo.DescribeFace(box.Faces()[0])
	pf := NewHoleFeatures(fs).addHole(&HoleDefinition{
		GeomFace: &ref,
		Diameter: constFloat(0.5),
		Depth:    constFloat(1),
		Type:     DrilledHole,
	})
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("geometric-face hole is not healthy: %+v", pf.Health())
	}
	boxVol := query.BodyGeometryProperties(geomRefBox(), ops.DefaultQuality()).Volume
	gotVol := query.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume
	if !(gotVol < boxVol) {
		t.Errorf("hole did not remove material: got %g, box %g", gotVol, boxVol)
	}
}

// TestGeometricFaceHoleRecipeRoundTrips checks the single-face geomFace encoding (a hole's
// placement face) round-trips through serialize → restore.
func TestGeometricFaceHoleRecipeRoundTrips(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	ref := topo.DescribeFace(geomRefBox().Faces()[0])
	pf := NewHoleFeatures(fs).addHole(&HoleDefinition{
		GeomFace: &ref, Diameter: constFloat(0.5), Depth: constFloat(1), Type: DrilledHole,
	})

	fd, err := serializeFeature(pf, nil, map[ID]int{})
	if err != nil {
		t.Fatalf("serializeFeature: %v", err)
	}
	if fd.Hole == nil || fd.Hole.GeomFace == nil {
		t.Fatalf("recipe did not carry the hole's geometric placement face: %+v", fd.Hole)
	}

	dst := NewPartFeatures(nil)
	rpf, err := buildFeature(dst, fd, nil, nil, nil)
	if err != nil {
		t.Fatalf("buildFeature: %v", err)
	}
	hf := rpf.feature.(*HoleFeature)
	if hf.def.GeomFace == nil {
		t.Fatal("restored hole lost its geometric placement face")
	}
	if hf.def.GeomFace.Centroid.DistanceTo(ref.Centroid) > 1e-9 {
		t.Errorf("restored centroid %v != original %v", hf.def.GeomFace.Centroid, ref.Centroid)
	}
}

// TestHoleExplicitCenterRoundTrips checks an externally-authored hole's explicit drill point
// survives serialize → restore (nil center stays nil).
func TestHoleExplicitCenterRoundTrips(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	ref := topo.DescribeFace(geomRefBox().Faces()[0])
	center := math.P3(0.2, 0.3, 0.5)
	pf := NewHoleFeatures(fs).addHole(&HoleDefinition{
		GeomFace: &ref, Center: &center, Diameter: constFloat(0.5), Depth: constFloat(1), Type: DrilledHole,
	})

	fd, err := serializeFeature(pf, nil, map[ID]int{})
	if err != nil {
		t.Fatalf("serializeFeature: %v", err)
	}
	if fd.Hole == nil || len(fd.Hole.Center) != 3 {
		t.Fatalf("recipe did not carry the hole's explicit center: %+v", fd.Hole)
	}

	dst := NewPartFeatures(nil)
	rpf, err := buildFeature(dst, fd, nil, nil, nil)
	if err != nil {
		t.Fatalf("buildFeature: %v", err)
	}
	got := rpf.feature.(*HoleFeature).def.Center
	if got == nil {
		t.Fatal("restored hole lost its explicit center")
	}
	if got.DistanceTo(center) > 1e-9 {
		t.Errorf("restored center %v != original %v", *got, center)
	}
}

// TestGeometricFaceShellRecipeRoundTrips checks the geomFaces encoding round-trips through
// serialize → restore.
func TestGeometricFaceShellRecipeRoundTrips(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	ref := topo.DescribeFace(geomRefBox().Faces()[0])
	pf := NewDressUpFeatures(fs).addShell(&ShellDefinition{
		GeomFaces: []topo.GeometricFaceRef{ref},
		Thickness: constFloat(0.5),
	})

	fd, err := serializeFeature(pf, nil, map[ID]int{})
	if err != nil {
		t.Fatalf("serializeFeature: %v", err)
	}
	if fd.Shell == nil || len(fd.Shell.GeomFaces) != 1 {
		t.Fatalf("recipe did not carry the geometric face ref: %+v", fd.Shell)
	}

	dst := NewPartFeatures(nil)
	rpf, err := buildFeature(dst, fd, nil, nil, nil)
	if err != nil {
		t.Fatalf("buildFeature: %v", err)
	}
	sf := rpf.feature.(*ShellFeature)
	if len(sf.def.GeomFaces) != 1 {
		t.Fatalf("restored shell has %d geom faces, want 1", len(sf.def.GeomFaces))
	}
	if sf.def.GeomFaces[0].Centroid.DistanceTo(ref.Centroid) > 1e-9 {
		t.Errorf("restored centroid %v != original %v", sf.def.GeomFaces[0].Centroid, ref.Centroid)
	}
}
