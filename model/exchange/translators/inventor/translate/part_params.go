// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/feature"
)

// Inventor .ipt part translator — PARAMETER / EXPRESSION MAPPING (M48 #2231 split of part.go).
// User parameters become Oblikovati parameters, and the extrude operation/extent/direction enums map to
// the kernel/feature expressions.

// addParameters emits each decoded user parameter (value in database cm).
func addParameters(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return nil
	}
	var warns []string
	for _, p := range ipt.DecodeParameters(seg) {
		// TODO: decode the parameter's unit kind; length (cm) is assumed for now.
		if _, err := def.Parameters().AddUserParameter(p.Name, fmt.Sprintf("%g cm", p.Value)); err != nil {
			warns = append(warns, fmt.Sprintf("parameter %q: %v", p.Name, err))
		}
	}
	return warns
}

// operationOf maps a decoded Inventor boolean operation to the kernel operation.
func operationOf(op int) ops.PartFeatureOperation {
	switch op {
	case ipt.OpCut:
		return ops.Cut
	case ipt.OpJoin:
		return ops.Join
	case ipt.OpIntersect:
		return ops.Intersect
	default:
		return ops.NewBody
	}
}

// extentOf turns a decoded extrude's termination into the feature engine's extent. A through-all
// extrude carries no distance — it runs until it leaves the material — so it must NOT be built as a
// length: its depth parameter decodes as 0, and a 0-length extrude is a degenerate zero-thickness
// body (that is what made BigChunkyPlate a surface rather than a solid).
func extentOf(ex ipt.Extrude) feature.Extent {
	if ex.ThroughAll {
		return feature.Extent{Type: feature.ThroughAllExtent, Direction: directionOf(ex)}
	}
	// A "To <face>" extrude terminates AT a plane and its Distance is a stale leftover, so it must be
	// built from the target, never as a length. Direction is deliberately left at its zero value:
	// toPlaneSpan derives the span from the SIGNED distance to the target, so which way it runs is
	// already decided by where that target is (see ipt.toTargetPlane).
	if ex.ToPlaneOK {
		if pl, ok := planeOf(ex.ToPlane); ok {
			return feature.Extent{Type: feature.ToFaceExtent, ToPlane: feature.NewFixedWorkPlane(pl)}
		}
	}
	dist := ex.Distance
	e := feature.Extent{
		Type:      feature.DistanceExtent,
		Direction: directionOf(ex),
		Distance:  func() float64 { return dist },
	}
	// A two-sided extrude grows dimLength2 the other way. Its own direction stays PositiveDir:
	// Distance2 IS the negative side, so pairing it with NegativeDir would grow both spans the
	// same way.
	if ex.Distance2 != 0 && !ex.Midplane {
		d2 := ex.Distance2
		e.Direction = feature.PositiveDir
		e.Distance2 = func() float64 { return d2 }
	}
	return e
}

// directionOf maps the extrude's own direction operands onto the extent direction. Midplane wins
// over reversed: straddling the sketch plane is symmetric, so which way it "grows" is moot.
//
// The extent direction is expressed in the SKETCH's frame, not the world's: buildExtrusionShell
// grows the prism along plane.Normal() scaled by the span, so PositiveDir means "along this
// sketch's own normal" wherever that points. Measured on all 517 corpus sketch placements, the
// DirectionAxis is the sketch's own normal in world coordinates (dot = +1.000 on every extrude of
// every part probed), so `dir` adds nothing to the comparison — only `reversed` decides, by
// flipping that vector to run against the normal.
//
// This deliberately replaces `sign(dir.z)` vs world +Z, which was only ever a shortcut valid while
// every sketch was forced onto XY (before ee6ac047 decoded the real planes). That shortcut is
// silently BLIND on any sketch whose normal is perpendicular to Z: dir.z is then 0, so it returns
// PositiveDir and ignores `reversed` entirely. CompressionRollerArmActuatorScrew is built from
// ±X-facing sketches, and its screwdriver-slot cut was placed OUTSIDE the head, removing nothing —
// scoring 1.033x on a no-op (proven: the volume is bit-identical across that feature).
func directionOf(ex ipt.Extrude) feature.ExtentDirection {
	if ex.Midplane {
		return feature.SymmetricDir
	}
	if ex.Reversed {
		return feature.NegativeDir
	}
	return feature.PositiveDir
}
