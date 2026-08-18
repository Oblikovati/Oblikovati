// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CORNER SEAM — lap / butt SOLID geometry (#2085, carved from #1964). The gap style cuts a notch on
// a single sheet; the other three styles finish the corner where two flange WALLS meet, so they need
// the two walls modelled as distinct material. That material is exactly what the auto-miter (#1961)
// already builds from each wall's [BendPlacement], so the seam reuses it:
//
//   - no-overlap    → the two walls are carried past the corner and butted (a watertight miter fill);
//   - overlap(p)    → the butt fill PLUS a proud tab that laps the first wall over the second's outer
//                     face by the overlap percent, so the finished solid genuinely gains material;
//   - reverse(p)    → the same tab lapping the OTHER wall, which distinguishes the two on a part
//                     whose walls differ.
//
// A watertight union hides an internal seam (why #1964 deferred this), so the lap is modelled as
// PROUD material — a tab sitting on the lapped wall's outer face — which the union keeps and which is
// therefore measurable and monotone in the percent. Without a bend junction there are no two walls to
// lap, so the feature keeps its honest "unmodelled" report rather than fake a flat-sheet corner.

// seamCornerMatchTol caps how far a selected corner edge may sit from a bend junction and still be
// finished by it. The vertical corner edge stands over the corner but is offset out by the walls'
// stand-offs, so the match is generous; the NEAREST junction wins, which keeps a multi-corner part's
// edges assigned to their own corners.
const seamCornerMatchTol = 3.0

// lapEmbed is how far the proud lap tab reaches back INTO the lapped wall (in thicknesses) so the
// Join welds cleanly instead of meeting a coincident face; the embedded part is inside existing
// material and so adds no volume — only the proud thickness does.
const lapEmbed = 1.0

// seamCorner pairs a selected corner edge with the bend junction beneath it.
type seamCorner struct {
	j    bendJunction
	edge *topo.Edge
}

// finishLapSeam models the butt / overlap / reverse-overlap styles as real solid (#2085). It finds
// the bent corners under the selected edges and finishes each; with no corner it records the
// deferral honestly, because a lap needs two walls a flat sheet does not have.
func (f *SheetMetalCornerSeamFeature) finishLapSeam(in Input, edges []*topo.Edge, gap float64,
	heals []ReferenceHeal) (Output, error) {
	corners := seamCorners(edges, in.PriorBends)
	if len(corners) == 0 {
		in.Diag.Recordf(codeCornerSeamUnmodeled, diag.Warning,
			"corner seam %q has no bent corner to finish: its lap/butt solid needs two flange walls "+
				"meeting at a bend junction; recorded overlap %g%%", f.def.Type, f.def.Overlap)
		return Output{Bodies: in.Bodies, Heals: heals}, nil
	}
	out := in.Bodies
	for i, c := range corners {
		var err error
		if out, err = f.finishOneCorner(out, c, gap, i); err != nil {
			return Output{}, err
		}
	}
	return Output{Bodies: out, Heals: heals}, nil
}

// finishOneCorner butts the two walls (all styles), laps the tab (overlap styles), and cuts the
// seam-root relief (when sized), in that order — the butt closes the corner, the tab adds proud
// material over it, and the relief opens the shared root.
func (f *SheetMetalCornerSeamFeature) finishOneCorner(bodies []*topo.Body, c seamCorner, gap float64,
	i int) ([]*topo.Body, error) {
	feat := seamCornerTag(f.featName, i)
	out, err := mitreFillAtJunction(bodies, c.j, gap, feat)
	if err != nil {
		return nil, err
	}
	if tab := f.lapTabFor(c, feat); tab != nil {
		if out, err = combine(Input{Bodies: out}, tab, ops.Join); err != nil {
			return nil, err
		}
	}
	tool, err := f.seamReliefTool(c, feat)
	if err != nil {
		return nil, err
	}
	if tool != nil {
		if out, err = cutFrom(out, tool); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// lapTabFor builds the proud lap tab for the current style, or nil for no-overlap. Overlap laps the
// first-placed wall (j.a) over the second (j.b); reverse-overlap swaps which wall is lapped, so the
// tab moves to the other face — the one difference between the two styles.
func (f *SheetMetalCornerSeamFeature) lapTabFor(c seamCorner, feat string) *topo.Body {
	var lapped BendPlacement
	switch f.def.Type {
	case OverlapSeam:
		lapped = c.j.b
	case ReverseOverlapSeam:
		lapped = c.j.a
	default: // NoOverlapSeam butts the walls with no lap.
		return nil
	}
	return lapTab(c.j, lapped, c.edge, f.def.Overlap, feat)
}

// seamReliefTool builds the seam-root relief cut when a relief size is given, reusing the corner-
// relief infrastructure (#2072). No size, or a tear shape, means no cut.
func (f *SheetMetalCornerSeamFeature) seamReliefTool(c seamCorner, feat string) (*topo.Body, error) {
	if f.def.ReliefSize == nil {
		return nil, nil
	}
	spec := CornerReliefSpec{Shape: f.def.ReliefShape, Size: evalFloat(f.def.ReliefSize)}
	if !spec.cuts() {
		return nil, nil
	}
	reach, err := cornerReach(c.j, spec)
	if err != nil {
		return nil, err
	}
	return cornerReliefTool(c.j, spec, reach, feat+"/seamRelief")
}

// lapTab is the proud tab that laps one wall over the lapped wall's FLAT outer face: a slab lying on
// that face, extending along the lapped wall's run from the corner by the overlap fraction of its
// stand-off, spanning the vertical face, and proud of the face by one thickness. The embedded root
// (lapEmbed thicknesses inward) welds it to the wall without adding volume, so the finished solid
// gains exactly thickness × lapLength × faceHeight — measurable and monotone in the percent.
//
// The tab must sit on the FLAT face only: the band curves through its bend near the base, so a tab
// carried down to the bend line juts into the concave corner and adds wall-independent bulk that
// hides the lap. The selected corner edge is the flat face's vertical edge, so its span sets where
// the tab starts and how tall it is — clear of the arc.
func lapTab(j bendJunction, lapped BendPlacement, edge *topo.Edge, overlapPct float64,
	feat string) *topo.Body {
	lapLen := overlapPct / 100 * wallStandOff(lapped)
	if lapLen <= 0 {
		return nil
	}
	away, err := awayFromCorner(j.at, lapped)
	if err != nil {
		return nil
	}
	// A horizontal in-face axis (Up × Outward) so the tab plane's Y is exactly Up; the run axis then
	// points toward the wall (runSign) rather than off its free end.
	ref, err := math.UnitVector3FromVector(lapped.Up.Cross(lapped.Outward))
	if err != nil {
		return nil
	}
	runSign := signOf(float64(ref.AsVector().Dot(away.AsVector())))
	start, height := edgeVerticalSpan(edge, j.at, lapped.Up)
	if height <= 0 {
		return nil
	}
	origin := j.at.TranslateBy(lapped.Up.AsVector().Scale(start))
	plane := planeFromFrame(origin, lapped.Outward, ref)
	poly := ensureCCW2([]math.Point2{
		math.P2(0, 0), math.P2(lapLen*runSign, 0), math.P2(lapLen*runSign, height), math.P2(0, height),
	})
	standOff, t := wallStandOff(lapped), lapped.Thickness
	return buildPrism(poly, plane, span{near: standOff - lapEmbed*t, far: standOff + t}, 0, feat+"/lap")
}

// edgeVerticalSpan returns where the corner edge starts above the junction along the wall's up
// direction and how tall it is — the flat vertical face the lap tab lies on.
func edgeVerticalSpan(edge *topo.Edge, at math.Point3, up math.UnitVector3) (start, height float64) {
	h0 := float64(at.VectorTo(edge.StartVertex().Point()).Dot(up.AsVector()))
	h1 := float64(at.VectorTo(edge.EndVertex().Point()).Dot(up.AsVector()))
	if h0 > h1 {
		h0, h1 = h1, h0
	}
	return h0, h1 - h0
}

// signOf is +1 for a non-negative value and -1 otherwise.
func signOf(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}

// seamCornerTag names one corner's tool bodies uniquely within the feature.
func seamCornerTag(feat string, i int) string { return fmt.Sprintf("%s/c%d", feat, i) }

// seamCorners matches each selected corner edge to the bend junction it stands over, finishing each
// junction ONCE: the two walls of a corner share several vertical edges, and all of them resolve to
// the same corner, so the first match wins and the rest are dropped.
func seamCorners(edges []*topo.Edge, prior []BendPlacement) []seamCorner {
	junctions := allBendJunctions(prior)
	out := make([]seamCorner, 0, len(edges))
	seen := map[string]bool{}
	for _, e := range edges {
		j, ok := junctionOnEdge(e, junctions)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%.4f,%.4f,%.4f", j.at.X, j.at.Y, j.at.Z)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, seamCorner{j: j, edge: e})
	}
	return out
}

// allBendJunctions returns every corner among the placed bends: each non-parallel pair whose bend
// lines share an end, with the earlier-placed bend as a. It complements findBendJunction (one bend
// vs the rest) for the corner seam, which owns no bend and must find the corners from the walls.
func allBendJunctions(prior []BendPlacement) []bendJunction {
	var out []bendJunction
	for i := 0; i < len(prior); i++ {
		for k := i + 1; k < len(prior); k++ {
			if parallelBends(prior[i], prior[k]) {
				continue
			}
			if at, ok := sharedEnd(prior[i], prior[k]); ok {
				out = append(out, bendJunction{at: at, a: prior[i], b: prior[k]})
			}
		}
	}
	return out
}

// junctionOnEdge picks the bend junction nearest the corner edge's horizontal footprint: the
// vertical edge stands over the corner, offset out by the walls' stand-offs, so the closest corner
// (within seamCornerMatchTol) is the one this seam finishes.
func junctionOnEdge(e *topo.Edge, junctions []bendJunction) (bendJunction, bool) {
	mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point())
	best, bestD := -1, seamCornerMatchTol
	for i, j := range junctions {
		if d := stdmath.Hypot(float64(j.at.X-mid.X), float64(j.at.Y-mid.Y)); d < bestD {
			bestD, best = d, i
		}
	}
	if best < 0 {
		return bendJunction{}, false
	}
	return junctions[best], true
}
