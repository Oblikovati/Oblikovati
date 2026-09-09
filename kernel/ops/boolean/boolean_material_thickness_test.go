// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// turnAbout is the rotation the rotation-invariance rows use: about (1,1,0), the axis that is NOT
// perpendicular to a Z-aligned tool. A turn about a coordinate axis leaves one world axis square to
// the tool, so the bounding box still measured it and the defect stayed hidden; this one tilts the
// body into every world axis at once.
func turnAbout(deg float64) math.Matrix4 {
	return math.Rotation4(math.Scalar(deg*stdmath.Pi/180), math.V3(1, 1, 0).AsUnit(), math.P3(0, 0, 0))
}

// turnedRingAndDrill is ringAndDrill built in a frame turned by deg. Both operands turn, so the
// operation is CONGRUENT to the unturned one and every honest measure of it must agree.
func turnedRingAndDrill(t *testing.T, bore, deg float64) (*topo.Body, *topo.Body) {
	t.Helper()
	m := turnAbout(deg)
	axis, err := m.TransformUnitVector(math.V3(0, 0, 1).AsUnit())
	if err != nil {
		t.Fatalf("axis at %g deg: %v", deg, err)
	}
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), axis.AsVector(), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring at %g deg: %v", deg, err)
	}
	drill, err := brep.SolidCylinder(m.TransformPoint(math.P3(5, 0, -4)), axis.AsVector(), bore, 8)
	if err != nil {
		t.Fatalf("drill at %g deg: %v", deg, err)
	}
	return ring, drill
}

// turnedBlock is a 10 x 4 x 0.25 slab built in a frame turned by deg — the PLANAR path's own
// rotation row. The slab is built from its turned corner points rather than transformed, so nothing
// but the frame differs.
func turnedBlock(t *testing.T, deg float64) *topo.Body {
	t.Helper()
	m := turnAbout(deg)
	corners := [8]math.Point3{}
	for i := range 8 {
		p := math.P3(math.Scalar(10*float64(i&1)), math.Scalar(4*float64((i>>1)&1)), math.Scalar(0.25*float64((i>>2)&1)))
		corners[i] = m.TransformPoint(p)
	}
	faces := [][]int{{0, 2, 3, 1}, {4, 5, 7, 6}, {0, 1, 5, 4}, {2, 6, 7, 3}, {0, 4, 6, 2}, {1, 3, 7, 5}}
	return subd.ToBody(subd.Mesh{Verts: corners[:], Faces: faces}, "slab")
}

// rotationDriftPerSpan is how far two measurements of ONE body in two frames may differ, per unit of
// the model's own extent. The drift is not zero and cannot be: the width is computed from rotated
// coordinates, so a 1e-10 feature carried on coordinates of size 13 loses ten digits to cancellation
// whichever way the part is turned. It is a bound on the ARITHMETIC, not a modelling tolerance.
//
// Measured over 0/13.7/37/90 degrees about (1,1,0): the drill's width moves 6.8e-18 absolute
// (2.0000068716156212e-10 to 2.0000136467170038e-10) and the slab's 5.6e-16. The AABB measure this
// replaces moved 3.4 — twelve orders of magnitude more, and enough to flip the verdict.
const rotationDriftPerSpan = 64 * 2.220446049250313e-16 // tol:numeric — a few ULPs of the model's own coordinates

// TestTheSizeClassificationIsRotationInvariant is the acceptance row for #3524, and the measurement
// that named the defect. solidThickness read the smallest side of the AXIS-ALIGNED bounding box, so
// it measured the frame as much as the body. Measured on the RING corpus pair, turning both operands
// about (1,1,0):
//
//	turn   drill thickness (was)     verdict (was)   drill thickness (now)  verdict (now)
//	 0°    2e-10                     REFUSE          2e-10                  REFUSE
//	37°    3.4043798713069418        BUILD           2e-10                  REFUSE
//	90°    2.0000046063728405e-10    REFUSE          2e-10                  REFUSE
//
// and for the ring itself, whose tube is 3 across at every angle: 3, 8.943664347897517,
// 10.071067811865476 — was; 3, 3, 3 — now. The same drill through the same ring decided two different
// ways depending on how the part happened to be turned.
//
// Both operands are measured, not just the tool: the ring is the row that fails if the axis path
// stops working, and the drill the row that fails if the width across an axis does.
func TestTheSizeClassificationIsRotationInvariant(t *testing.T) {
	t.Parallel()
	for _, bore := range []float64{1e-10, 0.8} {
		wantRing, wantDrill, wantRefused, span := turnedMeasurement(t, bore, 0)
		for _, deg := range []float64{13.7, 37, 90} {
			ring, drill, refused, _ := turnedMeasurement(t, bore, deg)
			assertSameWidth(t, "ring", bore, deg, wantRing, ring, span)
			assertSameWidth(t, "drill", bore, deg, wantDrill, drill, span)
			if refused != wantRefused {
				t.Errorf("bore %g: the size classification refuses=%v at %g° and %v at 0°",
					bore, refused, deg, wantRefused)
			}
		}
	}
}

// assertSameWidth compares one body's width in two frames, against the arithmetic's own floor at
// this model's scale.
func assertSameWidth(t *testing.T, role string, bore, deg, want, got, span float64) {
	t.Helper()
	if drift := stdmath.Abs(got - want); drift > rotationDriftPerSpan*span {
		t.Errorf("bore %g: the %s is %v thick at %g° and %v at 0° (drift %g, floor %g) — one body, one width",
			bore, role, got, deg, want, drift, rotationDriftPerSpan*span)
	}
}

// turnedMeasurement is both operands' measured widths, the pair's verdict, and the model's own extent
// at one turn.
func turnedMeasurement(t *testing.T, bore, deg float64) (ring, drill float64, refused bool, span float64) {
	t.Helper()
	ringBody, drillBody := turnedRingAndDrill(t, bore, deg)
	ring, ok := solidThickness(ringBody, 0)
	if !ok {
		t.Fatalf("bore %g at %g°: the ring is a solid and must measure", bore, deg)
	}
	drill, ok = solidThickness(drillBody, 0)
	if !ok {
		t.Fatalf("bore %g at %g°: the drill is a solid and must measure", bore, deg)
	}
	_, err := classifyOperandSize(Cut, ringBody, drillBody, nil)
	box := ringBody.RangeBox().Union(drillBody.RangeBox())
	return ring, drill, err != nil, float64(box.Diagonal().Length())
}

// The PLANAR path gets its own row. The drill's width comes from a radial distance to its own axis
// and a slab's from projecting turned coordinates onto a turned normal — different arithmetic, and
// only the first is exercised by the row above. Neither is exact under a turn (measured: the drill
// drifts 6.8e-18 absolute, the slab 5.6e-16), so both are asserted against the same scaled floor.
//
// The first version of this test measured only a cylinder, whose width was then answered by its
// stored radius, so it would have passed with the whole planar path deleted (#3524 review I3).
func TestThePlanarWidthIsRotationInvariant(t *testing.T) {
	t.Parallel()
	want, ok := solidThickness(turnedBlock(t, 0), 0)
	if !ok || stdmath.Abs(want-0.25) > 1e-15 { // tol:numeric — an exact block extent
		t.Fatalf("the unturned 10x4x0.25 slab measures %v, %v; want 0.25", want, ok)
	}
	span := float64(turnedBlock(t, 0).RangeBox().Diagonal().Length())
	for _, deg := range []float64{13.7, 37, 90} {
		got, ok := solidThickness(turnedBlock(t, deg), 0)
		if !ok {
			t.Fatalf("the slab at %g° must measure", deg)
		}
		assertSameWidth(t, "slab", 0, deg, want, got, span)
	}
}

// A body's width comes from the DIRECTIONS its boundary supplies, and a body whose faces supply none
// that reach across it must still be measured — "unmeasured" reaches the caller as "not thin", which
// is the silent exit ADR-0061 stage 6 exists to close. These are the four shapes that were measured
// wrong or not at all when the width was gated on finding a pair of OPPOSED planar faces (#3524
// review C1); each is refused by name now, and base refused each too.
func TestTheShapesWithNoOpposedFacePairAreStillMeasured(t *testing.T) {
	t.Parallel()
	for _, row := range []struct {
		name string
		body func(*testing.T) *topo.Body
		want float64
	}{
		{"a cone collapsed to two coincident discs", func(t *testing.T) *topo.Body { return flatCone(t, 2e-9) }, 0},
		{"a cone 1e-8 tall", func(t *testing.T) *topo.Body { return flatCone(t, 1e-8) }, 1e-8},
		{"an ODD-sided prism, whose side normals have no antiparallel partner",
			func(*testing.T) *topo.Body { return ngonPrism(25, 1e-10, 12) }, 1.992114701314478e-10},
		{"an even-sided prism", func(*testing.T) *topo.Body { return ngonPrism(24, 1e-10, 12) }, 2e-10},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			body := row.body(t)
			got, ok := solidThickness(body, 0)
			if !ok {
				t.Fatalf("%s: a solid must always measure", row.name)
			}
			if stdmath.Abs(got-row.want) > 1e-3*row.want { // tol:numeric — the measured width, to 0.1%
				t.Errorf("%s measures %v; want %v", row.name, got, row.want)
			}
			cube, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "cube")
			if err != nil {
				t.Fatalf("cube: %v", err)
			}
			if _, err := classifyOperandSize(Cut, cube, body, nil); err == nil {
				t.Errorf("%s (%v thick) must be refused by name, not built silently", row.name, got)
			}
		})
	}
}

// flatCone is brep's own cone tool at a height that makes it sub-resolution. Below the model's weld
// the constructor collapses it into two coincident planar discs, which is a body with ONE boundary
// direction and no extent along it — zero is the honest width and the refusal names it.
func flatCone(t *testing.T, height float64) *topo.Body {
	t.Helper()
	cone, err := brep.SolidCylinderCone(math.P3(0, 0, 5-height), math.P3(0, 0, 5), 3, 0, "cone")
	if err != nil {
		t.Fatalf("cone %g tall: %v", height, err)
	}
	return cone
}

// ngonPrism is a faceted prism of n sides — the shape an import or a planarized tool produces. An ODD
// n is the interesting one: no side face has an antiparallel partner, so a measure that needed an
// opposed PAIR saw only the caps and read the prism's length instead of its width.
func ngonPrism(n int, radius, height float64) *topo.Body {
	verts := make([]math.Point3, 0, n*2)
	for _, z := range []float64{0, height} {
		for i := range n {
			a := 2 * stdmath.Pi * float64(i) / float64(n)
			verts = append(verts, math.P3(math.Scalar(radius*stdmath.Cos(a)), math.Scalar(radius*stdmath.Sin(a)), math.Scalar(z)))
		}
	}
	bottom, top := make([]int, n), make([]int, n)
	for i := range n {
		bottom[i], top[i] = n-1-i, n+i
	}
	faces := [][]int{bottom, top}
	for i := range n {
		next := (i + 1) % n
		faces = append(faces, []int{i, next, next + n, i + n})
	}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, "prism")
}

// The RING is measured by the direction its own torus supplies — its axis, across which the tube is
// 2 x MinorRadius — and not by a box that grows as the part turns.
func TestATorusIsAsThickAsItsTube(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	got, ok := solidThickness(ring, 0)
	if !ok || got != 3 {
		t.Errorf("solidThickness(torus R=5 r=1.5) = %v, %v; want 3", got, ok)
	}
}

// The width is GLOBAL and therefore the same for two representations of one shape. A spool with a
// thin neck is the case that decides it: read LOCALLY, the neck's own cylinder says 0.002 while the
// same neck built as a prism says the flange diameter, so two ways of modelling one part would
// classify differently — the representation-dependence the kernel ground rules exist to prevent
// (#3524 review I6). Read globally, both say what the spool is: 4 tall.
func TestOneShapeIsOneWidthInEitherRepresentation(t *testing.T) {
	t.Parallel()
	analytic := spoolByRevolution(t)
	got, ok := solidThickness(analytic, 0)
	if !ok || stdmath.Abs(got-4) > 1e-12 { // tol:numeric — an exact revolve extent
		t.Errorf("the revolved spool measures %v, %v; want its 4 of height", got, ok)
	}
	cube, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(40, 40, 40), "cube")
	if err != nil {
		t.Fatalf("cube: %v", err)
	}
	if _, err := classifyOperandSize(Cut, cube, analytic, nil); err != nil {
		t.Errorf("a spool with a 1e-3 neck is ordinary geometry and must not be refused: %v", err)
	}
}

// spoolByRevolution is two 10-wide flanges joined by a 1e-3-radius neck, revolved.
func spoolByRevolution(t *testing.T) *topo.Body {
	t.Helper()
	meridian := []math.Point2{
		math.P2(0, 0), math.P2(5, 0), math.P2(5, 1), math.P2(1e-3, 1),
		math.P2(1e-3, 3), math.P2(5, 3), math.P2(5, 4), math.P2(0, 4),
	}
	spool, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), meridian, "spool")
	if err != nil {
		t.Fatalf("spool: %v", err)
	}
	return spool
}

// The width is the body's own EXTENT along a direction, not the nearest opposed pair on it. The
// nearest-pair form is a LOCAL gap, and on a non-convex or multi-lump body it reads the space
// between two unrelated regions as a thickness: measured against the NopSCADlib corpus it reported a
// star washer as −0.5988740122992224 thick and an IDC transition as 6.938893903907228e-18. This pins
// the distinction with a plate that carries a shallow pocket.
func TestAPocketFloorIsNotAPlateThickness(t *testing.T) {
	t.Parallel()
	plate, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(20, 20, 10), "plate")
	if err != nil {
		t.Fatalf("plate: %v", err)
	}
	pocket, err := brep.SolidBlock(math.P3(5, 5, 9.5), math.P3(15, 15, 11), "pocket")
	if err != nil {
		t.Fatalf("pocket: %v", err)
	}
	pocketed, err := Boolean(Cut, plate, pocket)
	if err != nil {
		t.Fatalf("pocketed plate: %v", err)
	}
	got, ok := solidThickness(pocketed, 0)
	if !ok || stdmath.Abs(got-10) > 1e-9 { // tol:numeric — an exact block extent, float noise only
		t.Errorf("solidThickness(20x20x10 plate with a 0.5-deep pocket) = %v, %v; want 10", got, ok)
	}
}

// A body whose curved faces reach past every vertex must not be measured by its vertices alone: a
// half cylinder's flat face is bounded on one side by an ARC that carries no vertex of its own, so a
// vertex-only width reads zero and refuses ordinary geometry. The curved support points
// (topo.Body.CurvedSupportPoints) are what close that.
func TestACurvedBulgePastEveryVertexIsStillMeasured(t *testing.T) {
	t.Parallel()
	half, err := brep.SolidOfRevolutionSector(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), stdmath.Pi,
		[]brep.RevolveVertex{{P: math.P2(0, 0)}, {P: math.P2(2, 0)}, {P: math.P2(2, 3)}, {P: math.P2(0, 3)}}, "half")
	if err != nil {
		t.Fatalf("half cylinder: %v", err)
	}
	got, ok := solidThickness(half, 0)
	if !ok || stdmath.Abs(got-2) > 1e-9 { // tol:numeric — the half-round's own radius
		t.Errorf("solidThickness(half cylinder r=2 h=3) = %v, %v; want 2 (its radius)", got, ok)
	}
}

// A ball supplies no direction at all — no plane, and a sphere's every direction is an axis — so it
// is measured by the principal frame of its own support points. It must come out its diameter, and
// it must not be refused.
func TestABallIsMeasuredByItsOwnSupport(t *testing.T) {
	t.Parallel()
	ball, err := brep.SolidSphere(math.P3(0, 0, 0), 2.5, "ball")
	if err != nil {
		t.Fatalf("ball: %v", err)
	}
	got, ok := solidThickness(ball, 0)
	if !ok || stdmath.Abs(got-5) > 1e-9 { // tol:numeric — the sampled sphere's own diameter
		t.Errorf("solidThickness(ball r=2.5) = %v, %v; want 5", got, ok)
	}
}
