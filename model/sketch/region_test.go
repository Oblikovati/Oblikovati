// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

func TestEntityOutline(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pt := s.Points().Add(math.P2(1, 2))
	ln := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(3, 0))
	ci := s.Circles().AddByCenterRadius(math.P2(0, 0), 1)

	if got := EntityOutline(pt); len(got) != 1 || !got[0].IsEqualTo(math.P2(1, 2), 1e-9) {
		t.Errorf("point outline = %v, want just its position", got)
	}
	if got := EntityOutline(ln); len(got) < 2 {
		t.Errorf("line outline = %d points, want at least its 2 endpoints", len(got))
	}
	if got := EntityOutline(ci); len(got) < 8 {
		t.Errorf("circle outline = %d points, want a sampled polygon", len(got))
	}
}
