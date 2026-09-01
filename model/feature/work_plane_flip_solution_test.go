// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestFlipNormalReversesAndPersists: FlipNormal reverses the plane normal, the plane does not move,
// the flip survives a recompute, and a second flip restores the original (#1851).
func TestFlipNormalReversesAndPersists(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pl := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 3 })
	g.Recompute(nil)
	before, origin := pl.Plane().Normal(), pl.Plane().Origin()

	pl.FlipNormal()
	g.Recompute(nil) // the flip must re-apply on recompute, not just at the toggle
	if d := pl.Plane().Normal().AsVector().Dot(before.AsVector()); d > -1+wtol {
		t.Errorf("flip should reverse the normal (dot ≈ -1), got %v", d)
	}
	if !pl.Plane().Origin().IsEqualTo(origin, wtol) {
		t.Errorf("flip must not move the plane: origin %v, want %v", pl.Plane().Origin(), origin)
	}
	pl.FlipNormal()
	g.Recompute(nil)
	if d := pl.Plane().Normal().AsVector().Dot(before.AsVector()); d < 1-wtol {
		t.Errorf("a second flip should restore the original normal (dot ≈ 1), got %v", d)
	}
}

// TestFlipNormalRoundTrips: a flipped plane keeps its reversed normal across Marshal/Apply/recompute
// — the flip lives on the recipe, not just the live object (the fix for the earlier revert) (#1851).
func TestFlipNormalRoundTrips(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pl := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 2 })
	pl.FlipNormal()
	g.Recompute(nil)
	flipped := pl.Plane().Normal()

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	restored.Recompute(nil)
	rp := restored.WorkPlanes().Item(restored.WorkPlanes().Count() - 1)
	if !rp.Flipped() {
		t.Error("flip flag should survive the round-trip")
	}
	if d := rp.Plane().Normal().AsVector().Dot(flipped.AsVector()); d < 1-wtol {
		t.Errorf("restored normal should match the flipped normal (dot ≈ 1), got %v", d)
	}
}

// TestWorkPlaneDisplayStateRoundTrips: grounded / auto-resize / explicit size are gettable, settable,
// and survive a round-trip; SetSize turns off auto-resize (#1851).
func TestWorkPlaneDisplayStateRoundTrips(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pl := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	g.Recompute(nil)
	if pl.AutoResize() || pl.Grounded() {
		t.Fatal("a new user plane should default to auto-resize=false, grounded=false")
	}
	// The derived (unset) size is a square centred on the plane origin.
	c1, c2 := pl.Size()
	mid := math.P3((c1.X+c2.X)/2, (c1.Y+c2.Y)/2, (c1.Z+c2.Z)/2)
	if !mid.IsEqualTo(pl.Plane().Origin(), wtol) {
		t.Errorf("derived size should centre on the plane origin, got midpoint %v", mid)
	}

	pl.SetAutoResize(true)
	pl.SetGrounded(true)
	pl.SetSize(math.P3(1, 2, 0), math.P3(3, 4, 0))
	if pl.AutoResize() {
		t.Error("SetSize should turn off auto-resize")
	}
	gc1, gc2 := pl.Size()
	if !gc1.IsEqualTo(math.P3(1, 2, 0), wtol) || !gc2.IsEqualTo(math.P3(3, 4, 0), wtol) {
		t.Errorf("explicit size not stored: %v %v", gc1, gc2)
	}

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	restored.Recompute(nil)
	rp := restored.WorkPlanes().Item(restored.WorkPlanes().Count() - 1)
	if !rp.Grounded() {
		t.Error("grounded should survive the round-trip")
	}
	rc1, rc2 := rp.Size()
	if !rc1.IsEqualTo(math.P3(1, 2, 0), wtol) || !rc2.IsEqualTo(math.P3(3, 4, 0), wtol) {
		t.Errorf("explicit size should survive the round-trip: %v %v", rc1, rc2)
	}
}

// TestWorkPlaneAutoResizeRoundTrips: an auto-resize flag with no explicit size survives (#1851).
func TestWorkPlaneAutoResizeRoundTrips(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pl := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	pl.SetAutoResize(true)
	data, _ := MarshalWork(g)
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	if !restored.WorkPlanes().Item(restored.WorkPlanes().Count() - 1).AutoResize() {
		t.Error("auto-resize should survive the round-trip")
	}
}

// TestPlaneParallelTangentProximitySelectsSide: for a plane-parallel tangent on a cylinder, a
// proximity point on either side of the axis selects the tangent on that side (#1844).
func TestPlaneParallelTangentProximitySelectsSide(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2) // axis +Z, radius 2
	if err != nil {
		t.Fatal(err)
	}
	base, err := sketch.NewPlane(math.P3(0, 0, 0), mustUnit(0, 1, 0), mustUnit(0, 0, 1)) // normal +X
	if err != nil {
		t.Fatal(err)
	}
	plus, minus := math.P3(5, 0, 0), math.P3(-5, 0, 0)
	pPlus, err := planeParallelTangent(base, cyl, &plus)
	if err != nil {
		t.Fatal(err)
	}
	pMinus, err := planeParallelTangent(base, cyl, &minus)
	if err != nil {
		t.Fatal(err)
	}
	if pPlus.Origin().X < 1.9 {
		t.Errorf("proximity +X should land the tangent at x≈+2, got %v", pPlus.Origin())
	}
	if pMinus.Origin().X > -1.9 {
		t.Errorf("proximity -X should land the tangent at x≈-2, got %v", pMinus.Origin())
	}
	// No proximity → the deterministic default (+n side).
	def, err := planeParallelTangent(base, cyl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if def.Origin().X < 1.9 {
		t.Errorf("default tangent should be the +n side (x≈+2), got %v", def.Origin())
	}
}

// TestCylinderTangentNormalsGolden pins cylinderTangentNormals (#1844) to values verified against
// OpenCASCADE's GccAna_Lin2d2Tan (line tangent to a circle) via the oracle harness. Cylinder axis
// +Z radius 2 at the origin; external axis-parallel line at (5,0,0): the two tangent contact points
// are (0.8, ±1.833030277982336). The oracle-gated TestOracleCylinderTangentNormals (build tag
// "oracle", _oracles/oracle_service.py) re-derives these live; this frozen copy guards them in CI.
func TestCylinderTangentNormalsGolden(t *testing.T) {
	t.Parallel()
	const R = 2.0
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), R)
	if err != nil {
		t.Fatal(err)
	}
	m1, m2, err := cylinderTangentNormals(math.P3(5, 0, 0), cyl)
	if err != nil {
		t.Fatal(err)
	}
	// Contact point = axis foot (the origin here) + R·m, where the normal m is the contact radial.
	got := [][2]float64{
		{R * float64(m1.AsVector().X), R * float64(m1.AsVector().Y)},
		{R * float64(m2.AsVector().X), R * float64(m2.AsVector().Y)},
	}
	if got[0][1] > got[1][1] {
		got[0], got[1] = got[1], got[0]
	}
	want := [][2]float64{{0.8, -1.833030277982336}, {0.8, 1.833030277982336}} // OCCT GccAna_Lin2d2Tan
	for i := range want {
		if dx := got[i][0] - want[i][0]; dx > 1e-9 || dx < -1e-9 {
			t.Errorf("contact %d x = %v, want %v (OCCT)", i, got[i][0], want[i][0])
		}
		if dy := got[i][1] - want[i][1]; dy > 1e-9 || dy < -1e-9 {
			t.Errorf("contact %d y = %v, want %v (OCCT)", i, got[i][1], want[i][1])
		}
	}
}

// TestLineTangentProximitySelectsSide: for a through-line tangent on a cylinder, a proximity point
// selects which of the two tangent solutions is built (#1844).
func TestLineTangentProximitySelectsSide(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	line := NewDatumAxis(math.P3(5, 0, 0), mustUnit(0, 0, 1)) // parallel to the axis, outside
	pPos, pNeg := math.P3(0, 5, 0), math.P3(0, -5, 0)
	planePos, err := planeThroughLineTangent(line, cyl, &pPos)
	if err != nil {
		t.Fatal(err)
	}
	planeNeg, err := planeThroughLineTangent(line, cyl, &pNeg)
	if err != nil {
		t.Fatal(err)
	}
	if planePos.Normal().AsVector().Y <= 0 {
		t.Errorf("proximity +Y should pick the +Y tangent normal, got %v", planePos.Normal())
	}
	if planeNeg.Normal().AsVector().Y >= 0 {
		t.Errorf("proximity -Y should pick the -Y tangent normal, got %v", planeNeg.Normal())
	}
}

// TestBisectorQuadrantSelectsSolution: for two intersecting planes, the quadrant point picks between
// the two perpendicular bisector solutions (#1844).
func TestBisectorQuadrantSelectsSolution(t *testing.T) {
	t.Parallel()
	xy, xz := sketch.XYPlane(), sketch.XZPlane() // intersect on the X axis
	qSum, qDiff := math.P3(0, -1, 1), math.P3(0, 1, 1)
	pSum, err := bisectingPlane(xy, xz, &qSum)
	if err != nil {
		t.Fatal(err)
	}
	pDiff, err := bisectingPlane(xy, xz, &qDiff)
	if err != nil {
		t.Fatal(err)
	}
	if d := pSum.Normal().AsVector().Dot(pDiff.Normal().AsVector()); !math.IsNearZero(d, 1e-6) {
		t.Errorf("the two selected bisectors should be perpendicular, dot = %v", d)
	}
}

// TestTwoPlanesTowardRoundTrips: the chosen bisector quadrant is recorded on the definition and
// survives a Marshal/Apply cycle (#1844).
func TestTwoPlanesTowardRoundTrips(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pl := g.WorkPlanes().AddByTwoPlanesToward(OriginXYPlane, OriginXZPlane, math.P3(0, 1, 1))
	g.Recompute(nil)
	want := pl.Plane().Normal()

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	restored.Recompute(nil)
	rp := restored.WorkPlanes().Item(restored.WorkPlanes().Count() - 1)
	if d := rp.Plane().Normal().AsVector().Dot(want.AsVector()); d < 1-wtol {
		t.Errorf("bisector solution not preserved across round-trip: want %v got %v (dot %v)", want, rp.Plane().Normal(), d)
	}
}
