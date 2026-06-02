// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

func TestEllipseFull2dAxesHonored(t *testing.T) {
	e, err := NewEllipseFull2d(math.P2(0, 0), math.V2(1, 0), 2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Angle 0 → +major (length 2); angle π/2 (t=0.25) → +minor (length 1).
	if got := e.PointAt(0); !got.IsEqualTo(math.P2(2, 0), eqScalar) {
		t.Errorf("major end = %v, want {2 0}", got)
	}
	if got := e.PointAt(0.25); !got.IsEqualTo(math.P2(0, 1), eqScalar) {
		t.Errorf("minor end = %v, want {0 1}", got)
	}
}

func TestEllipseFull2dRatioWithRotatedAxis(t *testing.T) {
	// Major axis along +Y this time, so the minor (ratio 0.5) lands on −X.
	e, _ := NewEllipseFull2d(math.P2(0, 0), math.V2(0, 1), 4, 2)
	if got := e.PointAt(0); !got.IsEqualTo(math.P2(0, 4), eqScalar) {
		t.Errorf("major end = %v, want {0 4}", got)
	}
	if got := e.PointAt(0.25); !got.IsEqualTo(math.P2(-2, 0), eqScalar) {
		t.Errorf("minor end = %v, want {-2 0}", got)
	}
}

func TestEllipticalArc2dSweep(t *testing.T) {
	// Quarter of an ellipse, major +X (r=2), sweeping to the minor axis (r=1).
	e, _ := NewEllipticalArc2d(math.P2(0, 0), math.V2(1, 0), 2, 1, 0, stdmath.Pi/2)
	if got := e.PointAt(1); !got.IsEqualTo(math.P2(0, 1), 1e-9) {
		t.Errorf("arc end = %v, want {0 1}", got)
	}
}

func TestEllipseFull3dAxesHonored(t *testing.T) {
	e, err := NewEllipseFull(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := e.PointAt(0); !got.IsEqualTo(math.P3(2, 0, 0), eqScalar) {
		t.Errorf("major end = %v, want {2 0 0}", got)
	}
	// minor axis = normal × major = +Y, length 1.
	if got := e.PointAt(0.25); !got.IsEqualTo(math.P3(0, 1, 0), eqScalar) {
		t.Errorf("minor end = %v, want {0 1 0}", got)
	}
}

func TestNewEllipseFull2dZeroAxisFails(t *testing.T) {
	if _, err := NewEllipseFull2d(math.P2(0, 0), math.V2(0, 0), 2, 1); err == nil {
		t.Fatal("expected error for zero major axis")
	}
}
