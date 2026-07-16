// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"
	"sort"

	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/sketch"
)

// A sketch usually holds several closed regions; the extrude names the loops bounding the one it
// uses (see ipt.ExtrudeRegions). This picks the rebuilt regions that make it up.
//
// The named region is generally NOT one of our profiles. Our kernel decomposes a sketch into
// MINIMAL regions, while Inventor names a face bounded by an outer loop and its holes — and one
// such face can span several minimal regions. The linkage's obround is exactly this: bounded by two
// whole circles and two tangent lines, it comes back as three minimal regions (the rectangle
// between the circles, and an annulus around each bore) whose areas sum to its true volume.
// So the selection is a SET of profiles, extruded together.

// regionProfileIndices returns the profiles composing the named region, or nil when it can't be
// resolved (so the caller declines rather than extruding a guess).
//
// A profile belongs to the region when every curve bounding it is one the region names — it is
// then a piece of the region's interior — EXCEPT a profile that is exactly the inside of a hole,
// which the region cuts away.
func regionProfileIndices(sk *sketch.Sketch, region []ipt.RegionLoop) []int {
	if sk == nil || len(region) == 0 {
		return nil
	}
	named := namedCurves(region)
	holes := holeCurveSets(region)
	profiles := sk.Profiles()
	var out []int
	for i := 0; i < profiles.Count(); i++ {
		p := profiles.Item(i)
		// An OPEN profile is a connected-but-unclosed chain the sketch reports alongside its real
		// regions; it bounds no area, so it is never part of the material an extrude consumes. It
		// must be skipped rather than matched: the feature layer fails the WHOLE extrude when any
		// selected profile is open (resolveClosedProfiles), so one open chain whose curves the
		// region happens to name costs the entire feature its body. ReadWriteHead lost its second
		// extrude that way — sick "profile is open", no body, 0.17x.
		if !p.IsClosed() {
			continue
		}
		keys := profileCurveKeys(p)
		if len(keys) == 0 || !withinNamedCurves(keys, named) || isHoleInterior(keys, holes) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// namedCurves is the set of every curve the region names, outer boundary and holes alike.
func namedCurves(region []ipt.RegionLoop) map[string]bool {
	out := map[string]bool{}
	for _, l := range region {
		for _, k := range loopCurveKeys(l) {
			out[k] = true
		}
	}
	return out
}

// holeCurveSets is, per cut loop, the set of curves bounding it — used to recognise a profile that
// is nothing but the inside of a hole.
func holeCurveSets(region []ipt.RegionLoop) [][]string {
	var out [][]string
	for _, l := range region {
		if l.Cut {
			out = append(out, loopCurveKeys(l))
		}
	}
	return out
}

// withinNamedCurves reports whether every curve bounding a profile is one the region names.
func withinNamedCurves(keys []string, named map[string]bool) bool {
	for _, k := range keys {
		if !named[k] {
			return false
		}
	}
	return true
}

// isHoleInterior reports whether a profile is exactly the inside of one of the region's holes.
func isHoleInterior(keys []string, holes [][]string) bool {
	for _, h := range holes {
		if sameCurveSet(keys, h) {
			return true
		}
	}
	return false
}

// loopCurveKeys is the sorted set of geometry keys of the curves bounding one loop.
func loopCurveKeys(l ipt.RegionLoop) []string {
	out := make([]string, 0, len(l.Edges))
	for _, e := range l.Edges {
		switch e.Kind {
		case ipt.EdgeLine:
			out = append(out, lineKey(e.Line.A.X, e.Line.A.Y, e.Line.B.X, e.Line.B.Y))
		case ipt.EdgeCircle:
			out = append(out, circleKey(e.Circle.Center.X, e.Circle.Center.Y, e.Circle.Radius))
		case ipt.EdgeArc:
			out = append(out, circleKey(e.Arc.Center.X, e.Arc.Center.Y, e.Arc.Radius))
		}
	}
	return uniqueSorted(out)
}

// profileCurveKeys is the sorted set of geometry keys of every curve bounding a rebuilt profile,
// its outer loop and its holes.
func profileCurveKeys(p *sketch.Profile) []string {
	var out []string
	for _, l := range append([]sketch.Loop{p.OuterLoop()}, p.InnerLoops()...) {
		for _, pe := range l.Entities() {
			if k, ok := entityKey(pe.Entity); ok {
				out = append(out, k)
			}
		}
	}
	return uniqueSorted(out)
}

// uniqueSorted sorts the keys and drops repeats: one curve the file names once can bound a rebuilt
// region as several trimmed pieces, so the comparison is set-wise.
func uniqueSorted(keys []string) []string {
	sort.Strings(keys)
	out := keys[:0]
	for i, k := range keys {
		if i == 0 || k != keys[i-1] {
			out = append(out, k)
		}
	}
	return out
}

// entityKey describes one rebuilt sketch curve the same way loopCurveKeys describes a decoded one.
// An arc keys as its full circle, because the region names the whole circle the patch trims it
// from. An unknown entity kind yields ok=false, which makes its profile unmatchable — the honest
// outcome, since we cannot claim it belongs to the named region.
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

// lineKey identifies a segment by its endpoints, independently of which end is called the start.
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
