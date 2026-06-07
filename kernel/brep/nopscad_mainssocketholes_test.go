// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"oblikovati/kernel/ops"
	"oblikovati/math"
	"testing"
)

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
