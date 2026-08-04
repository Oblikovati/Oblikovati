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
	// Runout, when non-nil, carries the exact SETBACK-CLOSE run-out stations (fillet_runout_band.go)
	// the runoutCanalProvider lofts. Same nilable-payload discipline as Canal/Stations: only
	// extractSetbackPatches sets it and only runoutCanalProvider.Fits keys on the pointer, so every
	// other loop is byte-unaffected.
	Runout *RunoutCanal
	// Envelope, when non-nil, names what the rolling ball must stay at radius from — the extractor's
	// own statement of the patch's defining property, which the certificate's interior residual
	// measures against. It is deliberately EXTRACTOR-supplied: coons4-audit.md §B.4 measured a
	// certify-time GUESS at the roll hosts reading 5–19% residual even on OCCT's own CORRECT patches,
	// so a self-derived guess cannot be a gate.
	//
	// It is the SINGLE source of the envelope for every provider — Runout deliberately does NOT carry
	// its own copy. It used to, and certifyRunoutCanalPatch read the tangency host from the copy while
	// the interior residual read this pointer: a producer that populated Runout and forgot Envelope
	// would have made maxBallDev abstain (0) while Valid still passed, silently degrading the
	// certificate to the five boundary/structural fields this slice exists to replace. Now
	// runoutCanalProvider.Build requires this pointer and reads the radius from it too, so a run-out
	// band without an envelope cannot be lofted at all.
	Envelope *BallEnvelope
}

// BallEnvelope names the geometry a constant-radius rolling-ball patch is the envelope of: the
// surfaces the ball stays TANGENT to and the restriction curves it passes THROUGH. A SETBACK-CLOSE
// run-out needs both — its flank bands are tangent to one host plane and pass through the blocking
// boss's footprint conic, which is why a tangent-surfaces-only payload cannot express them.
type BallEnvelope struct {
	// Radius is the rolling-ball radius r; every listed host/curve must sit exactly r from the centre.
	Radius float64
	// Tangents are the surfaces the ball rolls ON (dist(centre, surface) == Radius).
	Tangents []geom.Surface
	// Through are the restriction curves the ball passes THROUGH: the ball's own SECTION plane meets
	// the curve at distance exactly Radius. Note this is NOT dist(centre, curve) == Radius — the ball
	// crosses a restriction curve transversally (only at a symmetry station is it tangent to it), so a
	// plain point-to-curve distance under-reads by up to 3.7% of r even on an exactly-correct band.
	// The section plane is what makes the statement true, which is why Spine is part of this payload.
	Through []geom.Curve3
	// Spine is the unit normal of the ball's SECTION planes — the fillet spine direction. Required
	// whenever Through is non-empty; unused for a pure-tangency envelope.
	Spine math.Vector3
}

// RunoutCanal carries one SETBACK-CLOSE run-out band's EXACT rolling-ball stations — centre plus both
// contacts per spine station, every one closed-form (fillet_runout_envelope.go), so
// geom.LoftCanalStations skins the true envelope instead of a Coons interpolant through the same four
// rails. That substitution is the whole of commit A: coons4-audit.md measured the Coons interior at
// 9–19% of r from OCCT's own surface on nine corpus greens whose RAILS were already right to 1e-14.
type RunoutCanal struct {
	// Centers are the ball centres c(s), one per station, ascending along the spine.
	Centers []math.Point3
	// FeetA / FeetB are the two contacts at each station, each algebraically at Radius from Centers
	// (geom.LoftCanalStations asserts it, so a mis-derived station is declined rather than lofted).
	//
	// The envelope this band is the envelope OF lives in RailLoop.Envelope, never here — see the note
	// on that field for why the pair must not be splittable.
	FeetA, FeetB []math.Point3
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
