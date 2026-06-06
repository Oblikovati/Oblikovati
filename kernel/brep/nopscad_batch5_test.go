// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

func TestNopBeltbCSG(t *testing.T) {
	body := annularPrismRange(t, 0.8, 0.68, 0, 0.6, "beltb-pulley-arc")
	body = joinOrFatal(t, body, box(-0.8, -0.06, 0, 1.8, 0.12, 0.6), "beltb straight run")

	requireValidNopSolid(t, "beltb", body)
	if got := vol(body); got <= 0.12*0.6*1.8 {
		t.Errorf("beltb volume = %.6f, want straight belt plus pulley arc", got)
	}
}

func TestNopExtrusionCenterSectionCSG(t *testing.T) {
	body := box(-0.1, -1.0, 0, 0.2, 2.0, 0.12)
	body = joinOrFatal(t, body, box(-1.0, -0.1, 0, 2.0, 0.2, 0.12), "extrusion center cross spar")
	for _, side := range []float64{-1, 1} {
		body = joinOrFatal(t, body, box(side*0.72-0.09, -0.55, 0, 0.18, 1.1, 0.12), "extrusion side tab")
		body = joinOrFatal(t, body, box(-0.55, side*0.72-0.09, 0, 1.1, 0.18, 0.12), "extrusion end tab")
	}
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.22, 32, 0), -0.05, 0.2, "extrusion center hole"), "extrusion center hole")

	requireValidNopSolid(t, "extrusion_center_section", body)
	if got := vol(body); got <= 0 {
		t.Errorf("extrusion_center_section volume = %.6f, want positive spar section", got)
	}
}

func TestNopMainsSocketHolesCSG(t *testing.T) {
	body := box(-1.8, -1.2, 0, 3.6, 2.4, 0.12)
	for _, x := range []float64{-1.25, 1.25} {
		body = cutOrFatal(t, body, cylinderZAt(x, 0, -0.05, 0.2, 0.16, "mains-socket-screw"), "mains socket screw")
	}
	left := prismBody(regularPolygonPoints(math.P3(-0.45, 0, 0), 0.45, 32, 0), -0.05, 0.2, "mains-socket-left")
	right := prismBody(regularPolygonPoints(math.P3(0.45, 0, 0), 0.45, 32, 0), -0.05, 0.2, "mains-socket-right")
	aperture, err := ops.ConvexHullOf("mains-socket-aperture", left, right)
	if err != nil {
		t.Fatalf("ConvexHullOf(mains socket aperture): %v", err)
	}
	body = cutOrFatal(t, body, aperture, "mains socket aperture")
	body = cutOrFatal(t, body, cylinderZAt(-1.25, -0.75, -0.05, 0.2, 0.22, "mains-socket-earth"), "mains socket earth")

	requireValidNopSolid(t, "mains_socket_holes", body)
	if got := vol(body); got >= 3.6*2.4*0.12 {
		t.Errorf("mains_socket_holes volume = %.6f, want panel with cutouts", got)
	}
}

func TestNopAdjustCSG(t *testing.T) {
	body := cylinderZAt(0, 0, -0.089, 0.089, 0.1385, "adjust-dial")
	body = cutOrFatal(t, body, box(-0.16, -0.032, 0, 0.32, 0.064, 0.11), "adjust slot x")
	body = cutOrFatal(t, body, box(-0.032, -0.16, 0, 0.064, 0.32, 0.11), "adjust slot y")

	requireValidNopSolid(t, "adjust", body)
	if got := vol(body); got <= 0 || got >= stdmath.Pi*0.1385*0.1385*0.178 {
		t.Errorf("adjust volume = %.6f, want slotted trimpot dial", got)
	}
}

func TestNopTrimpot3362CSG(t *testing.T) {
	body := box(-0.3495, -0.33, 0.019, 0.699, 0.66, 0.45)
	for _, p := range []math.Point3{math.P3(-0.26, -0.22, -0.019), math.P3(0.26, -0.22, -0.019), math.P3(0, 0.22, -0.019)} {
		body = joinOrFatal(t, body, box(p.X-0.019, p.Y-0.019, p.Z, 0.038, 0.038, 0.038), "trimpot foot")
	}
	body = cutOrFatal(t, body, cylinderZAt(0, 0, 0.36, 0.52, 0.1385, "trimpot adjust recess"), "trimpot adjust recess")
	body = cutOrFatal(t, body, box(-0.16, -0.03, 0.39, 0.32, 0.06, 0.16), "trimpot screw slot")

	requireValidNopSolid(t, "trimpot3362", body)
	if got := vol(body); got <= 0.699*0.66*0.3 {
		t.Errorf("trimpot3362 volume = %.6f, want body plus feet", got)
	}
}

func TestNopRadialProfileCSG(t *testing.T) {
	points := []math.Point3{math.P3(0.16, 0, 0), math.P3(0.5, 0, 0), math.P3(0.5, 1.0, 0), math.P3(0.42, 1.16, 0), math.P3(0.22, 1.16, 0), math.P3(0.16, 1.0, 0)}
	body := prismBody(points, -0.03, 0.03, "radial-profile")

	requireValidNopSolid(t, "profile", body)
	if got := vol(body); got <= 0 {
		t.Errorf("profile volume = %.6f, want positive radial half-profile", got)
	}
}

func TestNopRdElectrolyticCSG(t *testing.T) {
	body := frustumBody(0.48, 0.44, 0, 1.15, 48, "rd-electrolytic-can")
	body = joinOrFatal(t, body, annularPrismRange(t, 0.5, 0.16, 0.02, 1.1, "rd-electrolytic-jacket"), "rd electrolytic jacket")
	for _, x := range []float64{-0.125, 0.125} {
		body = joinOrFatal(t, body, cylinderZAt(x, 0, -0.3, 0.02, 0.025, "rd-electrolytic-lead"), "rd electrolytic lead")
	}
	body = joinOrFatal(t, body, box(0.18, -0.02, 1.08, 0.22, 0.04, 0.04), "rd electrolytic crimp")

	requireValidNopSolid(t, "rd_electrolytic", body)
	if got := vol(body); got <= stdmath.Pi*0.44*0.44*1.0 {
		t.Errorf("rd_electrolytic volume = %.6f, want can plus leads", got)
	}
}

func TestNopSmdDiodeCSG(t *testing.T) {
	body := taperedBoxBody(0.46, 0.28, 0.42, 0.24, 0.02, 0.18, "smd-diode-body")
	for _, x := range []float64{-0.26, 0.26} {
		body = joinOrFatal(t, body, box(x-0.09, -0.12, 0.02, 0.18, 0.24, 0.04), "smd diode lead")
	}
	body = cutOrFatal(t, body, box(-0.11, -0.14, -0.01, 0.22, 0.28, 0.08), "smd diode lead gap")

	requireValidNopSolid(t, "smd_diode", body)
	if got := vol(body); got <= 0 {
		t.Errorf("smd_diode volume = %.6f, want positive package", got)
	}
}

func TestNopSmdTantCSG(t *testing.T) {
	body := taperedBoxBody(0.72, 0.42, 0.64, 0.34, 0.02, 0.26, "smd-tant-body")
	for _, x := range []float64{-0.41, 0.41} {
		body = joinOrFatal(t, body, box(x-0.12, -0.17, 0.02, 0.24, 0.34, 0.05), "smd tant lead")
	}
	body = cutOrFatal(t, body, box(-0.17, -0.21, -0.01, 0.34, 0.42, 0.1), "smd tant lead gap")
	body = joinOrFatal(t, body, box(-0.31, -0.15, 0.26, 0.08, 0.3, 0.01), "smd tant stripe")

	requireValidNopSolid(t, "smd_tant", body)
	if got := vol(body); got <= 0 {
		t.Errorf("smd_tant volume = %.6f, want positive tantalum package", got)
	}
}

func TestNopSingleCableClipCSG(t *testing.T) {
	base := prismBody(roundedRectPoints(1.6, 0.18, 0.08, 4), 0, 0.5, "cable-clip-foot")
	post := prismBody(roundedRectPoints(0.4, 0.9, 0.08, 4), 0, 0.5, "cable-clip-post")
	top := prismBody(regularPolygonPoints(math.P3(-0.55, 0.62, 0), 0.22, 24, 0), 0, 0.5, "cable-clip-loop")
	body, err := ops.ConvexHullOf("single-cable-clip-hull", base, post, top)
	if err != nil {
		t.Fatalf("ConvexHullOf(single cable clip): %v", err)
	}
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(-0.45, 0.45, 0), 0.18, 24, 0), -0.05, 0.55, "cable-channel"), "single cable channel")
	body = cutOrFatal(t, body, cylinderZAt(0.45, 0.45, -0.05, 0.55, 0.15, "cable-clip-screw"), "single cable clip screw")

	requireValidNopSolid(t, "single_cable_clip", body)
	if got := vol(body); got <= 0 {
		t.Errorf("single_cable_clip volume = %.6f, want positive clip", got)
	}
}

func taperedBoxBody(w0, d0, w1, d1, z0, z1 float64, feat string) *topo.Body {
	verts := []math.Point3{
		math.P3(-w0/2, -d0/2, z0), math.P3(w0/2, -d0/2, z0), math.P3(w0/2, d0/2, z0), math.P3(-w0/2, d0/2, z0),
		math.P3(-w1/2, -d1/2, z1), math.P3(w1/2, -d1/2, z1), math.P3(w1/2, d1/2, z1), math.P3(-w1/2, d1/2, z1),
	}
	faces := [][]int{{3, 2, 1, 0}, {4, 5, 6, 7}, {0, 1, 5, 4}, {1, 2, 6, 5}, {2, 3, 7, 6}, {3, 0, 4, 7}}
	return subdBody(verts, faces, feat)
}
