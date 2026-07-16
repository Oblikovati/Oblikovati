// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"
	"sort"

	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/sketch"
)

// A sketch usually holds several closed regions; the extrude names the curves bounding the one it
// uses (see ipt.ExtrudeRegions). This matches that named curve set against the regions our kernel
// computed from the same sketch, so the feature consumes the region Inventor authored rather than
// whichever one happens to be first.

// regionProfileIndex returns the index of the profile whose boundary curves are exactly the
// region's edges, or -1 when none is (so the caller declines rather than extruding a guess).
// Matching is by curve GEOMETRY, which is identity here: both sides describe the same sketch
// curves, one read from the file and one rebuilt from it.
func regionProfileIndex(sk *sketch.Sketch, region []ipt.RegionEdge) int {
	if sk == nil || len(region) == 0 {
		return -1
	}
	want := regionCurveKeys(region)
	profiles := sk.Profiles()
	for i := 0; i < profiles.Count(); i++ {
		if sameCurveSet(profileCurveKeys(profiles.Item(i)), want) {
			return i
		}
	}
	return -1
}

// regionCurveKeys is the sorted geometry keys of the curves the file names as the region boundary.
func regionCurveKeys(region []ipt.RegionEdge) []string {
	out := make([]string, 0, len(region))
	for _, e := range region {
		switch e.Kind {
		case ipt.EdgeLine:
			out = append(out, lineKey(e.Line.A.X, e.Line.A.Y, e.Line.B.X, e.Line.B.Y))
		case ipt.EdgeCircle:
			out = append(out, circleKey(e.Circle.Center.X, e.Circle.Center.Y, e.Circle.Radius))
		case ipt.EdgeArc:
			out = append(out, circleKey(e.Arc.Center.X, e.Arc.Center.Y, e.Arc.Radius))
		}
	}
	sort.Strings(out)
	return out
}

// profileCurveKeys is the sorted geometry keys of every curve bounding a rebuilt profile — its
// outer loop and any holes, since the file lists a region's inner loops alongside its outer one.
func profileCurveKeys(p *sketch.Profile) []string {
	var out []string
	for _, l := range append([]sketch.Loop{p.OuterLoop()}, p.InnerLoops()...) {
		for _, pe := range l.Entities() {
			if k, ok := entityKey(pe.Entity); ok {
				out = append(out, k)
			}
		}
	}
	sort.Strings(out)
	return out
}

// entityKey describes one rebuilt sketch curve the same way regionCurveKeys describes a decoded
// one. An unknown entity kind yields ok=false, which makes its profile unmatchable — the honest
// outcome, since we cannot claim it is the named region.
func entityKey(e sketch.Entity) (string, bool) {
	switch c := e.(type) {
	case *sketch.Line:
		a, b := c.StartPoint().Position(), c.EndPoint().Position()
		return lineKey(float64(a.X), float64(a.Y), float64(b.X), float64(b.Y)), true
	case *sketch.Circle:
		p := c.CenterPoint().Position()
		return circleKey(float64(p.X), float64(p.Y), float64(c.CurveRadius())), true
	case *sketch.Arc:
		p := c.CenterPoint().Position()
		return circleKey(float64(p.X), float64(p.Y), float64(c.CurveRadius())), true
	}
	return "", false
}

// lineKey identifies a segment independently of which end is called the start.
func lineKey(ax, ay, bx, by float64) string {
	p, q := ptKey(ax, ay), ptKey(bx, by)
	if q < p {
		p, q = q, p
	}
	return "L" + p + q
}

func circleKey(cx, cy, r float64) string { return "C" + ptKey(cx, cy) + fmt.Sprintf("|%.6f", r) }

// ptKey quantises to a micron, well below sketch precision but far above float noise between the
// decoded coordinate and the one that made the round trip through the solver.
func ptKey(x, y float64) string { return fmt.Sprintf("|%.6f,%.6f", x, y) }

func sameCurveSet(a, b []string) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
