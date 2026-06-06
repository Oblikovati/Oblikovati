// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

func TestNopDoorLatchStlCSG(t *testing.T) {
	body := prismBody(roundedRectPoints(3.5, 1.2, 0.3, 8), 0, 0.5, "door-latch-rounded-base")
	body = joinOrFatal(t, body, box(-1.75, -0.2, 0.25, 3.5, 0.4, 0.35), "door-latch-ridge")
	body = joinOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.6, 48, 0), 0, 1.425, "door-latch-boss"), "door-latch-boss")
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.22, 32, 0), -0.05, 1.5, "door-latch-screw"), "door latch screw clearance")
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.42, 6, stdmath.Pi/6), 0.7, 1.5, "door-latch-nut-trap"), "door latch nut trap")

	requireValidNopSolid(t, "door_latch_stl", body)
	if got := vol(body); got <= 3.5*1.2*0.5 {
		t.Errorf("door_latch_stl volume = %.6f, want larger than latch plate", got)
	}
}

func TestNopLedBezelRetainerCSG(t *testing.T) {
	body := annularPrism(t, 0.45, 0.32, 0.4, "led-bezel-retainer")
	requireValidNopSolid(t, "led_bezel_retainer", body)
	want := (nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.45, 64, 0)) - nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.32, 12, stdmath.Pi/12))) * 0.4
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("led_bezel_retainer volume = %.6f, want %.6f", got, want)
	}
}

func TestNopTearslotCSG(t *testing.T) {
	body := tearSlotBody(t, 0.35, 0.8, 0.5, false, "tearslot")
	requireValidNopSolid(t, "tearslot", body)
	if got := vol(body); got <= 0 || got >= (0.8+0.7)*1.0*0.5 {
		t.Errorf("tearslot volume = %.6f, outside expected hulled teardrop range", got)
	}
}

func TestNopTearslot2DCSG(t *testing.T) {
	body := tearSlotBody(t, 0.35, 0.8, 0.08, false, "tearslot-2d")
	requireValidNopSolid(t, "tearslot_2d", body)
	if got := vol(body); got <= 0 {
		t.Errorf("tearslot_2d volume = %.6f, want positive thin solid", got)
	}
}

func TestNopVerticalTearslotCSG(t *testing.T) {
	body := tearSlotBody(t, 0.35, 0.8, 0.5, true, "vertical-tearslot")
	requireValidNopSolid(t, "vertical_tearslot", body)
	if got := vol(body); got <= 0 || got >= 1.0*(0.8+0.7)*0.5 {
		t.Errorf("vertical_tearslot volume = %.6f, outside expected hulled teardrop range", got)
	}
}

func TestNopVerticalTearslot2DCSG(t *testing.T) {
	body := tearSlotBody(t, 0.35, 0.8, 0.08, true, "vertical-tearslot-2d")
	requireValidNopSolid(t, "vertical_tearslot_2d", body)
	if got := vol(body); got <= 0 {
		t.Errorf("vertical_tearslot_2d volume = %.6f, want positive thin solid", got)
	}
}

func TestNopDimensionCSG(t *testing.T) {
	body := box(-0.75, -0.015, -0.015, 1.5, 0.03, 0.03)
	body = joinOrFatal(t, body, prismBody([]math.Point3{math.P3(-0.7, -0.08, 0), math.P3(-0.7, 0.08, 0), math.P3(-0.95, 0, 0)}, -0.015, 0.015, "dimension-left-arrow"), "dimension left arrow")
	body = joinOrFatal(t, body, prismBody([]math.Point3{math.P3(0.7, -0.08, 0), math.P3(0.7, 0.08, 0), math.P3(0.95, 0, 0)}, -0.015, 0.015, "dimension-right-arrow"), "dimension right arrow")

	requireValidNopSolid(t, "dimension", body)
	if got := vol(body); got <= 0 {
		t.Errorf("dimension volume = %.6f, want positive line plus arrowheads", got)
	}
}

func TestNopWireLinkCSG(t *testing.T) {
	body := cylinderZAt(-0.6, 0, -0.3, 0.9, 0.06, "wire-left-leg")
	body = joinOrFatal(t, body, cylinderZAt(0.6, 0, -0.3, 0.9, 0.06, "wire-right-leg"), "wire right leg")
	body = joinOrFatal(t, body, cylinderAlongX(0.06, -0.6, 0.6, 0.9, "wire-top-link"), "wire top link")

	requireValidNopSolid(t, "wire_link", body)
	if got := vol(body); got <= 2*stdmath.Pi*0.06*0.06*1.2 {
		t.Errorf("wire_link volume = %.6f, want legs plus top link", got)
	}
}

func TestNopE3dFanDuctCSG(t *testing.T) {
	body := e3dFanDuctBody(t)
	requireValidNopSolid(t, "e3d_fan_duct", body)
	if got := vol(body); got <= 0 {
		t.Errorf("e3d_fan_duct volume = %.6f, want positive duct", got)
	}
}

func TestNopE3dFanCSG(t *testing.T) {
	body := e3dFanDuctBody(t)
	body = joinOrFatal(t, body, box(1.5, -1.5, 0.2, 1.0, 3.0, 0.3), "e3d fan frame")
	body = cutOrFatal(t, body, cylinderZAt(2.0, 0, 0.15, 0.55, 1.1, "e3d fan aperture"), "e3d fan aperture")

	requireValidNopSolid(t, "e3d_fan", body)
	if got := vol(body); got <= 0 {
		t.Errorf("e3d_fan volume = %.6f, want positive duct plus fan assembly", got)
	}
}

func tearSlotBody(t *testing.T, radius, span, depth float64, vertical bool, feat string) *topo.Body {
	t.Helper()
	var a, b *topo.Body
	if vertical {
		a = prismBody(teardropPoints(0, -span/2, radius, true), -depth/2, depth/2, feat+"-a")
		b = prismBody(teardropPoints(0, span/2, radius, true), -depth/2, depth/2, feat+"-b")
	} else {
		a = prismBody(teardropPoints(-span/2, 0, radius, false), -depth/2, depth/2, feat+"-a")
		b = prismBody(teardropPoints(span/2, 0, radius, false), -depth/2, depth/2, feat+"-b")
	}
	body, err := ops.ConvexHullOf(feat+"-hull", a, b)
	if err != nil {
		t.Fatalf("ConvexHullOf(%s): %v", feat, err)
	}
	return body
}

func teardropPoints(cx, cy, r float64, vertical bool) []math.Point3 {
	points := make([]math.Point3, 0, 28)
	for i := 0; i <= 20; i++ {
		a := -5*stdmath.Pi/4 + 3*stdmath.Pi/2*float64(i)/20
		x, y := r*stdmath.Cos(a), r*stdmath.Sin(a)
		if vertical {
			x, y = -y, x
		}
		points = append(points, math.P3(cx+x, cy+y, 0))
	}
	tipX, tipY := 0.0, 1.35*r
	if vertical {
		tipX, tipY = -tipY, tipX
	}
	points = append(points, math.P3(cx+tipX, cy+tipY, 0))
	return points
}

func cylinderZAt(cx, cy, z0, z1, radius float64, feat string) *topo.Body {
	return prismBody(regularPolygonPoints(math.P3(cx, cy, 0), radius, 32, 0), z0, z1, feat)
}

func e3dFanDuctBody(t *testing.T) *topo.Body {
	t.Helper()
	left := box(-0.8, -1.15, 0, 0.05, 2.3, 2.6)
	right := box(1.5, -1.5, 0, 0.05, 3.0, 3.0)
	body, err := ops.ConvexHullOf("e3d-fan-duct-hull", left, right)
	if err != nil {
		t.Fatalf("ConvexHullOf(e3d fan duct): %v", err)
	}
	body = cutOrFatal(t, body, cylinderZAt(0, 0, -0.1, 3.1, 0.55, "e3d-radial-clearance"), "e3d radial clearance")
	body = cutOrFatal(t, body, cylinderAlongX(0.55, -1.0, 2.0, 1.5, "e3d-cross-duct"), "e3d cross duct")
	return body
}
