// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// coneWrapFixture builds a cone (apex at the origin, axis +Z, radial(0) = +X, the given half angle)
// and a sketch plane TANGENT to it along the u=0 generator, with the sketch origin at slant s0. The
// plane's in-plane X runs circumferentially and Y runs along the slant (the wrap convention).
func coneWrapFixture(t *testing.T, halfAngle, s0 float64) (coneWrapFrame, sketch.Plane) {
	t.Helper()
	cone, err := geom.NewConeWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), halfAngle)
	if err != nil {
		t.Fatalf("NewConeWithRef: %v", err)
	}
	sinA, cosA := stdmath.Sin(halfAngle), stdmath.Cos(halfAngle)
	g := math.V3(math.Scalar(sinA), 0, math.Scalar(cosA)) // unit generator ĝ at u=0
	origin := math.P3(math.Scalar(s0*sinA), 0, math.Scalar(s0*cosA))
	plane, err := sketch.NewPlane(origin, math.V3(0, 1, 0).AsUnit(), g.AsUnit())
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	fr, err := coneWrapFrameFor(cone, plane)
	if err != nil {
		t.Fatalf("coneWrapFrameFor: %v", err)
	}
	return fr, plane
}

// TestConeWrapFrameHugsAndIsIsometric pins the cone development map: mapped points lie ON the cone,
// a radial sketch segment keeps its length (the developable-wrap isometry), and a circumferential
// offset subtends the RIGHT angle about the axis (φ/sinα — the angle a unit arc subtends shrinks
// with slant). Each check is a distinct failure mode a wrong map would trip.
func TestConeWrapFrameHugsAndIsIsometric(t *testing.T) {
	t.Parallel()
	const halfAngle, s0 = 0.5, 4.0
	fr, plane := coneWrapFixture(t, halfAngle, s0)
	sinA := stdmath.Sin(halfAngle)
	tanA := stdmath.Tan(halfAngle)

	// 1) HUG: every mapped point (level 0) satisfies the cone equation radius = v·tanα.
	for _, sx := range []float64{-2, -1, 0, 1, 2} {
		for _, sy := range []float64{-1.5, 0, 1.5} {
			p := fr.at(math.P2(math.Scalar(sx), math.Scalar(sy)), plane, 0)
			r := stdmath.Hypot(float64(p.X), float64(p.Y))
			if off := stdmath.Abs(r - float64(p.Z)*tanA); off > 1e-9 {
				t.Errorf("sketch (%g,%g) maps %v — %g off the cone surface (radius ≠ v·tanα)", sx, sy, p, off)
			}
		}
	}
	// 2) RADIAL ISOMETRY: a radial sketch segment (Δslant along Y) maps to a 3D segment of EQUAL
	// length — the defining property of a developable wrap. A cosα slip here would stretch/shrink it.
	a := fr.at(math.P2(0, 0), plane, 0)
	b := fr.at(math.P2(0, 2), plane, 0)
	if got := float64(a.DistanceTo(b)); stdmath.Abs(got-2) > 1e-9 {
		t.Errorf("a radial sketch segment of length 2 maps to a 3D length of %g, want 2 (isometry)", got)
	}
	// 3) ANGLE-PER-ARC: sketch (sx,0) subtends apex-angle φ = atan2(sx, s0); it must land at azimuth
	// φ/sinα about the axis. A sinα slip (or dropping it) puts the arc at the wrong angle.
	for _, sx := range []float64{0.5, 1, 2} {
		phi := stdmath.Atan2(sx, s0)
		p := fr.at(math.P2(math.Scalar(sx), 0), plane, 0)
		az := stdmath.Atan2(float64(p.Y), float64(p.X))
		if want := phi / sinA; stdmath.Abs(az-want) > 1e-9 {
			t.Errorf("sketch (%g,0) lands at azimuth %g about the axis, want φ/sinα = %g", sx, az, want)
		}
	}
}

// TestConeWrapFrameOffsetsAlongNormal checks the emboss depth is applied along the cone's SURFACE
// NORMAL (cosα·radial − sinα·axis), which tilts back toward the apex — not radially as on a
// cylinder. A radial offset would leave the axial component at zero.
func TestConeWrapFrameOffsetsAlongNormal(t *testing.T) {
	t.Parallel()
	const halfAngle, s0 = 0.5, 4.0
	fr, plane := coneWrapFixture(t, halfAngle, s0)
	p2 := math.P2(1, math.Scalar(0.5))
	surface := fr.at(p2, plane, 0)
	const d = 0.3
	raised := fr.at(p2, plane, d)
	if got := float64(surface.DistanceTo(raised)); stdmath.Abs(got-d) > 1e-9 {
		t.Errorf("level %g offsets the point %g from the surface, want %g", d, got, d)
	}
	axial := float64(surface.VectorTo(raised).Dot(math.V3(0, 0, 1)))
	if want := -stdmath.Sin(halfAngle) * d; stdmath.Abs(axial-want) > 1e-9 {
		t.Errorf("the offset's axial component is %g, want −sinα·d = %g (the offset is along the cone normal, not radial)", axial, want)
	}
}

// coneFaceOf returns a body's single conical face's geometry and reference key.
func coneFaceOf(t *testing.T, b *topo.Body) (geom.Cone, []byte) {
	t.Helper()
	for _, f := range b.Faces() {
		switch c := f.Geometry().(type) {
		case geom.Cone:
			return c, f.ReferenceKey()
		case *geom.Cone:
			return *c, f.ReferenceKey()
		}
	}
	t.Fatal("no conical face on the body")
	return geom.Cone{}, nil
}

// tangentPlaneToCone builds a sketch plane tangent to the cone along its u=0 generator, with the
// sketch origin at slant s0 (in-plane X circumferential, Y along the slant) — the plane a cone wrap
// needs. Mirrors the cylinder's shaftTangentPlane.
func tangentPlaneToCone(t *testing.T, cone geom.Cone, s0 float64) sketch.Plane {
	t.Helper()
	axis, radial0 := cone.AxisDir.AsVector(), cone.Ref.AsVector()
	sinA, cosA := stdmath.Sin(cone.HalfAngle), stdmath.Cos(cone.HalfAngle)
	g := axis.Scale(math.Scalar(cosA)).Add(radial0.Scale(math.Scalar(sinA)))
	origin := cone.Apex.TranslateBy(g.Scale(math.Scalar(s0)))
	plane, err := sketch.NewPlane(origin, axis.Cross(radial0).AsUnit(), g.AsUnit())
	if err != nil {
		t.Fatalf("tangent plane: %v", err)
	}
	return plane
}

// TestWrappedEmbossOnConeIsAValidSolidThatAddsMaterial is the #2065 end-to-end acceptance: a small
// rectangle embossed onto a real conical face (the chamfer of a big cylinder rim) builds a valid
// watertight solid and adds a glyph-sized amount of material — proving the cone dispatch, tangency
// frame, tool build and boolean compose over the real kernel. The map's geometric correctness (hug,
// isometry) is pinned by the frame unit tests above.
func TestWrappedEmbossOnConeIsAValidSolidThatAddsMaterial(t *testing.T) {
	t.Parallel()
	fs, rim := extrudedCylinderTopRim(t, 20, 40)
	NewDressUpFeatures(fs).AddChamfer([][]byte{rim}, func() float64 { return 10 })
	fs.Recompute()
	// The emboss now joins onto the chamfer cone ANALYTICALLY (#3459): the result keeps the host's
	// cone, cylinder and caps as analytic faces and adds the pad's own, where it used to hand back a
	// 1234-face polyhedron. That changed what this test can honestly measure.
	//
	// It used to difference two whole-body volumes. That worked only while BOTH sides were faceted,
	// so the inscribed-polyhedron bias cancelled — the comments this replaces spent a paragraph
	// explaining that the baseline had to be planarized for exactly that reason. Against an analytic
	// result BOTH sides integrate in closed form and agree with the mesh to every digit (45029.796846,
	// the body closing to 1.2e-13 of its own area), so the difference is honest again — but it is still
	// the wrong instrument: a 0.3 cm³ raise is a 7e-6 relative change on a 45029 cm³ body, and meshed at
	// property quality the readout alone carries about 1 cm³ of noise, 3× the whole glyph.
	//
	// The kernel rules say as much: "Result gates are per-face (area, surface type, loop count)
	// against the oracle. A whole-body volume or area match is a smoke test, never a proof." So the
	// gate below is per-face, and it is a stronger statement than the band was: the glyph exists, it
	// is an analytic cone, its area is the sketch's 1 cm², and the host's cone carries the matching
	// footprint hole.
	cone, key := coneFaceOf(t, fs.Result()[0])
	bareConeArea, ok := query.AnalyticFaceArea(soleConeFace(t, fs.Result()[0]))
	if !ok {
		t.Fatal("the host chamfer cone's area is not analytically measurable before the emboss")
	}

	// Sketch origin mid-band on the chamfer cone (its slant runs ≈14→28); a 1×1 cm raise sits well
	// inside it. Depth 0.3 cm along the surface normal.
	pf := NewEmbossFeatures(fs).Add(rectSketchOn(tangentPlaneToCone(t, cone, 21), -0.5, -0.5, 0.5, 0.5),
		[]int{0}, func() float64 { return 0.3 }, EmbossFromFace, 0)
	pf.Definition().(*EmbossFeature).Definition().WrapFaceKey = key
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("wrapped cone emboss went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("cone-wrapped emboss is not a valid solid: %+v", r.Issues)
	}
	// The host's chamfer cone must SURVIVE as a cone and carry the glyph's footprint as a second
	// loop; the glyph's own top must be a cone of the sketch's area. Both areas are checked against
	// the analytic integrator, which is more exact than the tessellation it would otherwise be
	// measured by.
	var hostCone, glyphTop *topo.Face
	for _, f := range body.Faces() {
		if _, isCone := f.Geometry().(geom.Cone); !isCone {
			continue
		}
		if len(f.Loops()) > 1 {
			hostCone = f
			continue
		}
		glyphTop = f
	}
	if hostCone == nil {
		t.Fatalf("the host chamfer cone did not survive the emboss as a cone with a footprint hole; "+
			"faces are %v", faceKindCounts(body))
	}
	if glyphTop == nil {
		t.Fatalf("the raised glyph has no analytic cone top; faces are %v", faceKindCounts(body))
	}
	// The footprint must be REMOVED from the host cone, not added to it. The hole straddles the
	// cone's parameter seam, and an even-odd nesting test asked on one branch of the chart read it
	// as a top-level loop and ADDED its measure — the host came out 1333.86 cm² where its own
	// undrilled band is 1332.86 (Oblikovati/Oblikovati#3505).
	hostArea, ok := query.AnalyticFaceArea(hostCone)
	if !ok {
		t.Fatal("the host cone's area is not analytically measurable")
	}
	if removed := bareConeArea - hostArea; removed < 0.9 || removed > 1.15 {
		t.Errorf("the emboss removed %g cm² from the host cone (%g → %g), want the sketch's 1 cm²; a "+
			"NEGATIVE figure means the footprint hole was added to the face instead of subtracted",
			removed, bareConeArea, hostArea)
	}
	area, ok := query.AnalyticFaceArea(glyphTop)
	if !ok {
		t.Fatal("the glyph top's area is not analytically measurable")
	}
	if area < 0.9 || area > 1.15 {
		t.Errorf("the raised glyph's top is %g cm², want the sketch's 1 cm² (wrapping a 1×1 rectangle "+
			"onto a cone stretches it slightly)", area)
	}
	// No faceting: the only planar faces are the host's two caps and the pad's own walls.
	if n := faceKindCounts(body)["geom.Plane"]; n > 10 {
		t.Errorf("the emboss produced %d planar faces; the host was faceted instead of joined "+
			"analytically (a faceted result runs to ~1234)", n)
	}
}

// soleConeFace is the body's one and only cone face, so a later measurement of "the host cone" is
// anchored to the same surface the emboss is about to cut into.
func soleConeFace(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	var out *topo.Face
	for _, f := range b.Faces() {
		if _, isCone := f.Geometry().(geom.Cone); !isCone {
			continue
		}
		if out != nil {
			t.Fatalf("the fixture has more than one cone face; faces are %v", faceKindCounts(b))
		}
		out = f
	}
	if out == nil {
		t.Fatalf("the fixture has no cone face; faces are %v", faceKindCounts(b))
	}
	return out
}

// faceKindCounts tallies a body's faces by surface kind — the per-face view these gates read.
func faceKindCounts(b *topo.Body) map[string]int {
	out := map[string]int{}
	for _, f := range b.Faces() {
		out[fmt.Sprintf("%T", f.Geometry())]++
	}
	return out
}

// TestConeWrapRejectsNonTangentPlane refuses a plane the cone wrap is not defined on: a
// cylinder-style plane whose normal is ⟂ the axis is NOT tangent to a cone (a cone's tangent plane
// tilts by the half angle and passes through the apex).
func TestConeWrapRejectsNonTangentPlane(t *testing.T) {
	t.Parallel()
	cone, err := geom.NewConeWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 0.5)
	if err != nil {
		t.Fatal(err)
	}
	bad, err := sketch.NewPlane(math.P3(1, 0, math.Scalar(2)), math.V3(0, 1, 0).AsUnit(), math.V3(0, 0, 1).AsUnit())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coneWrapFrameFor(cone, bad); err == nil {
		t.Error("a plane whose normal is ⟂ the axis is not tangent to a cone; want a refusal")
	}
}
