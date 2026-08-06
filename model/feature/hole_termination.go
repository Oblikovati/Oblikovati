// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Where a hole STOPS (Oblikovati#1863). A hole could only be blind at a typed depth or through
// everything; anything that bottomed on the part's own geometry had to be authored as a number
// measured by hand, which then stopped following the face it was measured to.
//
// A hole drills along one straight axis, so a termination is just a pair of distances along it: the
// bore starts at `from` and ends at `to`. Every termination reduces to that pair, which is why the
// cutter below is untouched — only the (start, depth) it is handed changes.
//
// A named terminator must be square to the drill axis. A slanted face would bottom the bore on a
// slope, and the cutters build a flat or conical bottom at ONE depth; taking the axis crossing and
// pretending the rest is flat would silently under- or over-cut, so it is refused instead.

// holeBore is the resolved geometry of one bore: where it starts and how far it runs.
type holeBore struct {
	start math.Point3
	depth float64
	entry float64 // how far above start the cutter begins; see boreEntryOverhang
}

// resolveBore applies the definition's termination to a site. The default (and every recipe written
// before #1863) is a plain depth from the placement face, so it returns the site unchanged.
func (h *HoleFeature) resolveBore(centre math.Point3, into math.UnitVector3, depth float64) (holeBore, error) {
	switch h.def.Termination {
	case ToFaceExtent:
		return h.boreToPlane(centre, into)
	case FromToExtent:
		return h.boreFromTo(centre, into)
	default:
		return holeBore{start: centre, depth: depth}, nil
	}
}

// boreToPlane runs the bore from the placement face down to the named terminator.
func (h *HoleFeature) boreToPlane(centre math.Point3, into math.UnitVector3) (holeBore, error) {
	to, err := holeAxisDistance(h.def.ToPlane, centre, into, "to-face")
	if err != nil {
		return holeBore{}, err
	}
	if to <= 0 {
		return holeBore{}, fmt.Errorf("hole: the to-face terminator sits %g cm BEHIND the drill start, so the bore has no depth", -to)
	}
	return holeBore{start: centre, depth: to}, nil
}

// boreFromTo runs the bore between two named terminators, so neither the placement face nor a typed
// depth decides where it begins — the from-face does.
func (h *HoleFeature) boreFromTo(centre math.Point3, into math.UnitVector3) (holeBore, error) {
	from, err := holeAxisDistance(h.def.FromPlane, centre, into, "from-to start")
	if err != nil {
		return holeBore{}, err
	}
	to, err := holeAxisDistance(h.def.ToPlane, centre, into, "from-to end")
	if err != nil {
		return holeBore{}, err
	}
	if to-from <= 0 {
		return holeBore{}, fmt.Errorf("hole: the from-to terminators are %g cm apart along the drill axis, in the wrong order", from-to)
	}
	return holeBore{start: centre.TranslateBy(into.AsVector().Scale(math.Scalar(from))), depth: to - from}, nil
}

// holeAxisDistance is how far along the drill axis a terminator plane sits from the drill start.
// The plane must be square to that axis — see the file comment for why a slanted one is refused.
func holeAxisDistance(target *WorkPlane, centre math.Point3, into math.UnitVector3, what string) (float64, error) {
	if target == nil {
		return 0, fmt.Errorf("hole: %s terminator was not found on the current body", what)
	}
	n := target.Plane().Normal()
	if !n.IsParallelTo(into, math.DefaultTolerance) {
		return 0, fmt.Errorf("hole: the %s terminator is not square to the drill axis; "+
			"a bore bottoms at one depth, so a slanted terminator has no single answer", what)
	}
	axis := into.AsVector()
	return float64(centre.VectorTo(target.Plane().Origin()).Dot(axis)), nil
}

// boreEntryOverhang is how far ABOVE its start the cutter should begin. A bore that starts at a
// surface needs the overhang so its entry face is not coincident with the target's (which would
// leave a zero-thickness sliver); a bore that starts INSIDE material — what a from-to termination
// produces — must begin exactly where it was asked to, since anything above the start is real
// material the hole was never meant to remove.
func boreEntryOverhang(body *topo.Body, start math.Point3) float64 {
	if ops.PointInsideBody(body, start) {
		return 0
	}
	return cutterOverhang
}
