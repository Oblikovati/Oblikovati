// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"testing"
)

// angleParamBytes builds one parameter in its on-disk shape: value f64, an identical nominal
// duplicate 8 bytes later, then the unit-type id 20 bytes past the value — the shape soleSweepAngle
// keys on. Prefixed by a "Revolution" marker in the tests so HasRevolve is satisfied.
func angleParamBytes(value float64, unit byte) []byte {
	b := make([]byte, 40)
	binary.LittleEndian.PutUint64(b[0:], math.Float64bits(value)) // value
	binary.LittleEndian.PutUint64(b[8:], math.Float64bits(value)) // nominal duplicate at +8
	b[20] = unit                                                  // unit-type id at value+20
	return append(append([]byte{}, utf16Of("Revolution")...), b...)
}

// TestRevolveAngleUsesUnit locks the unit-based angle decode: RevolveAngle reports a partial angle
// only for a param carrying the ANGLE unit id — a shaft's leading profile LENGTH (0.4 cm) read as
// radians had turned a full 360° revolve into a sliver. A length-unit param is ignored (full turn),
// an angle-unit param is honoured.
func TestRevolveAngleUsesUnit(t *testing.T) {
	if a, ok := RevolveAngle(angleParamBytes(0.4, lengthUnitID)); ok {
		t.Errorf("length param read as angle %.3f rad; want full revolve (ok=false)", a)
	}
	if a, ok := RevolveAngle(angleParamBytes(1.5, angleUnitID)); !ok || math.Abs(a-1.5) > 1e-9 {
		t.Errorf("angle param = (%.3f, %v); want (1.5, true)", a, ok)
	}
}

// TestRevolveExtentEnumForcesFull locks the feature→extent binding: a part carrying the full-revolve
// extent enum (kind=12 value=3) is swept FULL even when an angle-unit param is present — the profile
// angle DIMENSION (a chamfer's) is not the sweep. Without the enum the lone angle param binds.
func TestRevolveExtentEnumForcesFull(t *testing.T) {
	extentEnum := func(value uint16) []byte { // enum node: dcNodeTag|id|nullRef|marker|kind=12|value|0x26
		b := make([]byte, 24)
		binary.LittleEndian.PutUint32(b[0:], dcNodeTag)
		binary.LittleEndian.PutUint32(b[8:], nullRef)
		binary.LittleEndian.PutUint32(b[12:], 0x0A96)
		binary.LittleEndian.PutUint16(b[16:], revolveExtentKind)
		binary.LittleEndian.PutUint16(b[18:], value)
		binary.LittleEndian.PutUint32(b[20:], enumTrailer)
		return b
	}
	withAngle := angleParamBytes(1.5, angleUnitID) // Revolution marker + a 1.5 rad angle param
	if !HasFullRevolveExtent(append(withAngle, extentEnum(revolveFullExtent)...)) {
		t.Error("kind=12 value=3 not recognised as a full-revolve extent")
	}
	if _, ok := RevolveAngle(append(withAngle, extentEnum(revolveFullExtent)...)); ok {
		t.Error("full-extent revolve reported a partial angle (the profile angle dim was mistaken for the sweep)")
	}
	if a, ok := RevolveAngle(append(withAngle, extentEnum(1)...)); !ok || math.Abs(a-1.5) > 1e-9 {
		t.Errorf("angle-extent revolve = (%.3f, %v); want the 1.5 rad param", a, ok)
	}
}

// utf16Of returns the UTF-16LE bytes of s (the on-disk encoding of an Inventor node name).
func utf16Of(s string) []byte {
	b := make([]byte, 0, 2*len(s))
	for _, r := range s {
		b = append(b, byte(r), byte(r>>8))
	}
	return b
}

func TestDecodeExtrude(t *testing.T) {
	cases := []struct {
		file     string
		wantOK   bool
		distance float64
	}{
		{"15_cylinder.ipt", true, 2.0}, // circle extruded 2 cm
		{"10_box.ipt", true, 1.0},      // rectangle extruded 1 cm
		{"sketch_line.ipt", false, 0},  // sketch only, no feature
		{"param_L10.ipt", false, 0},    // parameter only
	}
	for _, tc := range cases {
		d := openDoc(t, tc.file)
		seg, ok := d.Segment("PmDCSegment")
		if !ok {
			t.Fatalf("%s: no PmDCSegment", tc.file)
		}
		ex, ok := DecodeExtrude(seg)
		if ok != tc.wantOK {
			t.Errorf("%s: DecodeExtrude ok=%v, want %v", tc.file, ok, tc.wantOK)
			continue
		}
		if ok && absf(ex.Distance-tc.distance) > 1e-9 {
			t.Errorf("%s: distance = %.4f cm, want %.4f", tc.file, ex.Distance, tc.distance)
		}
	}
}

// TestRevolveAngle: a 270° revolve reports 3π/2 rad; a full revolve reports no angle.
func TestRevolveAngle(t *testing.T) {
	d := openDoc(t, "24_revolve_270.ipt")
	seg, _ := d.Segment("PmDCSegment")
	a, ok := RevolveAngle(seg)
	if !ok {
		t.Fatal("270° revolve: no angle decoded")
	}
	if absf(a-3*mathPi/2) > 1e-9 {
		t.Errorf("angle = %.6f rad, want 3π/2 (%.6f)", a, 3*mathPi/2)
	}
	full := openDoc(t, "16_revolve.ipt")
	fseg, _ := full.Segment("PmDCSegment")
	if _, ok := RevolveAngle(fseg); ok {
		t.Error("full revolve reported a partial angle")
	}
}

// TestRevolveAxisLine checks centreline detection: the isolated line (both endpoints unshared)
// is the revolve axis, while a plain closed loop has none. This is what lets a real shaft
// revolve about its drawn centreline instead of the fixed X origin axis.
func TestRevolveAxisLine(t *testing.T) {
	s := Sketch{Lines: []Line{
		{A: Point2D{1, 0}, B: Point2D{2, 0}}, // closed triangle loop
		{A: Point2D{2, 0}, B: Point2D{1, 1}},
		{A: Point2D{1, 1}, B: Point2D{1, 0}},
		{A: Point2D{0, 0}, B: Point2D{0, 2}}, // isolated centreline (endpoints shared with nothing)
	}}
	if i, ok := RevolveAxisLine(s); !ok || i != 3 {
		t.Errorf("RevolveAxisLine = (%d, %v), want (3, true)", i, ok)
	}
	if _, ok := RevolveAxisLine(Sketch{Lines: s.Lines[:3]}); ok {
		t.Error("a plain closed loop should report no centreline")
	}
}

// squareLoop is a unit CCW square from (x0,y0), 4 closed lines — a reusable revolve profile.
func squareLoop(x0, y0, w float64) []Line {
	return []Line{
		{A: Point2D{x0, y0}, B: Point2D{x0 + w, y0}},
		{A: Point2D{x0 + w, y0}, B: Point2D{x0 + w, y0 + w}},
		{A: Point2D{x0 + w, y0 + w}, B: Point2D{x0, y0 + w}},
		{A: Point2D{x0, y0 + w}, B: Point2D{x0, y0}},
	}
}

// TestRevolveProfileCaseA locks the shaft encoding: a closed loop offset from a vertical
// centreline drawn in the SAME sketch (e.g. ReelMotorBearingShaft) resolves to a revolve about
// that centreline.
func TestRevolveProfileCaseA(t *testing.T) {
	lines := append(squareLoop(0.5, 0, 1), Line{A: Point2D{0, 0}, B: Point2D{0, 3}}) // + vertical axis
	b, ok := RevolveProfile([]Sketch{{Resolved: true, Lines: lines}})
	if !ok || b != (RevolveBinding{ProfileSketch: 0, AxisSketch: 0, AxisLine: 4}) {
		t.Fatalf("case A: got %+v ok=%v, want profile 0 / axis line 4", b, ok)
	}
}

// TestRevolveProfileCaseB locks the corpus encoding: a clean closed profile in one sketch with
// the centreline drawn as a separate single-line sketch it is strictly offset from.
func TestRevolveProfileCaseB(t *testing.T) {
	profile := Sketch{Resolved: true, Lines: squareLoop(1, 0.5, 2)}
	axis := Sketch{Resolved: true, Lines: []Line{{A: Point2D{0.5, 0}, B: Point2D{3.5, 0}}}}
	b, ok := RevolveProfile([]Sketch{profile, axis})
	if !ok || b != (RevolveBinding{ProfileSketch: 0, AxisSketch: 1, AxisLine: 0}) {
		t.Fatalf("case B: got %+v ok=%v, want profile 0 / axis sketch 1", b, ok)
	}
}

// TestRevolveProfileCaseC locks the solid-shaft encoding: a fully-closed profile whose own edge
// lies on the vertical axis (x≈0), revolved about that edge (LeverShaft, PressureRollerMainShaft).
// There is no isolated centreline or separate axis sketch — the axis is the profile boundary.
func TestRevolveProfileCaseC(t *testing.T) {
	lines := squareLoop(-1, 0, 1) // corners (-1,0)-(0,0)-(0,1)-(-1,1); the (0,0)-(0,1) edge is on x=0
	b, ok := RevolveProfile([]Sketch{{Resolved: true, Lines: lines}})
	if !ok || b.ProfileSketch != 0 || b.AxisSketch != 0 {
		t.Fatalf("case C: got %+v ok=%v, want a self-axis revolve on sketch 0", b, ok)
	}
	if l := lines[b.AxisLine]; absf(l.A.X) > 1e-9 || absf(l.B.X) > 1e-9 {
		t.Errorf("axis line %d = %+v, want the edge on x=0", b.AxisLine, l)
	}
}

// TestRevolveProfileRejects covers every gate that must fall back to the mesh rather than emit a
// wrong revolve: an incomplete (open) profile, an ambiguous horizontal in-sketch centreline (the
// 1677K262 mis-pick), a separate axis line the profile touches (the PressureRollerMainShaft
// giant-torus mis-pick), and an unresolved sketch.
func TestRevolveProfileRejects(t *testing.T) {
	openChain := []Line{ // a gap between the second and third segment: not a closed ring
		{A: Point2D{0.5, 0}, B: Point2D{0.5, 1}}, {A: Point2D{0.5, 1}, B: Point2D{1.5, 1}},
		{A: Point2D{1.5, 2}, B: Point2D{1.5, 3}}, {A: Point2D{0, 0}, B: Point2D{0, 3}},
	}
	horizAxis := append(squareLoop(0.5, 0.9, 1), Line{A: Point2D{0, 0}, B: Point2D{-1.8, 0}})
	touchingAxis := []Sketch{
		{Resolved: true, Lines: squareLoop(1, 0.5, 2)},
		{Resolved: true, Lines: []Line{{A: Point2D{0.5, 0.5}, B: Point2D{3.5, 0.5}}}}, // on the profile edge
	}
	// The 1677K262 trap post-incidence: a closed profile with an edge ON x=0 (a case-C candidate)
	// AND a separate HORIZONTAL centreline it is offset from — two conflicting axes. It is really a
	// partial revolve about the horizontal line whose angle lives in no readable parameter, so either
	// axis swept full is a wrong solid; the ambiguity must decline to the mesh.
	ambiguousAxis := []Sketch{
		{Resolved: true, Lines: []Line{ // profile at x∈[-0.5,0], y∈[0.9,1.3], edge (0,0.9)-(0,1.3) on x=0
			{A: Point2D{0, 0.9}, B: Point2D{0, 1.3}}, {A: Point2D{0, 1.3}, B: Point2D{-0.5, 1.3}},
			{A: Point2D{-0.5, 1.3}, B: Point2D{-0.5, 0.9}}, {A: Point2D{-0.5, 0.9}, B: Point2D{0, 0.9}},
		}},
		{Resolved: true, Lines: []Line{{A: Point2D{0, 0}, B: Point2D{-1.8, 0}}}}, // separate horizontal axis
	}
	cases := []struct {
		name     string
		sketches []Sketch
	}{
		{"open profile", []Sketch{{Resolved: true, Lines: openChain}}},
		{"horizontal in-sketch axis", []Sketch{{Resolved: true, Lines: horizAxis}}},
		{"separate axis touches profile", touchingAxis},
		{"ambiguous x=0 edge vs separate horizontal axis", ambiguousAxis},
		{"unresolved", []Sketch{{Resolved: false, Lines: squareLoop(0.5, 0, 1)}}},
	}
	for _, tc := range cases {
		if _, ok := RevolveProfile(tc.sketches); ok {
			t.Errorf("%s: RevolveProfile ok=true, want false (must fall back to mesh)", tc.name)
		}
	}
}
