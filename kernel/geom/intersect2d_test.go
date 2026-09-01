// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

func TestSegment2dIntersection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		a, b   LineSegment2d
		wantOK bool
		wantPt math.Point2
		wantS  float64
		wantT  float64
	}{
		{
			name:   "cross at center",
			a:      NewLineSegment2d(math.P2(-1, 0), math.P2(1, 0)),
			b:      NewLineSegment2d(math.P2(0, -1), math.P2(0, 1)),
			wantOK: true, wantPt: math.P2(0, 0), wantS: 0.5, wantT: 0.5,
		},
		{
			name:   "T-junction: b's start lands on a's interior",
			a:      NewLineSegment2d(math.P2(0, 0), math.P2(10, 0)),
			b:      NewLineSegment2d(math.P2(4, 0), math.P2(4, 5)),
			wantOK: true, wantPt: math.P2(4, 0), wantS: 0.4, wantT: 0,
		},
		{
			name:   "parallel: no crossing",
			a:      NewLineSegment2d(math.P2(0, 0), math.P2(10, 0)),
			b:      NewLineSegment2d(math.P2(0, 1), math.P2(10, 1)),
			wantOK: false,
		},
		{
			name:   "disjoint: lines cross but segments do not reach",
			a:      NewLineSegment2d(math.P2(0, 0), math.P2(1, 0)),
			b:      NewLineSegment2d(math.P2(5, -1), math.P2(5, 1)),
			wantOK: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pt, s, tt, ok := Segment2dIntersection(c.a, c.b, 0)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !ok {
				return
			}
			if !pt.IsEqualTo(c.wantPt, 1e-9) {
				t.Errorf("pt = %v, want %v", pt, c.wantPt)
			}
			if !math.IsNearZero(s-c.wantS, 1e-9) || !math.IsNearZero(tt-c.wantT, 1e-9) {
				t.Errorf("s,t = %v,%v want %v,%v", s, tt, c.wantS, c.wantT)
			}
		})
	}
}
