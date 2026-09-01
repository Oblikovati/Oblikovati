// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"
	"time"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

func cylinderFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f.ReferenceKey()
		}
	}
	t.Fatal("no cylindrical face on the body")
	return nil
}

// TestParseThreadDesignation pins the ISO metric parser.
func TestParseThreadDesignation(t *testing.T) {
	t.Parallel()
	spec, err := ParseThreadDesignation("M8x1.25")
	if err != nil {
		t.Fatalf("M8x1.25: %v", err)
	}
	if spec.MajorDiameter != 8 || spec.Pitch != 1.25 || !spec.RightHanded {
		t.Errorf("M8x1.25 = %+v", spec)
	}
	if want := 8 - 1.0825*1.25; stdmath.Abs(spec.MinorDiameter-want) > 1e-9 {
		t.Errorf("minor = %g, want %g", spec.MinorDiameter, want)
	}
	if s, _ := ParseThreadDesignation("M6"); s.Pitch != 1.0 { // coarse table
		t.Errorf("M6 coarse pitch = %g, want 1.0", s.Pitch)
	}
	if s, _ := ParseThreadDesignation("M10x1.5-LH"); s.RightHanded {
		t.Error("M10x1.5-LH should be left-handed")
	}
	for _, bad := range []string{"8x1.25", "M0x1", "M8x0", "UNC8", "Mx"} {
		if _, err := ParseThreadDesignation(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

// TestThreadCosmeticOnCylinder applies a thread to a cylinder's side face: the feature is
// healthy, the solid is unchanged, and the resolved spec is recorded.
func TestThreadCosmeticOnCylinder(t *testing.T) {
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.4, 3)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	before := ops.BodyGeometryProperties(cyl, ops.DefaultQuality()).Volume
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(cyl)
	th := NewDressUpFeatures(fs).AddThreadDef(&ThreadDefinition{FaceKey: cylinderFaceKey(t, cyl), Designation: "M8x1.25", Cut: false})
	fs.Recompute()

	if !th.Health().OK() {
		t.Fatalf("cosmetic thread went sick: %+v", th.Health())
	}
	if got := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-before) > 1e-9 {
		t.Errorf("cosmetic thread changed the volume: %.6f → %.6f", before, got)
	}
	spec := th.Definition().(*ThreadFeature).Spec()
	if spec == nil || spec.MajorDiameter != 8 || spec.Internal {
		t.Errorf("recorded spec = %+v, want external M8", spec)
	}
}

// TestThreadSickOnPlanarFace rejects a thread on a non-cylindrical face.
func TestThreadSickOnPlanarFace(t *testing.T) {
	t.Parallel()
	box := prismBody() // a unit prism: every face is planar
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	th := NewDressUpFeatures(fs).AddThreadDef(&ThreadDefinition{FaceKey: box.Faces()[0].ReferenceKey(), Designation: "M8x1.25", Cut: false})
	fs.Recompute()
	if th.Health().Status != health.Sick {
		t.Errorf("thread on a planar face should be Sick, got %v", th.Health().Status)
	}
}

// TestThreadSickOnBadDesignation rejects an unparseable designation.
func TestThreadSickOnBadDesignation(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.4, 3)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(cyl)
	th := NewDressUpFeatures(fs).AddThreadDef(&ThreadDefinition{FaceKey: cylinderFaceKey(t, cyl), Designation: "garbage", Cut: false})
	fs.Recompute()
	if th.Health().Status != health.Sick {
		t.Errorf("bad designation should be Sick, got %v", th.Health().Status)
	}
}

// TestThreadDisplayHelixOnSurface checks the cosmetic thread produces a helix display curve on
// its cylindrical face (the data the head renders).
func TestThreadDisplayHelixOnSurface(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.4, 1.0)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(cyl)
	NewDressUpFeatures(fs).AddThreadDef(&ThreadDefinition{FaceKey: cylinderFaceKey(t, cyl), Designation: "M8x1.25", Cut: false})
	fs.Recompute()

	curves := ThreadDisplayCurves(fs)
	if len(curves) != 1 {
		t.Fatalf("got %d thread display curves, want 1", len(curves))
	}
	h := curves[0]
	if len(h) < 8 { // ~8 turns over 1 cm at pitch 0.125 cm
		t.Fatalf("helix too short: %d points", len(h))
	}
	for _, p := range h {
		if r := stdmath.Hypot(float64(p.X), float64(p.Y)); stdmath.Abs(r-0.4) > 1e-6 {
			t.Errorf("helix point off the surface: r=%.5f, want 0.4", r)
		}
		if z := float64(p.Z); z < -1e-9 || z > 1.0+1e-9 {
			t.Errorf("helix point out of axial extent: z=%.5f", z)
		}
	}
}

// TestThreadDisplayPartialSpan checks a cosmetic thread limited by Offset/Length draws its helix
// only over that axial window (Inventor's ThreadOffset/ThreadDepth) — the double-ended-stud case,
// where the two end threads must not spill into the plain middle.
func TestThreadDisplayPartialSpan(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.4, 3.0)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(cyl)
	NewDressUpFeatures(fs).AddThreadDef(&ThreadDefinition{
		FaceKey:     cylinderFaceKey(t, cyl),
		Designation: "M8x1.25",
		Offset:      constFloat(2.0), // start 2 cm up the 3 cm cylinder
		Length:      constFloat(0.5), // run 0.5 cm — the nut-end band
	})
	fs.Recompute()

	curves := ThreadDisplayCurves(fs)
	if len(curves) != 1 {
		t.Fatalf("got %d thread display curves, want 1", len(curves))
	}
	for _, p := range curves[0] {
		if z := float64(p.Z); z < 2.0-1e-6 || z > 2.5+1e-6 {
			t.Errorf("helix point outside the [2.0,2.5] thread span: z=%.5f", z)
		}
	}
}

// TestThreadCutModelsRealThreadFast checks the modeled (cut) thread retypes the face to a
// threaded surface in O(1) — microseconds, no boolean — giving a valid solid whose volume
// drops (the grooves are real geometry).
func TestThreadCutModelsRealThreadFast(t *testing.T) {
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5, 2.0)
	before := ops.BodyGeometryProperties(cyl, ops.DefaultQuality()).Volume
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(cyl)
	th := NewDressUpFeatures(fs).AddThreadDef(&ThreadDefinition{FaceKey: cylinderFaceKey(t, cyl), Designation: "M8x1.25", Cut: true})

	start := time.Now()
	fs.Recompute()
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Errorf("modeled thread recompute took %v, want ≪ a few ms (no boolean)", d)
	}
	if !th.Health().OK() {
		t.Fatalf("modeled thread sick: %+v", th.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("modeled thread not a valid solid: %+v", r)
	}
	after := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if after >= before || after <= 0 {
		t.Errorf("modeled thread volume %.4f not in (0, %.4f) — grooves should remove material", after, before)
	}
}
