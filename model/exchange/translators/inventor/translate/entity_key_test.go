// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// entityKey is the join between a REBUILT sketch curve and the curves an extrude's region NAMES
// (loopCurveKeys), so the two must describe the same curve identically or a region resolves to no
// profile. It had no test of its own while the containment rule (#26) short-circuited the curve-set
// rule it feeds, which is exactly when a silent drift would go unnoticed — these pin the contract.

// TestEntityKeyMatchesTheDecodedForm pins that a rebuilt curve keys the same way loopCurveKeys keys
// the decoded curve it came from, for every kind the region matcher understands.
func TestEntityKeyMatchesTheDecodedForm(t *testing.T) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a := s.Points().Add(math.P2(1, 2))
	b := s.Points().Add(math.P2(4, 6))
	line := s.Lines().Add(a, b)
	circle := s.Circles().AddByCenterRadius(math.P2(3, 3), 2.5)
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(2, 0), math.P2(0, 2), true)

	cases := []struct {
		name string
		e    sketch.Entity
		want string
	}{
		{"line keys by its endpoints", line, lineKey(1, 2, 4, 6)},
		{"circle keys by centre+radius", circle, circleKey(3, 3, 2.5)},
		// The arc keys as its WHOLE circle: the region names the circle the patch trims it from, so
		// an arc that keyed by its endpoints would never match its own region.
		{"arc keys as its full circle", arc, circleKey(0, 0, 2)},
	}
	for _, c := range cases {
		got, ok := entityKey(c.e)
		if !ok {
			t.Errorf("%s: entityKey declined a kind the matcher supports", c.name)
			continue
		}
		if got != c.want {
			t.Errorf("%s: entityKey = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestEntityKeyIsEndpointOrderIndependent pins that a line keys the same whichever end the rebuilt
// sketch calls the start — the decoded loop and the solved sketch need not agree on direction.
func TestEntityKeyIsEndpointOrderIndependent(t *testing.T) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	p, q := s.Points().Add(math.P2(1, 2)), s.Points().Add(math.P2(4, 6))
	fwd, _ := entityKey(s.Lines().Add(p, q))
	rev, _ := entityKey(s.Lines().Add(q, p))
	if fwd != rev {
		t.Errorf("reversed line keyed %q, want %q — the key must not depend on which end is the start", rev, fwd)
	}
}

// TestEntityKeyDeclinesUnsupportedKinds pins the honest outcome for a kind the region matcher has no
// key for: ok=false, which makes the profile unmatchable, rather than a key that could collide.
func TestEntityKeyDeclinesUnsupportedKinds(t *testing.T) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	if _, ok := entityKey(s.Points().Add(math.P2(1, 1))); ok {
		t.Error("entityKey accepted a Point — only line/circle/arc have a region key")
	}
	if _, ok := entityKey(s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 2, 1)); ok {
		t.Error("entityKey accepted an Ellipse — only line/circle/arc have a region key")
	}
}
