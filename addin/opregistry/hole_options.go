// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"fmt"
	"strings"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The hole's two independent axes and its termination, over the JSON API (Oblikovati#1862, #1863).
//
// Inventor keeps the SEAT (drilled / counterbore / countersink / spotface) and the TAP (none /
// tapped / taper-tapped) on separate axes, so a counterbored tapped hole is an ordinary thing. The
// older args spelled "tapped" as a seat, which made that combination impossible to say; that
// spelling is still accepted and now means "a drilled hole that is also tapped".

// applyHoleOptions records the tap, the clearance sizing and the termination on a built hole.
func applyHoleOptions(part *compdef.PartComponentDefinition, pf *feature.PartFeature,
	in featureargs.Hole) error {
	def, ok := pf.Definition().(*feature.HoleFeature)
	if !ok {
		return fmt.Errorf("hole: %q is not a hole feature", pf.Kind())
	}
	tap, err := holeTapInfo(in)
	if err != nil {
		return err
	}
	d := def.Definition()
	d.Tap, d.Clearance = tap, holeClearanceInfo(in)
	return bindHoleTermination(part, d, in)
}

// holeTapInfo reads the tap axis. The seat spelling "tapped" is the older way of saying the same
// thing about a drilled hole, so it is folded in rather than rejected.
func holeTapInfo(in featureargs.Hole) (feature.HoleTapInfo, error) {
	kind := strings.TrimSpace(in.Tap)
	if kind == "" && strings.EqualFold(strings.TrimSpace(in.Type), "tapped") {
		kind = "tapped"
	}
	switch kind {
	case "", "none":
		return feature.HoleTapInfo{}, nil
	case "tapped", "taperTapped":
		if strings.TrimSpace(in.Designation) == "" {
			return feature.HoleTapInfo{}, fmt.Errorf("hole: tap %q needs a designation, e.g. \"M5x0.8\"", kind)
		}
		return feature.HoleTapInfo{Tapped: true, Designation: in.Designation, Tapered: kind == "taperTapped",
			Class: in.ThreadClass, LeftHanded: in.LeftHanded}, nil
	default:
		return feature.HoleTapInfo{}, fmt.Errorf("hole: unknown tap %q (want none, tapped or taperTapped)", in.Tap)
	}
}

// holeClearanceInfo reads the fastener the bore must pass. The FASTENER is recorded, not the
// diameter it resolves to, so the hole follows the fastener when it is edited.
func holeClearanceInfo(in featureargs.Hole) feature.HoleClearanceInfo {
	if in.Clearance == nil {
		return feature.HoleClearanceInfo{}
	}
	return feature.HoleClearanceInfo{
		Standard: in.Clearance.Standard, Fastener: in.Clearance.Fastener, Fit: in.Clearance.Fit,
	}
}

// bindHoleTermination records where the bore stops and resolves whichever terminator plane(s) it
// needs. "distance" and "through-all" name no geometry, so they bind nothing.
func bindHoleTermination(part *compdef.PartComponentDefinition, def *feature.HoleDefinition,
	in featureargs.Hole) error {
	switch strings.TrimSpace(in.Termination) {
	case "", "distance":
		return nil
	case "through-all":
		def.ThroughAll = true
		return nil
	case "to-face":
		def.Termination = feature.ToFaceExtent
		return bindHoleStop(part, def, in)
	case "from-to":
		def.Termination = feature.FromToExtent
		from, err := extentTargetPlane(part, "hole", "fromFace", in.FromFace, in.FromFaceGeom)
		if err != nil {
			return err
		}
		def.FromPlane = from
		return bindHoleStop(part, def, in)
	default:
		return fmt.Errorf("hole: unknown termination %q (want distance, through-all, to-face or from-to)", in.Termination)
	}
}

// bindHoleStop resolves the terminator the bore bottoms on — the "to" end of both geometric
// terminations, named exactly as an extrude's to-face target is.
func bindHoleStop(part *compdef.PartComponentDefinition, def *feature.HoleDefinition,
	in featureargs.Hole) error {
	stop, err := extentTargetPlane(part, "hole", "toFace", in.ToFace, in.ToFaceGeom)
	def.ToPlane = stop
	return err
}
