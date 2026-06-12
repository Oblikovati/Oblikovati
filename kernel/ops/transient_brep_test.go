// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestSectionWithPlaneBoxRectangle: a mid-height plane sections the box into
// one closed rectangular chain of perimeter 2(sx+sy).
func TestSectionWithPlaneBoxRectangle(t *testing.T) {
	b := boxBody(math.P3(0, 0, 0), 3, 2, 4)
	sec, err := SectionWithPlane(b, math.P3(0, 0, 2), math.V3(0, 0, 1), DefaultQuality())
	if err != nil {
		t.Fatalf("SectionWithPlane: %v", err)
	}
	wires := sec.Wires()
	if len(wires) != 1 {
		t.Fatalf("box section has %d wires, want 1", len(wires))
	}
	if !wires[0].IsClosed() {
		t.Error("box section wire should be closed")
	}
	if l := wireLength(wires[0]); stdmath.Abs(l-10) > 1e-6 {
		t.Errorf("section perimeter = %g, want 10", l)
	}
	if _, err := SectionWithPlane(b, math.P3(0, 0, 50), math.V3(0, 0, 1), DefaultQuality()); err == nil {
		t.Error("a plane missing the body must error")
	}
}

// boxBody stitches the six outward quads of a sx×sy×sz box at p — the
// established cubeFaces idiom, scaled.
func boxBody(p math.Point3, sx, sy, sz float64) *topo.Body {
	s := func(x, y, z float64) math.Point3 {
		return math.P3(p.X+math.Scalar(x*sx), p.Y+math.Scalar(y*sy), p.Z+math.Scalar(z*sz))
	}
	faces := []*topo.Body{
		quadBody("bottom", s(0, 0, 0), s(0, 1, 0), s(1, 1, 0), s(1, 0, 0)),
		quadBody("top", s(0, 0, 1), s(1, 0, 1), s(1, 1, 1), s(0, 1, 1)),
		quadBody("front", s(0, 0, 0), s(1, 0, 0), s(1, 0, 1), s(0, 0, 1)),
		quadBody("back", s(0, 1, 0), s(0, 1, 1), s(1, 1, 1), s(1, 1, 0)),
		quadBody("left", s(0, 0, 0), s(0, 0, 1), s(0, 1, 1), s(0, 1, 0)),
		quadBody("right", s(1, 0, 0), s(1, 1, 0), s(1, 1, 1), s(1, 0, 1)),
	}
	body, _ := Stitch(faces, 0, false, "box")
	return body
}

// TestFaceSilhouetteCylinderRulings: a cylinder side face viewed from +X has
// two vertical silhouette rulings (at y = ±r).
func TestFaceSilhouetteCylinderRulings(t *testing.T) {
	cyl := cylinderBody(t, 2, 5)
	var side *topo.Face
	for _, f := range cyl.Faces() {
		if !isPlanarFace(f) {
			side = f
		}
	}
	if side == nil {
		t.Fatal("cylinder has no curved side face")
	}
	sil, err := FaceSilhouetteWires(side, math.V3(1, 0, 0), true, DefaultQuality())
	if err != nil {
		t.Fatalf("FaceSilhouetteWires: %v", err)
	}
	if n := len(sil.Wires()); n != 2 {
		t.Fatalf("cylinder silhouette has %d wires, want 2 rulings", n)
	}
	for _, w := range sil.Wires() {
		if l := wireLength(w); stdmath.Abs(l-5) > 0.1 {
			t.Errorf("silhouette ruling length = %g, want 5", l)
		}
	}
}

func cylinderBody(t *testing.T, r, h float64) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	return b
}

func isPlanarFace(f *topo.Face) bool {
	_, ok := f.Geometry().(geom.Plane)
	return ok
}

// TestRuledSurfaceBetweenSquares: ruling two parallel unit squares 2 apart
// gives a surface body of area ~8 (four 1×2 walls).
func TestRuledSurfaceBetweenSquares(t *testing.T) {
	_, w1 := squareWireBody(1)
	_, w2raw := squareWireBody(1)
	lifted, err := TransformBody(w2raw.Body(), math.Translation4(math.V3(0, 0, 2)), func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		t.Fatal(err)
	}
	w2 := lifted.Wires()[0]
	surf, err := RuledSurfaceBetweenWires(w1, w2)
	if err != nil {
		t.Fatalf("RuledSurfaceBetweenWires: %v", err)
	}
	if surf.IsSolid() {
		t.Error("a ruled surface is a surface body, not a solid")
	}
	props := BodyGeometryProperties(surf, DefaultQuality())
	if stdmath.Abs(props.Area-8) > 0.1 {
		t.Errorf("ruled surface area = %g, want ~8", props.Area)
	}
}

// TestGroupIdenticalBodies: a box equals its translate and rotate, not a
// different box; reflection matching is controllable.
func TestGroupIdenticalBodies(t *testing.T) {
	a := boxBody(math.P3(0, 0, 0), 1, 2, 3)
	moved, _ := TransformBody(a, math.Translation4(math.V3(10, 0, 0)), func(l topo.Lineage) topo.Lineage { return l })
	axis, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	rotated, _ := TransformBody(a, math.Rotation4(0.7, axis, math.P3(5, 5, 5)), func(l topo.Lineage) topo.Lineage { return l })
	other := boxBody(math.P3(0, 0, 0), 1, 2, 4)
	groups := GroupIdenticalBodies([]*topo.Body{a, moved, rotated, other},
		IdenticalBodiesOptions{MatchReflection: true}, DefaultQuality())
	if len(groups) != 2 {
		t.Fatalf("grouping = %v, want 2 groups", groups)
	}
	if len(groups[0]) != 3 {
		t.Errorf("identical group = %v, want the three congruent boxes", groups[0])
	}
}

// TestDropFacesKeepsSelectionSemantics: deleting one face opens the body;
// keep-instead retains only the selection.
func TestDropFacesKeepsSelectionSemantics(t *testing.T) {
	b := boxBody(math.P3(0, 0, 0), 1, 1, 1)
	key := b.Faces()[0].ReferenceKey()
	open, err := DropFaces(b, [][]byte{key}, false)
	if err != nil {
		t.Fatalf("DropFaces: %v", err)
	}
	if len(open.Faces()) != 5 || open.IsSolid() {
		t.Errorf("drop-one result: %d faces solid=%v, want 5 faces surface body", len(open.Faces()), open.IsSolid())
	}
	if len(BoundaryEdges(open)) != 4 {
		t.Errorf("opened box has %d boundary edges, want 4", len(BoundaryEdges(open)))
	}
	only, err := DropFaces(b, [][]byte{key}, true)
	if err != nil {
		t.Fatalf("DropFaces keep: %v", err)
	}
	if len(only.Faces()) != 1 {
		t.Errorf("keep-instead result has %d faces, want 1", len(only.Faces()))
	}
	if _, err := DropFaces(only, [][]byte{only.Faces()[0].ReferenceKey()}, false); err == nil {
		t.Error("removing every face must error")
	}
}
