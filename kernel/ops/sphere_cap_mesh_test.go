// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Regression for the flat-cap tessellation bug (M2 Phase 1, Oblikovati/Oblikovati#1334): a sphere
// trimmed by one plane used to mesh as a flat disk (its coplanar rim lifted to no bulge → zero cap
// volume). sphereCapFan must mesh the true spherical cap, so the body's volume matches the analytic
// spherical-cap formula V = π·h²·(3R − h)/3 for the kept (−normal) side, height h = R + d where
// d = (planePoint − centre)·normal (signed centre→plane distance along the normal). Accuracy is
// held to 2%: the kernel's own DefaultQuality full sphere is ~1.6% under analytic (inscribed
// chords), so a cap meshed at the same density can be no better; the regression that matters is
// that the cap is no longer flat (volume ≈ 0).

// capBody builds the cap of a sphere kept on the side OPPOSITE planeNormal: the spherical cap face
// (rim = the cut circle) closed by a planar disk lid whose outward normal is +planeNormal.
func capBody(t *testing.T, center math.Point3, radius float64, planePoint math.Point3, planeNormal math.Vector3) *topo.Body {
	t.Helper()
	n, err := math.UnitVector3FromVector(planeNormal)
	if err != nil {
		t.Fatalf("degenerate plane normal %v", planeNormal)
	}
	sphere, err := geom.NewSphere(center, radius)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	d := float64(center.VectorTo(planePoint).Dot(n.AsVector())) // signed center→plane distance along n
	r0 := stdmath.Sqrt(radius*radius - d*d)
	circleCenter := center.TranslateBy(n.AsVector().Scale(math.Scalar(d)))
	circle, err := geom.NewCircle(circleCenter, n.AsVector(), r0)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	disk, _ := geom.NewPlane(planePoint, n.AsVector()) // outward = +n (material on −n side)
	lin := func(s string) topo.Lineage { return topo.NewLineage(topo.Tok("cap", s, 0)) }
	bld := topo.NewBuilder(true, lin("body"))
	v := bld.AddVertex(circle.PointAt(0), lin("v"))
	e := bld.AddEdge(circle, v, v, lin("e"))
	bld.AddFace(disk, lin("disk"), topo.OuterLoop(topo.Fwd(e)))
	bld.AddFace(sphere, lin("sph"), topo.OuterLoop(topo.Rev(e)))
	return bld.Build()
}

func TestSphereCapTessellatesToAnalyticVolume(t *testing.T) {
	const R = 5.0
	cases := []struct {
		name        string
		planePoint  math.Point3
		planeNormal math.Vector3
		d           float64 // signed center→plane distance along the (unit) normal
	}{
		{"hemisphere", math.P3(0, 0, 0), math.V3(0, 0, 1), 0},      // h = 5
		{"small cap", math.P3(0, 0, 3.5), math.V3(0, 0, 1), 3.5},   // kept −z side, h = 8.5
		{"large cap", math.P3(0, 0, -2.5), math.V3(0, 0, 1), -2.5}, // kept −z side, h = 2.5
		{"oblique axis", math.P3(0, 0, 0), math.V3(1, 1, 1), 0},    // hemisphere, tilted cut
		{"oblique offset", math.P3(2, 0, 0), math.V3(1, 0, 0), 2},  // h = 7 along +x
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := capBody(t, math.P3(0, 0, 0), R, tc.planePoint, tc.planeNormal)
			if r := Validate(body); !r.Valid || !r.Closed || !r.Manifold || !body.IsSolid() {
				t.Fatalf("cap body not a valid closed manifold solid: %+v", r)
			}
			got := BodyGeometryProperties(body, DefaultQuality()).Volume
			h := R + tc.d
			want := stdmath.Pi * h * h * (3*R - h) / 3
			if got < 0.5*want {
				t.Fatalf("cap meshed flat: volume %.4f is far below analytic %.4f (the #1334 bug)", got, want)
			}
			if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
				t.Errorf("cap volume %.4f, want %.4f (rel err %.4f > 2%%)", got, want, rel)
			}
		})
	}
}
