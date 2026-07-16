// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/sketch"
)

// TestRegionNeverSelectsAnOpenProfile pins that an OPEN chain is never selected as part of an
// extrude's region.
//
// A sketch reports its connected-but-unclosed chains as profiles alongside its real regions. Such a
// chain bounds no area, so it is never material — and matching one is not merely useless, it is
// destructive: the feature layer fails the WHOLE extrude when any selected profile is open
// (resolveClosedProfiles), so the feature goes sick and contributes no body at all. ReadWriteHead's
// second extrude died exactly that way.
func TestRegionNeverSelectsAnOpenProfile(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	// a closed square...
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(4, 0))
	c := sk.Points().Add(math.P2(4, 4))
	d := sk.Points().Add(math.P2(0, 4))
	sk.Lines().Add(a, b)
	sk.Lines().Add(b, c)
	sk.Lines().Add(c, d)
	sk.Lines().Add(d, a)
	// ...and a detached open chain well outside it.
	e := sk.Points().Add(math.P2(10, 0))
	f := sk.Points().Add(math.P2(12, 0))
	g := sk.Points().Add(math.P2(12, 2))
	sk.Lines().Add(e, f)
	sk.Lines().Add(f, g)

	if !hasOpenProfile(sk) {
		t.Skip("sketch reported no open profile; the guard has nothing to bite on here")
	}
	// a region naming EVERY curve, the square's and the open chain's alike
	region := []ipt.RegionLoop{{Edges: []ipt.RegionEdge{
		lineEdge(0, 0, 4, 0), lineEdge(4, 0, 4, 4), lineEdge(4, 4, 0, 4), lineEdge(0, 4, 0, 0),
		lineEdge(10, 0, 12, 0), lineEdge(12, 0, 12, 2),
	}}}
	for _, i := range regionProfileIndices(sk, region) {
		if !sk.Profiles().Item(i).IsClosed() {
			t.Errorf("selected profile %d is OPEN; an open chain bounds no material and sickens the whole extrude", i)
		}
	}
}

func hasOpenProfile(sk *sketch.Sketch) bool {
	ps := sk.Profiles()
	for i := 0; i < ps.Count(); i++ {
		if !ps.Item(i).IsClosed() {
			return true
		}
	}
	return false
}

func lineEdge(ax, ay, bx, by float64) ipt.RegionEdge {
	return ipt.RegionEdge{Kind: ipt.EdgeLine, Line: ipt.Line{
		A: ipt.Point2D{X: ax, Y: ay}, B: ipt.Point2D{X: bx, Y: by}}}
}
