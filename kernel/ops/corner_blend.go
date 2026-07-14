// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved corner/miter blend engine (design: docs/superpowers/specs/2026-07-13-curved-corner-miter-
// blend-engine-design.md). Where a rolling-ball fillet runs into a junction on NON-planar hosts, the
// swept arms stop short and a corner/miter PATCH must fill the gap. The kernel used to honest-reject
// these (curvedAdjacentError = miter, curvedEndpointError = corner). This file is the SEAM: a request
// value, a certificate, and a tier of interchangeable providers. The analytic-vs-approximation choice
// lives in the ORDER of the tier (analytic-known-part first, bspline-general last), NOT in any caller,
// so assembleBody/orientFilletShell stay agnostic and a family can be promoted to exact in place
// without touching assembly (ADR-1/2). A junction with no valid certificate honest-rejects (ADR-3).

// CornerBlendKind labels which provider produced a patch — provenance/telemetry ONLY. Assembly and
// the topological-naming pass (ADR-0043) MUST NOT read it: lineage keys on the generating tokens, so
// promoting a family from bspline to an exact surface changes geometry, never a name (lineage
// invariance). A stubborn corner-twist rejection under BlendKindBSpline is the signal to add an
// analytic tier for that family.
type CornerBlendKind string

const (
	BlendKindNone    CornerBlendKind = ""                 // no provider claimed / produced the patch
	BlendKindBSpline CornerBlendKind = "bspline-general"  // the universal Coons+certify fallback tier
	BlendKindSphere  CornerBlendKind = "analytic-sphere"  // exact corner sphere (equal-radius trihedral)
	BlendKindCoons4  CornerBlendKind = "coons4-general"   // general 4-sided ribbon-G1 Coons fill over a RailLoop
	BlendKindTri3    CornerBlendKind = "tri3-degenerate4" // 3-sided fill: a degenerate-4 Coons patch (one corner → a pole)
)

// BlendArm is one fillet converging on the junction: its rolling-ball contact path (spine), the
// cross-section arc at its setback-trimmed end (the seam the patch must weld to), the constant blend
// radius, and the two host faces the arm's fillet lies between. The patch must be G1-tangent to this
// arm's surface along EndSection.
type BlendArm struct {
	Spine        geom.Curve3
	EndSection   geom.Curve3
	Radius       float64
	HostL, HostR *topo.Face
}

// CornerBlendRequest is the junction to fill: the converging arms (len 2 ⇒ miter, ≥3 ⇒ corner), the
// host faces meeting there (read-only), and the model-relative setback the arms were trimmed to
// (ADR-0042 — the scale every tolerance is taken relative to).
type CornerBlendRequest struct {
	Junction math.Point3
	Arms     []BlendArm
	Hosts    []*topo.Face
	Setback  Resolution
	// ObstacleFeature, when non-nil, marks this as a MID-SPAN OBSTACLE request (ADR-4): a straight
	// fillet whose planar host face is notched by a through-feature, NOT a junction of arms. Junction
	// requests leave it nil and behave exactly as before. Only the obstacle provider reads it.
	ObstacleFeature *ObstacleFeature
}

// CornerBlendPatch is a provider's output, shaped EXACTLY like what assembleBody already consumes (a
// surface + boundary loops), so the rebuild/orientation layer never learns a provider produced it.
type CornerBlendPatch struct {
	Surface geom.Surface
	Loops   []filletLoop
	Kind    CornerBlendKind
}

// Certificate is a patch's admissibility proof (ADR-3): it is admitted because it PROVES it is sound,
// not because "the code ran". Closed/WeldsArms are structural (watertight loop, every arm spanned);
// NoFold gates the winding/orientation invariant hardened in B2 (S_u×S_v keeps one sign — no self-
// overlap); MaxDev/MaxAngleDev are the measured G0/G1 residuals thresholded by Valid.
type Certificate struct {
	Closed      bool    // the trim loop closes
	WeldsArms   bool    // every arm's EndSection is spanned by a boundary side
	NoFold      bool    // the patch Jacobian keeps a consistent sign over the domain
	MaxDev      float64 // G0 positional max deviation from the boundary curves (model units)
	MaxAngleDev float64 // G1 angular tangent deviation vs the neighbour surfaces (radians)
}

// Valid reports whether the patch is admissible at the junction's model scale: structurally sound AND
// within tolerance — G0 within the model weld (ADR-0042), G1 below seamAngularTol (a G0-tight but
// tangent-kinked patch shades as a crease and is rejected). scale carries the model-relative weld.
func (c Certificate) Valid(scale Resolution) bool {
	return c.Closed && c.WeldsArms && c.NoFold &&
		c.MaxDev <= scale.Weld() && c.MaxAngleDev <= seamAngularTol
}

// CornerBlendProvider produces the corner/miter patch for one junction, or declines. Fits is a cheap
// classification (no heavy math); Build returns the patch plus its certificate, or ok=false to
// decline and let the next tier try. A provider depends only on geom+math — never on assembly.
type CornerBlendProvider interface {
	Name() CornerBlendKind
	Fits(req CornerBlendRequest) bool
	Build(req CornerBlendRequest) (CornerBlendPatch, Certificate, bool)
}

// resolveCornerBlend walks the tiers in priority order and returns the first patch that both builds
// and passes its certificate. No such patch ⇒ ok=false, and the caller honest-rejects (ADR-3). The
// tier ORDER is the whole classifier: an analytic provider promoted ahead of the fallback wins for
// its family simply by being tried first (ADR-2).
func resolveCornerBlend(req CornerBlendRequest, tiers []CornerBlendProvider) (CornerBlendPatch, bool) {
	for _, p := range tiers {
		if !p.Fits(req) {
			continue
		}
		if patch, cert, ok := p.Build(req); ok && cert.Valid(req.Setback) {
			return patch, true
		}
	}
	return CornerBlendPatch{}, false
}
