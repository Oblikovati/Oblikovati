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

// RailSignature classifies a RailLoop's GEOMETRIC FAMILY as recognized by the extractor that built
// it — the ONLY channel an extractor uses to hand a provider-specific hint downstream (ADR-0051 M6):
// a provider's Fits reads Signature, never topo/extractor internals, so providers stay topo-free.
type RailSignature int

const (
	// RailSignatureGeneral is the zero value: no extractor claimed a special family for this loop.
	// Every existing extractor (extractOctantCorner, coons4/tri3's own fixtures, etc.) leaves it
	// unset, so every pre-M6 loop is unaffected by the new plate tier (corpus-neutral by default).
	RailSignatureGeneral RailSignature = iota
	// RailSignatureTangentPlate marks a tangent-degenerate valence-4 corner (N7's family): the
	// extractor confirmed wallFeetSplit's degenerate topology, the plate tier's ONLY recognition
	// signal (kernel/ops/corner_extract_tangent.go). See plateProvider.Fits.
	RailSignatureTangentPlate
)

// RailLoop is the single request type every junction valence (bevel, 3-way,
// n-sided) is expressed as (ADR-0051): an ordered closed cycle of [Side]s bounding
// one fill patch.
type RailLoop struct {
	// Sides are ordered so consecutive sides share an endpoint (see [RailLoop.Closed]).
	Sides []Side
	// Provenance carries the generating tokens for ADR-0043 topological naming.
	// It is identity/history bookkeeping only — a fill provider never reads it.
	Provenance topo.Lineage
	// Signature is the extractor's classification hint (RailSignatureGeneral by default). It is the
	// ONLY channel from extractor to provider (ADR-0051 M6) — geom never reads it, and a provider's
	// Fits may key on it instead of/alongside loop shape.
	Signature RailSignature
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
