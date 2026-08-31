// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
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
	fs, rim := extrudedCylinderTopRim(t, 20, 40)
	NewDressUpFeatures(fs).AddChamfer([][]byte{rim}, func() float64 { return 10 })
	fs.Recompute()
	// The raise is measured against the host the emboss's boolean ACTUALLY consumes, which is the
	// planarized host, not the analytic one. The glyph tool is a planar wrapped prism, so combine
	// takes the planar path and facets the cylinder+cone host first — a declared degradation
	// (planarizedDiag records CodeBooleanAnalyticFaceted), permanent, and 87.09 cm³ of real material,
	// 280× the glyph. Differencing the two bodies as-measured therefore reports the FACETING, not the
	// emboss: the analytic host integrates to the exact 45029.494701 (π·20²·40 minus the 45° chamfer)
	// while the result is a 175-face polyhedron at 44942.403579, so the raise read −87.09 cm³. Both
	// numbers are right for their own body; the difference is not a raise. Against the host the
	// operation consumed — planarized to 44942.091693, which is also exactly what the mesh integrator
	// used to report for the analytic host, so this is the comparison the pre-analytic test was making
	// all along — the raise is the glyph alone. (Making the emboss keep the analytic host is
	// Oblikovati/Oblikovati#1601's curved-boolean gap, not this test's subject.)
	before := ops.BodyGeometryProperties(planarized(fs.Result()[0], "combine-target"), ops.DefaultQuality()).Volume
	cone, key := coneFaceOf(t, fs.Result()[0])

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
	raise := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume - before
	if raise < 0.1 || raise > 0.6 {
		t.Errorf("cone emboss raised %g cm³, want a ~1cm²×0.3cm glyph-sized raise in [0.1, 0.6]", raise)
	}
}

// TestConeWrapRejectsNonTangentPlane refuses a plane the cone wrap is not defined on: a
// cylinder-style plane whose normal is ⟂ the axis is NOT tangent to a cone (a cone's tangent plane
// tilts by the half angle and passes through the apex).
func TestConeWrapRejectsNonTangentPlane(t *testing.T) {
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
