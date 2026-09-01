// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestUnmeshedWallWrapIsFlagged: if a full-wrap wall ever reaches the flat patch CDT again, the
// half-covered mesh it produces must ship as a Defect, not silently (#2038's third acceptance).
func TestUnmeshedWallWrapIsFlagged(t *testing.T) {
	t.Parallel()
	s := bandCylinder(10)
	wrapping := bridgedWallLoop(s, 0, 10, 96)
	m := &Mesh{}
	recordUnmeshedWallWrap(m, s, wrapping, 1)
	if len(m.Diagnostics) != 1 || m.Diagnostics[0].Code != CodeWallWrapUnmeshed {
		t.Fatalf("a full-wrap cylinder wall on the flat CDT recorded %v, want one %s defect",
			m.Diagnostics, CodeWallWrapUnmeshed)
	}
	if m.Diagnostics[0].Severity != diag.Defect {
		t.Errorf("severity %v, want Defect — a half-covered wall is not advisory", m.Diagnostics[0].Severity)
	}
}

// TestUnmeshedWallWrapIgnoresLegitimateFlatPatches: a sphere cap straddling its pole belongs on the
// flat CDT, and so does a wall patch that does not wrap, so neither may be flagged.
func TestUnmeshedWallWrapIgnoresLegitimateFlatPatches(t *testing.T) {
	t.Parallel()
	sph, err := geom.NewSphere(math.P3(0, 0, 0), 3)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	cyl := bandCylinder(10)
	for _, tc := range []struct {
		name string
		s    geom.Surface
		loop []math.Point3
	}{
		{"sphere cap over the pole", sph, bridgedWallLoop(sph, 0.4, 1.0, 48)},
		{"cylinder patch that does not wrap", cyl,
			[]math.Point3{cyl.PointAt(0, 0), cyl.PointAt(1, 0), cyl.PointAt(1, 10), cyl.PointAt(0, 10)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &Mesh{}
			recordUnmeshedWallWrap(m, tc.s, tc.loop, 0)
			if len(m.Diagnostics) != 0 {
				t.Errorf("flagged a legitimate flat patch: %v", m.Diagnostics)
			}
		})
	}
}
