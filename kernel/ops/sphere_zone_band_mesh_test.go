// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Spherical ZONE tessellation (Oblikovati#2061). The belt between two coaxial circles on a sphere had no
// mesher of its own: sphereCapFan wants ONE planar rim, sphereZoneCapFan one rim plus an enclosed pole,
// sphereSeamedCapFan a seam ending at a pole. The face fell to spherePatchMesh's gnomonic chart, which
// covers less than a hemisphere — fine for a belt on one side of its own equator, hopeless for one that
// straddles it. The numbers below are the measured "before": 0.617 through a seamed loop and 2.809
// through outer+inner loops (the inner rim ignored outright) against a true 2.513.

// zoneTestSphere is the reference belt: a R=0.5 ball with rims at y=±0.4, so the belt straddles the
// equator of its own band axis — the case that used to have no mesh.
const (
	zoneTestR    = 0.5
	zoneTestHalf = 0.4
)

func zoneTestRims(t *testing.T) (sph geom.Sphere, hi, lo geom.Circle) {
	t.Helper()
	sph, err := geom.NewSphere(math.P3(0, 0, 0), zoneTestR)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	r := stdmath.Sqrt(zoneTestR*zoneTestR - zoneTestHalf*zoneTestHalf)
	hi, err = geom.NewCircle(math.P3(0, zoneTestHalf, 0), math.V3(0, 1, 0), r)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	lo = geom.Circle{Center: math.P3(0, -zoneTestHalf, 0), Normal: hi.Normal, RefDir: hi.RefDir, Radius: r}
	return sph, hi, lo
}

// zoneTestArea is Archimedes' 2πRh for the belt between the two rim planes.
func zoneTestArea() float64 { return 2 * stdmath.Pi * zoneTestR * (2 * zoneTestHalf) }

// TestSphereZoneMeshesFromEitherLoopShape: a belt reaches the tessellator two ways — as two separate
// loops (what the coaxial boolean builds) and as one loop with a doubled seam edge bridging the rims
// (what revolution.go builds) — and BOTH must mesh to the same true area. The mesher reads the rims off
// the face's edges rather than off the loop structure precisely so the two agree.
func TestSphereZoneMeshesFromEitherLoopShape(t *testing.T) {
	sph, hi, lo := zoneTestRims(t)
	for _, c := range []struct {
		name string
		body func() *topo.Body
	}{
		{"two loops", func() *topo.Body { return twoLoopZone(t, sph, hi, lo) }},
		{"one seamed loop", func() *topo.Body { return seamedZone(t, sph, hi, lo) }},
	} {
		area := BodyGeometryProperties(c.body(), DefaultQuality()).Area
		want := zoneTestArea()
		if rel := (area - want) / want; rel < -0.02 || rel > 0 {
			t.Errorf("%s: belt area %.6f, want %.6f (%.2f%% off; an inscribed mesh may run under, never over)",
				c.name, area, want, 100*rel)
		}
	}
}

// TestSphereZoneAreaConvergesWithQuality: the old failure was FLAT in quality — the gnomonic chart lost
// the same three quarters of the belt however fine the facets got. An exact mesh must converge.
func TestSphereZoneAreaConvergesWithQuality(t *testing.T) {
	sph, hi, lo := zoneTestRims(t)
	body := twoLoopZone(t, sph, hi, lo)
	coarse := BodyGeometryProperties(body, DefaultQuality()).Area
	fine := BodyGeometryProperties(body, PropertyQuality()).Area
	if fine <= coarse {
		t.Fatalf("refining did not raise the belt area (%.6f → %.6f)", coarse, fine)
	}
	if rel := stdmath.Abs(fine-zoneTestArea()) / zoneTestArea(); rel > 1e-3 {
		t.Errorf("at PropertyQuality the belt area is %.6f, want %.6f (off by %.3f%%)",
			fine, zoneTestArea(), 100*rel)
	}
}

// TestSphereCapFanDeclinesAHoledFace guards the gate that lets the belt reach its own mesher at all. A
// cap fan sweeps its rim straight to the enclosed pole, so it would pave right over any hole; before
// this it claimed a two-loop belt and returned the outer cap, silently 12% over the true area.
func TestSphereCapFanDeclinesAHoledFace(t *testing.T) {
	sph, hi, lo := zoneTestRims(t)
	outer := circleRing(hi, 64)
	if _, ok := sphereCapFan(sph, outer, nil, DefaultQuality()); !ok {
		t.Fatal("the cap fan declined a plain single-rim cap")
	}
	if _, ok := sphereCapFan(sph, outer, [][]math.Point3{circleRing(lo, 64)}, DefaultQuality()); ok {
		t.Error("the cap fan claimed a HOLED face; it would pave over the hole")
	}
}

// TestSphereZoneBandFanDeclinesNonCoaxialRims: two rims that are not coaxial bound a lune, not a belt,
// and the latitude sweep would be meaningless there — it must decline so the gnomonic CDT takes over.
func TestSphereZoneBandFanDeclinesNonCoaxialRims(t *testing.T) {
	sph, hi, _ := zoneTestRims(t)
	tilted, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(1, 0, 0), zoneTestR)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	body := twoLoopZone(t, sph, hi, tilted)
	if _, ok := sphereZoneBandFan(body.Faces()[0], sph, DefaultQuality()); ok {
		t.Error("the belt fan claimed two rims whose planes are not parallel")
	}
}

// twoLoopZone builds the belt as an outer loop plus a hole — the shape the coaxial boolean assembles.
func twoLoopZone(t *testing.T, sph geom.Sphere, outer, inner geom.Circle) *topo.Body {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("zonetest", "x", 0))
	bld := topo.NewBuilder(true, lin)
	vo := bld.AddVertex(outer.PointAt(0), lin)
	vi := bld.AddVertex(inner.PointAt(0), lin)
	eo := bld.AddEdge(outer, vo, vo, lin)
	ei := bld.AddEdge(inner, vi, vi, lin)
	bld.AddFace(sph, lin, topo.OuterLoop(topo.Fwd(eo)), topo.InnerLoop(topo.Rev(ei)))
	return bld.Build()
}

// seamedZone builds the belt as ONE loop, the two rims bridged by a meridian seam walked twice — the
// shape revolution.go's addRevolutionSphereZone assembles.
func seamedZone(t *testing.T, sph geom.Sphere, hi, lo geom.Circle) *topo.Body {
	t.Helper()
	pHi, pLo := hi.PointAt(0), lo.PointAt(0)
	mid := sph.Center.TranslateBy(meridianPerp(sph, pHi, hi.Normal.AsVector()).Scale(math.Scalar(sph.Radius)))
	arc, err := geom.Arc3dByThreePoints(pHi, mid, pLo)
	if err != nil {
		t.Fatalf("Arc3dByThreePoints: %v", err)
	}
	lin := topo.NewLineage(topo.Tok("zonetest", "x", 0))
	bld := topo.NewBuilder(true, lin)
	vHi := bld.AddVertex(pHi, lin)
	vLo := bld.AddVertex(pLo, lin)
	eHi := bld.AddEdge(hi, vHi, vHi, lin)
	eLo := bld.AddEdge(lo, vLo, vLo, lin)
	seam := bld.AddEdge(arc, vHi, vLo, lin)
	bld.AddFace(sph, lin, topo.OuterLoop(topo.Fwd(eHi), topo.Fwd(seam), topo.Rev(eLo), topo.Rev(seam)))
	return bld.Build()
}

// circleRing samples a circle at n stations — a stand-in for an edge's own discretization.
func circleRing(c geom.Circle, n int) []math.Point3 {
	out := make([]math.Point3, n)
	for i := range out {
		out[i] = c.PointAt(float64(i) / float64(n))
	}
	return out
}
