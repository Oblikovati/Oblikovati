// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// General curved-crossing imprint (ADR-0058 phase 3). The three per-pair crossing imprints —
// cylinder∩cylinder (crossingCylinderLoops), cone∩cylinder, cone∩cone — traced the SAME thing: the
// shared surface-surface intersection of the two operands' primary curved side surfaces, windowed to
// the base operand's own extent. They differed only in which primitive extractor and band each hard-
// coded. curvedImprintLoops is that one trace, dispatched generally: it pulls each operand's primary
// curved surface, windows the base surface to the base BODY's own box with geom.SurfaceWindow (the same
// full-angle + axial-band the per-pair code hand-built), and marches the loops through the shared,
// diagnostic-carrying imprintTraceLoops. One entry now serves every curved-crossing pair, and grows to
// new pairs (a sphere/torus side) the moment primaryCurvedSurface recognises them — no new imprint.
//
// The per-pair wrappers that used to sit in front of it — each with a type guard (both cones / both
// cylinders / a cone and a cylinder) and its own conditioning gate (the Steinmetz snap ceiling, the
// near-pinch decline) — are deleted with the recognizers they served (ADR-0061 stage 4). What is left is
// this trace: a pair with no primary curved surface declines here, and every pair that has one is
// traced the same way.

// primaryCurvedSurface returns the operand's principal curved side surface — the surface whose SSI with
// the other operand is the boolean imprint. It recognises a bare cone/frustum side and a bare cylinder
// side (the pairs the general curved∩curved pipeline builds today); ok=false for a body with no such
// side, so the caller declines to its fallback. It grows one case per primitive, exactly like
// curvedSolidMembership, keeping the two in step.
func primaryCurvedSurface(b *topo.Body) (geom.Surface, bool) {
	faces := facesOfAny(b)
	if cone, _, _, ok := coneSolidParams(faces); ok {
		return cone, true
	}
	if cyl, _, _, ok := cylinderSolidParams(faces); ok {
		return cyl, true
	}
	return nil, false
}

// curvedImprintLoops traces the shared intersection loops of two operands' primary curved surfaces as
// closed polylines, or ok=false when either operand has no recognised curved side or no loop closes. The
// base surface (operand a's) is windowed to a's own body box — full periodic angle plus a's axial band,
// the exact window the per-pair imprints computed by hand — so the marcher never sweeps beyond the
// operand. Diagnostics (fallback-contour, unclosed-chain) flow through imprintTraceLoops unchanged.
func curvedImprintLoops(a, b *topo.Body, rec *diag.Recorder) ([]geom.Curve3, bool) {
	sa, oka := primaryCurvedSurface(a)
	sb, okb := primaryCurvedSurface(b)
	if !oka || !okb {
		return nil, false
	}
	res := geom.ResolutionForBox(a.RangeBox().Union(b.RangeBox())) // model-relative loop-closure weld (#1399)
	// SurfaceWindowTight (no pad): the window is a's own caps EXACTLY, reproducing the per-primitive imprint
	// band bit-for-bit — the padded window's 5% over-sweep shifts the marched loop and breaks the
	// sampling-sensitive near-pinch weld (#1818, ADR-0058 phase 3).
	loops := imprintTraceLoops(sa, sb, geom.SurfaceWindowTight(sa, a.RangeBox()), res, rec)
	if len(loops) == 0 {
		return nil, false
	}
	return loops, true
}
