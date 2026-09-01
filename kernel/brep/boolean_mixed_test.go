// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Per-face-dispatch mixed boolean (ADR-0058): a body with curved faces booleans EXACTLY through the
// planar pipeline when the curved (and curved-edged planar) faces are clear of the tool — they pass
// through whole into the unified stitch. Out-of-scope configurations decline with ErrNonPlanar so the
// curved/CSG fallbacks run as before.

// bossedBlock is the mixed fixture: a 10³ block with an r=2 h=3 cylindrical boss on its top face —
// straight-edged planar sides, plus a curved wall, a circular-hole seat and a circular-rim cap.
func bossedBlock(t *testing.T) *topo.Body {
	t.Helper()
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	cyl, err := brep.SolidCylinder(math.P3(5, 5, 10), math.V3(0, 0, 1), 2, 3)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	bossed, ok := brep.JoinCylindricalBoss(block, cyl)
	if !ok || bossed == nil {
		t.Fatalf("JoinCylindricalBoss declined; fixture unavailable")
	}
	return bossed
}

func cylinderFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// TestMixedBooleanNotchAwayFromBoss: cutting a notch into the bossed block's side, far from the boss,
// runs the exact planar pipeline on the block faces while the boss (wall, seat hole, cap) passes
// through analytically — valid solid, exact volume, cylinder wall intact.
func TestMixedBooleanNotchAwayFromBoss(t *testing.T) {
	t.Parallel()
	bossed := bossedBlock(t)
	notch, _ := brep.SolidBlock(math.P3(-1, 4, 1), math.P3(2, 6, 3), "notch")
	res, err := brep.BooleanDiag(brep.Difference, bossed, notch, nil)
	if err != nil {
		t.Fatalf("mixed difference declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("mixed difference result is not a solid")
	}
	if n := cylinderFaceCount(res); n != 1 {
		t.Errorf("boss wall did not survive analytically: %d cylinder faces, want 1", n)
	}
	want := 1000 + stdmath.Pi*4*3 - 2*2*2 // block + boss − notch overlap
	if got := vol(res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("mixed difference volume = %g, want %g", got, want)
	}
}

// TestMixedBooleanUnionAwayFromBoss: welding an add-on block onto a side face away from the boss.
func TestMixedBooleanUnionAwayFromBoss(t *testing.T) {
	t.Parallel()
	bossed := bossedBlock(t)
	addon, _ := brep.SolidBlock(math.P3(10, 4, 1), math.P3(12, 6, 3), "addon")
	res, err := brep.BooleanDiag(brep.Union, bossed, addon, nil)
	if err != nil {
		t.Fatalf("mixed union declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("mixed union result is not a solid")
	}
	if n := cylinderFaceCount(res); n != 1 {
		t.Errorf("boss wall did not survive analytically: %d cylinder faces, want 1", n)
	}
	want := 1000 + stdmath.Pi*4*3 + 2*2*2
	if got := vol(res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("mixed union volume = %g, want %g", got, want)
	}
}

// TestMixedBooleanDeclinesCurvedInteraction: a cut that genuinely CROSSES the boss wall must DECLINE
// (ErrNonPlanar) — imprinting a cylinder wall is the bespoke curved handlers' and the reconstruction
// path's job. The tool reaches x=8, well past the wall (radius 2 about (5,5)), and the exact
// interaction gate proves the contact (a ruling-line pair inside the tool face's trim and the band).
func TestMixedBooleanDeclinesCurvedInteraction(t *testing.T) {
	t.Parallel()
	bossed := bossedBlock(t)
	through, _ := brep.SolidBlock(math.P3(4, 4, 9), math.P3(8, 6, 12), "through")
	if _, err := brep.BooleanDiag(brep.Difference, bossed, through, nil); err == nil {
		t.Fatal("cut crossing the boss wall did not decline; want ErrNonPlanar (conservative scope)")
	}
}

// TestMixedBooleanCavityThroughSeatHole: a tool box passing through the SEAT PLANE inside the boss's
// hole region — fully interior to the block∪boss solid, touching no face (the exact interaction gate
// proves the wall clear; the exact trim clipping mints no imprint inside the hole void; the membership
// oracle sees the TRUE hole boundary). The cut removes EXACTLY the tool volume as an embedded cavity.
// Asserted as a volume DELTA against the uncut fixture, isolating the boolean from any fixture bias.
func TestMixedBooleanCavityThroughSeatHole(t *testing.T) {
	t.Parallel()
	bossed := bossedBlock(t)
	before := vol(bossed)
	tool, _ := brep.SolidBlock(math.P3(4, 4, 9), math.P3(6, 6, 12), "tool")
	res, err := brep.BooleanDiag(brep.Difference, bossed, tool, nil)
	if err != nil {
		t.Fatalf("through-seat-hole cavity declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("cavity result is not a solid")
	}
	if removed := before - vol(res); stdmath.Abs(removed-12) > 1e-6 {
		t.Errorf("cavity removed %g, want exactly 12 (the embedded tool's volume)", removed)
	}
}

// TestMixedBooleanDisjointDifference: an all-pass-through operand (a lone cylinder — curved wall +
// circular-rim caps) minus a disjoint box keeps every face whole through the unified stitch.
func TestMixedBooleanDisjointDifference(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	far, _ := brep.SolidBlock(math.P3(20, 20, 20), math.P3(22, 22, 22), "far")
	res, err := brep.BooleanDiag(brep.Difference, cyl, far, nil)
	if err != nil {
		t.Fatalf("disjoint mixed difference declined: %v", err)
	}
	if !res.IsSolid() || cylinderFaceCount(res) != 1 {
		t.Fatalf("disjoint difference did not keep the cylinder whole (solid=%v cyls=%d)", res.IsSolid(), cylinderFaceCount(res))
	}
	if got, want := vol(res), stdmath.Pi*4*4; stdmath.Abs(got-want) > 0.5 {
		t.Errorf("disjoint difference volume = %g, want %g", got, want)
	}
}

// TestMixedBooleanEmbeddedCavityCut: subtracting a cylinder wholly inside the block cuts an exact
// cylindrical cavity — the tool's pass-through faces (curved wall + circular-edged caps) are kept
// REVERSED into the void, welded by the unified stitch as an inner shell.
func TestMixedBooleanEmbeddedCavityCut(t *testing.T) {
	t.Parallel()
	block, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	tool, _ := brep.SolidCylinder(math.P3(5, 5, 3), math.V3(0, 0, 1), 1, 4)
	res, err := brep.BooleanDiag(brep.Difference, block, tool, nil)
	if err != nil {
		t.Fatalf("embedded cavity cut declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("cavity result is not a solid")
	}
	if n := cylinderFaceCount(res); n != 1 {
		t.Errorf("cavity wall not analytic: %d cylinder faces, want 1", n)
	}
	want := 1000 - stdmath.Pi*1*4
	if got := vol(res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("cavity volume = %g, want %g", got, want)
	}
}

// TestMixedBooleanPocketOnSeatFace: cutting a pocket into the TOP face of the bossed block — the face
// that carries the boss's rim circle as a hole. The curved hole is detached, the seat splits through
// the exact polygonal pipeline, and the rim circle re-attaches EXACTLY to the fragment containing it,
// so it still welds with the pass-through boss wall (previously this declined: the seat face was
// pass-through-only and its box spans the whole top).
func TestMixedBooleanPocketOnSeatFace(t *testing.T) {
	t.Parallel()
	bossed := bossedBlock(t)
	pocket, _ := brep.SolidBlock(math.P3(1, 1, 9), math.P3(2.8, 2.8, 11), "pocket")
	res, err := brep.BooleanDiag(brep.Difference, bossed, pocket, nil)
	if err != nil {
		t.Fatalf("seat-face pocket declined: %v", err)
	}
	if !res.IsSolid() {
		t.Fatal("seat-face pocket result is not a solid")
	}
	if n := cylinderFaceCount(res); n != 1 {
		t.Errorf("boss wall did not survive analytically: %d cylinder faces, want 1", n)
	}
	want := 1000 + stdmath.Pi*4*3 - 1.8*1.8*1 // pocket bites 1 deep into the top
	if got := vol(res); stdmath.Abs(got-want) > 0.5 {
		t.Errorf("seat-face pocket volume = %g, want %g", got, want)
	}
}

// TestMixedBooleanCavityInsideCylinder: a box wholly inside a cylinder — the box's faces overlap the
// wall's bounding box everywhere, so the retired box gate always declined; the exact interaction gate
// (plane∩cylinder ruling/circle curves against both trims, OCCT IntTools style) proves the wall clear
// and the cut becomes an exact embedded cavity in a CURVED body.
func TestMixedBooleanCavityInsideCylinder(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	before := vol(cyl)
	tool, _ := brep.SolidBlock(math.P3(-0.5, -0.5, 2), math.P3(0.5, 0.5, 3), "tool")
	res, err := brep.BooleanDiag(brep.Difference, cyl, tool, nil)
	if err != nil {
		t.Fatalf("cavity inside cylinder declined: %v", err)
	}
	if !res.IsSolid() || cylinderFaceCount(res) != 1 {
		t.Fatalf("cavity-in-cylinder invalid (solid=%v cyls=%d)", res.IsSolid(), cylinderFaceCount(res))
	}
	// A volume DELTA against the uncut fixture: the cut must remove exactly the tool's volume,
	// isolating the boolean from a pre-existing cylinder-wall mass-properties bias (~0.08·h at r=2).
	if removed := before - vol(res); stdmath.Abs(removed-1) > 1e-6 {
		t.Errorf("cavity-in-cylinder removed %g, want exactly 1", removed)
	}
}

// TestMixedBooleanPocketInCylinderCap: a pocket cut into a cylinder's TOP CAP — the cap (a planar
// face with a full-circle OUTER loop) splits through the exact-frame chart (planeFaceUV): its rim
// stays the exact circle, the pocket rim is the shared plane∩plane segments both sides split on, and
// the untouched wall proves clear through the exact interaction gate. The pre-B3 dispatch declined
// this whole class.
func TestMixedBooleanPocketInCylinderCap(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	before := vol(cyl)
	tool, _ := brep.SolidBlock(math.P3(-0.8, -0.8, 4), math.P3(0.8, 0.8, 6), "tool")
	res, err := brep.BooleanDiag(brep.Difference, cyl, tool, nil)
	if err != nil {
		t.Fatalf("cap pocket declined: %v", err)
	}
	if !res.IsSolid() || cylinderFaceCount(res) != 1 {
		t.Fatalf("cap pocket invalid (solid=%v cyls=%d)", res.IsSolid(), cylinderFaceCount(res))
	}
	// The pocket removes exactly its overlap with the solid: 1.6×1.6×1 (z∈[4,5]).
	if removed := before - vol(res); stdmath.Abs(removed-2.56) > 1e-6 {
		t.Errorf("cap pocket removed %g, want exactly 2.56", removed)
	}
}

// TestMixedBooleanSlotThroughCylinderWall: a full-height slot box crossing a cylinder's WALL — the
// wall splits through the ruled chart by its exact ruling-line imprints, the caps split through the
// exact-frame chart (their rims staying exact circles crossed at the shared ruling azimuths), and the
// tool's faces split on the mirrored segments. Removal matches the closed-form circular-segment
// prism: 5·(∫√(4−y²)dy − 1.5·Δy) over y∈[−0.6, 0.6].
func TestMixedBooleanSlotThroughCylinderWall(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	before := vol(cyl)
	tool, _ := brep.SolidBlock(math.P3(1.5, -0.6, -1), math.P3(2.5, 0.6, 6), "tool")
	res, err := brep.BooleanDiag(brep.Difference, cyl, tool, nil)
	if err != nil {
		t.Fatalf("wall slot declined: %v", err)
	}
	if !res.IsSolid() || cylinderFaceCount(res) != 1 {
		t.Fatalf("wall slot invalid (solid=%v cyls=%d)", res.IsSolid(), cylinderFaceCount(res))
	}
	// The oracle integrates the analytic B-rep (#3453), so the removal is asserted against the closed
	// form itself. The loose 0.15 bound this replaces existed only while the oracle was a 32-gon
	// quadrature — and it hid #3488: the split caps' loops did not close, so the body silently fell
	// back to that tessellated quadrature and the delta came out 0.30 too large.
	want := 5 * (0.6*stdmath.Sqrt(4-0.36) + 4*stdmath.Asin(0.3) - 1.8)
	if removed := before - vol(res); stdmath.Abs(removed-want) > 1e-9 {
		t.Errorf("wall slot removed %.17g, want the closed form %.17g", removed, want)
	}
	if len(res.Faces()) != 6 {
		t.Fatalf("wall slot has %d faces, want 6 (3 cavity walls + 2 split caps + breached wall)", len(res.Faces()))
	}
	assertSlotGeometry(t, res)
	assertSlotLoopsClose(t, res)
}

// assertSlotGeometry pins the slot result's EXACT geometry: the breach's ruling x sits at √(4−0.36)
// on every touching face box, and each split cap keeps exactly one rim arc whose points stay on the
// r=2 circle (the exact-frame guarantee).
func assertSlotGeometry(t *testing.T, res *topo.Body) {
	t.Helper()
	xr := stdmath.Sqrt(4 - 0.36)
	arcsSeen := 0
	for _, f := range res.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); isCyl {
			if bx := f.RangeBox(); stdmath.Abs(float64(bx.Max.X)-xr) > 1e-9 {
				t.Errorf("breached wall x-max = %g, want the exact ruling %g", float64(bx.Max.X), xr)
			}
			continue
		}
		for _, l := range f.Loops() {
			for _, u := range l.EdgeUses() {
				if arc, isArc := u.Edge().Geometry().(geom.Arc3d); isArc {
					arcsSeen++
					for _, tp := range []float64{0, 0.5, 1} {
						p := arc.PointAt(tp)
						r := stdmath.Hypot(float64(p.X), float64(p.Y))
						if stdmath.Abs(r-2) > 1e-9 {
							t.Errorf("cap rim arc point off the r=2 circle: r=%g", r)
						}
					}
				}
			}
		}
	}
	if arcsSeen != 2 {
		t.Errorf("split caps carry %d rim arcs, want 2 (one per cap, exact)", arcsSeen)
	}
}

// assertSlotLoopsClose walks every face loop in traversal order and asserts each edge ends where the
// next begins, and that every breach vertex it passes is the closed form. The split caps used to leave
// a 1.5e-4 gap where a straight imprint met the rim arc — one vertex represented twice, neither copy
// satisfying both constraints it lies on (#3488) — invisible to every mesh check but fatal to the
// analytic reader, which cannot integrate an open loop.
func assertSlotLoopsClose(t *testing.T, res *topo.Body) {
	t.Helper()
	xr := stdmath.Sqrt(4 - 0.36)
	for i, f := range res.Faces() {
		for _, l := range f.Loops() {
			ends := loopEndpointsInOrder(l)
			assertChainCloses(t, ends, i)
			for _, e := range ends {
				assertBreachVertexExact(t, e[0], xr)
			}
		}
	}
}

// assertChainCloses asserts the loop's edges form a closed chain: edge j ends where edge j+1 begins.
func assertChainCloses(t *testing.T, ends [][2]math.Point3, face int) {
	t.Helper()
	for j := range ends {
		if d := float64(ends[j][1].DistanceTo(ends[(j+1)%len(ends)][0])); d > 1e-12 {
			t.Errorf("face %d loop does not close: gap %g after edge %d", face, d, j)
		}
	}
}

// assertBreachVertexExact pins a vertex sitting where the tool plane |y|=0.6 meets the rim circle r=2
// (screened loosely, so a vertex that MISSES either constraint is still caught): it must satisfy BOTH
// at once, x=√(4−0.36) with |y|=0.6 and r=2. #3488 produced one copy 1.45e-4 off the cylinder and
// another 4.4e-5 off the plane.
func assertBreachVertexExact(t *testing.T, p math.Point3, xr float64) {
	t.Helper()
	y, r := stdmath.Abs(float64(p.Y)), stdmath.Hypot(float64(p.X), float64(p.Y))
	if stdmath.Abs(y-0.6) > 1e-3 || stdmath.Abs(r-2) > 1e-3 {
		return // not a breach vertex
	}
	if stdmath.Abs(float64(p.X)-xr) > 1e-12 || stdmath.Abs(y-0.6) > 1e-12 || stdmath.Abs(r-2) > 1e-12 {
		t.Errorf("breach vertex (%.17g, %.17g) is not the closed form (%.17g, ±0.6): r=%.17g", float64(p.X), float64(p.Y), xr, r)
	}
}

// loopEndpointsInOrder lists a loop's (start, end) points in traversal order, honouring edge-use sense.
func loopEndpointsInOrder(l *topo.Loop) [][2]math.Point3 {
	uses := l.EdgeUses()
	out := make([][2]math.Point3, 0, len(uses))
	for _, u := range uses {
		s, e := u.Edge().StartVertex().Point(), u.Edge().EndVertex().Point()
		if u.Reversed() {
			s, e = e, s
		}
		out = append(out, [2]math.Point3{s, e})
	}
	return out
}
