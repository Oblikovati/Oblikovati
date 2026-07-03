// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func twoCapTarget(t *testing.T) *topo.Body {
	t.Helper()
	tg, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	return tg
}

// TestTwoCapAcceptsCapToCapTunnel: a steep (20 deg) tool entering one cap and exiting the other, wall
// untouched, builds a four-face result (whole wall + 2 holed caps + tunnel).
func TestTwoCapAcceptsCapToCapTunnel(t *testing.T) {
	th := 20.0 * stdmath.Pi / 180
	ux, uz := stdmath.Sin(th), stdmath.Cos(th)
	tool, err := SolidCylinder(math.P3(-2.416, 0, -2.518), math.V3(math.Scalar(ux), 0, math.Scalar(uz)), 0.7, 16)
	if err != nil {
		t.Fatalf("tool: %v", err)
	}
	res, ok := TwoCapCrossingCutGeneral(twoCapTarget(t), tool, &diag.Recorder{})
	if !ok {
		t.Fatal("two-cap cut declined a genuine cap-to-cap tool")
	}
	if n := len(res.Faces()); n != 4 {
		t.Errorf("two-cap result has %d faces; want 4", n)
	}
}

// TestTwoCapDeclinesSingleCapExit: the slice-1/2 tool (45 deg, base -6.5) enters the wall and exits ONE cap,
// so only one cap qualifies — the two-cap recognizer must decline (slice 1/2 handle it).
func TestTwoCapDeclinesSingleCapExit(t *testing.T) {
	s := 1 / stdmath.Sqrt2
	tool, err := SolidCylinder(math.P3(-6.5, 0, 2), math.V3(math.Scalar(s), 0, math.Scalar(s)), 0.9, 16)
	if err != nil {
		t.Fatalf("tool: %v", err)
	}
	if _, ok := TwoCapCrossingCutGeneral(twoCapTarget(t), tool, &diag.Recorder{}); ok {
		t.Error("two-cap cut accepted a single-cap-exit tool; want decline")
	}
}

// TestTwoCapDeclinesWallDrill: a transverse tool that drills through the WALL (no cap exit) must decline.
func TestTwoCapDeclinesWallDrill(t *testing.T) {
	tool, err := SolidCylinder(math.P3(-6, 0, 5), math.V3(1, 0, 0), 1, 12)
	if err != nil {
		t.Fatalf("tool: %v", err)
	}
	if _, ok := TwoCapCrossingCutGeneral(twoCapTarget(t), tool, &diag.Recorder{}); ok {
		t.Error("two-cap cut accepted a wall drill; want decline")
	}
}
