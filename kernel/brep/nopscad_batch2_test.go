// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

func TestNopJackCSG(t *testing.T) {
	body := prismBody(rectPoints(0.6, 0.7), 0, 0.6, "jack-body")
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.175, 48, 0), -0.05, 0.65, "jack-bore"), "jack bore")
	tube := annularPrism(t, 0.3, 0.175, 0.85, "jack-front-tube")
	body = joinOrFatal(t, body, tube, "jack tube")
	body = joinOrFatal(t, body, box(-0.3, -0.35, -0.3, 0.6, 0.7, 0.3), "jack rear block")

	requireValidNopSolid(t, "jack", body)
	if got := vol(body); got <= 0.6*0.7*0.6 || got >= 0.6*0.7*0.9+stdmath.Pi*0.3*0.3*0.85 {
		t.Errorf("jack volume = %.6f, outside expected source-construction range", got)
	}
}

func TestNopStandoffCSG(t *testing.T) {
	post := prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.25, 48, 0), 0, 1.2, "standoff-post")
	capsule, err := ops.ConvexHull(standoffSphereSamples(0.18, -0.06, 1.56), "standoff-capsule")
	if err != nil {
		t.Fatalf("ConvexHull(standoff capsule): %v", err)
	}
	body := joinOrFatal(t, post, capsule, "standoff capsule")

	requireValidNopSolid(t, "standoff", body)
	if got := vol(body); got <= vol(post) || got >= vol(post)+vol(capsule) {
		t.Errorf("standoff volume = %.6f, want between post %.6f and sum %.6f", got, vol(post), vol(post)+vol(capsule))
	}
}

func TestNopPiCutoutCSG(t *testing.T) {
	left := prismBody(regularPolygonPoints(math.P3(-0.35, 0, 0), 0.18, 32, 0), 0, 0.35, "pi-cutout-left")
	right := prismBody(regularPolygonPoints(math.P3(0.35, 0, 0), 0.18, 32, 0), 0, 0.35, "pi-cutout-right")
	hull, err := ops.ConvexHullOf("pi-cutout-base-hull", left, right)
	if err != nil {
		t.Fatalf("ConvexHullOf(pi holes): %v", err)
	}
	body := joinOrFatal(t, hull, box(-0.45, -0.55, 0, 0.9, 0.12, 0.9), "pi lower stem")
	body = joinOrFatal(t, body, box(-0.45, 0.43, 0, 0.9, 0.12, 0.9), "pi upper stem")

	requireValidNopSolid(t, "pi_cutout", body)
	if got := vol(body); got <= vol(hull) {
		t.Errorf("pi_cutout volume = %.6f, want larger than hole hull %.6f", got, vol(hull))
	}
}

func TestNopStarWasherCSG(t *testing.T) {
	outerR, innerR, thickness := 0.9, 0.31, 0.12
	body := annularPrism(t, outerR, innerR, thickness, "star-washer")
	inner := (innerR + outerR) / 2
	spoke := outerR - innerR
	for a := 0.0; a < 360; a += 30 {
		tool := rotatedBox(spoke, 2*stdmath.Pi*inner/36, thickness+0.1, inner+spoke/2, a*stdmath.Pi/180, "star-slot")
		body = cutOrFatal(t, body, tool, "star slot")
	}

	requireValidNopSolid(t, "star_washer", body)
	if got := vol(body); got <= 0 || got >= stdmath.Pi*(outerR*outerR-innerR*innerR)*thickness {
		t.Errorf("star_washer volume = %.6f, want below uncut annulus", got)
	}
}

func TestNopZiptieCSG(t *testing.T) {
	outer := stadiumBandPoints(0, 0, 1.0, 0.45, 24)
	inner := stadiumBandPoints(0, 0, 0.82, 0.27, 24)
	body := prismBody(outer, -0.18, 0.18, "ziptie-outer")
	body = cutOrFatal(t, body, prismBody(inner, -0.22, 0.22, "ziptie-inner"), "ziptie inner offset")
	strapVolume := vol(body)
	body = joinOrFatal(t, body, box(0.65, -0.16, -0.3, 0.35, 0.32, 0.6), "ziptie latch")

	requireValidNopSolid(t, "ziptie", body)
	if got := vol(body); got <= strapVolume {
		t.Errorf("ziptie volume = %.6f, want larger than strap band %.6f", got, strapVolume)
	}
}

func TestNopRibbonGrommetCSG(t *testing.T) {
	outer := ribbonGrommetProfile(2.75, 0.42, 0.16, 16)
	inner := []math.Point3{math.P3(-0.95, 0.08, 0), math.P3(0.95, 0.08, 0), math.P3(0.95, 0.24, 0), math.P3(-0.95, 0.24, 0)}
	body := prismBody(outer, -0.15, 0.15, "ribbon-grommet-side")
	body = cutOrFatal(t, body, prismBody(inner, -0.2, 0.2, "ribbon-grommet-slot"), "ribbon slot")

	requireValidNopSolid(t, "ribbon_grommet", body)
	want := (nopPolygonArea(outer) - nopPolygonArea(inner)) * 0.3
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("ribbon_grommet volume = %.6f, want %.6f", got, want)
	}
}

func TestNopRibbonGrommetHoleCSG(t *testing.T) {
	body := prismBody(ribbonGrommetProfile(2.72, 0.405, 0.15, 16), 0, 5.0, "ribbon-grommet-hole")
	requireValidNopSolid(t, "ribbon_grommet_hole", body)
	if got, want := vol(body), nopPolygonArea(ribbonGrommetProfile(2.72, 0.405, 0.15, 16))*5.0; stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("ribbon_grommet_hole volume = %.6f, want %.6f", got, want)
	}
}

func TestNopPolyRingCSG(t *testing.T) {
	body := annularPrism(t, 0.7, 0.35, 0.12, "poly-ring")
	requireValidNopSolid(t, "poly_ring", body)
	want := (nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.7, 64, 0)) - nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.35, 12, stdmath.Pi/12))) * 0.12
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("poly_ring volume = %.6f, want %.6f", got, want)
	}
}

func TestNopQuadrantCSG(t *testing.T) {
	body := prismBody(roundedCornerRectPoints(2.0, 1.4, 0.5, 20), 0, 0.2, "quadrant")
	requireValidNopSolid(t, "quadrant", body)
	if got := vol(body); got <= 0 || got >= 2.0*1.4*0.2 {
		t.Errorf("quadrant volume = %.6f, want below square blank", got)
	}
}

func TestNopRoundedCornerCSG(t *testing.T) {
	body := prismBody(roundedCornerRectPoints(1.6, 2.4, 0.35, 20), 0, 0.2, "rounded-corner")
	requireValidNopSolid(t, "rounded_corner", body)
	if got := vol(body); got <= 0 || got >= 1.6*2.4*0.2 {
		t.Errorf("rounded_corner volume = %.6f, want below square blank", got)
	}
}

func annularPrism(t *testing.T, outerR, innerR, height float64, feat string) *topo.Body {
	t.Helper()
	body := prismBody(regularPolygonPoints(math.P3(0, 0, 0), outerR, 64, 0), 0, height, feat+"-outer")
	inner := prismBody(regularPolygonPoints(math.P3(0, 0, 0), innerR, 12, stdmath.Pi/12), -0.05, height+0.05, feat+"-inner")
	return cutOrFatal(t, body, inner, feat+" inner")
}

func cutOrFatal(t *testing.T, body, tool *topo.Body, label string) *topo.Body {
	t.Helper()
	out, err := ops.Boolean(ops.Cut, body, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut %s): %v", label, err)
	}
	return out
}

func joinOrFatal(t *testing.T, body, tool *topo.Body, label string) *topo.Body {
	t.Helper()
	out, err := ops.Boolean(ops.Join, body, tool)
	if err != nil {
		t.Fatalf("Boolean(Join %s): %v", label, err)
	}
	return out
}

func rectPoints(width, height float64) []math.Point3 {
	return []math.Point3{math.P3(-width/2, -height/2, 0), math.P3(width/2, -height/2, 0), math.P3(width/2, height/2, 0), math.P3(-width/2, height/2, 0)}
}

func standoffSphereSamples(radius, z0, z1 float64) []math.Point3 {
	var points []math.Point3
	for _, z := range []float64{z0, z1} {
		points = append(points, regularPolygonPoints(math.P3(0, 0, z), radius, 16, 0)...)
		points = append(points, math.P3(0, 0, z-radius), math.P3(0, 0, z+radius))
	}
	return points
}

func rotatedBox(width, height, depth, cx, angle float64, feat string) *topo.Body {
	ca, sa := stdmath.Cos(angle), stdmath.Sin(angle)
	local := rectPoints(width, height)
	points := make([]math.Point3, len(local))
	for i, p := range local {
		x := p.X + cx
		points[i] = math.P3(x*ca-p.Y*sa, x*sa+p.Y*ca, 0)
	}
	return prismBody(points, -depth/2, depth/2, feat)
}

func stadiumBandPoints(cx, cy, halfLength, halfWidth float64, steps int) []math.Point3 {
	points := make([]math.Point3, 0, 2*steps+2)
	for i := 0; i <= steps; i++ {
		a := stdmath.Pi/2 - stdmath.Pi*float64(i)/float64(steps)
		points = append(points, math.P3(cx+halfLength+halfWidth*stdmath.Cos(a), cy+halfWidth*stdmath.Sin(a), 0))
	}
	for i := 0; i <= steps; i++ {
		a := -stdmath.Pi/2 - stdmath.Pi*float64(i)/float64(steps)
		points = append(points, math.P3(cx-halfLength+halfWidth*stdmath.Cos(a), cy+halfWidth*stdmath.Sin(a), 0))
	}
	return points
}

func ribbonGrommetProfile(length, height, radius float64, steps int) []math.Point3 {
	points := []math.Point3{math.P3(-length/2, 0, 0), math.P3(length/2, 0, 0), math.P3(length/2, height-radius, 0)}
	for i := 0; i <= steps; i++ {
		a := stdmath.Pi * float64(i) / float64(steps)
		points = append(points, math.P3(length/2-radius+radius*stdmath.Cos(a), height-radius+radius*stdmath.Sin(a), 0))
	}
	points = append(points, math.P3(-length/2, height-radius, 0))
	return points
}

func roundedCornerRectPoints(width, height, radius float64, steps int) []math.Point3 {
	points := []math.Point3{math.P3(0, 0, 0), math.P3(width, 0, 0), math.P3(width, height-radius, 0)}
	for i := 0; i <= steps; i++ {
		a := stdmath.Pi / 2 * float64(i) / float64(steps)
		points = append(points, math.P3(width-radius+radius*stdmath.Cos(a), height-radius+radius*stdmath.Sin(a), 0))
	}
	points = append(points, math.P3(0, height, 0))
	return points
}
