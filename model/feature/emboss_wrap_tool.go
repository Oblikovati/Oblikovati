// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Building the wrapped emboss tool from the correspondence in emboss_wrap.go: each profile's
// outline is laid on the face at two radii and skinned between them, giving a pad that follows the
// curvature. The profile's inner loops — a glyph's counters, the hole in an "o" — are cut back out
// of it, exactly as the flat prism path drills them (#1893).

// wrappedEmbossTool merges every profile's wrapped pad into one tool body.
func wrappedEmbossTool(profiles []*sketch.Profile, plane sketch.Plane, fr embossWrapFrame,
	d float64, engrave bool, feat string, rec *diag.Recorder) (*topo.Body, error) {
	inner, outer, err := wrapRadii(fr.cyl.Radius, d, engrave)
	if err != nil {
		return nil, err
	}
	pads := make([]*topo.Body, len(profiles))
	for i, p := range profiles {
		pads[i], err = wrappedProfilePad(p, plane, fr, inner, outer, prismName(feat, i, len(profiles)), rec)
		if err != nil {
			return nil, err
		}
	}
	return mergePrisms(pads, feat), nil
}

// wrapRadii is the pad's inner and outer radius on the wrap face.
//
// Each end is pushed past the face by twice the discretization's SAGITTA — the exact worst-case
// gap between the chorded loop and the true cylinder, R(1−cos(step/2)). That is what makes a raise
// always overlap the material it joins, and a cut always break its skin, instead of leaving the
// hair-thin sliver a chorded loop resting exactly on the nominal radius would. Deriving the pad
// from the sagitta rather than a fixed epsilon keeps it right at any radius.
func wrapRadii(radius, d float64, engrave bool) (inner, outer float64, err error) {
	pad := 2 * radius * (1 - stdmath.Cos(wrapAngularStep/2))
	if !engrave {
		return radius - pad, radius + d, nil
	}
	if d >= radius {
		return 0, 0, fmt.Errorf("emboss: a wrapped engrave of depth %g would cut past the axis of a "+
			"radius-%g face; the depth must stay under the radius", d, radius)
	}
	return radius - d, radius + pad, nil
}

// wrappedProfilePad is one profile's pad with its inner loops cut back out.
func wrappedProfilePad(p *sketch.Profile, plane sketch.Plane, fr embossWrapFrame,
	inner, outer float64, feat string, rec *diag.Recorder) (*topo.Body, error) {
	pad, err := wrappedSkin(p.OuterLoop().Polygon(), plane, fr, inner, outer, feat)
	if err != nil {
		return nil, err
	}
	return drillWrappedHoles(pad, p.InnerLoops(), plane, fr, inner, outer, feat, rec)
}

// drillWrappedHoles cuts each inner loop out of the pad. Every hole tool overshoots BOTH radial
// ends by the pad's own thickness so the cut passes clean through, the radial counterpart of
// drillProfileHoles' axial overshoot.
func drillWrappedHoles(pad *topo.Body, inner []sketch.Loop, plane sketch.Plane, fr embossWrapFrame,
	lo, hi float64, feat string, rec *diag.Recorder) (*topo.Body, error) {
	margin := hi - lo
	for j, loop := range inner {
		hole, err := wrappedSkin(loop.Polygon(), plane, fr, lo-margin, hi+margin,
			fmt.Sprintf("%s/hole%d", feat, j))
		if err != nil {
			return nil, err
		}
		if res, err := ops.BooleanWithDiagnostics(ops.Cut, pad, hole, rec); err == nil && res != nil {
			pad = res
		}
	}
	return pad, nil
}

// wrappedSkin lays one closed loop on the face at two radii and skins between them: the wrapped
// loops become the pad's inner and outer faces, and the swept solid's caps close its walls.
func wrappedSkin(poly []math.Point2, plane sketch.Plane, fr embossWrapFrame,
	inner, outer float64, feat string) (*topo.Body, error) {
	sections := [][]math.Point3{
		wrappedLoop(poly, plane, fr, inner),
		wrappedLoop(poly, plane, fr, outer),
	}
	body, err := sweptSolid(sections, false, feat)
	if err != nil {
		return nil, fmt.Errorf("emboss: wrap %s: %w", feat, err)
	}
	return body, nil
}
