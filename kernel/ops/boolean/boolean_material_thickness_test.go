// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// turnAbout is the rotation the rotation-invariance rows use: about (1,1,0), the axis that is NOT
// perpendicular to the drill's own axis. A turn about a coordinate axis leaves one world axis square
// to the drill, so the bounding box still measured the bore and the defect stayed hidden; this one
// tilts the drill into every world axis at once.
func turnAbout(deg float64) math.Matrix4 {
	return math.Rotation4(math.Scalar(deg*stdmath.Pi/180), math.V3(1, 1, 0).AsUnit(), math.P3(0, 0, 0))
}

// turnedRingAndDrill is ringAndDrill built in a frame turned by deg. Both operands turn, so the
// operation is CONGRUENT to the untuned one and every honest measure of it must agree.
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
// ways depending on how the part happened to be turned, and the numbers that agreed only agreed to
// six figures. The measure is now the body's own face geometry, so the value is BIT-IDENTICAL at
// every angle, not merely close.
func TestTheSizeClassificationIsRotationInvariant(t *testing.T) {
	t.Parallel()
	for _, bore := range []float64{1e-10, 0.8} {
		want, wantRefused := turnedMeasurement(t, bore, 0)
		for _, deg := range []float64{37, 90} {
			got, refused := turnedMeasurement(t, bore, deg)
			if got != want {
				t.Errorf("bore %g: thickness at %g° is %v, at 0° it is %v — one body, one thickness",
					bore, deg, got, want)
			}
			if refused != wantRefused {
				t.Errorf("bore %g: the size classification refuses=%v at %g° and %v at 0°",
					bore, refused, deg, wantRefused)
			}
		}
	}
}

// turnedMeasurement is the drill's measured thickness and the pair's verdict at one turn.
func turnedMeasurement(t *testing.T, bore, deg float64) (float64, bool) {
	t.Helper()
	ring, drill := turnedRingAndDrill(t, bore, deg)
	thickness, ok := solidThickness(drill, 0)
	if !ok {
		t.Fatalf("bore %g at %g°: the drill is a solid and must measure", bore, deg)
	}
	_, err := classifyOperandSize(Cut, ring, drill, nil)
	return thickness, err != nil
}

// The RING is measured by its own torus face — the tube it is made of, 2 x MinorRadius across — and
// not by a box that grows as the part turns.
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

// A tube's material is its WALL, which no bounding box can see: the box of a 10/9.999 tube is 20
// across, four orders of magnitude above the 1e-3 of material it actually holds. The wall is the
// thickness a coaxial cylinder PAIR reports (geom.OpposedSpan).
func TestATubeIsAsThickAsItsWall(t *testing.T) {
	t.Parallel()
	tube := coaxialTube(t, 10, 9.999, 4)
	got, ok := solidThickness(tube, 0)
	if !ok || stdmath.Abs(got-1e-3) > 1e-12 { // tol:numeric — an exact radius difference, float noise only
		t.Errorf("solidThickness(tube wall 1e-3) = %v, %v; want 1e-3", got, ok)
	}
}

// coaxialTube builds a solid tube by revolving its rectangular meridian: an outer cylinder facing
// out, an inner one facing in, and two annular caps.
func coaxialTube(t *testing.T, outer, inner, height float64) *topo.Body {
	t.Helper()
	meridian := []math.Point2{
		math.P2(math.Scalar(inner), 0), math.P2(math.Scalar(outer), 0),
		math.P2(math.Scalar(outer), math.Scalar(height)), math.P2(math.Scalar(inner), math.Scalar(height)),
	}
	tube, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), meridian, "tube")
	if err != nil {
		t.Fatalf("tube: %v", err)
	}
	return tube
}

// A small closed curved face must NOT be read as the body's thickness unless its trim wraps the whole
// surface: a 1e-9 fillet lies on a cylinder of that radius while the plate it rounds is 10 thick. The
// wrap test (faceWrapsItsSurface) is what keeps the classification off ordinary geometry.
func TestAPartialCurvedFaceIsNotABodyThickness(t *testing.T) {
	t.Parallel()
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatalf("block: %v", err)
	}
	for _, f := range block.Faces() {
		if faceWrapsItsSurface(f) {
			t.Errorf("a planar block face %q must not count as a closed wrap", f.ReferenceKey())
		}
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	if !anyFaceWraps(rod) {
		t.Error("a whole cylinder's periodic side face wraps its surface and must be recognised")
	}
}

func anyFaceWraps(b *topo.Body) bool {
	for _, f := range b.Faces() {
		if faceWrapsItsSurface(f) {
			return true
		}
	}
	return false
}

// The width across a direction group is the body's own EXTENT along it, not the nearest opposed pair.
// Measured with the nearest-pair form against the NopSCADlib corpus, the local reading called a star
// washer −0.5988740122992224 thick and an IDC transition 6.938893903907228e-18 — a gap between two
// unrelated regions of one body, and a pair of coincident facets. This pins the distinction with a
// plate that carries a shallow pocket: the pocket floor is nearer the top than the plate is thick.
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
