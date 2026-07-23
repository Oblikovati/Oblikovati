// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Continuity is the required smoothness across a [Side]'s rail, mapped 1:1 onto
// geom.FillSurface's per-side Order (ADR-0051).
type Continuity int

const (
	// G0 is a position-only crease: the fill meets Adjacent along Curve but may
	// kink (tangent planes need not agree).
	G0 Continuity = 0
	// G1 requires the fill's tangent plane to agree with Adjacent along Curve.
	G1 Continuity = 1
)

// Side is one edge of a [RailLoop]: the exact boundary rail a fill surface
// interpolates, plus what continuity it must hold against what.
//
// For a corner patch, Adjacent is the fillet ARM surface (cylinder/cone/torus),
// NOT the host face — the two arms meeting at a shared patch corner already agree
// on the host tangent plane, so pairing against the arm (not the host) is what
// makes the G1 ribbons twist-compatible (ADR-0051; Port Contract 1).
type Side struct {
	// Curve is the EXACT boundary rail the fill interpolates.
	Curve geom.Curve3
	// Adjacent is the surface across this rail — the arm/host geometry itself, not a
	// topo.Face: a fill provider depends only on geom+math (ADR-0051 dependency rule),
	// and NormalAt/DerivativesAt are all it needs to match a ribbon or recognise a
	// known part. The extractor holds the topo.Face and supplies the surface oriented
	// so NormalAt points material-outward; topo identity lives in RailLoop.Provenance.
	// It may be nil for a pure-G0 side, where no tangent-plane agreement is required.
	Adjacent geom.Surface
	// Cont is the continuity required to Adjacent along Curve.
	Cont Continuity
}

// CanalCorner carries what the canal provider needs BEYOND the rails (ADR-C1, M6'
// canal-corner-seam-architecture.md): the surfaces the rolling ball stays tangent to (the spine is
// their ±Radius offset intersection) and the ball radius. nil for EVERY non-canal loop, so the
// canal provider declines and the corpus is unaffected — it mirrors CornerBlendRequest.ObstacleFeature,
// a nilable provider-scoped payload only its extractor sets and only its provider reads.
type CanalCorner struct {
	// Rolls are the roll HOSTS the ball stays tangent to (len 2 for the N7 family: the wall + the
	// non-wall arm's own surface, e.g. wall+s_10 in the OCCT recipe naming — see
	// canal-corner-math.md STEP 2 / blend-sweep-spike-report.md). These are the surfaces the spine's
	// offset-SSI is built from, NOT the Side.Adjacent ARM surfaces the rails already carry —
	// building the spine from the rails/arms missed the oracle area by +75% (blend-sweep-spike-
	// report.md); geom.Surface only (never topo.Face), so the provider stays topo-free.
	Rolls []geom.Surface
	// Radius is the rolling-ball radius r, EXPLICIT: the offset is ±r and reading r off a rational
	// arc rail is fragile, whereas the extractor knows w.radius exactly and for free. Zero for
	// non-canal loops (Canal itself is nil there, so this field is never read).
	Radius float64
	// Ends are the two endpoint ball-centres the offset-SSI spine is trimmed to (the reflected-family
	// centres of the two WALL-sharing arms; N7: C and C″). EXPLICIT for the SAME reason Radius is
	// (ADR-C1): reading them off the rails is fragile — the mid arm's reflected centre is ALSO tangent
	// to both roll hosts, and only the host-offset SIGN distinguishes it from the two true ends, so a
	// rail-centre scan cannot pick the spine ends without re-deriving the topology the extractor
	// already knows. The extractor holds them exactly and for free (centres[wa[0]], centres[wa[1]]).
	Ends [2]math.Point3
}

// RailLoop is the single request type every junction valence (bevel, 3-way,
// n-sided) is expressed as (ADR-0051): an ordered closed cycle of [Side]s bounding
// one fill patch.
type RailLoop struct {
	// Sides are ordered so consecutive sides share an endpoint (see [RailLoop.Closed]).
	Sides []Side
	// Provenance carries the generating tokens for ADR-0043 topological naming.
	// It is identity/history bookkeeping only — a fill provider never reads it.
	Provenance topo.Lineage
	// Canal, when non-nil, carries the canal provider's payload (roll hosts + radius) — nil for
	// every non-canal loop. It is the ONLY channel from extractor to the canal provider (ADR-C1/
	// ADR-C2, M6') — geom never reads it, and canalProvider.Fits keys on the pointer itself.
	Canal *CanalCorner
	// Stations, when non-nil, carries the EXACT rolling-ball cross-section stations (centre + both
	// host feet per z) the canalStationProvider skins into the faithful dual-host CORE panel via
	// geom.LoftCanalStations (U4-4b, #2007 Group C). It is the analogue of Canal for the closed-form
	// station loft: nil for every non-core loop, so the provider declines and the corpus is
	// unaffected — only buildCoreLoop sets it, and canalStationProvider.Fits keys on the pointer.
	Stations *CanalStationFill
}

// Valence returns the number of sides in the loop (a triangle corner is 3, a
// bevel end-cap runout can be 4, etc.).
func (l RailLoop) Valence() int {
	return len(l.Sides)
}

// Closed reports whether the loop is a single ordered cycle: each side's end
// meets the next side's start within tol, wrapping last-to-first. tol is
// model-relative (ADR-0042) — callers pass scale.Weld(), never a bare constant.
// A loop with fewer than 2 sides cannot form a cycle and is never Closed.
func (l RailLoop) Closed(tol float64) bool {
	if len(l.Sides) < 2 {
		return false
	}
	for i, side := range l.Sides {
		next := l.Sides[(i+1)%len(l.Sides)]
		if curveEnd(side.Curve).DistanceTo(curveStart(next.Curve)) > tol {
			return false
		}
	}
	return true
}

// curveStart returns the point at the low end of c's parameter domain.
func curveStart(c geom.Curve3) math.Point3 {
	lo, _ := c.Domain()
	return c.PointAt(lo)
}

// curveEnd returns the point at the high end of c's parameter domain.
func curveEnd(c geom.Curve3) math.Point3 {
	_, hi := c.Domain()
	return c.PointAt(hi)
}
